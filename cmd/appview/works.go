package main

import (
	"database/sql"
	"fmt"
	"html"
	"net/http"
)

// The Beta 1 aggregation page: "who has this specific work," grouped by
// work instead of listed by event — the opposite of GET /shelf, which is
// "everything, everyone." This is the definitional AppView query: it only
// makes sense once data from more than one independent account exists.
func setupWorks(mux *http.ServeMux, db *sql.DB) {
	mux.HandleFunc("GET /works/{provider}/{id}", func(w http.ResponseWriter, r *http.Request) {
		provider := r.PathValue("provider")
		id := r.PathValue("id")

		items, err := listShelfItemsByWork(db, provider, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		title, poster := displayWork(db, provider, id)
		img := ""
		if poster != "" {
			img = fmt.Sprintf(`<img src="%s" height="180"><br>`, html.EscapeString(poster))
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<!doctype html>%s<h1>%s</h1><p>%d account(s) have this on their shelf:</p><ul>",
			img, html.EscapeString(title), len(items))
		for _, it := range items {
			fmt.Fprintf(w, "<li>%s (added %s)</li>", it.DID, it.CreatedAt)
		}
		if len(items) == 0 {
			fmt.Fprint(w, "<li>nobody has this yet</li>")
		}
		fmt.Fprint(w, "</ul>")
	})
}
