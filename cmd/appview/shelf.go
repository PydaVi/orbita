package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// Substitui o curl manual: escreve social.orbita.shelf.item de verdade,
// autenticado via sessão OAuth (DPoP, PAR, PKCE já resolvidos pela lib
// no login — aqui só reaproveitamos a sessão salva).
func setupShelf(mux *http.ServeMux) {
	mux.HandleFunc("GET /shelf/add", handleShelfAddForm)
	mux.HandleFunc("POST /shelf/add", handleShelfAdd)
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
		http.Error(w, "não autenticado — faça login em /oauth/login primeiro", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!doctype html>
<p>Logado como %s</p>
<form method="POST" action="/shelf/add">
  <label>Provider: <input name="provider" value="tmdb-movie"></label>
  <label>ID (ex: 603 = Matrix na TMDB): <input name="id" placeholder="603"></label>
  <button type="submit">Adicionar à estante</button>
</form>`, did.String())
}

func handleShelfAdd(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	did, sessionID := currentSessionDID(r)
	if did == nil {
		http.Error(w, "não autenticado", http.StatusUnauthorized)
		return
	}

	oauthSess, err := oauthClient.ResumeSession(ctx, *did, sessionID)
	if err != nil {
		http.Error(w, fmt.Sprintf("sessão inválida, faça login de novo: %v", err), http.StatusUnauthorized)
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

	log.Printf("escrevendo shelf.item via OAuth (DPoP): provider=%s id=%s", provider, workID)
	if err := c.Post(ctx, "com.atproto.repo.createRecord", body, nil); err != nil {
		http.Error(w, fmt.Sprintf("falha ao escrever registro: %v", err), http.StatusBadRequest)
		return
	}

	fmt.Fprintf(w, "adicionado à estante: %s/%s — confira no seu PDS real", provider, workID)
}
