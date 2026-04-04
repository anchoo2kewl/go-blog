package blog

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// Post is a single blog entry. Fields populated from YAML-ish frontmatter at
// the top of the markdown file, plus computed values (slug, excerpt, reading
// time).
type Post struct {
	Slug        string    // URL slug (derived from filename if not set)
	Title       string    // Required. Falls back to slug if empty.
	Author      string    // Author display name.
	Date        time.Time // Publication date.
	Updated     time.Time // Last-edit date (optional).
	Cover       string    // Cover / featured image URL.
	Tags        []string  // Tags shown as pills.
	Excerpt     string    // Short summary (auto-extracted if empty).
	Draft       bool      // Hidden from listings when true.
	Featured    bool      // Pinned as lead article when true.
	Content     string    // Raw markdown body.
	ReadingMins int       // Estimated reading time in minutes.
}

// URL returns the URL to this post under basePath.
func (p *Post) URL(basePath string) string {
	return strings.TrimRight(basePath, "/") + "/" + p.Slug
}

// DisplayDate renders the date as "Jan 2, 2006".
func (p *Post) DisplayDate() string {
	if p.Date.IsZero() {
		return ""
	}
	return p.Date.Format("Jan 2, 2006")
}

// ISO8601 renders the date as RFC3339 for <time datetime="...">.
func (p *Post) ISO8601() string {
	if p.Date.IsZero() {
		return ""
	}
	return p.Date.Format(time.RFC3339)
}

// ParsePost parses the raw file bytes of a markdown file with YAML frontmatter
// and returns a Post. defaultSlug is used if frontmatter has no explicit slug.
func ParsePost(raw []byte, defaultSlug string) (*Post, error) {
	body, meta, err := splitFrontmatter(raw)
	if err != nil {
		return nil, err
	}
	p := &Post{Content: body}
	for k, v := range meta {
		switch strings.ToLower(k) {
		case "title":
			p.Title = scalar(v)
		case "author":
			p.Author = scalar(v)
		case "slug":
			p.Slug = scalar(v)
		case "cover", "cover_image", "featured_image":
			p.Cover = scalar(v)
		case "excerpt", "description", "summary":
			p.Excerpt = scalar(v)
		case "draft":
			p.Draft = truthy(scalar(v))
		case "featured":
			p.Featured = truthy(scalar(v))
		case "date", "published", "publication_date":
			p.Date = parseDate(scalar(v))
		case "updated", "modified", "last_edit":
			p.Updated = parseDate(scalar(v))
		case "tags", "categories":
			p.Tags = parseList(v)
		}
	}
	if p.Slug == "" {
		p.Slug = defaultSlug
	}
	if p.Title == "" {
		p.Title = titleFromSlug(p.Slug)
	}
	if p.Excerpt == "" {
		p.Excerpt = autoExcerpt(body, 240)
	}
	p.ReadingMins = readingTime(body)
	return p, nil
}

// splitFrontmatter separates a leading YAML-ish frontmatter block (delimited
// by --- lines) from the markdown body. Returns (body, metaMap, error).
// metaMap values are raw strings (possibly multi-line for lists).
func splitFrontmatter(raw []byte) (string, map[string]string, error) {
	s := string(raw)
	// Trim a leading BOM if present.
	s = strings.TrimPrefix(s, "\ufeff")
	if !strings.HasPrefix(s, "---") {
		// No frontmatter — whole file is body.
		return strings.TrimLeft(s, "\r\n"), map[string]string{}, nil
	}
	// Find end of opening fence line.
	nl := strings.IndexByte(s, '\n')
	if nl < 0 {
		return s, map[string]string{}, nil
	}
	rest := s[nl+1:]
	// Find closing "---" line.
	end := findClosingFence(rest)
	if end < 0 {
		// No closing fence — treat whole file as body.
		return strings.TrimLeft(s, "\r\n"), map[string]string{}, nil
	}
	fm := rest[:end]
	body := rest[end:]
	// Skip the "---\n" closing fence.
	if i := strings.IndexByte(body, '\n'); i >= 0 {
		body = body[i+1:]
	}
	meta, err := parseFrontmatter(fm)
	if err != nil {
		return "", nil, err
	}
	return strings.TrimLeft(body, "\r\n"), meta, nil
}

func findClosingFence(s string) int {
	// Scan line-by-line; return byte offset of a line equal to "---" or "...".
	i := 0
	for i < len(s) {
		nl := strings.IndexByte(s[i:], '\n')
		var line string
		if nl < 0 {
			line = s[i:]
			if isFence(line) {
				return i
			}
			return -1
		}
		line = s[i : i+nl]
		if isFence(line) {
			return i
		}
		i += nl + 1
	}
	return -1
}

func isFence(line string) bool {
	line = strings.TrimRight(line, "\r")
	return line == "---" || line == "..."
}

// parseFrontmatter handles a tiny subset of YAML sufficient for blog
// frontmatter: scalar strings, numbers, booleans, inline lists `[a, b]`, and
// block lists (next lines beginning with "- "). No nested maps.
func parseFrontmatter(fm string) (map[string]string, error) {
	out := map[string]string{}
	sc := bufio.NewScanner(strings.NewReader(fm))
	// Increase buffer for long lines.
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var curKey string
	var listBuf []string
	flush := func() {
		if curKey != "" && len(listBuf) > 0 {
			out[curKey] = "[" + strings.Join(listBuf, ", ") + "]"
		}
		curKey = ""
		listBuf = nil
	}
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		// Block list continuation: "  - value" under a key with empty value.
		trimmed := strings.TrimLeft(line, " \t")
		if curKey != "" && strings.HasPrefix(trimmed, "- ") {
			listBuf = append(listBuf, strings.TrimSpace(trimmed[2:]))
			continue
		}
		// New key.
		flush()
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			continue
		}
		key := strings.TrimSpace(line[:colon])
		val := strings.TrimSpace(line[colon+1:])
		if key == "" {
			continue
		}
		if val == "" {
			curKey = key
			continue
		}
		out[key] = val
	}
	flush()
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("frontmatter scan: %w", err)
	}
	return out, nil
}

// scalar strips surrounding quotes from a scalar value.
func scalar(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 {
		if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
			return v[1 : len(v)-1]
		}
	}
	return v
}

func truthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "yes", "on", "1":
		return true
	}
	return false
}

// parseList accepts inline `[a, b, "c d"]` or already-normalized `[a, b]`.
func parseList(v string) []string {
	v = strings.TrimSpace(v)
	if strings.HasPrefix(v, "[") && strings.HasSuffix(v, "]") {
		inner := v[1 : len(v)-1]
		parts := splitCSV(inner)
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = scalar(strings.TrimSpace(p))
			if p != "" {
				out = append(out, p)
			}
		}
		return out
	}
	// Fallback: comma separated scalar.
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = scalar(strings.TrimSpace(p))
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// splitCSV splits a comma-separated list, honoring double and single quotes.
func splitCSV(s string) []string {
	var parts []string
	var cur strings.Builder
	quote := byte(0)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote != 0 {
			cur.WriteByte(c)
			if c == quote {
				quote = 0
			}
			continue
		}
		switch c {
		case '"', '\'':
			quote = c
			cur.WriteByte(c)
		case ',':
			parts = append(parts, cur.String())
			cur.Reset()
		default:
			cur.WriteByte(c)
		}
	}
	if cur.Len() > 0 || len(parts) > 0 {
		parts = append(parts, cur.String())
	}
	return parts
}

var dateLayouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	"2006-01-02",
	"2006/01/02",
	"Jan 2, 2006",
	"January 2, 2006",
	"2 Jan 2006",
}

func parseDate(v string) time.Time {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}
	}
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, v); err == nil {
			return t
		}
	}
	return time.Time{}
}

var stripMD = regexp.MustCompile("(?s)```.*?```|`[^`]*`|!\\[[^\\]]*\\]\\([^)]*\\)|\\[([^\\]]*)\\]\\([^)]*\\)|^#+\\s*|[*_~>\\[\\]]|<[^>]+>")

func autoExcerpt(body string, max int) string {
	// Take text up to the first blank line, then strip markdown syntax.
	cut := body
	if i := strings.Index(body, "\n\n"); i > 0 {
		cut = body[:i]
	}
	cut = stripMD.ReplaceAllStringFunc(cut, func(m string) string {
		// For [text](url) keep the text group if present.
		if strings.HasPrefix(m, "[") && !strings.HasPrefix(m, "![") {
			if closer := strings.Index(m, "]("); closer > 0 {
				return m[1:closer]
			}
		}
		return " "
	})
	cut = strings.TrimSpace(collapseWS(cut))
	if len(cut) <= max {
		return cut
	}
	// Cut on a word boundary.
	if i := strings.LastIndexByte(cut[:max], ' '); i > 0 {
		return strings.TrimRight(cut[:i], ".,;:") + "…"
	}
	return cut[:max] + "…"
}

func collapseWS(s string) string {
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteRune(' ')
			}
			prevSpace = true
			continue
		}
		prevSpace = false
		b.WriteRune(r)
	}
	return b.String()
}

// readingTime estimates minutes at ~225 wpm, minimum 1.
func readingTime(body string) int {
	words := 0
	inWord := false
	for _, r := range body {
		if unicode.IsSpace(r) {
			if inWord {
				words++
			}
			inWord = false
			continue
		}
		inWord = true
	}
	if inWord {
		words++
	}
	mins := words / 225
	if mins < 1 {
		mins = 1
	}
	return mins
}

func titleFromSlug(slug string) string {
	parts := strings.FieldsFunc(slug, func(r rune) bool { return r == '-' || r == '_' })
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}
