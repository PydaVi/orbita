package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

// Saved notes are deliberately *not* an AT Protocol record. Every other
// piece of data this appview writes (shelf items, nooks, notes, reposts)
// is meant to be public — that's the entire point of a repo on the
// network. A private bookmark list is the opposite kind of thing: it
// reveals quiet interest a person may not want visible to anyone,
// themselves included in the sense of "not performed" — the same category
// of problem flagged for direct messages (see docs/ROADMAP.md's Beta 14),
// just with an obvious answer here instead of an open one. Real platforms
// that have shipped a "save/bookmark" feature (Bluesky's own included)
// store it privately, server-side, never as a public repo record. So this
// lives only in this appview's own SQLite — a real limitation (it won't
// follow you to a different AppView the way your shelf does), accepted
// deliberately rather than solved by making bookmarks public just to fit
// the federated-record pattern everything else here follows.
const savedNotesSchema = `
CREATE TABLE IF NOT EXISTS saved_notes (
	did        TEXT NOT NULL,
	note_uri   TEXT NOT NULL,
	saved_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
	PRIMARY KEY (did, note_uri)
);
`

func insertSavedNote(db *sql.DB, did, noteURI string) error {
	_, err := db.Exec(
		`INSERT INTO saved_notes (did, note_uri) VALUES (?, ?) ON CONFLICT(did, note_uri) DO NOTHING`,
		did, noteURI,
	)
	return err
}

func deleteSavedNote(db *sql.DB, did, noteURI string) error {
	_, err := db.Exec(`DELETE FROM saved_notes WHERE did = ? AND note_uri = ?`, did, noteURI)
	return err
}

// listSavedNoteURIs is deliberately cheap (just the URIs) — pages that
// only need to know "is this note saved," to render a button's state, use
// this instead of paying for every saved note's full resolved shape.
func listSavedNoteURIs(db *sql.DB, did string) ([]string, error) {
	rows, err := db.Query(`SELECT note_uri FROM saved_notes WHERE did = ? ORDER BY saved_at DESC`, did)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var uris []string
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err != nil {
			return nil, err
		}
		uris = append(uris, u)
	}
	return uris, rows.Err()
}

func setupSaved(mux *http.ServeMux, db *sql.DB) {
	mux.HandleFunc("POST /api/notes/save", func(w http.ResponseWriter, r *http.Request) {
		did, _ := currentSessionDID(r)
		if did == nil {
			http.Error(w, "not authenticated", http.StatusUnauthorized)
			return
		}
		var body struct {
			URI string `json:"uri"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := insertSavedNote(db, did.String(), body.URI); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("POST /api/notes/unsave", func(w http.ResponseWriter, r *http.Request) {
		did, _ := currentSessionDID(r)
		if did == nil {
			http.Error(w, "not authenticated", http.StatusUnauthorized)
			return
		}
		var body struct {
			URI string `json:"uri"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := deleteSavedNote(db, did.String(), body.URI); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("GET /api/saved/uris", func(w http.ResponseWriter, r *http.Request) {
		did, _ := currentSessionDID(r)
		if did == nil {
			http.Error(w, "not authenticated", http.StatusUnauthorized)
			return
		}
		uris, err := listSavedNoteURIs(db, did.String())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(uris)
	})

	mux.HandleFunc("GET /api/saved", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		did, _ := currentSessionDID(r)
		if did == nil {
			http.Error(w, "not authenticated", http.StatusUnauthorized)
			return
		}
		uris, err := listSavedNoteURIs(db, did.String())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		entries := make([]feedNoteEntry, 0, len(uris))
		for _, uri := range uris {
			n, err := getNoteByURI(db, uri)
			if err != nil {
				continue // saved, then deleted at the source since — skipped silently, not an error
			}
			entries = append(entries, buildFeedEntry(ctx, db, *n, "", ""))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entries)
	})
}
