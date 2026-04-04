---
title: "Writing Posts"
author: "Anshuman Biswas"
date: 2026-04-02
tags: [guide]
excerpt: "How frontmatter, drafts, and tags work in go-blog."
---

Every post is a markdown file with a frontmatter block at the top. Only `title` is strictly required — everything else has sensible defaults.

## Supported frontmatter fields

| Field | Type | Notes |
|-------|------|-------|
| `title` | string | Shown in the header and list. |
| `author` | string | Falls back to `WithAuthorName`. |
| `date` | date | YYYY-MM-DD, RFC3339, or "Jan 2, 2006". |
| `cover` | URL | Featured image. |
| `tags` | list | `[a, b]` inline or `- a` / `- b` block. |
| `excerpt` | string | Auto-generated if omitted. |
| `draft` | bool | Hides the post from listings. |
| `featured` | bool | Pins to top of the feed. |

## Drafts

Set `draft: true` to hide a post from listings and feeds. It's still accessible at its URL, so you can share a preview link.

## Slugs

The slug defaults to the filename without extension. Override with `slug: my-custom-slug` in frontmatter.
