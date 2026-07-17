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
	// Minimal scope: only create + delete on our own collection, no
	// access to the whole account, no "update" either since we don't
	// have that feature yet. REPO_ACTIONS is create/update/delete
	// (confirmed against @atproto/oauth-scopes' source); multiple
	// actions on the same collection repeat the "action" param, not a
	// comma list. Found this the hard way: a real delete attempt failed
	// with "ScopeMissingError: Missing required scope
	// repo:social.orbita.shelf.item?action=delete" against the actual
	// Bluesky server, because Beta 0 only ever requested "create".
	// Beta 2: a separate repo scope entry for the second collection —
	// each collection gets its own "repo:" scope, they don't combine
	// into one string.
	scopes := []string{
		"atproto",
		"repo:social.orbita.shelf.item?action=create&action=delete",
		"repo:social.orbita.note?action=create",
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
	http.Redirect(w, r, "/shelf/add", http.StatusFound)
}
