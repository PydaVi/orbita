package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// workRef decodes a {provider, id} pair as it appears inside a nook's
// works array — same shape as shelf.item's own work field, just named
// "id" in the wire JSON rather than "workID".
type workRef struct {
	Provider string `json:"provider"`
	ID       string `json:"id"`
}

// Real shape observed in Tap events (see docs/architecture-beta0-local.md) —
// two types exist, "identity" and "record"; only the second one matters here.
type tapEvent struct {
	Type   string         `json:"type"`
	Record *tapRecordData `json:"record"`
}

type tapRecordData struct {
	DID        string      `json:"did"`
	Collection string      `json:"collection"`
	Rkey       string      `json:"rkey"`
	Action     string      `json:"action"`
	CID        string      `json:"cid"`
	Record     recordValue `json:"record"`
}

// recordValue is the wire shape of a record's own value — identical
// whether it arrives as a Tap webhook event's "record" field or as the
// "value" of one entry from com.atproto.repo.listRecords (resync.go uses
// it that way, reading straight from a PDS instead of the firehose). One
// decode shape, one indexing function (indexRecord below), so the webhook
// path and the resync path can never quietly drift apart.
type recordValue struct {
	CreatedAt string `json:"createdAt"`
	Text      string `json:"text"`
	Season    *int   `json:"season"`
	Episode   *int   `json:"episode"`
	Work      struct {
		Provider string `json:"provider"`
		ID       string `json:"id"`
	} `json:"work"`
	Reply *struct {
		Root struct {
			URI string `json:"uri"`
			CID string `json:"cid"`
		} `json:"root"`
		Parent struct {
			URI string `json:"uri"`
			CID string `json:"cid"`
		} `json:"parent"`
	} `json:"reply"`
	Subject struct {
		URI string `json:"uri"`
		CID string `json:"cid"`
	} `json:"subject"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Works       []workRef `json:"works"`
	Order       *int      `json:"order"`
	Style       *struct {
		Theme string `json:"theme"`
	} `json:"style"`
}

// indexRecord writes one record into whichever local table its collection
// belongs to — the single place that decides how a record's value maps to
// a row, called from both the live webhook and resync.go's PDS re-read.
func indexRecord(db *sql.DB, uri, cid, did, collection string, rec recordValue) error {
	switch collection {
	case "social.orbita.shelf.item":
		return insertShelfItem(db, uri, cid, did, rec.Work.Provider, rec.Work.ID, rec.CreatedAt)
	case "social.orbita.note":
		var rootURI, rootCID, parentURI, parentCID *string
		if rec.Reply != nil {
			rootURI, rootCID = &rec.Reply.Root.URI, &rec.Reply.Root.CID
			parentURI, parentCID = &rec.Reply.Parent.URI, &rec.Reply.Parent.CID
		}
		return insertNote(db, uri, cid, did, rec.Work.Provider, rec.Work.ID,
			rec.Season, rec.Episode, rec.Text, rec.CreatedAt,
			rootURI, rootCID, parentURI, parentCID)
	case "social.orbita.repost":
		return insertRepost(db, uri, cid, did, rec.Subject.URI, rec.Subject.CID, rec.CreatedAt)
	case "social.orbita.shelf.nook":
		theme := "default"
		if rec.Style != nil && rec.Style.Theme != "" {
			theme = rec.Style.Theme
		}
		works := make([]WorkRef, 0, len(rec.Works))
		for _, w := range rec.Works {
			works = append(works, WorkRef{Provider: w.Provider, WorkID: w.ID})
		}
		return insertNook(db, uri, cid, did, rec.Name, rec.Description, theme, rec.CreatedAt, rec.Order, works)
	}
	return nil
}

// Beta 2: indexes two collections now, not one. Same "only create,
// nothing else yet" limitation as before — a real gap for delete/update
// on notes, matching the same gap already named for shelf_items.
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

		// Beta 7: a nook's "edit" is a putRecord (action "update"), not a
		// new record — the same collections that only ever handled
		// "create" before now also need to handle nooks being replaced in
		// place. Other collections still only ever arrive as "create" in
		// practice (no edit path exists for them yet), so this is scoped
		// to what actually needs it rather than assuming "update" is
		// meaningful for every collection.
		if evt.Type == "record" && evt.Record != nil && (evt.Record.Action == "create" || evt.Record.Action == "update") {
			rec := evt.Record
			uri := fmt.Sprintf("at://%s/%s/%s", rec.DID, rec.Collection, rec.Rkey)
			if err := indexRecord(db, uri, rec.CID, rec.DID, rec.Collection, rec.Record); err != nil {
				log.Printf("failed to index %s: %v", uri, err)
			} else {
				log.Printf("indexed: %s", uri)
			}
		}

		w.WriteHeader(http.StatusOK)
	})
}
