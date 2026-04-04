package blog

import (
	"embed"
	"encoding/json"
	"html/template"
	"strings"
)

//go:embed templates/*.gohtml
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

var tmplFuncs = template.FuncMap{
	"html": func(s string) template.HTML { return template.HTML(s) },
	// json returns a JSON-encoded value suitable for embedding directly in a
	// <script type="application/ld+json"> block or similar. Returns
	// template.JS so the double-quoted string is emitted verbatim.
	"json": func(v any) template.JS {
		b, err := json.Marshal(v)
		if err != nil {
			return template.JS(`""`)
		}
		return template.JS(b)
	},
	"joinCSV": func(xs []string) string { return strings.Join(xs, ", ") },
}

// Each page is its own template set, parsed with layout + that page's file.
// This avoids name collisions on blocks (title, content, meta) across pages.
var (
	listTmpl = template.Must(template.New("list.gohtml").Funcs(tmplFuncs).
			ParseFS(templateFS, "templates/layout.gohtml", "templates/list.gohtml"))
	postTmpl = template.Must(template.New("post.gohtml").Funcs(tmplFuncs).
			ParseFS(templateFS, "templates/layout.gohtml", "templates/post.gohtml"))
)

func templateFor(name string) *template.Template {
	switch name {
	case "list.gohtml":
		return listTmpl
	case "post.gohtml":
		return postTmpl
	}
	return nil
}
