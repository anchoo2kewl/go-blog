---
title: "Hello, World"
author: "Anshuman Biswas"
date: 2026-04-04
tags: [intro, meta]
featured: true
cover: "https://images.unsplash.com/photo-1499750310107-5fef28a66643?w=1200&q=80"
excerpt: "Welcome to go-blog — a file-backed markdown blog for Go web apps."
---

Welcome to **go-blog**, a small library that turns a directory of markdown files into a real blog. Drop a `.md` file in the posts directory with a frontmatter block and it appears as a new article — no database, no CMS, no build step.

## Why another blog thing?

Because most blog engines assume too much. `go-blog` does the minimum:

- Read `.md` files from a directory
- Parse YAML-ish frontmatter (title, author, date, cover, tags)
- Render them with the full `go-wiki` markdown pipeline
- Serve editorial-style list and article pages
- Expose a Store interface so the database-backed path is a drop-in swap

## What you get

1. **File watching** — edit a post, refresh the browser. Cache invalidates on mtime changes.
2. **go-draw embeds** — drop `[draw:my-diagram]` into a post and render a live canvas viewer.
3. **RSS + JSON API** — `/blog/feed.xml` and `/blog/api/posts` come free.
4. **Tag pages** — each tag gets a `/blog/tags/{name}` listing.

## A code block

```go
b, _ := blog.New(
    blog.WithBasePath("/blog"),
    blog.WithPostsDir("./posts"),
    blog.WithSiteTitle("My Blog"),
)
http.Handle("/blog/", b.Handler())
```

That's the entire integration. Have fun.
