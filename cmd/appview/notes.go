package main

import "database/sql"

// Beta 2's write path originally lived here as a classic form POST
// (/notes/add); Beta 3 replaced it with the JSON endpoint in api.go
// (/api/notes/add), used by the new frontend. This file is left with just
// the storage/read side, shared by both the webhook indexer and the API.
const notesSchema = `
CREATE TABLE IF NOT EXISTS notes (
	uri        TEXT PRIMARY KEY,
	cid        TEXT NOT NULL,
	did        TEXT NOT NULL,
	provider   TEXT NOT NULL,
	work_id    TEXT NOT NULL,
	season     INTEGER,
	episode    INTEGER,
	text       TEXT NOT NULL,
	created_at TEXT NOT NULL,
	indexed_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
`

type Note struct {
	URI       string
	DID       string
	Text      string
	CreatedAt string
}

func insertNote(db *sql.DB, uri, cid, did, provider, workID string, season, episode *int, text, createdAt string) error {
	_, err := db.Exec(
		`INSERT INTO notes (uri, cid, did, provider, work_id, season, episode, text, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(uri) DO NOTHING`,
		uri, cid, did, provider, workID, season, episode, text, createdAt,
	)
	return err
}

// listNotes covers both cases with the same query shape: season == nil
// means a whole-work note (matches rows where season/episode are both
// NULL), season != nil means a specific episode (matches an exact pair).
// Season-only notes (a note about a season as a whole, no episode) are
// deliberately not wired up yet — the Lexicon already allows it, the UI
// just doesn't ask for it, same "don't build what wasn't asked for" bar
// as everything else here.
func listNotes(db *sql.DB, provider, workID string, season, episode *int) ([]Note, error) {
	var rows *sql.Rows
	var err error
	if season == nil {
		rows, err = db.Query(
			`SELECT uri, did, text, created_at FROM notes
			 WHERE provider = ? AND work_id = ? AND season IS NULL AND episode IS NULL
			 ORDER BY created_at ASC`,
			provider, workID)
	} else {
		rows, err = db.Query(
			`SELECT uri, did, text, created_at FROM notes
			 WHERE provider = ? AND work_id = ? AND season = ? AND episode = ?
			 ORDER BY created_at ASC`,
			provider, workID, *season, *episode)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.URI, &n.DID, &n.Text, &n.CreatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}
