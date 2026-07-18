package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
)

// The nook page is the first server-templated one in this frontend — every
// other page is a static shell that fetches its own JSON client-side. Link
// unfurlers (Slack, iMessage, Bluesky itself) never run JavaScript, so a
// shareable "small manifesto of taste" needs its og:title/og:description/
// og:image burned into the raw HTML response, not filled in after load.
var nookPageTemplate = template.Must(template.ParseFiles("frontend/nook.html"))

type nookPageData struct {
	Title       string
	Description string
	Image       string
	URL         string
}

func setupNookPage(mux *http.ServeMux, db *sql.DB) {
	mux.HandleFunc("GET /profile/{handle}/nook/{rkey}", func(w http.ResponseWriter, r *http.Request) {
		handle := r.PathValue("handle")
		rkey := r.PathValue("rkey")

		did, err := resolveHandleToDID(r.Context(), handle)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		uri := fmt.Sprintf("at://%s/social.orbita.shelf.nook/%s", did, rkey)
		n, err := getNook(db, uri)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		description := n.Description
		if description == "" {
			description = fmt.Sprintf("%d works, curated on Órbita.", len(n.Works))
		}
		var image string
		if len(n.Works) > 0 {
			_, poster, _ := displayWork(db, n.Works[0].Provider, n.Works[0].WorkID)
			image = poster
		}

		scheme := "https"
		if r.TLS == nil && r.Header.Get("X-Forwarded-Proto") != "https" {
			scheme = "http"
		}

		data := nookPageData{
			Title:       fmt.Sprintf("%s — a nook by @%s", n.Name, handle),
			Description: description,
			Image:       image,
			URL:         fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL.Path),
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := nookPageTemplate.Execute(w, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}
