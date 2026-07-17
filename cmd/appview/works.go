package main

import "net/http"

// The work page and its season/episode sub-pages all serve the same static
// shell now — provider/id/season/episode are read from the URL client-side
// by frontend/app.js, which fetches the matching /api/works/... endpoint
// for the actual data. No server-side path parsing needed here anymore.
func setupWorks(mux *http.ServeMux) {
	mux.HandleFunc("GET /works/{provider}/{id}", serveWorkPage)
	mux.HandleFunc("GET /works/{provider}/{id}/season/{season}", serveWorkPage)
	mux.HandleFunc("GET /works/{provider}/{id}/season/{season}/episode/{episode}", serveWorkPage)
}

func serveWorkPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "frontend/work.html")
}
