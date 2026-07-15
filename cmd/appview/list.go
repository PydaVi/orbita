package main

import (
	"database/sql"
	"fmt"
	"net/http"
)

// Sole purpose: prove that what the webhook indexed can be read back.
// No styling, no pagination — this is the minimal "Beta 0 done" criterion
// (docs/BETA0-PLAN.md), not a product UI.
func setupList(mux *http.ServeMux, db *sql.DB) {
	mux.HandleFunc("GET /shelf", func(w http.ResponseWriter, r *http.Request) {
		items, err := listShelfItems(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, "<!doctype html><h1>Synced shelf</h1><ul>")
		for _, it := range items {
			fmt.Fprintf(w, "<li><b>%s/%s</b> — %s (indexed %s)<br><small>%s</small></li>",
				it.Provider, it.WorkID, it.DID, it.IndexedAt, it.URI)
		}
		if len(items) == 0 {
			fmt.Fprint(w, "<li>nothing synced yet</li>")
		}
		fmt.Fprint(w, "</ul>")
	})
}
