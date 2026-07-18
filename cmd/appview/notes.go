package main

import (
	"database/sql"
	"fmt"
	"strings"
)

// Beta 2's write path originally lived here as a classic form POST
// (/notes/add); Beta 3 replaced it with the JSON endpoint in api.go
// (/api/notes/add), used by the new frontend. This file is left with just
// the storage/read side, shared by both the webhook indexer and the API.
//
// Beta 6 (revised): notes can reply to other notes — root/parent, the same
// shape AT Protocol already uses everywhere (app.bsky.feed.post's own
// reply field, and this project's earlier product work, ADR-011). A note
// with reply_parent_uri set is a reply, not a root-level note — the
// various "notes about a work/account" queries below all exclude replies,
// which surface only nested under their parent (see listReplies).
const notesSchema = `
CREATE TABLE IF NOT EXISTS notes (
	uri             TEXT PRIMARY KEY,
	cid             TEXT NOT NULL,
	did             TEXT NOT NULL,
	provider        TEXT NOT NULL,
	work_id         TEXT NOT NULL,
	season          INTEGER,
	episode         INTEGER,
	text            TEXT NOT NULL,
	created_at      TEXT NOT NULL,
	reply_root_uri   TEXT,
	reply_root_cid   TEXT,
	reply_parent_uri TEXT,
	reply_parent_cid TEXT,
	indexed_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
`

type Note struct {
	URI       string
	CID       string
	DID       string
	Text      string
	CreatedAt string
}

func insertNote(db *sql.DB, uri, cid, did, provider, workID string, season, episode *int, text, createdAt string, replyRootURI, replyRootCID, replyParentURI, replyParentCID *string) error {
	_, err := db.Exec(
		`INSERT INTO notes (uri, cid, did, provider, work_id, season, episode, text, created_at,
		                     reply_root_uri, reply_root_cid, reply_parent_uri, reply_parent_cid)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(uri) DO NOTHING`,
		uri, cid, did, provider, workID, season, episode, text, createdAt,
		replyRootURI, replyRootCID, replyParentURI, replyParentCID,
	)
	return err
}

// listNotes covers both cases with the same query shape: season == nil
// means a whole-work note (matches rows where season/episode are both
// NULL), season != nil means a specific episode (matches an exact pair).
// Season-only notes (a note about a season as a whole, no episode) are
// deliberately not wired up yet — the Lexicon already allows it, the UI
// just doesn't ask for it, same "don't build what wasn't asked for" bar
// as everything else here. Replies are excluded — they surface nested
// under their parent (see listReplies), not as their own top-level row.
func listNotes(db *sql.DB, provider, workID string, season, episode *int) ([]Note, error) {
	var rows *sql.Rows
	var err error
	if season == nil {
		rows, err = db.Query(
			`SELECT uri, cid, did, text, created_at FROM notes
			 WHERE provider = ? AND work_id = ? AND season IS NULL AND episode IS NULL
			   AND reply_parent_uri IS NULL
			 ORDER BY created_at ASC`,
			provider, workID)
	} else {
		rows, err = db.Query(
			`SELECT uri, cid, did, text, created_at FROM notes
			 WHERE provider = ? AND work_id = ? AND season = ? AND episode = ?
			   AND reply_parent_uri IS NULL
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
		if err := rows.Scan(&n.URI, &n.CID, &n.DID, &n.Text, &n.CreatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

// listReplies returns the direct replies to one note, chronological
// (oldest first, reading like a conversation) — only one level deep for
// now. A reply to a reply is stored correctly (its own parent points at
// the reply, its root points at the original), it just isn't fetched
// recursively here yet.
func listReplies(db *sql.DB, parentURI string) ([]Note, error) {
	rows, err := db.Query(
		`SELECT uri, cid, did, text, created_at FROM notes
		 WHERE reply_parent_uri = ? ORDER BY created_at ASC`,
		parentURI)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.URI, &n.CID, &n.DID, &n.Text, &n.CreatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

// noteRootRef looks up what a reply's root should be when replying to
// uri: if uri is itself a reply, its own root is reused (so a whole
// thread shares one root, matching AT Protocol convention); otherwise uri
// is a root-level note and becomes the root itself.
func noteRootRef(db *sql.DB, uri string) (cid, rootURI, rootCID string, err error) {
	var replyRootURI, replyRootCID sql.NullString
	row := db.QueryRow(`SELECT cid, reply_root_uri, reply_root_cid FROM notes WHERE uri = ?`, uri)
	if err := row.Scan(&cid, &replyRootURI, &replyRootCID); err != nil {
		return "", "", "", err
	}
	if replyRootURI.Valid && replyRootURI.String != "" {
		return cid, replyRootURI.String, replyRootCID.String, nil
	}
	return cid, uri, cid, nil
}

// getNoteByURI resolves a repost's subject back to the actual note —
// reposts only ever store the URI/CID they point at, not a copy of the
// content itself.
func getNoteByURI(db *sql.DB, uri string) (*FeedNote, error) {
	var n FeedNote
	row := db.QueryRow(
		`SELECT uri, cid, did, provider, work_id, season, episode, text, created_at FROM notes WHERE uri = ?`,
		uri)
	if err := row.Scan(&n.URI, &n.CID, &n.DID, &n.Provider, &n.WorkID, &n.Season, &n.Episode, &n.Text, &n.CreatedAt); err != nil {
		return nil, err
	}
	return &n, nil
}

// FeedNote backs the feed: notes across a set of followed accounts, not
// just one — the DID is part of the result here, since (unlike
// listNotesByAccount) it isn't already known by the caller.
type FeedNote struct {
	URI       string
	CID       string
	DID       string
	Provider  string
	WorkID    string
	Season    *int
	Episode   *int
	Text      string
	CreatedAt string
}

// listNotesByDIDs is the feed's actual query: chronological, deterministic
// — no ranking, matching this product's own non-negotiable shape for any
// feed. An empty dids slice (follows nobody, or nobody followed has ever
// used this appview) returns no rows rather than every note that exists;
// there's no query to run if there's nothing to filter by. Replies are
// excluded — same reasoning as listNotes.
func listNotesByDIDs(db *sql.DB, dids []string, limit int) ([]FeedNote, error) {
	if len(dids) == 0 {
		return nil, nil
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(dids)), ",")
	args := make([]any, 0, len(dids)+1)
	for _, d := range dids {
		args = append(args, d)
	}
	args = append(args, limit)

	query := fmt.Sprintf(
		`SELECT uri, cid, did, provider, work_id, season, episode, text, created_at FROM notes
		 WHERE did IN (%s) AND reply_parent_uri IS NULL ORDER BY created_at DESC LIMIT ?`, placeholders)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []FeedNote
	for rows.Next() {
		var n FeedNote
		if err := rows.Scan(&n.URI, &n.CID, &n.DID, &n.Provider, &n.WorkID, &n.Season, &n.Episode, &n.Text, &n.CreatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

// listNotesByWorks backs the feed's main "Shelf" tab: notes from *anyone*
// about works on the viewer's own shelf — obra-first, same as the rest of
// this product, not organized around who wrote something. works is a set
// of (provider, work_id) pairs; matched with an OR chain rather than
// SQLite's row-value IN syntax, to stay unambiguous across driver
// versions for what's normally a short list (one person's shelf). Replies
// are excluded — same reasoning as listNotes.
func listNotesByWorks(db *sql.DB, works []ShelfItem, limit int) ([]FeedNote, error) {
	if len(works) == 0 {
		return nil, nil
	}

	clauses := make([]string, 0, len(works))
	args := make([]any, 0, len(works)*2+1)
	for _, w := range works {
		clauses = append(clauses, "(provider = ? AND work_id = ?)")
		args = append(args, w.Provider, w.WorkID)
	}
	args = append(args, limit)

	query := fmt.Sprintf(
		`SELECT uri, cid, did, provider, work_id, season, episode, text, created_at FROM notes
		 WHERE (%s) AND reply_parent_uri IS NULL ORDER BY created_at DESC LIMIT ?`, strings.Join(clauses, " OR "))
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []FeedNote
	for rows.Next() {
		var n FeedNote
		if err := rows.Scan(&n.URI, &n.CID, &n.DID, &n.Provider, &n.WorkID, &n.Season, &n.Episode, &n.Text, &n.CreatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

// AccountNote backs the profile page — every note a given account has
// written, across every work, regardless of season/episode. Replies are
// included here (a person's replies are still part of what they've said),
// unlike the work/feed queries above.
type AccountNote struct {
	URI       string
	Provider  string
	WorkID    string
	Season    *int
	Episode   *int
	Text      string
	CreatedAt string
}

func listNotesByAccount(db *sql.DB, did string) ([]AccountNote, error) {
	rows, err := db.Query(
		`SELECT uri, provider, work_id, season, episode, text, created_at FROM notes
		 WHERE did = ? ORDER BY created_at DESC`,
		did)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []AccountNote
	for rows.Next() {
		var n AccountNote
		if err := rows.Scan(&n.URI, &n.Provider, &n.WorkID, &n.Season, &n.Episode, &n.Text, &n.CreatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}
