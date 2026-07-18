package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/gorilla/sessions"
)

var (
	oauthClient *oauth.ClientApp
	cookieStore *sessions.CookieStore
)

// Checkpoint 1 of the plan: registers the handlers and confirms the
// redirect URL gets generated. StartAuthFlow (login) already does PAR +
// PKCE + DPoP internally — we don't write any of that by hand, just call
// the lib.
func setupOAuth(mux *http.ServeMux, sessionSecret string) {
	// REPO_ACTIONS is create/update/delete (confirmed against
	// @atproto/oauth-scopes' source); multiple actions on the same
	// collection repeat the "action" param, not a comma list. Found this
	// the hard way: a real delete attempt failed with "ScopeMissingError:
	// Missing required scope repo:social.orbita.shelf.item?action=delete"
	// against the actual Bluesky server, because Beta 0 only ever
	// requested "create". Each collection gets its own "repo:" scope
	// entry, they don't combine into one string — caught the same class
	// of bug again in Beta 7 (nook creation failing with the same
	// ScopeMissingError shape) after adding two new collections without
	// updating this list: repost only ever needs "create", but nook needs
	// "update" too, since editing a nook is a whole-record replacement
	// (com.atproto.repo.putRecord), the first write path in this project
	// that isn't just create/delete.
	scopes := []string{
		"atproto",
		"repo:social.orbita.shelf.item?action=create&action=delete",
		"repo:social.orbita.note?action=create",
		"repo:social.orbita.repost?action=create",
		"repo:social.orbita.shelf.nook?action=create&action=update&action=delete",
	}

	// NewLocalhostConfig uses the spec's local-development exception —
	// client_id becomes a special "http://localhost", with no public
	// domain or HTTPS required.
	// "localhost" (tested) was rejected by Bluesky's real server — PAR
	// only accepts the literal forms 127.0.0.1/[::1]. But the author's
	// browser only reaches WSL2 via "localhost", not via 127.0.0.1 —
	// hypothesis: "localhost" resolves via IPv6 (::1) first in this
	// environment, and it's specifically IPv4 (127.0.0.1) that doesn't
	// cross the WSL2↔Windows boundary here. Testing [::1], the other
	// literal form the spec accepts.
	config := oauth.NewLocalhostConfig(
		"http://[::1]:8092/oauth/callback",
		scopes,
	)
	oauthClient = oauth.NewClientApp(&config, oauth.NewMemStore())
	cookieStore = sessions.NewCookieStore([]byte(sessionSecret))

	mux.HandleFunc("GET /oauth/client-metadata.json", handleClientMetadata)
	mux.HandleFunc("GET /oauth/login", handleLoginForm)
	mux.HandleFunc("POST /oauth/login", handleLoginStart)
	mux.HandleFunc("GET /oauth/callback", handleCallback)
}

func handleClientMetadata(w http.ResponseWriter, r *http.Request) {
	meta := oauthClient.Config.ClientMetadata()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(meta); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleLoginForm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!doctype html>
<form method="POST" action="/oauth/login">
  <label>Handle or DID: <input name="handle" placeholder="alice.test"></label>
  <button type="submit">Sign in with AT Protocol</button>
</form>`)
}

func handleLoginStart(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	handle := r.PostFormValue("handle")

	redirectURL, err := oauthClient.StartAuthFlow(r.Context(), handle)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to start login: %v", err), http.StatusBadRequest)
		return
	}
	log.Printf("login started for %q, redirecting to PDS: %s", handle, redirectURL)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func handleCallback(w http.ResponseWriter, r *http.Request) {
	sessData, err := oauthClient.ProcessCallback(r.Context(), r.URL.Query())
	if err != nil {
		http.Error(w, fmt.Sprintf("OAuth callback failed: %v", err), http.StatusBadRequest)
		return
	}

	sess, _ := cookieStore.Get(r, "orbita-oauth")
	sess.Values["account_did"] = sessData.AccountDID.String()
	sess.Values["session_id"] = sessData.SessionID
	if err := sess.Save(r, w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("login successful: %s", sessData.AccountDID.String())
	// /shelf/add (the manual raw-id form) was retired in Beta 3 — /search
	// is the real entry point for adding something to your shelf now.
	http.Redirect(w, r, "/search", http.StatusFound)
}
