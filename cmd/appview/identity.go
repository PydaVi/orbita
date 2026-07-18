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
	bio        TEXT NOT NULL DEFAULT '',
	cached_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
`

func getCachedIdentity(db *sql.DB, did string) (handle, avatarURL, bio string, ok bool) {
	row := db.QueryRow(`SELECT handle, avatar_url, bio FROM identity_cache WHERE did = ?`, did)
	if err := row.Scan(&handle, &avatarURL, &bio); err != nil {
		return "", "", "", false
	}
	return handle, avatarURL, bio, true
}

func setCachedIdentity(db *sql.DB, did, handle, avatarURL, bio string) error {
	_, err := db.Exec(
		`INSERT INTO identity_cache (did, handle, avatar_url, bio) VALUES (?, ?, ?, ?)
		 ON CONFLICT(did) DO UPDATE SET handle = excluded.handle, avatar_url = excluded.avatar_url, bio = excluded.bio`,
		did, handle, avatarURL, bio,
	)
	return err
}

// resolveIdentity is cache-first and fail-open, same shape as displayWork:
// on any failure it falls back to showing the raw DID as the "handle" and
// no avatar/bio, rather than breaking the page.
func resolveIdentity(ctx context.Context, db *sql.DB, didStr string) (handle, avatarURL string) {
	h, a, _, _ := resolveIdentityFull(ctx, db, didStr)
	return h, a
}

// resolveIdentityFull is the same resolution, exposing the bio too — kept
// separate from resolveIdentity so call sites that don't need a bio (shelf
// rows, note bylines) don't have to thread an unused value through.
func resolveIdentityFull(ctx context.Context, db *sql.DB, didStr string) (handle, avatarURL, bio string, ok bool) {
	if h, a, b, cached := getCachedIdentity(db, didStr); cached {
		return h, a, b, true
	}

	did, err := syntax.ParseDID(didStr)
	if err != nil {
		return didStr, "", "", false
	}

	ident, err := identityDirectory.LookupDID(ctx, did)
	if err != nil {
		return didStr, "", "", false
	}
	handle = ident.Handle.String()
	if handle == "" || handle == "handle.invalid" {
		handle = didStr
	}

	pds, hasPDS := ident.Services["atproto_pds"]
	if !hasPDS {
		setCachedIdentity(db, didStr, handle, "", "")
		return handle, "", "", true
	}

	avatarURL, bio = fetchBlueskyProfile(pds.URL, didStr)
	if err := setCachedIdentity(db, didStr, handle, avatarURL, bio); err != nil {
		// Not fatal — just means this DID gets re-resolved next time.
		_ = err
	}
	return handle, avatarURL, bio, true
}

// resolveHandleToDID is the reverse lookup profile pages need: given a
// handle typed into a URL, find the DID it currently belongs to (bi-
// directionally verified by the same CacheDirectory resolveIdentity uses,
// so a handle that doesn't resolve, or a handle/DID mismatch, is a real
// error here — there's no fallback string to show for an identity that
// doesn't exist).
func resolveHandleToDID(ctx context.Context, handleStr string) (string, error) {
	h, err := syntax.ParseHandle(handleStr)
	if err != nil {
		return "", err
	}
	ident, err := identityDirectory.LookupHandle(ctx, h)
	if err != nil {
		return "", err
	}
	return ident.DID.String(), nil
}

// resolvePDSURL is what the feed needs to read someone's follow list
// (follows.go) — a DID document's declared PDS endpoint, resolved through
// the same in-memory-cached directory as everything else here.
func resolvePDSURL(ctx context.Context, didStr string) (string, error) {
	did, err := syntax.ParseDID(didStr)
	if err != nil {
		return "", err
	}
	ident, err := identityDirectory.LookupDID(ctx, did)
	if err != nil {
		return "", err
	}
	pds, ok := ident.Services["atproto_pds"]
	if !ok {
		return "", fmt.Errorf("no PDS declared for %s", didStr)
	}
	return pds.URL, nil
}

// fetchBlueskyProfile reads the account's own app.bsky.actor.profile
// record straight from its own PDS and returns the avatar (as a resolved
// getBlob URL) and the bio text — no Bluesky-specific API call, just the
// well-known record shape any AT Protocol account may have on its own
// repo. Returns ("", "") on any failure (no profile record, PDS
// unreachable): fail-open, same as everywhere else in this codebase.
func fetchBlueskyProfile(pdsURL, did string) (avatarURL, bio string) {
	u := fmt.Sprintf("%s/xrpc/com.atproto.repo.getRecord?repo=%s&collection=app.bsky.actor.profile&rkey=self",
		pdsURL, url.QueryEscape(did))
	resp, err := http.Get(u)
	if err != nil {
		return "", ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", ""
	}

	var body struct {
		Value struct {
			Avatar struct {
				Ref struct {
					Link string `json:"$link"`
				} `json:"ref"`
			} `json:"avatar"`
			Description string `json:"description"`
		} `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", ""
	}
	bio = body.Value.Description
	cid := body.Value.Avatar.Ref.Link
	if cid == "" {
		return "", bio
	}
	avatarURL = fmt.Sprintf("%s/xrpc/com.atproto.sync.getBlob?did=%s&cid=%s",
		pdsURL, url.QueryEscape(did), url.QueryEscape(cid))
	return avatarURL, bio
}
