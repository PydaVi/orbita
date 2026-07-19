package main

import "database/sql"

// nookWorksLimit mirrors lexicons/social/orbita/shelf/nook.json's own
// works.maxLength — see that file's description for the actual reasoning
// (a nook is a curated gesture, not the shelf again; the real ceiling
// there is the Sight & Sound poll's 10-title ballot and "best of the
// year/decade" lists, not any technical constraint). This appview never
// validated an incoming record against its own Lexicon on write, so this
// was only ever enforced client-side (shelf.js's NOOK_WORKS_LIMIT) until
// now — a real gap, since nothing stopped a direct API call from
// exceeding it.
//
// maxNooksPerAccount exists for the same reason spread across nooks
// instead of within one: this product has exactly 7 curated nook themes
// (style.theme's knownValues), and the constellation (see
// constellation.go/js) anchors on theme — many nooks sharing few themes
// crowd the same region and stop being legible as distinct shapes. 7 was
// chosen to echo that same curated theme count directly (roughly one
// standout nook per mood), sitting within Miller's classic "7±2" chunking
// estimate for how many categories stay easy to hold in mind at once
// (itself later refined tighter, to around 4, by Cowan 2001 — 7 is the
// generous end of a real range, not picked in isolation).
const (
	nookWorksLimit     = 50
	maxNooksPerAccount = 7
)

// Beta 7: a nook is the primary way a shelf is organized and shown to
// visitors — not an optional side list. The whole works array lives in one
// record; "editing" a nook (renaming, reordering, adding/removing a work)
// is just writing a new version of that same record via
// com.atproto.repo.putRecord, not a family of granular membership
// records. nook_items exists purely to preserve order locally and to
// answer "which works are already organized" — it's fully rebuilt every
// time a nook is (re)indexed, matching putRecord's own "whole record
// replaced" semantics.
// sort_order (not "order" — a reserved SQL word) is gapped (multiples of
// 1000) so moving one nook to a new position only ever rewrites that one
// record: the new value is the midpoint between its new neighbors, not a
// renumbering of every nook. NULL for nooks written before this field
// existed — they sort after everything with a real value, by created_at.
const nooksSchema = `
CREATE TABLE IF NOT EXISTS nooks (
	uri         TEXT PRIMARY KEY,
	cid         TEXT NOT NULL,
	did         TEXT NOT NULL,
	name        TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	theme       TEXT NOT NULL DEFAULT 'default',
	sort_order  INTEGER,
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
// than tracking incremental add/remove operations. order is a pointer so a
// nook written without one (older client, or the field genuinely omitted)
// stores a real NULL instead of a misleading 0.
func insertNook(db *sql.DB, uri, cid, did, name, description, theme, createdAt string, order *int, works []WorkRef) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO nooks (uri, cid, did, name, description, theme, sort_order, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(uri) DO UPDATE SET
		   cid = excluded.cid, name = excluded.name, description = excluded.description,
		   theme = excluded.theme, sort_order = excluded.sort_order, created_at = excluded.created_at`,
		uri, cid, did, name, description, theme, order, createdAt,
	)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM nook_items WHERE nook_uri = ?`, uri); err != nil {
		return err
	}
	// Defensive, not just optimistic: this indexes whatever a record
	// actually contains, from any source (the live webhook, a resync read
	// straight from the PDS) — a duplicate entry within one nook's works
	// array has nowhere else to be caught before it reaches here.
	seen := make(map[WorkRef]bool, len(works))
	position := 0
	for _, w := range works {
		if seen[w] {
			continue
		}
		seen[w] = true
		if _, err := tx.Exec(
			`INSERT INTO nook_items (nook_uri, position, provider, work_id) VALUES (?, ?, ?, ?)`,
			uri, position, w.Provider, w.WorkID,
		); err != nil {
			return err
		}
		position++
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
	Order       *int
	CreatedAt   string
	Works       []WorkRef
}

// Nooks with a real sort_order come first (lowest first); nooks without
// one (written before this field existed) sort after all of those, oldest
// first — a reasonable default, not a data loss.
// getNook fetches a single nook by its at:// URI — the shareable nook page
// needs one record, not the whole account's list.
func getNook(db *sql.DB, uri string) (*Nook, error) {
	var n Nook
	row := db.QueryRow(
		`SELECT uri, name, description, theme, sort_order, created_at FROM nooks WHERE uri = ?`,
		uri)
	if err := row.Scan(&n.URI, &n.Name, &n.Description, &n.Theme, &n.Order, &n.CreatedAt); err != nil {
		return nil, err
	}
	works, err := listNookWorks(db, n.URI)
	if err != nil {
		return nil, err
	}
	n.Works = works
	return &n, nil
}

func listNooksByAccount(db *sql.DB, did string) ([]Nook, error) {
	rows, err := db.Query(
		`SELECT uri, name, description, theme, sort_order, created_at FROM nooks
		 WHERE did = ?
		 ORDER BY (sort_order IS NULL) ASC, sort_order ASC, created_at ASC`,
		did)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nooks []Nook
	for rows.Next() {
		var n Nook
		if err := rows.Scan(&n.URI, &n.Name, &n.Description, &n.Theme, &n.Order, &n.CreatedAt); err != nil {
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

// nextNookOrder is where a newly created nook lands: after everything that
// already has a real position, gapped so future reorders have room either
// side of it.
func nextNookOrder(db *sql.DB, did string) int {
	var max sql.NullInt64
	row := db.QueryRow(`SELECT MAX(sort_order) FROM nooks WHERE did = ?`, did)
	if err := row.Scan(&max); err != nil || !max.Valid {
		return 1000
	}
	return int(max.Int64) + 1000
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
