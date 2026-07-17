package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// The manual raw-id add form (type a provider and id by hand) is retired —
// it was the exact path that caused the "Titanic" garbage-record incident
// in docs/BETA1-PLAN.md, and /search plus POST /api/shelf/add fully
// replace it now. Delete stays here, still used by the classic GET /shelf
// debug page from Beta 0 (see list.go).
func setupShelf(mux *http.ServeMux, db *sql.DB) {
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
