package blog

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// routes builds the blog's http.Handler.
func (b *Blog) routes() http.Handler {
	mux := http.NewServeMux()

	// Static asset handler (CSS, etc.).
	sub, _ := fs.Sub(staticFS, "static")
	assetPrefix := b.basePath + "/static/"
	mux.Handle(assetPrefix, http.StripPrefix(assetPrefix, http.FileServer(http.FS(sub))))

	// Core routes under basePath.
	mux.HandleFunc(b.basePath+"/", func(w http.ResponseWriter, r *http.Request) {
		// Exact list page.
		if r.URL.Path == b.basePath+"/" || r.URL.Path == b.basePath {
			b.handleList(w, r)
			return
		}
		rest := strings.TrimPrefix(r.URL.Path, b.basePath+"/")
		switch {
		case rest == "feed.xml" || rest == "rss.xml":
			b.handleFeed(w, r)
		case rest == "api/posts":
			b.handleAPIList(w, r)
		case strings.HasPrefix(rest, "api/posts/"):
			b.handleAPIGet(w, r, strings.TrimPrefix(rest, "api/posts/"))
		case strings.HasPrefix(rest, "tags/"):
			b.handleTag(w, r, strings.TrimPrefix(rest, "tags/"))
		default:
			b.handlePost(w, r, rest)
		}
	})

	return mux
}

type pageData struct {
	Site       siteData
	Posts      []postView
	Post       *postView
	Tag        string
	Page       int
	HasPrev    bool
	HasNext    bool
	PrevURL    string
	NextURL    string
	AllTags    []tagCount
	ExtraHead  template.HTML
	ExtraFoot  template.HTML
	Year       int
}

type siteData struct {
	Title    string
	Tagline  string
	BasePath string
	Accent   string
	FeedURL  string
	Author   string
}

type postView struct {
	*Post
	HTML   template.HTML
	URL    string
	IsLead bool
}

type tagCount struct {
	Name  string
	Count int
	URL   string
}

func (b *Blog) baseSite() siteData {
	return siteData{
		Title:    b.siteTitle,
		Tagline:  b.siteTagline,
		BasePath: b.basePath,
		Accent:   b.accentColor,
		FeedURL:  b.feedOrDefault(),
		Author:   b.authorName,
	}
}

func (b *Blog) feedOrDefault() string {
	if b.feedURL != "" {
		return b.feedURL
	}
	return b.basePath + "/feed.xml"
}

func (b *Blog) toView(p *Post, lead bool) postView {
	return postView{
		Post:   p,
		HTML:   template.HTML(b.renderer.RenderContent(p.Content)),
		URL:    p.URL(b.basePath),
		IsLead: lead,
	}
}

// handleList renders the main list page, optionally filtered by tag.
func (b *Blog) handleList(w http.ResponseWriter, r *http.Request) {
	posts, err := b.store.List()
	if err != nil {
		http.Error(w, "load posts: "+err.Error(), http.StatusInternalServerError)
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	start := (page - 1) * b.perPage
	if start > len(posts) {
		start = len(posts)
	}
	end := start + b.perPage
	if end > len(posts) {
		end = len(posts)
	}
	slice := posts[start:end]
	views := make([]postView, len(slice))
	for i, p := range slice {
		views[i] = b.toView(p, i == 0 && page == 1)
	}
	data := pageData{
		Site:      b.baseSite(),
		Posts:     views,
		Page:      page,
		HasPrev:   page > 1,
		HasNext:   end < len(posts),
		AllTags:   b.tagCounts(posts),
		ExtraHead: b.extraHead,
		ExtraFoot: b.extraFooter,
		Year:      time.Now().Year(),
	}
	if data.HasPrev {
		data.PrevURL = fmt.Sprintf("%s/?page=%d", b.basePath, page-1)
	}
	if data.HasNext {
		data.NextURL = fmt.Sprintf("%s/?page=%d", b.basePath, page+1)
	}
	b.render(w, "list.gohtml", data)
}

// handleTag renders posts filtered by a single tag.
func (b *Blog) handleTag(w http.ResponseWriter, r *http.Request, tag string) {
	tag = strings.TrimSuffix(tag, "/")
	if tag == "" {
		http.Redirect(w, r, b.basePath+"/", http.StatusSeeOther)
		return
	}
	all, err := b.store.List()
	if err != nil {
		http.Error(w, "load posts: "+err.Error(), http.StatusInternalServerError)
		return
	}
	var filtered []*Post
	for _, p := range all {
		for _, t := range p.Tags {
			if strings.EqualFold(t, tag) {
				filtered = append(filtered, p)
				break
			}
		}
	}
	views := make([]postView, len(filtered))
	for i, p := range filtered {
		views[i] = b.toView(p, false)
	}
	b.render(w, "list.gohtml", pageData{
		Site:      b.baseSite(),
		Posts:     views,
		Tag:       tag,
		AllTags:   b.tagCounts(all),
		ExtraHead: b.extraHead,
		ExtraFoot: b.extraFooter,
		Year:      time.Now().Year(),
	})
}

// handlePost renders a single post page.
func (b *Blog) handlePost(w http.ResponseWriter, r *http.Request, slug string) {
	slug = strings.Trim(slug, "/")
	if slug == "" {
		http.Redirect(w, r, b.basePath+"/", http.StatusSeeOther)
		return
	}
	p, err := b.store.Get(slug)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "load post: "+err.Error(), http.StatusInternalServerError)
		return
	}
	view := b.toView(p, false)
	all, _ := b.store.List()
	b.render(w, "post.gohtml", pageData{
		Site:      b.baseSite(),
		Post:      &view,
		AllTags:   b.tagCounts(all),
		ExtraHead: b.extraHead,
		ExtraFoot: b.extraFooter,
		Year:      time.Now().Year(),
	})
}

// handleAPIList returns all posts as JSON (no rendered HTML).
func (b *Blog) handleAPIList(w http.ResponseWriter, r *http.Request) {
	posts, err := b.store.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type item struct {
		Slug        string    `json:"slug"`
		Title       string    `json:"title"`
		Author      string    `json:"author,omitempty"`
		Date        time.Time `json:"date"`
		Cover       string    `json:"cover,omitempty"`
		Tags        []string  `json:"tags,omitempty"`
		Excerpt     string    `json:"excerpt,omitempty"`
		ReadingMins int       `json:"reading_mins"`
		URL         string    `json:"url"`
		Featured    bool      `json:"featured,omitempty"`
	}
	out := make([]item, len(posts))
	for i, p := range posts {
		out[i] = item{
			Slug: p.Slug, Title: p.Title, Author: p.Author, Date: p.Date,
			Cover: p.Cover, Tags: p.Tags, Excerpt: p.Excerpt,
			ReadingMins: p.ReadingMins, URL: p.URL(b.basePath), Featured: p.Featured,
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"posts": out})
}

// handleAPIGet returns a single rendered post as JSON.
func (b *Blog) handleAPIGet(w http.ResponseWriter, r *http.Request, slug string) {
	slug = strings.Trim(slug, "/")
	p, err := b.store.Get(slug)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"slug":         p.Slug,
		"title":        p.Title,
		"author":       p.Author,
		"date":         p.Date,
		"updated":      p.Updated,
		"cover":        p.Cover,
		"tags":         p.Tags,
		"excerpt":      p.Excerpt,
		"reading_mins": p.ReadingMins,
		"content_md":   p.Content,
		"content_html": b.renderer.RenderContent(p.Content),
	})
}

// handleFeed serves an RSS 2.0 feed.
func (b *Blog) handleFeed(w http.ResponseWriter, r *http.Request) {
	posts, err := b.store.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type feedItem struct {
		XMLName     xml.Name `xml:"item"`
		Title       string   `xml:"title"`
		Link        string   `xml:"link"`
		GUID        string   `xml:"guid"`
		PubDate     string   `xml:"pubDate"`
		Description string   `xml:"description"`
	}
	type channel struct {
		XMLName xml.Name   `xml:"channel"`
		Title   string     `xml:"title"`
		Link    string     `xml:"link"`
		Desc    string     `xml:"description"`
		Items   []feedItem `xml:"item"`
	}
	type rss struct {
		XMLName xml.Name `xml:"rss"`
		Version string   `xml:"version,attr"`
		Channel channel  `xml:"channel"`
	}
	base := absURL(r, b.basePath)
	items := make([]feedItem, 0, len(posts))
	for _, p := range posts {
		link := base + "/" + p.Slug
		items = append(items, feedItem{
			Title:       p.Title,
			Link:        link,
			GUID:        link,
			PubDate:     p.Date.Format(time.RFC1123Z),
			Description: p.Excerpt,
		})
	}
	f := rss{Version: "2.0", Channel: channel{
		Title: b.siteTitle, Link: base, Desc: b.siteTagline, Items: items,
	}}
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	_, _ = w.Write([]byte(xml.Header))
	_ = xml.NewEncoder(w).Encode(f)
}

func absURL(r *http.Request, path string) string {
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	host := r.Host
	if h := r.Header.Get("X-Forwarded-Host"); h != "" {
		host = h
	}
	return scheme + "://" + host + path
}

func (b *Blog) tagCounts(posts []*Post) []tagCount {
	m := map[string]int{}
	for _, p := range posts {
		for _, t := range p.Tags {
			m[t]++
		}
	}
	out := make([]tagCount, 0, len(m))
	for name, c := range m {
		out = append(out, tagCount{
			Name:  name,
			Count: c,
			URL:   b.basePath + "/tags/" + name,
		})
	}
	// Stable sort by count desc, then name.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0; j-- {
			if out[j].Count > out[j-1].Count ||
				(out[j].Count == out[j-1].Count && out[j].Name < out[j-1].Name) {
				out[j], out[j-1] = out[j-1], out[j]
			} else {
				break
			}
		}
	}
	return out
}

func (b *Blog) render(w http.ResponseWriter, name string, data pageData) {
	t := templateFor(name)
	if t == nil {
		http.Error(w, "unknown template: "+name, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "template: "+err.Error(), http.StatusInternalServerError)
	}
}
