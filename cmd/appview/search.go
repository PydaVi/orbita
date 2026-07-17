package main

import "net/http"

// Beta 1, item 1: search before writing — the only thing that guarantees
// the {provider, id} pair written to the PDS is one TMDB actually
// recognizes (see the "Titanic" incident in docs/BETA1-PLAN.md, where a
// raw text id got written straight to a real PDS record with nothing
// stopping it). Beta 3: converted to the same JSON-API-plus-static-shell
// shape as the work page — GET /api/search does the querying now.
func setupSearch(mux *http.ServeMux) {
	mux.HandleFunc("GET /search", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "frontend/search.html")
	})
}
