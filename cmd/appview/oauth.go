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

// Checkpoint 1 do plano: registra os handlers e confirma que a URL de
// redirect é gerada. StartAuthFlow (login) já faz PAR + PKCE + DPoP por
// dentro — não escrevemos nada disso na mão, só chamamos a lib.
func setupOAuth(mux *http.ServeMux, sessionSecret string) {
	// Escopo mínimo: só permissão de criar registro na nossa própria coleção,
	// nada de acesso à conta inteira.
	scopes := []string{"atproto", "repo:social.orbita.shelf.item?action=create"}

	// NewLocalhostConfig usa a exceção de desenvolvimento local da spec —
	// client_id vira um "http://localhost" especial, sem exigir domínio
	// público nem HTTPS.
	// "localhost" (testado) foi recusado pelo servidor real da Bluesky —
	// PAR só aceita as formas literais 127.0.0.1/[::1]. Mas o navegador do
	// autor só alcança o WSL2 via "localhost", não via 127.0.0.1 — hipótese:
	// "localhost" resolve primeiro por IPv6 (::1) nesse ambiente, e é o IPv4
	// (127.0.0.1) especificamente que não atravessa WSL2↔Windows aqui.
	// Testando [::1], a outra forma literal que a spec aceita.
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
  <label>Handle ou DID: <input name="handle" placeholder="alice.test"></label>
  <button type="submit">Entrar com AT Protocol</button>
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
		http.Error(w, fmt.Sprintf("falha ao iniciar login: %v", err), http.StatusBadRequest)
		return
	}
	log.Printf("login iniciado pra %q, redirecionando pro PDS: %s", handle, redirectURL)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func handleCallback(w http.ResponseWriter, r *http.Request) {
	sessData, err := oauthClient.ProcessCallback(r.Context(), r.URL.Query())
	if err != nil {
		http.Error(w, fmt.Sprintf("falha no callback OAuth: %v", err), http.StatusBadRequest)
		return
	}

	sess, _ := cookieStore.Get(r, "orbita-oauth")
	sess.Values["account_did"] = sessData.AccountDID.String()
	sess.Values["session_id"] = sessData.SessionID
	if err := sess.Save(r, w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("login bem-sucedido: %s", sessData.AccountDID.String())
	fmt.Fprintf(w, "login OK — DID: %s", sessData.AccountDID.String())
}
