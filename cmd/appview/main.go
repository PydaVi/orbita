package main

import (
	"log"
	"net/http"
	"os"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	db, err := openDB("orbita.db")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Beta 3/4: static frontend assets, served by the same binary — no
	// separate nginx-style service, no build step. Registered as specific
	// patterns (not a "/" catch-all) so they can't shadow API/OAuth routes.
	// serveFrontend keeps this list from growing one hand-written closure
	// per file as pages keep being added (feed, profile, and whatever
	// comes after).
	serveFrontend := func(urlPath, file string) {
		mux.HandleFunc("GET "+urlPath, func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "frontend/"+file)
		})
	}
	mux.Handle("GET /fonts/", http.StripPrefix("/fonts/", http.FileServer(http.Dir("frontend/fonts"))))
	serveFrontend("/styles.css", "styles.css")
	serveFrontend("/common.js", "common.js")
	serveFrontend("/app.js", "app.js")
	serveFrontend("/search.js", "search.js")
	serveFrontend("/feed.js", "feed.js")
	serveFrontend("/profile.js", "profile.js")
	serveFrontend("/shelf.js", "shelf.js")
	serveFrontend("/nook.js", "nook.js")
	serveFrontend("/profile-shelf.js", "profile-shelf.js")

	// Beta 4: the basic site layout — Feed and Profile exist as real pages
	// (with a real nav to reach them). Beta 5: /profile/{handle} shows any
	// account's profile, not just the viewer's own.
	serveFrontend("/feed", "feed.html")
	serveFrontend("/profile", "profile.html")
	mux.HandleFunc("GET /profile/{handle}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "frontend/profile.html")
	})
	// Beta 7 (post-launch): /profile/{handle} only shows a summary (a
	// couple of nook cards) — this is the complete counterpart, every nook
	// in full plus the whole Unsorted grid, one click away via "see full
	// shelf" rather than the default every profile pays for.
	mux.HandleFunc("GET /profile/{handle}/shelf", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "frontend/profile-shelf.html")
	})

	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		log.Fatal("SESSION_SECRET not set — generate one with `openssl rand -hex 16`")
	}
	setupOAuth(mux, sessionSecret)
	setupShelf(mux, db)
	setupWebhook(mux, db)
	setupList(mux)
	setupWorks(mux)
	setupSearch(mux)
	setupNookPage(mux, db)
	setupResync(mux, db)
	setupAPI(mux, db)

	addr := ":8092" // 8000 is already taken by comum's api-gateway, running on the same machine
	log.Printf("orbita appview listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
