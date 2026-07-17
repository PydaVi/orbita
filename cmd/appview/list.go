package main

import "net/http"

// Beta 0's "prove the webhook indexed it" debug page — Beta 3/4 left it as
// raw HTML on purpose (never a real product surface), but shelf is now one
// of the product's four core surfaces, so it gets the same treatment as
// everything else: GET /api/shelf backs a real page now.
func setupList(mux *http.ServeMux) {
	mux.HandleFunc("GET /shelf", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "frontend/shelf.html")
	})
}
