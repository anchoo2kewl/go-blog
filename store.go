package blog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ErrNotFound is returned when a post cannot be located.
var ErrNotFound = errors.New("go-blog: post not found")

// Store is the interface a post backend must implement. A file-backed default
// is provided; callers can plug in SQL, KV, or remote stores.
type Store interface {
	List() ([]*Post, error)
	Get(slug string) (*Post, error)
	Save(p *Post) error
	Delete(slug string) error
}

// FileStore reads posts from a directory of .md files with YAML frontmatter.
// Files are cached in memory and invalidated on mtime changes.
type FileStore struct {
	dir string
	mu  sync.RWMutex
	// cache keyed by slug
	cache map[string]cachedPost
	// filename index: slug -> filepath
	index map[string]string
}

type cachedPost struct {
	post  *Post
	mtime time.Time
}

// NewFileStore creates a FileStore rooted at dir, creating the directory if
// needed.
func NewFileStore(dir string) (*FileStore, error) {
	if dir == "" {
		return nil, fmt.Errorf("go-blog: posts dir is required")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("go-blog: create posts dir: %w", err)
	}
	return &FileStore{
		dir:   dir,
		cache: map[string]cachedPost{},
		index: map[string]string{},
	}, nil
}

// Dir returns the root directory.
func (f *FileStore) Dir() string { return f.dir }

// List returns all non-draft posts, newest first.
func (f *FileStore) List() ([]*Post, error) {
	if err := f.refresh(); err != nil {
		return nil, err
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	out := make([]*Post, 0, len(f.cache))
	for _, c := range f.cache {
		if c.post.Draft {
			continue
		}
		out = append(out, c.post)
	}
	sort.Slice(out, func(i, j int) bool {
		// Featured posts sort first, then by date desc, then by title.
		if out[i].Featured != out[j].Featured {
			return out[i].Featured
		}
		if !out[i].Date.Equal(out[j].Date) {
			return out[i].Date.After(out[j].Date)
		}
		return out[i].Title < out[j].Title
	})
	return out, nil
}

// Get returns a single post by slug (including drafts).
func (f *FileStore) Get(slug string) (*Post, error) {
	if err := f.refresh(); err != nil {
		return nil, err
	}
	f.mu.RLock()
	c, ok := f.cache[slug]
	f.mu.RUnlock()
	if !ok {
		return nil, ErrNotFound
	}
	return c.post, nil
}

// Save writes a post to disk as {slug}.md.
func (f *FileStore) Save(p *Post) error {
	if p.Slug == "" {
		return fmt.Errorf("go-blog: post slug required")
	}
	path := filepath.Join(f.dir, p.Slug+".md")
	data := serialize(p)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("go-blog: write post: %w", err)
	}
	f.mu.Lock()
	f.cache[p.Slug] = cachedPost{post: p, mtime: time.Now()}
	f.index[p.Slug] = path
	f.mu.Unlock()
	return nil
}

// Delete removes a post file.
func (f *FileStore) Delete(slug string) error {
	f.mu.Lock()
	path, ok := f.index[slug]
	f.mu.Unlock()
	if !ok {
		// Try to find it on disk.
		path = filepath.Join(f.dir, slug+".md")
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("go-blog: delete post: %w", err)
	}
	f.mu.Lock()
	delete(f.cache, slug)
	delete(f.index, slug)
	f.mu.Unlock()
	return nil
}

// refresh scans the directory, loading new/changed files and evicting deleted ones.
func (f *FileStore) refresh() error {
	entries, err := os.ReadDir(f.dir)
	if err != nil {
		return fmt.Errorf("go-blog: read posts dir: %w", err)
	}
	seen := make(map[string]struct{}, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".md") || strings.HasPrefix(name, ".") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		path := filepath.Join(f.dir, name)
		defaultSlug := strings.TrimSuffix(name, filepath.Ext(name))
		defaultSlug = slugify(defaultSlug)

		f.mu.RLock()
		// Look up cache by any existing slug that points at this path.
		var existing cachedPost
		var existingSlug string
		for slug, p := range f.index {
			if p == path {
				existingSlug = slug
				existing = f.cache[slug]
				break
			}
		}
		f.mu.RUnlock()

		if existingSlug != "" && existing.mtime.Equal(info.ModTime()) {
			seen[existingSlug] = struct{}{}
			continue
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		post, err := ParsePost(raw, defaultSlug)
		if err != nil {
			continue
		}
		f.mu.Lock()
		// If slug changed, evict the old entry.
		if existingSlug != "" && existingSlug != post.Slug {
			delete(f.cache, existingSlug)
			delete(f.index, existingSlug)
		}
		f.cache[post.Slug] = cachedPost{post: post, mtime: info.ModTime()}
		f.index[post.Slug] = path
		f.mu.Unlock()
		seen[post.Slug] = struct{}{}
	}
	// Evict any slugs whose files have vanished.
	f.mu.Lock()
	for slug := range f.cache {
		if _, ok := seen[slug]; !ok {
			delete(f.cache, slug)
			delete(f.index, slug)
		}
	}
	f.mu.Unlock()
	return nil
}

// slugify normalizes a filename stem into a URL-safe slug.
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevDash = false
		case r == '-' || r == '_' || r == ' ' || r == '.':
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.TrimRight(b.String(), "-")
}

// serialize writes a Post back to markdown + YAML frontmatter.
func serialize(p *Post) []byte {
	var b strings.Builder
	b.WriteString("---\n")
	if p.Title != "" {
		fmt.Fprintf(&b, "title: %q\n", p.Title)
	}
	if p.Author != "" {
		fmt.Fprintf(&b, "author: %q\n", p.Author)
	}
	if !p.Date.IsZero() {
		fmt.Fprintf(&b, "date: %s\n", p.Date.Format("2006-01-02"))
	}
	if !p.Updated.IsZero() {
		fmt.Fprintf(&b, "updated: %s\n", p.Updated.Format("2006-01-02"))
	}
	if p.Cover != "" {
		fmt.Fprintf(&b, "cover: %q\n", p.Cover)
	}
	if p.Excerpt != "" {
		fmt.Fprintf(&b, "excerpt: %q\n", p.Excerpt)
	}
	if len(p.Tags) > 0 {
		quoted := make([]string, len(p.Tags))
		for i, t := range p.Tags {
			quoted[i] = fmt.Sprintf("%q", t)
		}
		fmt.Fprintf(&b, "tags: [%s]\n", strings.Join(quoted, ", "))
	}
	if p.Draft {
		b.WriteString("draft: true\n")
	}
	if p.Featured {
		b.WriteString("featured: true\n")
	}
	b.WriteString("---\n\n")
	b.WriteString(p.Content)
	if !strings.HasSuffix(p.Content, "\n") {
		b.WriteByte('\n')
	}
	return []byte(b.String())
}
