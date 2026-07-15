package main

import (
	"log"
	"net/http"
	"os"
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
		log.Fatal("SESSION_SECRET não definido — gere um com `openssl rand -hex 16`")
	}
	setupOAuth(mux, sessionSecret)
	setupShelf(mux)
	setupWebhook(mux, db)
	setupList(mux, db)

	addr := ":8092" // 8000 já é do api-gateway de comum, rodando na mesma máquina
	log.Printf("orbita appview ouvindo em %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
