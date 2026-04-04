// Example: standalone go-blog server.
//
//	cd _examples/standalone && go run main.go
//	Visit: http://localhost:8080/blog/
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
		blog.WithSiteTitle("Demo Blog"),
		blog.WithSiteTagline("A file-backed markdown blog for Go web apps."),
		blog.WithAuthorName("Anshuman Biswas"),
		blog.WithAccentColor("#2563eb"),
		// Enable this once a go-draw handler is mounted at /draw/:
		// blog.WithDrawBasePath("/draw"),
	)
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/blog/", b.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/blog/", http.StatusSeeOther)
			return
		}
		http.NotFound(w, r)
	})

	log.Println("Listening on :8080  ->  http://localhost:8080/blog/")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
