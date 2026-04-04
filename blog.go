// Package blog provides an embeddable, file-backed markdown blog for Go web
// applications. It follows the same conventions as go-wiki and go-draw: assets
// are embedded via Go's embed package, zero npm/build step, and a single
// http.Handler is returned for mounting on any mux.
//
// By default, blog posts are stored as markdown files with YAML frontmatter on
// disk — drop a new .md file in the posts directory and it appears as a new
// article. A Store interface is exposed so a database-backed implementation
// can be swapped in without touching handlers or templates.
//
// Quick start:
//
//	b, err := blog.New(
//	    blog.WithBasePath("/blog"),
//	    blog.WithPostsDir("./posts"),
//	    blog.WithSiteTitle("My Blog"),
//	)
//	if err != nil { log.Fatal(err) }
//	http.Handle("/blog/", b.Handler())
//
// Then drop posts/hello.md with a frontmatter block and visit /blog/.
package blog

import (
	"fmt"
	"html/template"
	"net/http"

	gowiki "github.com/anchoo2kewl/go-wiki"
)

// Renderer converts markdown content to HTML.
type Renderer interface {
	RenderContent(md string) string
}

// Blog is the top-level object. Create one with New() and mount its Handler.
type Blog struct {
	store        Store
	renderer     Renderer
	basePath     string
	siteTitle    string
	siteTagline  string
	authorName   string
	accentColor  string
	drawBasePath string
	feedURL      string
	perPage      int
	homeURL      string
	homeLabel    string
	siteURL      string
	extraHead    template.HTML
	extraFooter  template.HTML
}

// New creates a Blog with the provided options.
// If no WithStore is given, a FileStore rooted at ./posts is used.
// If no WithRenderer is given, a go-wiki renderer is installed with sensible
// defaults (plus go-draw embeds if WithDrawBasePath was set).
func New(opts ...Option) (*Blog, error) {
	b := &Blog{
		basePath:    "/blog",
		siteTitle:   "Blog",
		siteTagline: "Writing, notes, and deep dives.",
		perPage:     10,
	}
	for _, o := range opts {
		o(b)
	}
	if b.store == nil {
		fs, err := NewFileStore("./posts")
		if err != nil {
			return nil, fmt.Errorf("go-blog: default file store: %w", err)
		}
		b.store = fs
	}
	if b.renderer == nil {
		var wopts []gowiki.Option
		if b.drawBasePath != "" {
			wopts = append(wopts, gowiki.WithDrawBasePath(b.drawBasePath))
		}
		b.renderer = gowiki.New(wopts...)
	}
	return b, nil
}

// Handler returns an http.Handler that serves all blog routes. Mount with a
// trailing-slash prefix matching the base path:
//
//	http.Handle("/blog/", b.Handler())
func (b *Blog) Handler() http.Handler {
	return b.routes()
}

// Store returns the underlying post store so callers can list/get/save/delete
// programmatically (e.g. for admin UIs or migration jobs).
func (b *Blog) Store() Store { return b.store }

// BasePath returns the configured base URL path (no trailing slash).
func (b *Blog) BasePath() string { return b.basePath }
