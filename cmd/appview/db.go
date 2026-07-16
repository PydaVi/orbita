package main

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Local database: just the indexed copy of what the Tap webhook delivers.
// The PDS is the source of truth (see docs/BETA0-PLAN.md, "Where each
// piece of data lives") — this file can be deleted and rebuilt from
// scratch by replaying Tap, it's never the only place a piece of data
// exists.
const schema = `
CREATE TABLE IF NOT EXISTS shelf_items (
	uri        TEXT PRIMARY KEY,
	cid        TEXT NOT NULL,
	did        TEXT NOT NULL,
	provider   TEXT NOT NULL,
	work_id    TEXT NOT NULL,
	created_at TEXT NOT NULL,
	indexed_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
`

// Beta 1, item 2: a lightweight cache, not comum's growing public
// catalog. Only ever holds {provider, id} pairs that actually showed up
// in shelf_items — resolved once against TMDB, reused after. Disposable
// like everything else here: dropping this table just means the next
// read re-resolves and re-fills it.
const workCacheSchema = `
CREATE TABLE IF NOT EXISTS work_cache (
	provider   TEXT NOT NULL,
	work_id    TEXT NOT NULL,
	title      TEXT NOT NULL,
	poster_url TEXT NOT NULL,
	year       TEXT NOT NULL,
	cached_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
	PRIMARY KEY (provider, work_id)
);
`

func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("creating schema: %w", err)
	}
	if _, err := db.Exec(workCacheSchema); err != nil {
		return nil, fmt.Errorf("creating work_cache schema: %w", err)
	}
	return db, nil
}

func insertShelfItem(db *sql.DB, uri, cid, did, provider, workID, createdAt string) error {
	_, err := db.Exec(
		`INSERT INTO shelf_items (uri, cid, did, provider, work_id, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(uri) DO NOTHING`,
		uri, cid, did, provider, workID, createdAt,
	)
	return err
}

// Removes our local indexed copy. Does NOT touch the PDS — that's a
// separate call (com.atproto.repo.deleteRecord), made by the caller
// first. If Tap ever redelivers a delete for this uri (it isn't handled
// in webhook.go yet — a known gap, only "create" is handled today),
// this is naturally a no-op by then.
func deleteShelfItem(db *sql.DB, uri string) error {
	_, err := db.Exec(`DELETE FROM shelf_items WHERE uri = ?`, uri)
	return err
}

type ShelfItem struct {
	URI       string
	CID       string
	DID       string
	Provider  string
	WorkID    string
	CreatedAt string
	IndexedAt string
}

func listShelfItems(db *sql.DB) ([]ShelfItem, error) {
	rows, err := db.Query(`SELECT uri, cid, did, provider, work_id, created_at, indexed_at
	                        FROM shelf_items ORDER BY created_at DESC`)
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

func getCachedWork(db *sql.DB, provider, workID string) (title, posterURL, year string, ok bool) {
	row := db.QueryRow(`SELECT title, poster_url, year FROM work_cache WHERE provider = ? AND work_id = ?`,
		provider, workID)
	if err := row.Scan(&title, &posterURL, &year); err != nil {
		return "", "", "", false
	}
	return title, posterURL, year, true
}

func setCachedWork(db *sql.DB, provider, workID, title, posterURL, year string) error {
	_, err := db.Exec(
		`INSERT INTO work_cache (provider, work_id, title, poster_url, year)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(provider, work_id) DO UPDATE SET
		   title = excluded.title, poster_url = excluded.poster_url, year = excluded.year`,
		provider, workID, title, posterURL, year,
	)
	return err
}

// The Beta 1 "aggregation" query: grouped by work instead of listed by
// event. This is the first query where looking at the whole network of
// shelves is more useful than looking at what one person did.
func listShelfItemsByWork(db *sql.DB, provider, workID string) ([]ShelfItem, error) {
	rows, err := db.Query(`SELECT uri, cid, did, provider, work_id, created_at, indexed_at
	                        FROM shelf_items WHERE provider = ? AND work_id = ?
	                        ORDER BY created_at ASC`,
		provider, workID)
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
