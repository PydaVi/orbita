package main

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Banco local: só a cópia indexada do que o webhook do Tap entrega.
// PDS é a fonte da verdade (ver docs/BETA0-PLAN.md, "Onde cada dado mora") —
// esse arquivo pode ser apagado e reconstruído do zero replayando o Tap,
// nunca é o único lugar onde um dado existe.
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

func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("abrindo banco: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("criando schema: %w", err)
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
