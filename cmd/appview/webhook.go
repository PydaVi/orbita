package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// Real shape observed in Tap events (see docs/architecture-beta0-local.md) —
// two types exist, "identity" and "record"; only the second one matters here.
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
		log.Printf("webhook received: %s", body)

		var evt tapEvent
		if err := json.Unmarshal(body, &evt); err != nil {
			// Don't fail the webhook over this — Tap would keep resending forever.
			// Just log it and confirm receipt (200), like we already did.
			log.Printf("unrecognized event, ignoring: %v", err)
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
				log.Printf("failed to index %s: %v", uri, err)
			} else {
				log.Printf("indexed: %s", uri)
			}
		}

		w.WriteHeader(http.StatusOK)
	})
}
