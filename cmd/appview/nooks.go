package main

import "database/sql"

// Beta 7: a nook is the primary way a shelf is organized and shown to
// visitors — not an optional side list. The whole works array lives in one
// record; "editing" a nook (renaming, reordering, adding/removing a work)
// is just writing a new version of that same record via
// com.atproto.repo.putRecord, not a family of granular membership
// records. nook_items exists purely to preserve order locally and to
// answer "which works are already organized" — it's fully rebuilt every
// time a nook is (re)indexed, matching putRecord's own "whole record
// replaced" semantics.
const nooksSchema = `
CREATE TABLE IF NOT EXISTS nooks (
	uri         TEXT PRIMARY KEY,
	cid         TEXT NOT NULL,
	did         TEXT NOT NULL,
	name        TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	theme       TEXT NOT NULL DEFAULT 'default',
	created_at  TEXT NOT NULL,
	indexed_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
`

const nookItemsSchema = `
CREATE TABLE IF NOT EXISTS nook_items (
	nook_uri TEXT NOT NULL,
	position INTEGER NOT NULL,
	provider TEXT NOT NULL,
	work_id  TEXT NOT NULL,
	PRIMARY KEY (nook_uri, position)
);
`

type WorkRef struct {
	Provider string
	WorkID   string
}

// insertNook is also the update path: it replaces nook_items wholesale,
// matching putRecord's "the whole record was replaced" semantics rather
// than tracking incremental add/remove operations.
func insertNook(db *sql.DB, uri, cid, did, name, description, theme, createdAt string, works []WorkRef) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO nooks (uri, cid, did, name, description, theme, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(uri) DO UPDATE SET
		   cid = excluded.cid, name = excluded.name, description = excluded.description,
		   theme = excluded.theme, created_at = excluded.created_at`,
		uri, cid, did, name, description, theme, createdAt,
	)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM nook_items WHERE nook_uri = ?`, uri); err != nil {
		return err
	}
	for i, w := range works {
		if _, err := tx.Exec(
			`INSERT INTO nook_items (nook_uri, position, provider, work_id) VALUES (?, ?, ?, ?)`,
			uri, i, w.Provider, w.WorkID,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func deleteNook(db *sql.DB, uri string) error {
	if _, err := db.Exec(`DELETE FROM nook_items WHERE nook_uri = ?`, uri); err != nil {
		return err
	}
	_, err := db.Exec(`DELETE FROM nooks WHERE uri = ?`, uri)
	return err
}

type Nook struct {
	URI         string
	Name        string
	Description string
	Theme       string
	CreatedAt   string
	Works       []WorkRef
}

func listNooksByAccount(db *sql.DB, did string) ([]Nook, error) {
	rows, err := db.Query(
		`SELECT uri, name, description, theme, created_at FROM nooks
		 WHERE did = ? ORDER BY created_at ASC`,
		did)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nooks []Nook
	for rows.Next() {
		var n Nook
		if err := rows.Scan(&n.URI, &n.Name, &n.Description, &n.Theme, &n.CreatedAt); err != nil {
			return nil, err
		}
		nooks = append(nooks, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range nooks {
		works, err := listNookWorks(db, nooks[i].URI)
		if err != nil {
			return nil, err
		}
		nooks[i].Works = works
	}
	return nooks, nil
}

func listNookWorks(db *sql.DB, nookURI string) ([]WorkRef, error) {
	rows, err := db.Query(
		`SELECT provider, work_id FROM nook_items WHERE nook_uri = ? ORDER BY position ASC`,
		nookURI)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var works []WorkRef
	for rows.Next() {
		var w WorkRef
		if err := rows.Scan(&w.Provider, &w.WorkID); err != nil {
			return nil, err
		}
		works = append(works, w)
	}
	return works, rows.Err()
}

// listUnsortedShelfItems backs the profile's catch-all section: shelf
// items that aren't in any of the account's own nooks yet. A freshly
// shelved work starts here, not trapped inside some default grouping.
func listUnsortedShelfItems(db *sql.DB, did string) ([]ShelfItem, error) {
	rows, err := db.Query(
		`SELECT s.uri, s.cid, s.did, s.provider, s.work_id, s.created_at, s.indexed_at
		 FROM shelf_items s
		 WHERE s.did = ?
		   AND NOT EXISTS (
		     SELECT 1 FROM nook_items ni
		     JOIN nooks n ON n.uri = ni.nook_uri
		     WHERE n.did = s.did AND ni.provider = s.provider AND ni.work_id = s.work_id
		   )
		 ORDER BY s.created_at DESC`,
		did)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ShelfItem
	for rows.Next() {
		var it ShelfItem
		if err := rows.Scan(&it.URI, &it.CID, &it.DID, &it.Provider, &it.WorkID, &it.CreatedAt, &it.IndexedAt); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// shelfItemExists is the write-time guard the Lexicon itself can't
// express: a nook may only ever contain works already on the author's own
// shelf.
func shelfItemExists(db *sql.DB, did, provider, workID string) bool {
	var one int
	row := db.QueryRow(
		`SELECT 1 FROM shelf_items WHERE did = ? AND provider = ? AND work_id = ? LIMIT 1`,
		did, provider, workID)
	return row.Scan(&one) == nil
}
