package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// Beta 3: the appview's read/write surface moves from raw HTML strings to
// JSON, so a real frontend (frontend/) can be built against it instead of
// the server hand-assembling markup. The work page and its season/episode
// sub-pages are converted here; shelf reads/writes are still next.

type profileShelfEntry struct {
	Provider string `json:"provider"`
	ID       string `json:"id"`
	Title    string `json:"title"`
	Poster   string `json:"poster,omitempty"`
}

type profileNoteEntry struct {
	URI       string `json:"uri"`
	Provider  string `json:"provider"`
	ID        string `json:"id"`
	Title     string `json:"title"`
	Season    *int   `json:"season,omitempty"`
	Episode   *int   `json:"episode,omitempty"`
	Text      string `json:"text"`
	CreatedAt string `json:"createdAt"`
}

type profileResponse struct {
	DID       string              `json:"did"`
	Handle    string              `json:"handle"`
	AvatarURL string              `json:"avatarUrl,omitempty"`
	Bio       string              `json:"bio,omitempty"`
	Shelf     []profileShelfEntry `json:"shelf"`
	Notes     []profileNoteEntry  `json:"notes"`
}

type accountEntry struct {
	URI       string `json:"uri"`
	DID       string `json:"did"`
	Handle    string `json:"handle"`
	AvatarURL string `json:"avatarUrl,omitempty"`
	AddedAt   string `json:"addedAt"`
}

type shelfEntry struct {
	URI       string `json:"uri"`
	Provider  string `json:"provider"`
	ID        string `json:"id"`
	Title     string `json:"title"`
	Poster    string `json:"poster,omitempty"`
	DID       string `json:"did"`
	Handle    string `json:"handle"`
	AvatarURL string `json:"avatarUrl,omitempty"`
	AddedAt   string `json:"addedAt"`
}

type noteEntry struct {
	URI       string `json:"uri"`
	DID       string `json:"did"`
	Handle    string `json:"handle"`
	AvatarURL string `json:"avatarUrl,omitempty"`
	Text      string `json:"text"`
	CreatedAt string `json:"createdAt"`
}

// Both accounts and notes carry a handle/avatar resolved straight from the
// author's own DID document and PDS (see identity.go) — never the Bluesky
// API, so this works the same for any AT Protocol account, not just ones
// registered with Bluesky specifically.
func toAccountEntry(ctx context.Context, db *sql.DB, it ShelfItem) accountEntry {
	handle, avatar := resolveIdentity(ctx, db, it.DID)
	return accountEntry{URI: it.URI, DID: it.DID, Handle: handle, AvatarURL: avatar, AddedAt: it.CreatedAt}
}

func toNoteEntry(ctx context.Context, db *sql.DB, n Note) noteEntry {
	handle, avatar := resolveIdentity(ctx, db, n.DID)
	return noteEntry{URI: n.URI, DID: n.DID, Handle: handle, AvatarURL: avatar, Text: n.Text, CreatedAt: n.CreatedAt}
}

type workResponse struct {
	Provider string         `json:"provider"`
	ID       string         `json:"id"`
	Title    string         `json:"title"`
	Poster   string         `json:"poster,omitempty"`
	Overview string         `json:"overview,omitempty"`
	Accounts []accountEntry `json:"accounts"`
	Seasons  []season       `json:"seasons,omitempty"`
	Notes    []noteEntry    `json:"notes"`
}

type seasonResponse struct {
	Provider string    `json:"provider"`
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Season   int       `json:"season"`
	Episodes []episode `json:"episodes"`
}

type episodeResponse struct {
	Provider string      `json:"provider"`
	ID       string      `json:"id"`
	Title    string      `json:"title"`
	Season   int         `json:"season"`
	Episode  int         `json:"episode"`
	Name     string      `json:"name"`
	Overview string      `json:"overview"`
	AirDate  string      `json:"airDate"`
	StillURL string      `json:"stillUrl,omitempty"`
	Notes    []noteEntry `json:"notes"`
}

func setupAPI(mux *http.ServeMux, db *sql.DB) {
	// Tells the frontend who's signed in (if anyone), so it can offer an
	// "add to shelf" control and compare the viewer's own DID against a
	// work's account list, entirely client-side.
	mux.HandleFunc("GET /api/me", func(w http.ResponseWriter, r *http.Request) {
		did, _ := currentSessionDID(r)
		if did == nil {
			http.Error(w, "not authenticated", http.StatusUnauthorized)
			return
		}
		handle, avatar := resolveIdentity(r.Context(), db, did.String())
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"did": did.String(), "handle": handle, "avatarUrl": avatar})
	})

	mux.HandleFunc("POST /api/shelf/add", func(w http.ResponseWriter, r *http.Request) {
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

		var body struct {
			Provider string `json:"provider"`
			ID       string `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		c := oauthSess.APIClient()
		apiBody := map[string]any{
			"repo":       c.AccountDID.String(),
			"collection": "social.orbita.shelf.item",
			"record": map[string]any{
				"$type": "social.orbita.shelf.item",
				"work": map[string]any{
					"provider": body.Provider,
					"id":       body.ID,
				},
				"createdAt": syntax.DatetimeNow(),
			},
		}

		log.Printf("writing shelf.item via OAuth (DPoP), JSON API: provider=%s id=%s", body.Provider, body.ID)
		var created struct {
			URI string `json:"uri"`
		}
		if err := c.Post(ctx, "com.atproto.repo.createRecord", apiBody, &created); err != nil {
			http.Error(w, fmt.Sprintf("failed to write record: %v", err), http.StatusBadRequest)
			return
		}

		handle, avatar := resolveIdentity(ctx, db, c.AccountDID.String())
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(accountEntry{
			URI: created.URI, DID: c.AccountDID.String(), Handle: handle, AvatarURL: avatar,
			AddedAt: syntax.DatetimeNow().String(),
		})
	})

	mux.HandleFunc("POST /api/shelf/delete", func(w http.ResponseWriter, r *http.Request) {
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

		var body struct {
			URI string `json:"uri"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		atURI, err := syntax.ParseATURI(body.URI)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid AT-URI: %v", err), http.StatusBadRequest)
			return
		}

		c := oauthSess.APIClient()
		apiBody := map[string]any{
			"repo":       c.AccountDID.String(),
			"collection": atURI.Collection().String(),
			"rkey":       atURI.RecordKey().String(),
		}

		log.Printf("deleting %s via OAuth (DPoP), JSON API", body.URI)
		if err := c.Post(ctx, "com.atproto.repo.deleteRecord", apiBody, nil); err != nil {
			http.Error(w, fmt.Sprintf("failed to delete record: %v", err), http.StatusBadRequest)
			return
		}
		if err := deleteShelfItem(db, body.URI); err != nil {
			log.Printf("deleted from PDS but failed to remove local index for %s: %v", body.URI, err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	// Beta 0's "prove Tap indexed it" listing showed everyone's shelf
	// activity — reconsidered mid-Beta-4 (2026-07-17): unscoped, that page
	// serves none of the product's four core surfaces, it's not "your
	// shelf" (Beta 5, profile) or a followed-accounts feed (Beta 6). What
	// has a real purpose is your OWN shelf as a plain list, so that's what
	// this became instead — auth required, scoped to the viewer's own DID.
	mux.HandleFunc("GET /api/shelf", func(w http.ResponseWriter, r *http.Request) {
		did, _ := currentSessionDID(r)
		if did == nil {
			http.Error(w, "not authenticated", http.StatusUnauthorized)
			return
		}

		items, err := listShelfItemsByAccount(db, did.String())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		entries := make([]shelfEntry, 0, len(items))
		for _, it := range items {
			title, poster, _ := displayWork(db, it.Provider, it.WorkID)
			handle, avatar := resolveIdentity(r.Context(), db, it.DID)
			entries = append(entries, shelfEntry{
				URI: it.URI, Provider: it.Provider, ID: it.WorkID, Title: title, Poster: poster,
				DID: it.DID, Handle: handle, AvatarURL: avatar, AddedAt: it.CreatedAt,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entries)
	})

	// Beta 5: a page about a *person*, not a work — any account, not just
	// the viewer's own, reachable by handle. Only ever shows data this
	// appview already has locally (small-scale, same as everything so
	// far) — an account that's never logged in here comes back empty, not
	// an error, since it genuinely may just not have shown up yet.
	mux.HandleFunc("GET /api/profile/{handle}", func(w http.ResponseWriter, r *http.Request) {
		handleParam := r.PathValue("handle")
		did, err := resolveHandleToDID(r.Context(), handleParam)
		if err != nil {
			http.Error(w, fmt.Sprintf("could not resolve handle: %v", err), http.StatusNotFound)
			return
		}

		handle, avatar, bio, _ := resolveIdentityFull(r.Context(), db, did)

		shelfItems, err := listShelfItemsByAccount(db, did)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		shelf := make([]profileShelfEntry, 0, len(shelfItems))
		for _, it := range shelfItems {
			title, poster, _ := displayWork(db, it.Provider, it.WorkID)
			shelf = append(shelf, profileShelfEntry{Provider: it.Provider, ID: it.WorkID, Title: title, Poster: poster})
		}

		accountNotes, err := listNotesByAccount(db, did)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		notes := make([]profileNoteEntry, 0, len(accountNotes))
		for _, n := range accountNotes {
			title, _, _ := displayWork(db, n.Provider, n.WorkID)
			notes = append(notes, profileNoteEntry{
				URI: n.URI, Provider: n.Provider, ID: n.WorkID, Title: title,
				Season: n.Season, Episode: n.Episode, Text: n.Text, CreatedAt: n.CreatedAt,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(profileResponse{
			DID: did, Handle: handle, AvatarURL: avatar, Bio: bio, Shelf: shelf, Notes: notes,
		})
	})

	mux.HandleFunc("GET /api/search", func(w http.ResponseWriter, r *http.Request) {
		did, _ := currentSessionDID(r)
		if did == nil {
			http.Error(w, "not authenticated", http.StatusUnauthorized)
			return
		}

		query := r.URL.Query().Get("q")
		if query == "" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]searchResult{})
			return
		}

		results, err := searchTMDB(query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	})

	mux.HandleFunc("GET /api/works/{provider}/{id}", func(w http.ResponseWriter, r *http.Request) {
		provider := r.PathValue("provider")
		id := r.PathValue("id")

		items, err := listShelfItemsByWork(db, provider, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		accounts := make([]accountEntry, 0, len(items))
		for _, it := range items {
			accounts = append(accounts, toAccountEntry(r.Context(), db, it))
		}

		notes, err := listNotes(db, provider, id, nil, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		noteEntries := make([]noteEntry, 0, len(notes))
		for _, n := range notes {
			noteEntries = append(noteEntries, toNoteEntry(r.Context(), db, n))
		}

		title, poster, overview := displayWork(db, provider, id)

		resp := workResponse{
			Provider: provider,
			ID:       id,
			Title:    title,
			Poster:   poster,
			Overview: overview,
			Accounts: accounts,
			Notes:    noteEntries,
		}
		if provider == "tmdb-tv" {
			resp.Seasons = displaySeasons(db, provider, id)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("GET /api/works/{provider}/{id}/season/{season}", func(w http.ResponseWriter, r *http.Request) {
		provider := r.PathValue("provider")
		id := r.PathValue("id")
		seasonNum, err := strconv.Atoi(r.PathValue("season"))
		if err != nil {
			http.Error(w, "invalid season number", http.StatusBadRequest)
			return
		}

		title, _, _ := displayWork(db, provider, id)
		episodes := displayEpisodes(db, provider, id, seasonNum)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(seasonResponse{
			Provider: provider,
			ID:       id,
			Title:    title,
			Season:   seasonNum,
			Episodes: episodes,
		})
	})

	mux.HandleFunc("GET /api/works/{provider}/{id}/season/{season}/episode/{episode}", func(w http.ResponseWriter, r *http.Request) {
		provider := r.PathValue("provider")
		id := r.PathValue("id")
		seasonNum, err := strconv.Atoi(r.PathValue("season"))
		if err != nil {
			http.Error(w, "invalid season number", http.StatusBadRequest)
			return
		}
		episodeNum, err := strconv.Atoi(r.PathValue("episode"))
		if err != nil {
			http.Error(w, "invalid episode number", http.StatusBadRequest)
			return
		}

		title, _, _ := displayWork(db, provider, id)
		episodes := displayEpisodes(db, provider, id, seasonNum)

		var ep *episode
		for i := range episodes {
			if episodes[i].Number == episodeNum {
				ep = &episodes[i]
				break
			}
		}
		if ep == nil {
			http.Error(w, "episode not found", http.StatusNotFound)
			return
		}

		notes, err := listNotes(db, provider, id, &seasonNum, &episodeNum)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		noteEntries := make([]noteEntry, 0, len(notes))
		for _, n := range notes {
			noteEntries = append(noteEntries, toNoteEntry(r.Context(), db, n))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(episodeResponse{
			Provider: provider,
			ID:       id,
			Title:    title,
			Season:   seasonNum,
			Episode:  episodeNum,
			Name:     ep.Name,
			Overview: ep.Overview,
			AirDate:  ep.AirDate,
			StillURL: ep.StillURL,
			Notes:    noteEntries,
		})
	})

	mux.HandleFunc("POST /api/notes/add", func(w http.ResponseWriter, r *http.Request) {
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

		var body struct {
			Provider string `json:"provider"`
			ID       string `json:"id"`
			Season   *int   `json:"season"`
			Episode  *int   `json:"episode"`
			Text     string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		createdAt := syntax.DatetimeNow()
		record := map[string]any{
			"$type": "social.orbita.note",
			"work": map[string]any{
				"provider": body.Provider,
				"id":       body.ID,
			},
			"text":      body.Text,
			"createdAt": createdAt.String(),
		}
		if body.Season != nil {
			record["season"] = *body.Season
		}
		if body.Episode != nil {
			record["episode"] = *body.Episode
		}

		c := oauthSess.APIClient()
		apiBody := map[string]any{
			"repo":       c.AccountDID.String(),
			"collection": "social.orbita.note",
			"record":     record,
		}

		log.Printf("writing note via OAuth (DPoP), JSON API: provider=%s id=%s", body.Provider, body.ID)
		var created struct {
			URI string `json:"uri"`
		}
		if err := c.Post(ctx, "com.atproto.repo.createRecord", apiBody, &created); err != nil {
			http.Error(w, fmt.Sprintf("failed to write note: %v", err), http.StatusBadRequest)
			return
		}

		// Returned directly instead of making the frontend re-fetch the
		// notes list: the write above already succeeded against the real
		// PDS (the source of truth), but the local SQLite index only
		// catches up once Tap's webhook delivers the event — an
		// eventual-consistency lag that would otherwise make a freshly
		// posted note briefly invisible.
		handle, avatar := resolveIdentity(ctx, db, c.AccountDID.String())
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(noteEntry{
			URI:       created.URI,
			DID:       c.AccountDID.String(),
			Handle:    handle,
			AvatarURL: avatar,
			Text:      body.Text,
			CreatedAt: createdAt.String(),
		})
	})
}
