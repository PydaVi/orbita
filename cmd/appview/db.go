package main

import (
	"database/sql"
	"fmt"
	"log"
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

// work_tags is the real data the archetype work needed and never had —
// TMDB has always returned genres in its own movie/tv responses
// (confirmed against the real API), this appview just never extracted
// them until now. One row per (work, tag) rather than a comma-joined
// column so a future "which works share this tag" query stays a plain
// SQL query, not string parsing. position preserves TMDB's own genre
// order (index in resolveWork's Genres slice) — the constellation's
// dominantFamily() picks "the first recognized tag" to decide a work's
// single anchor family, and that's only a meaningful choice if "first"
// means TMDB's own primary genre, not whatever a plain SELECT happens to
// return.
const workTagsSchema = `
CREATE TABLE IF NOT EXISTS work_tags (
	provider TEXT NOT NULL,
	work_id  TEXT NOT NULL,
	tag      TEXT NOT NULL,
	position INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (provider, work_id, tag)
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
	// Same reasoning as notes/nooks below: an existing work_cache table
	// (any database from before `overview` was added to workCacheSchema)
	// needs the column added explicitly. Found live: setCachedWork was
	// silently failing on every cache miss for lack of this column, so
	// every uncached work was being re-fetched from TMDB on every lookup.
	if err := ensureColumn(db, "work_cache", "overview", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return nil, fmt.Errorf("migrating work_cache.overview: %w", err)
	}
	if _, err := db.Exec(workTagsSchema); err != nil {
		return nil, fmt.Errorf("creating work_tags schema: %w", err)
	}
	// Found live: every work_tags row written before `position` existed was
	// read back in plain alphabetical order (SQLite's own index order for
	// this lookup, not insertion order), which silently broke
	// dominantFamily()'s "first tag" rule into "alphabetically-first tag" —
	// Drama, this catalog's most common secondary genre, almost never won
	// alphabetically, so the constellation's anchors and the archetype's
	// own dominant-family text could genuinely disagree. Distinguishing
	// "just added the column" from "already migrated" (ensureColumn's own
	// swallowed-error return can't tell them apart) matters here: only a
	// fresh ALTER means the existing rows predate real ordering and are
	// worth clearing. work_tags is a disposable cache like work_cache —
	// wiping it just means the next read re-resolves through TMDB, this
	// time with position recorded correctly.
	alterErr := func() error {
		_, err := db.Exec(`ALTER TABLE work_tags ADD COLUMN position INTEGER NOT NULL DEFAULT 0`)
		return err
	}()
	if alterErr == nil {
		if _, err := db.Exec(`DELETE FROM work_tags`); err != nil {
			return nil, fmt.Errorf("clearing stale work_tags after position migration: %w", err)
		}
	} else if !strings.Contains(alterErr.Error(), "duplicate column name") {
		return nil, fmt.Errorf("migrating work_tags.position: %w", alterErr)
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
	if _, err := db.Exec(savedNotesSchema); err != nil {
		return nil, fmt.Errorf("creating saved_notes schema: %w", err)
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

// getShelfItem is the idempotency check POST /api/shelf/add needs: nothing
// on the write path used to stop the same {provider, id} from being added
// twice as two separate PDS records (there's no protocol-level uniqueness
// constraint on record content, only on the URI, and every createRecord
// call mints a fresh rkey) — a repeat "+ Add to shelf" click, from a
// second visit or a UI that hadn't yet learned the work was already
// there, silently produced a real duplicate.
func getShelfItem(db *sql.DB, did, provider, workID string) (*ShelfItem, error) {
	var it ShelfItem
	row := db.QueryRow(
		`SELECT uri, cid, did, provider, work_id, created_at, indexed_at
		 FROM shelf_items WHERE did = ? AND provider = ? AND work_id = ? LIMIT 1`,
		did, provider, workID,
	)
	if err := row.Scan(&it.URI, &it.CID, &it.DID, &it.Provider, &it.WorkID, &it.CreatedAt, &it.IndexedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &it, nil
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

func setWorkTags(db *sql.DB, provider, workID string, tags []string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM work_tags WHERE provider = ? AND work_id = ?`, provider, workID); err != nil {
		return err
	}
	for i, t := range tags {
		if _, err := tx.Exec(
			`INSERT INTO work_tags (provider, work_id, tag, position) VALUES (?, ?, ?, ?) ON CONFLICT DO NOTHING`,
			provider, workID, t, i,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func queryWorkTags(db *sql.DB, provider, workID string) []string {
	rows, err := db.Query(`SELECT tag FROM work_tags WHERE provider = ? AND work_id = ? ORDER BY position`, provider, workID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var tags []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err == nil {
			tags = append(tags, t)
		}
	}
	return tags
}

// getWorkTags resolves directly (not through displayWorkFull's own
// work_cache check) whenever no tag rows exist yet — work_tags is what
// this appview didn't have until now, so checking work_cache's own
// presence isn't a safe proxy for "tags already handled": every work
// cached before this feature existed already passes that check while
// genuinely having zero rows in work_tags. A work with real, legitimately
// zero genres would keep re-resolving here, but that's rare enough in
// practice (TMDB titles almost always carry at least one) to accept
// rather than build a separate "tags were attempted" marker for.
func getWorkTags(db *sql.DB, provider, workID string) []string {
	if tags := queryWorkTags(db, provider, workID); len(tags) > 0 {
		return tags
	}
	w, err := resolveWork(provider, workID)
	if err != nil {
		return nil
	}
	if err := setWorkTags(db, provider, workID, w.Genres); err != nil {
		log.Printf("failed to cache tags for %s/%s: %v", provider, workID, err)
	}
	return w.Genres
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
