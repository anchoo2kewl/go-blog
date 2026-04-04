package blog

import (
	"html/template"
	"strings"
)

// Option is a functional option for configuring a Blog.
type Option func(*Blog)

// WithBasePath sets the URL prefix where the blog is mounted. Default: "/blog".
// Do not include a trailing slash.
func WithBasePath(p string) Option {
	return func(b *Blog) { b.basePath = strings.TrimRight(p, "/") }
}

// WithPostsDir sets the directory used by the default FileStore. Ignored if
// WithStore is also given.
func WithPostsDir(dir string) Option {
	return func(b *Blog) {
		if b.store == nil {
			if fs, err := NewFileStore(dir); err == nil {
				b.store = fs
			}
		}
	}
}

// WithStore installs a custom post store (e.g. a SQL-backed implementation).
func WithStore(s Store) Option {
	return func(b *Blog) { b.store = s }
}

// WithRenderer installs a custom markdown renderer. If unset, a go-wiki
// renderer is used.
func WithRenderer(r Renderer) Option {
	return func(b *Blog) { b.renderer = r }
}

// WithSiteTitle sets the blog's site title shown in the header and <title>.
func WithSiteTitle(t string) Option {
	return func(b *Blog) { b.siteTitle = t }
}

// WithSiteTagline sets the blog's tagline / hero subtitle.
func WithSiteTagline(t string) Option {
	return func(b *Blog) { b.siteTagline = t }
}

// WithAuthorName sets the default author name used when a post omits one.
func WithAuthorName(name string) Option {
	return func(b *Blog) { b.authorName = name }
}

// WithAccentColor sets the CSS accent color (used for links, hero accent).
// Accepts any CSS color (e.g. "#2563eb", "rebeccapurple").
func WithAccentColor(c string) Option {
	return func(b *Blog) { b.accentColor = c }
}

// WithDrawBasePath enables go-draw shortcode rendering in posts. The value
// should match the base path where the go-draw handler is mounted.
// Only applied if a custom renderer was NOT provided.
func WithDrawBasePath(p string) Option {
	return func(b *Blog) { b.drawBasePath = strings.TrimRight(p, "/") }
}

// WithFeedURL sets the public absolute URL of the RSS feed (for <link rel>).
// Leave empty to use the relative path.
func WithFeedURL(u string) Option {
	return func(b *Blog) { b.feedURL = u }
}

// WithPerPage sets how many posts to show per listing page.
func WithPerPage(n int) Option {
	return func(b *Blog) {
		if n > 0 {
			b.perPage = n
		}
	}
}

// WithHomeLink adds a "back to main site" link in the blog header nav.
// url is the target (absolute or relative). label is the link text shown.
// Leave unset to omit the link.
//
//	blog.WithHomeLink("/", "← Home")
//	blog.WithHomeLink("https://example.com", "example.com")
func WithHomeLink(url, label string) Option {
	return func(b *Blog) {
		b.homeURL = url
		b.homeLabel = label
	}
}

// WithExtraHead injects raw HTML into <head> of every blog page (analytics, meta).
func WithExtraHead(h template.HTML) Option {
	return func(b *Blog) { b.extraHead = h }
}

// WithExtraFooter injects raw HTML into the footer of every blog page.
func WithExtraFooter(h template.HTML) Option {
	return func(b *Blog) { b.extraFooter = h }
}
