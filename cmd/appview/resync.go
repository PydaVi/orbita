package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// Beta 7 (post-launch): the local SQLite index is a cache of the PDS, not
// the source of truth — but Tap's webhook delivery only tries so hard.
// Twice in one afternoon of local development, restarting the appview to
// pick up new code left a window where Tap's webhook POSTs came back
// "connection refused"; Tap retries for a while, and if the appview comes
// back up in time the record shows up late, but if the window runs past
// Tap's retry budget, the record is silently and permanently missing from
// the index — confirmed both ways by comparing the account's real PDS
// records (com.atproto.repo.listRecords) against orbita.db by hand. This
// endpoint makes that comparison a one-click operation instead of a
// manual diagnosis: re-read every record this account has, straight from
// their own PDS, and reconcile the local index against it — catching up
// anything the webhook missed, and removing anything indexed locally
// that's been deleted at the source. Scoped to the signed-in account's
// own data only; resyncing isn't something one account can trigger for
// another.
var resyncedCollections = []string{
	"social.orbita.shelf.item",
	"social.orbita.note",
	"social.orbita.repost",
	"social.orbita.shelf.nook",
}

type pdsRecord struct {
	uri   string
	cid   string
	value recordValue
}

// fetchAllRecords pages through com.atproto.repo.listRecords — a public
// read endpoint, no OAuth session needed, same as any AT Protocol client
// reading a repo it doesn't own.
func fetchAllRecords(ctx context.Context, pdsURL, did, collection string) ([]pdsRecord, error) {
	var out []pdsRecord
	cursor := ""
	for {
		u := fmt.Sprintf("%s/xrpc/com.atproto.repo.listRecords?repo=%s&collection=%s&limit=100",
			pdsURL, url.QueryEscape(did), url.QueryEscape(collection))
		if cursor != "" {
			u += "&cursor=" + url.QueryEscape(cursor)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		var body struct {
			Records []struct {
				URI   string          `json:"uri"`
				CID   string          `json:"cid"`
				Value json.RawMessage `json:"value"`
			} `json:"records"`
			Cursor string `json:"cursor"`
		}
		decodeErr := json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()
		if decodeErr != nil {
			return nil, decodeErr
		}

		for _, r := range body.Records {
			var v recordValue
			if err := json.Unmarshal(r.Value, &v); err != nil {
				continue
			}
			out = append(out, pdsRecord{uri: r.URI, cid: r.CID, value: v})
		}

		if body.Cursor == "" || len(body.Records) == 0 {
			break
		}
		cursor = body.Cursor
	}
	return out, nil
}

// pruneMissing deletes rows this account owns that are no longer live on
// the PDS — one query per table rather than a single query built from a
// table-name variable, so nothing here ever interpolates a string into
// SQL.
func pruneMissing(db *sql.DB, did string, live map[string]bool) error {
	prune := func(query string) error {
		rows, err := db.Query(query, did)
		if err != nil {
			return err
		}
		var toDelete []string
		for rows.Next() {
			var uri string
			if err := rows.Scan(&uri); err != nil {
				rows.Close()
				return err
			}
			if !live[uri] {
				toDelete = append(toDelete, uri)
			}
		}
		rows.Close()
		for _, uri := range toDelete {
			if _, err := db.Exec(`DELETE FROM shelf_items WHERE uri = ?`, uri); err != nil {
				return err
			}
			if _, err := db.Exec(`DELETE FROM notes WHERE uri = ?`, uri); err != nil {
				return err
			}
			if _, err := db.Exec(`DELETE FROM reposts WHERE uri = ?`, uri); err != nil {
				return err
			}
			if err := deleteNook(db, uri); err != nil {
				return err
			}
		}
		return nil
	}

	for _, q := range []string{
		`SELECT uri FROM shelf_items WHERE did = ?`,
		`SELECT uri FROM notes WHERE did = ?`,
		`SELECT uri FROM reposts WHERE did = ?`,
		`SELECT uri FROM nooks WHERE did = ?`,
	} {
		if err := prune(q); err != nil {
			return err
		}
	}
	return nil
}

// resyncAccount is the reconciliation itself: read every record this
// account has for every collection this appview indexes, upsert each one
// (indexRecord is the same function the webhook calls, so there's exactly
// one definition of "how a record becomes a row"), then remove anything
// local that's no longer live. Returns how many live records were found
// per collection, for a status line the caller can show.
func resyncAccount(ctx context.Context, db *sql.DB, did string) (map[string]int, error) {
	pdsURL, err := resolvePDSURL(ctx, did)
	if err != nil {
		return nil, fmt.Errorf("resolving PDS: %w", err)
	}

	counts := map[string]int{}
	live := map[string]bool{}
	for _, collection := range resyncedCollections {
		records, err := fetchAllRecords(ctx, pdsURL, did, collection)
		if err != nil {
			return nil, fmt.Errorf("listing %s: %w", collection, err)
		}
		counts[collection] = len(records)
		for _, rec := range records {
			live[rec.uri] = true
			if err := indexRecord(db, rec.uri, rec.cid, did, collection, rec.value); err != nil {
				return nil, fmt.Errorf("indexing %s: %w", rec.uri, err)
			}
		}
	}

	if err := pruneMissing(db, did, live); err != nil {
		return nil, fmt.Errorf("pruning stale rows: %w", err)
	}
	return counts, nil
}

func setupResync(mux *http.ServeMux, db *sql.DB) {
	mux.HandleFunc("POST /api/resync", func(w http.ResponseWriter, r *http.Request) {
		did, _ := currentSessionDID(r)
		if did == nil {
			http.Error(w, "not authenticated", http.StatusUnauthorized)
			return
		}

		counts, err := resyncAccount(r.Context(), db, did.String())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(counts)
	})
}
