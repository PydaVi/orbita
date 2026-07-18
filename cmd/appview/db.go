package main

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

// ensureColumn adds a column to an already-existing table if it isn't
// there yet — SQLite has no "ADD COLUMN IF NOT EXISTS," so the "column
// already exists" error from a repeat run is caught and ignored instead.
func ensureColumn(db *sql.DB, table, column, definition string) error {
	_, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition))
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return err
	}
	return nil
}

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
	overview   TEXT NOT NULL DEFAULT '',
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
	if _, err := db.Exec(seasonCacheSchema); err != nil {
		return nil, fmt.Errorf("creating season_cache schema: %w", err)
	}
	if _, err := db.Exec(episodeCacheSchema); err != nil {
		return nil, fmt.Errorf("creating episode_cache schema: %w", err)
	}
	if _, err := db.Exec(notesSchema); err != nil {
		return nil, fmt.Errorf("creating notes schema: %w", err)
	}
	// notesSchema's CREATE TABLE IF NOT EXISTS only applies to a fresh
	// database — an existing notes table (any beta before this one) needs
	// these columns added explicitly, since it already exists.
	for _, col := range []string{"reply_root_uri", "reply_root_cid", "reply_parent_uri", "reply_parent_cid"} {
		if err := ensureColumn(db, "notes", col, "TEXT"); err != nil {
			return nil, fmt.Errorf("migrating notes.%s: %w", col, err)
		}
	}
	if _, err := db.Exec(repostsSchema); err != nil {
		return nil, fmt.Errorf("creating reposts schema: %w", err)
	}
	if _, err := db.Exec(nooksSchema); err != nil {
		return nil, fmt.Errorf("creating nooks schema: %w", err)
	}
	// Same reasoning as notes above: an existing nooks table (any Beta 7
	// database from before nook ordering existed) needs sort_order added
	// explicitly.
	if err := ensureColumn(db, "nooks", "sort_order", "INTEGER"); err != nil {
		return nil, fmt.Errorf("migrating nooks.sort_order: %w", err)
	}
	if _, err := db.Exec(nookItemsSchema); err != nil {
		return nil, fmt.Errorf("creating nook_items schema: %w", err)
	}
	if _, err := db.Exec(identityCacheSchema); err != nil {
		return nil, fmt.Errorf("creating identity_cache schema: %w", err)
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

// listShelfItemsByAccount backs the real "my shelf" page — an earlier,
// unscoped "everyone's shelf" version never served any real purpose and
// was removed (see api.go's GET /api/shelf).
func listShelfItemsByAccount(db *sql.DB, did string) ([]ShelfItem, error) {
	rows, err := db.Query(`SELECT uri, cid, did, provider, work_id, created_at, indexed_at
	                        FROM shelf_items WHERE did = ? ORDER BY created_at DESC`, did)
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

func getCachedWork(db *sql.DB, provider, workID string) (title, posterURL, year, overview string, ok bool) {
	row := db.QueryRow(`SELECT title, poster_url, year, overview FROM work_cache WHERE provider = ? AND work_id = ?`,
		provider, workID)
	if err := row.Scan(&title, &posterURL, &year, &overview); err != nil {
		return "", "", "", "", false
	}
	return title, posterURL, year, overview, true
}

func setCachedWork(db *sql.DB, provider, workID, title, posterURL, year, overview string) error {
	_, err := db.Exec(
		`INSERT INTO work_cache (provider, work_id, title, poster_url, year, overview)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(provider, work_id) DO UPDATE SET
		   title = excluded.title, poster_url = excluded.poster_url, year = excluded.year, overview = excluded.overview`,
		provider, workID, title, posterURL, year, overview,
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
