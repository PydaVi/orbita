package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// Formato real observado nos eventos do Tap (ver docs/architecture-beta0-local.md) —
// dois tipos existem, "identity" e "record"; só nos importa o segundo aqui.
type tapEvent struct {
	Type   string         `json:"type"`
	Record *tapRecordData `json:"record"`
}

type tapRecordData struct {
	DID        string `json:"did"`
	Collection string `json:"collection"`
	Rkey       string `json:"rkey"`
	Action     string `json:"action"`
	CID        string `json:"cid"`
	Record     struct {
		CreatedAt string `json:"createdAt"`
		Work      struct {
			Provider string `json:"provider"`
			ID       string `json:"id"`
		} `json:"work"`
	} `json:"record"`
}

func setupWebhook(mux *http.ServeMux, db *sql.DB) {
	mux.HandleFunc("POST /webhook", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Printf("webhook recebido: %s", body)

		var evt tapEvent
		if err := json.Unmarshal(body, &evt); err != nil {
			// Não falha o webhook por isso — Tap reenviaria pra sempre.
			// Só loga e confirma recebimento (200), como já fazíamos.
			log.Printf("evento não reconhecido, ignorando: %v", err)
			w.WriteHeader(http.StatusOK)
			return
		}

		if evt.Type == "record" && evt.Record != nil &&
			evt.Record.Collection == "social.orbita.shelf.item" &&
			evt.Record.Action == "create" {
			rec := evt.Record
			uri := fmt.Sprintf("at://%s/%s/%s", rec.DID, rec.Collection, rec.Rkey)
			err := insertShelfItem(db, uri, rec.CID, rec.DID, rec.Record.Work.Provider, rec.Record.Work.ID, rec.Record.CreatedAt)
			if err != nil {
				log.Printf("erro ao indexar %s: %v", uri, err)
			} else {
				log.Printf("indexado: %s", uri)
			}
		}

		w.WriteHeader(http.StatusOK)
	})
}
