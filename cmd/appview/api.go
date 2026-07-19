package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
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

type feedNoteEntry struct {
	URI              string      `json:"uri"`
	CID              string      `json:"cid"`
	DID              string      `json:"did"`
	Handle           string      `json:"handle"`
	AvatarURL        string      `json:"avatarUrl,omitempty"`
	Provider         string      `json:"provider"`
	ID               string      `json:"id"`
	Title            string      `json:"title"`
	Poster           string      `json:"poster,omitempty"`
	Season           *int        `json:"season,omitempty"`
	Episode          *int        `json:"episode,omitempty"`
	Text             string      `json:"text"`
	CreatedAt        string      `json:"createdAt"`
	Replies          []noteEntry `json:"replies,omitempty"`
	RepostedByHandle string      `json:"repostedByHandle,omitempty"`
	RepostedAt       string      `json:"repostedAt,omitempty"`
}

// buildFeedEntry resolves everything a feed card needs (author identity,
// work title/poster, direct replies) for one note. repostedByHandle/At are
// only set when this entry reached the feed via a repost, not by the
// original author being followed/on-shelf directly.
func buildFeedEntry(ctx context.Context, db *sql.DB, n FeedNote, repostedByHandle, repostedAt string) feedNoteEntry {
	handle, avatar := resolveIdentity(ctx, db, n.DID)
	title, poster, _ := displayWork(db, n.Provider, n.WorkID)
	entry := feedNoteEntry{
		URI: n.URI, CID: n.CID, DID: n.DID, Handle: handle, AvatarURL: avatar,
		Provider: n.Provider, ID: n.WorkID, Title: title, Poster: poster,
		Season: n.Season, Episode: n.Episode, Text: n.Text, CreatedAt: n.CreatedAt,
		RepostedByHandle: repostedByHandle, RepostedAt: repostedAt,
	}
	if replies, err := listReplies(db, n.URI); err == nil {
		for _, rep := range replies {
			entry.Replies = append(entry.Replies, toReplyEntry(ctx, db, rep))
		}
	}
	return entry
}

// feedSortKey is what merges notes-by-followed-authors and
// reposts-by-followed-accounts into one chronological list: a repost
// sorts by when it was shared, not when the original note was written —
// that's when it actually showed up for the viewer.
func feedSortKey(e feedNoteEntry) string {
	if e.RepostedAt != "" {
		return e.RepostedAt
	}
	return e.CreatedAt
}

// Beta 7: nooks are the primary way a shelf is organized and shown to
// visitors — the profile groups by nook, not a single flat grid anymore.
// Unsorted is the honest catch-all: works on the shelf that aren't in any
// nook yet, never hidden or forced into a default grouping.
type nookEntry struct {
	URI         string              `json:"uri"`
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Theme       string              `json:"theme"`
	Order       *int                `json:"order,omitempty"`
	Works       []profileShelfEntry `json:"works"`
}

// buildNookEntry resolves a stored Nook's works into the title/poster shape
// the frontend renders — shared by the profile listing and the single-nook
// detail endpoint so both stay in sync.
func buildNookEntry(db *sql.DB, n Nook) nookEntry {
	works := make([]profileShelfEntry, 0, len(n.Works))
	for _, w := range n.Works {
		title, poster, _ := displayWork(db, w.Provider, w.WorkID)
		works = append(works, profileShelfEntry{Provider: w.Provider, ID: w.WorkID, Title: title, Poster: poster})
	}
	return nookEntry{URI: n.URI, Name: n.Name, Description: n.Description, Theme: n.Theme, Order: n.Order, Works: works}
}

// nookDetailResponse is a single nook plus enough of its owner's identity to
// credit the card ("<handle>'s nook") without a second round-trip.
type nookDetailResponse struct {
	nookEntry
	OwnerHandle string `json:"ownerHandle"`
	OwnerAvatar string `json:"ownerAvatar,omitempty"`
}

type profileResponse struct {
	DID       string              `json:"did"`
	Handle    string              `json:"handle"`
	AvatarURL string              `json:"avatarUrl,omitempty"`
	Bio       string              `json:"bio,omitempty"`
	Nooks     []nookEntry         `json:"nooks"`
	Unsorted  []profileShelfEntry `json:"unsorted"`
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
	URI       string      `json:"uri"`
	CID       string      `json:"cid"`
	DID       string      `json:"did"`
	Handle    string      `json:"handle"`
	AvatarURL string      `json:"avatarUrl,omitempty"`
	Text      string      `json:"text"`
	CreatedAt string      `json:"createdAt"`
	Replies   []noteEntry `json:"replies,omitempty"`
}

// Both accounts and notes carry a handle/avatar resolved straight from the
// author's own DID document and PDS (see identity.go) — never the Bluesky
// API, so this works the same for any AT Protocol account, not just ones
// registered with Bluesky specifically.
func toAccountEntry(ctx context.Context, db *sql.DB, it ShelfItem) accountEntry {
	handle, avatar := resolveIdentity(ctx, db, it.DID)
	return accountEntry{URI: it.URI, DID: it.DID, Handle: handle, AvatarURL: avatar, AddedAt: it.CreatedAt}
}

// toReplyEntry renders a note with no further reply-fetching — the
// building block for one level of nesting, used both directly and inside
// toNoteEntry below.
func toReplyEntry(ctx context.Context, db *sql.DB, n Note) noteEntry {
	handle, avatar := resolveIdentity(ctx, db, n.DID)
	return noteEntry{URI: n.URI, CID: n.CID, DID: n.DID, Handle: handle, AvatarURL: avatar, Text: n.Text, CreatedAt: n.CreatedAt}
}

// toNoteEntry additionally fetches direct replies (one level deep, see
// listReplies) — a reply's own replies aren't fetched here, keeping this
// a bounded, non-recursive query regardless of how deep a thread actually
// goes at the data layer.
func toNoteEntry(ctx context.Context, db *sql.DB, n Note) noteEntry {
	entry := toReplyEntry(ctx, db, n)
	if replies, err := listReplies(db, n.URI); err == nil {
		for _, rep := range replies {
			entry.Replies = append(entry.Replies, toReplyEntry(ctx, db, rep))
		}
	}
	return entry
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

		// Idempotent by (did, provider, id) — a repeat "+ Add to shelf"
		// for something already there returns the existing record instead
		// of minting a second one. The protocol has no uniqueness
		// constraint on record content, only on URI, so this check is the
		// only thing that ever stood between here and a real duplicate.
		if existing, err := getShelfItem(db, did.String(), body.Provider, body.ID); err == nil && existing != nil {
			handle, avatar := resolveIdentity(ctx, db, did.String())
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(accountEntry{
				URI: existing.URI, DID: did.String(), Handle: handle, AvatarURL: avatar,
				AddedAt: existing.CreatedAt,
			})
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

	// Beta 7: nooks. A nook's "edit" (rename, reorder, add/remove a work)
	// is a whole-record replacement (com.atproto.repo.putRecord) — there's
	// no separate membership record type, so editing just means resending
	// the full desired works array. shelfItemExists is the guard the
	// Lexicon itself can't express: only works already on the author's
	// own shelf may go into a nook.
	type nookRequestBody struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Theme       string `json:"theme"`
		Order       *int   `json:"order"`
		Works       []struct {
			Provider string `json:"provider"`
			ID       string `json:"id"`
		} `json:"works"`
	}

	buildNookRecord := func(did, createdAt string, order *int, body nookRequestBody) (map[string]any, []WorkRef, error) {
		theme := body.Theme
		if theme == "" {
			theme = "default"
		}
		works := make([]WorkRef, 0, len(body.Works))
		workRefs := make([]map[string]any, 0, len(body.Works))
		seen := make(map[WorkRef]bool, len(body.Works))
		for _, w := range body.Works {
			if !shelfItemExists(db, did, w.Provider, w.ID) {
				return nil, nil, fmt.Errorf("%s/%s is not on your shelf", w.Provider, w.ID)
			}
			ref := WorkRef{Provider: w.Provider, WorkID: w.ID}
			if seen[ref] {
				continue // a work can only appear once per nook — silently collapsed, not an error
			}
			seen[ref] = true
			works = append(works, ref)
			workRefs = append(workRefs, map[string]any{"provider": w.Provider, "id": w.ID})
		}
		record := map[string]any{
			"$type":       "social.orbita.shelf.nook",
			"name":        body.Name,
			"description": body.Description,
			"works":       workRefs,
			"style":       map[string]any{"theme": theme},
			"createdAt":   createdAt,
		}
		if order != nil {
			record["order"] = *order
		}
		return record, works, nil
	}

	toNookEntry := func(uri, name, description, theme string, order *int, works []WorkRef) nookEntry {
		workEntries := make([]profileShelfEntry, 0, len(works))
		for _, w := range works {
			title, poster, _ := displayWork(db, w.Provider, w.WorkID)
			workEntries = append(workEntries, profileShelfEntry{Provider: w.Provider, ID: w.WorkID, Title: title, Poster: poster})
		}
		return nookEntry{URI: uri, Name: name, Description: description, Theme: theme, Order: order, Works: workEntries}
	}

	mux.HandleFunc("POST /api/nooks", func(w http.ResponseWriter, r *http.Request) {
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

		var body nookRequestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		createdAt := syntax.DatetimeNow().String()
		order := body.Order
		if order == nil {
			next := nextNookOrder(db, did.String())
			order = &next
		}
		record, works, err := buildNookRecord(did.String(), createdAt, order, body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		c := oauthSess.APIClient()
		apiBody := map[string]any{
			"repo":       c.AccountDID.String(),
			"collection": "social.orbita.shelf.nook",
			"record":     record,
		}
		log.Printf("writing nook via OAuth (DPoP), JSON API: name=%q, %d work(s)", body.Name, len(works))
		var created struct {
			URI string `json:"uri"`
		}
		if err := c.Post(ctx, "com.atproto.repo.createRecord", apiBody, &created); err != nil {
			http.Error(w, fmt.Sprintf("failed to write nook: %v", err), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(toNookEntry(created.URI, body.Name, body.Description, record["style"].(map[string]any)["theme"].(string), order, works))
	})

	mux.HandleFunc("PUT /api/nooks/{rkey}", func(w http.ResponseWriter, r *http.Request) {
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

		rkey := r.PathValue("rkey")
		uri := fmt.Sprintf("at://%s/social.orbita.shelf.nook/%s", did.String(), rkey)

		var body nookRequestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// The original createdAt is preserved across edits — this is when
		// the nook was made, not when it was last touched. Same for order,
		// unless this request is specifically a reorder (body.Order set):
		// most edits (rename, add/remove a work) shouldn't silently bump a
		// nook to a new position. Both fall back to fresh values only if
		// the local index hasn't caught up with this nook yet.
		createdAt := syntax.DatetimeNow().String()
		order := body.Order
		if existing, existErr := listNooksByAccount(db, did.String()); existErr == nil {
			for _, n := range existing {
				if n.URI == uri {
					createdAt = n.CreatedAt
					if order == nil {
						order = n.Order
					}
					break
				}
			}
		}

		record, works, err := buildNookRecord(did.String(), createdAt, order, body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		c := oauthSess.APIClient()
		apiBody := map[string]any{
			"repo":       c.AccountDID.String(),
			"collection": "social.orbita.shelf.nook",
			"rkey":       rkey,
			"record":     record,
		}
		log.Printf("updating nook via OAuth (DPoP), JSON API: uri=%s", uri)
		if err := c.Post(ctx, "com.atproto.repo.putRecord", apiBody, nil); err != nil {
			http.Error(w, fmt.Sprintf("failed to update nook: %v", err), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(toNookEntry(uri, body.Name, body.Description, record["style"].(map[string]any)["theme"].(string), order, works))
	})

	mux.HandleFunc("POST /api/nooks/delete", func(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("deleting nook %s via OAuth (DPoP), JSON API", body.URI)
		if err := c.Post(ctx, "com.atproto.repo.deleteRecord", apiBody, nil); err != nil {
			http.Error(w, fmt.Sprintf("failed to delete nook: %v", err), http.StatusBadRequest)
			return
		}
		if err := deleteNook(db, body.URI); err != nil {
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

		rawNooks, err := listNooksByAccount(db, did)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		nooks := make([]nookEntry, 0, len(rawNooks))
		for _, n := range rawNooks {
			nooks = append(nooks, buildNookEntry(db, n))
		}

		unsortedItems, err := listUnsortedShelfItems(db, did)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		unsorted := make([]profileShelfEntry, 0, len(unsortedItems))
		for _, it := range unsortedItems {
			title, poster, _ := displayWork(db, it.Provider, it.WorkID)
			unsorted = append(unsorted, profileShelfEntry{Provider: it.Provider, ID: it.WorkID, Title: title, Poster: poster})
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
			DID: did, Handle: handle, AvatarURL: avatar, Bio: bio, Nooks: nooks, Unsorted: unsorted, Notes: notes,
		})
	})

	// Beta 7: a single nook, addressable on its own — what the shareable
	// nook page (nookpage.go) renders both server-side (for OG tags) and
	// client-side (for the interactive card). ownerHandle/ownerAvatar ride
	// along so the card can credit whose taste this is without a second
	// request.
	mux.HandleFunc("GET /api/profile/{handle}/nook/{rkey}", func(w http.ResponseWriter, r *http.Request) {
		handleParam := r.PathValue("handle")
		rkey := r.PathValue("rkey")
		did, err := resolveHandleToDID(r.Context(), handleParam)
		if err != nil {
			http.Error(w, fmt.Sprintf("could not resolve handle: %v", err), http.StatusNotFound)
			return
		}

		ownerHandle, ownerAvatar := resolveIdentity(r.Context(), db, did)
		uri := fmt.Sprintf("at://%s/social.orbita.shelf.nook/%s", did, rkey)

		n, err := getNook(db, uri)
		if err != nil {
			http.Error(w, "nook not found", http.StatusNotFound)
			return
		}

		entry := buildNookEntry(db, *n)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(nookDetailResponse{
			nookEntry:   entry,
			OwnerHandle: ownerHandle,
			OwnerAvatar: ownerAvatar,
		})
	})

	// Beta 6: chronological, deterministic, no ranking — the product's own
	// non-negotiable shape for any feed. Two tabs for now: "shelf" (the
	// main one — notes from *anyone* about works on the viewer's own
	// shelf, obra-first) and "following" (notes from people the viewer
	// follows, plus notes anyone followed reposted — reusing the existing
	// Bluesky follow graph instead of inventing a parallel one). A third
	// tab, "affinity," belongs here too but needs the Jaccard computation
	// that doesn't exist yet (Beta 13) — the frontend shows it as an
	// honest placeholder rather than this endpoint faking a response for
	// it. Only ever pulls from social.orbita.note — forum comments,
	// whenever they exist, are deliberately not feed material. Scoped,
	// for now, to accounts that have already used this appview — real
	// fan-out is Beta 11.
	mux.HandleFunc("GET /api/feed", func(w http.ResponseWriter, r *http.Request) {
		did, _ := currentSessionDID(r)
		if did == nil {
			http.Error(w, "not authenticated", http.StatusUnauthorized)
			return
		}

		tab := r.URL.Query().Get("tab")
		if tab == "" {
			tab = "shelf"
		}

		var entries []feedNoteEntry
		switch tab {
		case "shelf":
			shelfItems, shelfErr := listShelfItemsByAccount(db, did.String())
			if shelfErr != nil {
				http.Error(w, shelfErr.Error(), http.StatusInternalServerError)
				return
			}
			notes, err := listNotesByWorks(db, shelfItems, 50)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			for _, n := range notes {
				entries = append(entries, buildFeedEntry(r.Context(), db, n, "", ""))
			}

		case "following":
			pdsURL, pdsErr := resolvePDSURL(r.Context(), did.String())
			if pdsErr != nil {
				http.Error(w, fmt.Sprintf("could not resolve your PDS: %v", pdsErr), http.StatusInternalServerError)
				return
			}
			followedDIDs, followErr := fetchFollowedDIDs(r.Context(), pdsURL, did.String())
			if followErr != nil {
				http.Error(w, fmt.Sprintf("could not read your follows: %v", followErr), http.StatusInternalServerError)
				return
			}

			notes, err := listNotesByDIDs(db, followedDIDs, 50)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			for _, n := range notes {
				entries = append(entries, buildFeedEntry(r.Context(), db, n, "", ""))
			}

			// Reposts by anyone followed surface the original note too —
			// from *anyone*, not just other followed accounts, since
			// that's the whole point: a friend found it worth sharing.
			// Sort key for these is the repost's own time, not the
			// note's — that's when it actually showed up for the viewer.
			reposts, repostErr := listRepostsByDIDs(db, followedDIDs, 50)
			if repostErr != nil {
				http.Error(w, repostErr.Error(), http.StatusInternalServerError)
				return
			}
			for _, rp := range reposts {
				n, noteErr := getNoteByURI(db, rp.SubjectURI)
				if noteErr != nil {
					continue // the reposted note isn't indexed locally — skip, don't break the feed
				}
				reposterHandle, _ := resolveIdentity(r.Context(), db, rp.DID)
				entries = append(entries, buildFeedEntry(r.Context(), db, *n, reposterHandle, rp.CreatedAt))
			}

			sort.Slice(entries, func(i, j int) bool {
				return feedSortKey(entries[i]) > feedSortKey(entries[j])
			})
			if len(entries) > 50 {
				entries = entries[:50]
			}

		default:
			http.Error(w, fmt.Sprintf("unknown feed tab %q", tab), http.StatusBadRequest)
			return
		}

		if entries == nil {
			entries = []feedNoteEntry{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"notes": entries})
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
			ReplyTo  *struct {
				URI string `json:"uri"`
				CID string `json:"cid"`
			} `json:"replyTo"`
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
		if body.ReplyTo != nil {
			// A whole thread shares one root: if the note being replied to
			// is itself a reply, its own root is reused rather than
			// starting a new one — matching AT Protocol's own reply
			// convention (app.bsky.feed.post does the same).
			_, rootURI, rootCID, rootErr := noteRootRef(db, body.ReplyTo.URI)
			if rootErr != nil {
				http.Error(w, fmt.Sprintf("could not find the note being replied to: %v", rootErr), http.StatusBadRequest)
				return
			}
			record["reply"] = map[string]any{
				"root":   map[string]any{"uri": rootURI, "cid": rootCID},
				"parent": map[string]any{"uri": body.ReplyTo.URI, "cid": body.ReplyTo.CID},
			}
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
			CID string `json:"cid"`
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
			CID:       created.CID,
			DID:       c.AccountDID.String(),
			Handle:    handle,
			AvatarURL: avatar,
			Text:      body.Text,
			CreatedAt: createdAt.String(),
		})
	})

	// A repost is its own record (social.orbita.repost), not a flag on the
	// note — see reposts.go. No count is ever computed or returned here;
	// this only ever exists so the note surfaces in the reposter's
	// followers' Following feed, attributed to who shared it.
	mux.HandleFunc("POST /api/notes/repost", func(w http.ResponseWriter, r *http.Request) {
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
			CID string `json:"cid"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		c := oauthSess.APIClient()
		apiBody := map[string]any{
			"repo":       c.AccountDID.String(),
			"collection": "social.orbita.repost",
			"record": map[string]any{
				"$type":     "social.orbita.repost",
				"subject":   map[string]any{"uri": body.URI, "cid": body.CID},
				"createdAt": syntax.DatetimeNow(),
			},
		}

		log.Printf("writing repost via OAuth (DPoP), JSON API: subject=%s", body.URI)
		if err := c.Post(ctx, "com.atproto.repo.createRecord", apiBody, nil); err != nil {
			http.Error(w, fmt.Sprintf("failed to write repost: %v", err), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
}
