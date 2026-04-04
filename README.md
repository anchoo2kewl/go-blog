# go-blog

Embeddable, file-backed markdown blog for Go web apps — drop a `.md` file, get a new article. Designed to sit alongside [go-wiki](https://github.com/anchoo2kewl/go-wiki) and [go-draw](https://github.com/anchoo2kewl/go-draw).

Default storage is markdown files with YAML frontmatter on disk. A `Store` interface is exposed so a database-backed implementation can be swapped in without touching handlers or templates.

## Features

- **File-backed** — posts are `.md` files with frontmatter; drop one in and it appears. Cache invalidates on mtime.
- **Editorial layout** — hero, lead article card, typographic article page with cover image, meta, tag pills.
- **go-wiki rendering** — full 12-stage markdown pipeline: code blocks, blockquotes, task lists, mermaid, lightbox galleries.
- **go-draw embeds** — `[draw:diagram-id]` shortcodes in posts render as live canvas viewers.
- **RSS 2.0 feed** at `/feed.xml` and **JSON API** at `/api/posts` and `/api/posts/{slug}`.
- **Tag pages** at `/tags/{name}` — generated automatically from frontmatter.
- **Drafts** — set `draft: true` to hide from listings (still accessible by direct URL).
- **Dark mode** — CSS uses `prefers-color-scheme`.
- **Zero frontend build** — CSS embedded via `embed.FS`, no npm.
- **Swappable Store** — plug in SQL, KV, or remote stores.

## Install

```sh
go get github.com/anchoo2kewl/go-blog
```

## Quick start

```go
package main

import (
    "log"
    "net/http"

    blog "github.com/anchoo2kewl/go-blog"
)

func main() {
    b, err := blog.New(
        blog.WithBasePath("/blog"),
        blog.WithPostsDir("./posts"),
        blog.WithSiteTitle("My Blog"),
        blog.WithSiteTagline("Notes on engineering, architecture, agents."),
        blog.WithAuthorName("Jane Doe"),
        blog.WithAccentColor("#7c3aed"),
    )
    if err != nil { log.Fatal(err) }

    http.Handle("/blog/", b.Handler())
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

Create `posts/hello.md`:

```markdown
---
title: "Hello, World"
author: "Jane Doe"
date: 2026-04-04
tags: [intro]
cover: "https://example.com/cover.jpg"
featured: true
---

Welcome to my blog.

## First heading

Some **markdown** content.
```

Visit `http://localhost:8080/blog/` — your article is live.

## Frontmatter reference

| Field | Type | Notes |
|-------|------|-------|
| `title` | string | Shown in header and list. Defaults to filename. |
| `author` | string | Falls back to `WithAuthorName`. |
| `date` | date | `2026-04-04`, RFC3339, or "Jan 2, 2006". |
| `updated` | date | Last edit date (optional). |
| `cover` | URL | Featured image; also used as OG image. |
| `tags` | list | `[a, b, c]` inline or block-style `- a`. |
| `excerpt` | string | Auto-generated if omitted. |
| `slug` | string | Defaults to filename stem. |
| `draft` | bool | Hides from listings. |
| `featured` | bool | Pins to top of feed as lead article. |

## Routes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Article list (paginated) |
| GET | `/{slug}` | Single article |
| GET | `/tags/{tag}` | Articles tagged with `{tag}` |
| GET | `/feed.xml` | RSS 2.0 feed |
| GET | `/api/posts` | JSON list of posts (metadata only) |
| GET | `/api/posts/{slug}` | Single post as JSON (with rendered HTML) |
| GET | `/static/blog.css` | Theme stylesheet |

Paths are relative to the configured base path (default `/blog`).

## Options

```go
blog.New(
    blog.WithBasePath("/blog"),              // URL prefix (default "/blog")
    blog.WithPostsDir("./posts"),            // file store root
    blog.WithStore(myStore),                 // custom Store implementation
    blog.WithRenderer(myRenderer),           // custom markdown renderer
    blog.WithSiteTitle("My Blog"),
    blog.WithSiteTagline("Notes on things"),
    blog.WithAuthorName("Jane Doe"),         // default author
    blog.WithAccentColor("#2563eb"),         // CSS accent
    blog.WithDrawBasePath("/draw"),          // enable [draw:id] shortcodes
    blog.WithFeedURL("https://..."),         // absolute RSS URL
    blog.WithPerPage(10),                    // posts per listing page
    blog.WithExtraHead(template.HTML("...")),// inject into <head>
    blog.WithExtraFooter(template.HTML("...")),
)
```

## Integrating with go-draw

Mount a go-draw handler anywhere, then tell go-blog where it lives:

```go
d, _ := godraw.New(godraw.WithBasePath("/draw"))
http.Handle("/draw/", d.Handler())

b, _ := blog.New(
    blog.WithBasePath("/blog"),
    blog.WithDrawBasePath("/draw"),
)
http.Handle("/blog/", b.Handler())
```

Now any post containing `[draw:my-diagram]` renders an embedded, resizable go-draw viewport. Use `[draw:my-diagram:edit]` for the editor.

## Custom Store

Implement the `Store` interface:

```go
type Store interface {
    List() ([]*Post, error)
    Get(slug string) (*Post, error)
    Save(p *Post) error
    Delete(slug string) error
}
```

Pass it via `blog.WithStore(myStore)`. The default `FileStore` lives in `store.go` as a reference.

## Custom renderer

By default, go-blog uses [go-wiki](https://github.com/anchoo2kewl/go-wiki) for markdown. Swap in any renderer:

```go
type Renderer interface {
    RenderContent(md string) string
}
```

```go
blog.New(blog.WithRenderer(myRenderer))
```

## Package layout

```
go-blog/
├── blog.go            Top-level Blog struct, New(), Handler()
├── options.go         Functional options
├── handler.go         HTTP routes (list, post, tag, feed, api)
├── post.go            Post struct + frontmatter parser
├── store.go           Store interface + FileStore
├── embed.go           //go:embed templates/static
├── templates/
│   ├── layout.gohtml
│   ├── list.gohtml
│   └── post.gohtml
├── static/
│   └── blog.css
└── _examples/
    └── standalone/
        ├── main.go
        └── posts/*.md
```

## License

MIT
