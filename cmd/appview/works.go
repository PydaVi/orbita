package main

import (
	"database/sql"
	"fmt"
	"html"
	"net/http"
	"strconv"
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

		// Beta 2: season navigation, TV only — a movie/album/book has
		// nothing to browse into.
		if provider == "tmdb-tv" {
			seasons := displaySeasons(db, provider, id)
			fmt.Fprint(w, "<h2>Seasons</h2><ul>")
			for _, s := range seasons {
				fmt.Fprintf(w, `<li><a href="/works/%s/%s/season/%d">%s</a> (%d episodes)</li>`,
					provider, id, s.Number, html.EscapeString(s.Name), s.EpisodeCount)
			}
			if len(seasons) == 0 {
				fmt.Fprint(w, "<li>could not load seasons</li>")
			}
			fmt.Fprint(w, "</ul>")
		}

		// Beta 2: notes about the work as a whole, not anchored to any
		// episode — season/episode both nil.
		renderNotesSection(w, db, provider, id, nil, nil)
	})

	mux.HandleFunc("GET /works/{provider}/{id}/season/{season}", func(w http.ResponseWriter, r *http.Request) {
		provider := r.PathValue("provider")
		id := r.PathValue("id")
		seasonNum, err := strconv.Atoi(r.PathValue("season"))
		if err != nil {
			http.Error(w, "invalid season number", http.StatusBadRequest)
			return
		}

		title, _ := displayWork(db, provider, id)
		episodes := displayEpisodes(db, provider, id, seasonNum)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!doctype html><p><a href="/works/%s/%s">&laquo; %s</a></p><h1>%s — Season %d</h1><ul>`,
			provider, id, html.EscapeString(title), html.EscapeString(title), seasonNum)
		for _, e := range episodes {
			fmt.Fprintf(w, `<li><a href="/works/%s/%s/season/%d/episode/%d">Episode %d: %s</a> (%s)</li>`,
				provider, id, seasonNum, e.Number, e.Number, html.EscapeString(e.Name), html.EscapeString(e.AirDate))
		}
		if len(episodes) == 0 {
			fmt.Fprint(w, "<li>could not load episodes</li>")
		}
		fmt.Fprint(w, "</ul>")
	})

	mux.HandleFunc("GET /works/{provider}/{id}/season/{season}/episode/{episode}", func(w http.ResponseWriter, r *http.Request) {
		provider := r.PathValue("provider")
		id := r.PathValue("id")
		seasonNum, err := strconv.Atoi(r.PathValue("season"))
		if err != nil {
			http.Error(w, "invalid season number", http.StatusBadRequest)
			return
		}
		episodeNum, err := strconv.Atoi(r.PathValue("episode"))
		if err != nil {
			http.Error(w, "invalid episode number", http.StatusBadRequest)
			return
		}

		title, _ := displayWork(db, provider, id)
		episodes := displayEpisodes(db, provider, id, seasonNum)

		var ep *episode
		for i := range episodes {
			if episodes[i].Number == episodeNum {
				ep = &episodes[i]
				break
			}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!doctype html><p><a href="/works/%s/%s/season/%d">&laquo; %s Season %d</a></p>`,
			provider, id, seasonNum, html.EscapeString(title), seasonNum)
		if ep == nil {
			fmt.Fprint(w, "<p>episode not found</p>")
			return
		}
		fmt.Fprintf(w, "<h1>%s — S%dE%d: %s</h1><p>%s</p><p><small>%s</small></p>",
			html.EscapeString(title), seasonNum, episodeNum, html.EscapeString(ep.Name),
			html.EscapeString(ep.Overview), html.EscapeString(ep.AirDate))

		renderNotesSection(w, db, provider, id, &seasonNum, &episodeNum)
	})
}
