package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// Replaces the manual curl: writes a real social.orbita.shelf.item,
// authenticated via the OAuth session (DPoP, PAR, PKCE already resolved
// by the lib at login — here we just reuse the saved session).
func setupShelf(mux *http.ServeMux, db *sql.DB) {
	mux.HandleFunc("GET /shelf/add", handleShelfAddForm)
	mux.HandleFunc("POST /shelf/add", handleShelfAdd)
	mux.HandleFunc("POST /shelf/delete", func(w http.ResponseWriter, r *http.Request) {
		handleShelfDelete(w, r, db)
	})
}

func currentSessionDID(r *http.Request) (*syntax.DID, string) {
	sess, _ := cookieStore.Get(r, "orbita-oauth")
	didStr, ok := sess.Values["account_did"].(string)
	if !ok || didStr == "" {
		return nil, ""
	}
	did, err := syntax.ParseDID(didStr)
	if err != nil {
		return nil, ""
	}
	sessionID, ok := sess.Values["session_id"].(string)
	if !ok || sessionID == "" {
		return nil, ""
	}
	return &did, sessionID
}

func handleShelfAddForm(w http.ResponseWriter, r *http.Request) {
	did, _ := currentSessionDID(r)
	if did == nil {
		http.Error(w, "not authenticated — sign in at /oauth/login first", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!doctype html>
<p>Signed in as %s</p>
<p><a href="/search">Search for a title</a> instead of typing a raw id below.</p>
<form method="POST" action="/shelf/add">
  <label>Provider: <input name="provider" value="tmdb-movie"></label>
  <label>ID (e.g. 603 = The Matrix on TMDB): <input name="id" placeholder="603"></label>
  <button type="submit">Add to shelf</button>
</form>`, did.String())
}

func handleShelfAdd(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	did, sessionID := currentSessionDID(r)
	if did == nil {
		http.Error(w, "not authenticated", http.StatusUnauthorized)
		return
	}

	oauthSess, err := oauthClient.ResumeSession(ctx, *did, sessionID)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid session, please sign in again: %v", err), http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	provider := r.PostFormValue("provider")
	workID := r.PostFormValue("id")

	c := oauthSess.APIClient()
	body := map[string]any{
		"repo":       c.AccountDID.String(),
		"collection": "social.orbita.shelf.item",
		"record": map[string]any{
			"$type": "social.orbita.shelf.item",
			"work": map[string]any{
				"provider": provider,
				"id":       workID,
			},
			"createdAt": syntax.DatetimeNow(),
		},
	}

	log.Printf("writing shelf.item via OAuth (DPoP): provider=%s id=%s", provider, workID)
	if err := c.Post(ctx, "com.atproto.repo.createRecord", body, nil); err != nil {
		http.Error(w, fmt.Sprintf("failed to write record: %v", err), http.StatusBadRequest)
		return
	}

	fmt.Fprintf(w, "added to shelf: %s/%s — check your real PDS", provider, workID)
}

// Deletes the real PDS record first — com.atproto.repo.deleteRecord only
// ever succeeds against the authenticated account's own repo, so there's
// no separate ownership check to write here, the protocol already
// enforces it. Only removes our local indexed copy after that succeeds.
func handleShelfDelete(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	ctx := r.Context()
	did, sessionID := currentSessionDID(r)
	if did == nil {
		http.Error(w, "not authenticated", http.StatusUnauthorized)
		return
	}

	oauthSess, err := oauthClient.ResumeSession(ctx, *did, sessionID)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid session, please sign in again: %v", err), http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	uri := r.PostFormValue("uri")
	atURI, err := syntax.ParseATURI(uri)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid AT-URI: %v", err), http.StatusBadRequest)
		return
	}

	c := oauthSess.APIClient()
	body := map[string]any{
		"repo":       c.AccountDID.String(),
		"collection": atURI.Collection().String(),
		"rkey":       atURI.RecordKey().String(),
	}

	log.Printf("deleting %s via OAuth (DPoP)", uri)
	if err := c.Post(ctx, "com.atproto.repo.deleteRecord", body, nil); err != nil {
		http.Error(w, fmt.Sprintf("failed to delete record: %v", err), http.StatusBadRequest)
		return
	}

	if err := deleteShelfItem(db, uri); err != nil {
		log.Printf("deleted from PDS but failed to remove local index for %s: %v", uri, err)
	}

	http.Redirect(w, r, "/shelf", http.StatusFound)
}
