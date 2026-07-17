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

	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		log.Fatal("SESSION_SECRET not set — generate one with `openssl rand -hex 16`")
	}
	setupOAuth(mux, sessionSecret)
	setupShelf(mux, db)
	setupWebhook(mux, db)
	setupList(mux, db)
	setupWorks(mux, db)
	setupSearch(mux)
	setupNotes(mux, db)

	addr := ":8092" // 8000 is already taken by comum's api-gateway, running on the same machine
	log.Printf("orbita appview listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
