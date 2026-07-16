package main

import (
	"fmt"
	"html"
	"net/http"
)

// Beta 1, item 1: search before writing. The point isn't just UX — it's
// the only thing that guarantees the {provider, id} pair written to the
// PDS is one TMDB actually recognizes (see the "Titanic" incident in
// docs/BETA1-PLAN.md, where a raw text id got written straight to a
// real PDS record with nothing stopping it).
func setupSearch(mux *http.ServeMux) {
	mux.HandleFunc("GET /search", func(w http.ResponseWriter, r *http.Request) {
		did, _ := currentSessionDID(r)
		if did == nil {
			http.Error(w, "not authenticated — sign in at /oauth/login first", http.StatusUnauthorized)
			return
		}

		query := r.URL.Query().Get("q")

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!doctype html>
<form method="GET" action="/search">
  <input name="q" placeholder="search a title..." value="%s">
  <button type="submit">Search</button>
</form>`, html.EscapeString(query))

		if query == "" {
			return
		}

		results, err := searchTMDB(query)
		if err != nil {
			fmt.Fprintf(w, "<p>search failed: %s</p>", html.EscapeString(err.Error()))
			return
		}

		fmt.Fprint(w, "<ul>")
		for _, res := range results {
			img := ""
			if res.PosterURL != "" {
				img = fmt.Sprintf(`<img src="%s" height="120"><br>`, html.EscapeString(res.PosterURL))
			}
			fmt.Fprintf(w, `<li>%s%s (%s)
				<form method="POST" action="/shelf/add" style="display:inline">
				  <input type="hidden" name="provider" value="%s">
				  <input type="hidden" name="id" value="%s">
				  <button type="submit">Add to shelf</button>
				</form></li>`,
				img, html.EscapeString(res.Title), html.EscapeString(res.Year),
				res.Provider, html.EscapeString(res.ID))
		}
		if len(results) == 0 {
			fmt.Fprint(w, "<li>no results</li>")
		}
		fmt.Fprint(w, "</ul>")
	})
}
