package main

import (
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
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

	// Recebe os eventos que o Tap entrega depois de filtrar o firehose pela
	// nossa coleção. Só loga por enquanto — indexar num banco é o próximo
	// passo, depois de ver isso chegar de verdade. Tap só marca o evento como
	// entregue (ack) se a resposta for 2xx, então status 200 aqui importa.
	mux.HandleFunc("POST /webhook", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Printf("webhook recebido: %s", body)
		w.WriteHeader(http.StatusOK)
	})

	addr := ":8092" // 8000 já é do api-gateway de comum, rodando na mesma máquina
	log.Printf("orbita appview ouvindo em %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
