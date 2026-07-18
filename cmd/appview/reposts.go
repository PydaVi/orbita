package main

import (
	"database/sql"
	"fmt"
	"strings"
)

// A repost is its own record (social.orbita.repost), not a flag on the
// note it points at — same shape AT Protocol already uses
// (app.bsky.feed.repost), and the only way this stays true to the "no
// public popularity metrics" principle: nothing here is ever aggregated
// into a count anywhere, it only ever surfaces as "reposted by @handle" in
// the Following feed of someone who follows the reposter.
const repostsSchema = `
CREATE TABLE IF NOT EXISTS reposts (
	uri         TEXT PRIMARY KEY,
	cid         TEXT NOT NULL,
	did         TEXT NOT NULL,
	subject_uri TEXT NOT NULL,
	subject_cid TEXT NOT NULL,
	created_at  TEXT NOT NULL,
	indexed_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
`

func insertRepost(db *sql.DB, uri, cid, did, subjectURI, subjectCID, createdAt string) error {
	_, err := db.Exec(
		`INSERT INTO reposts (uri, cid, did, subject_uri, subject_cid, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(uri) DO NOTHING`,
		uri, cid, did, subjectURI, subjectCID, createdAt,
	)
	return err
}

type Repost struct {
	URI        string
	DID        string
	SubjectURI string
	CreatedAt  string
}

// listRepostsByDIDs backs the Following feed tab: reposts made by accounts
// the viewer follows, regardless of who wrote the note being reposted —
// that's the whole point, it surfaces things a friend found worth
// sharing, from anyone.
func listRepostsByDIDs(db *sql.DB, dids []string, limit int) ([]Repost, error) {
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
		`SELECT uri, did, subject_uri, created_at FROM reposts
		 WHERE did IN (%s) ORDER BY created_at DESC LIMIT ?`, placeholders)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reposts []Repost
	for rows.Next() {
		var rp Repost
		if err := rows.Scan(&rp.URI, &rp.DID, &rp.SubjectURI, &rp.CreatedAt); err != nil {
			return nil, err
		}
		reposts = append(reposts, rp)
	}
	return reposts, rows.Err()
}
