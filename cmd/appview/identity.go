package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// Handle resolution is pure protocol: a DID document declares its handle
// (alsoKnownAs) and its PDS endpoint, both bi-directionally verifiable —
// indigo's identity.CacheDirectory does the lookup and caches it in memory.
// Avatar resolution deliberately does NOT call Bluesky's app.bsky.actor
// API — it reads the account's own app.bsky.actor.profile record straight
// from its own PDS (the well-known convention for a profile picture in
// this ecosystem, not a Bluesky-specific one) and builds a blob URL
// against that same PDS. No dependency on any single company's service.
var identityDirectory = identity.NewCacheDirectory(
	&identity.BaseDirectory{},
	0,             // unlimited cache size
	24*time.Hour,  // successful lookups stay cached a day
	5*time.Minute, // failed lookups retried sooner
	1*time.Hour,
)

const identityCacheSchema = `
CREATE TABLE IF NOT EXISTS identity_cache (
	did        TEXT PRIMARY KEY,
	handle     TEXT NOT NULL,
	avatar_url TEXT NOT NULL,
	cached_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
`

func getCachedIdentity(db *sql.DB, did string) (handle, avatarURL string, ok bool) {
	row := db.QueryRow(`SELECT handle, avatar_url FROM identity_cache WHERE did = ?`, did)
	if err := row.Scan(&handle, &avatarURL); err != nil {
		return "", "", false
	}
	return handle, avatarURL, true
}

func setCachedIdentity(db *sql.DB, did, handle, avatarURL string) error {
	_, err := db.Exec(
		`INSERT INTO identity_cache (did, handle, avatar_url) VALUES (?, ?, ?)
		 ON CONFLICT(did) DO UPDATE SET handle = excluded.handle, avatar_url = excluded.avatar_url`,
		did, handle, avatarURL,
	)
	return err
}

// resolveIdentity is cache-first and fail-open, same shape as displayWork:
// on any failure it falls back to showing the raw DID as the "handle" and
// no avatar, rather than breaking the page.
func resolveIdentity(ctx context.Context, db *sql.DB, didStr string) (handle, avatarURL string) {
	if h, a, ok := getCachedIdentity(db, didStr); ok {
		return h, a
	}

	did, err := syntax.ParseDID(didStr)
	if err != nil {
		return didStr, ""
	}

	ident, err := identityDirectory.LookupDID(ctx, did)
	if err != nil {
		return didStr, ""
	}
	handle = ident.Handle.String()
	if handle == "" || handle == "handle.invalid" {
		handle = didStr
	}

	pds, ok := ident.Services["atproto_pds"]
	if !ok {
		setCachedIdentity(db, didStr, handle, "")
		return handle, ""
	}

	avatarURL = fetchAvatarURL(pds.URL, didStr)
	if err := setCachedIdentity(db, didStr, handle, avatarURL); err != nil {
		// Not fatal — just means this DID gets re-resolved next time.
		_ = err
	}
	return handle, avatarURL
}

// fetchAvatarURL reads the account's own app.bsky.actor.profile record
// straight from its own PDS and builds the getBlob URL for the avatar —
// no Bluesky-specific API call. Returns "" on any failure (no profile
// record, no avatar set, PDS unreachable): fail-open, same as everywhere
// else in this codebase.
func fetchAvatarURL(pdsURL, did string) string {
	u := fmt.Sprintf("%s/xrpc/com.atproto.repo.getRecord?repo=%s&collection=app.bsky.actor.profile&rkey=self",
		pdsURL, url.QueryEscape(did))
	resp, err := http.Get(u)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var body struct {
		Value struct {
			Avatar struct {
				Ref struct {
					Link string `json:"$link"`
				} `json:"ref"`
			} `json:"avatar"`
		} `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return ""
	}
	cid := body.Value.Avatar.Ref.Link
	if cid == "" {
		return ""
	}
	return fmt.Sprintf("%s/xrpc/com.atproto.sync.getBlob?did=%s&cid=%s",
		pdsURL, url.QueryEscape(did), url.QueryEscape(cid))
}
