package main

import (
	"database/sql"
	"fmt"
	"html"
	"log"
	"net/http"
	"strconv"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// Beta 2's write path: mirrors shelf.go's pattern exactly (same
// currentSessionDID, same ResumeSession + APIClient().Post shape),
// against the second collection. season/episode are optional in the
// Lexicon, so they're only added to the record map when actually
// provided — never sent as a bare 0.
const notesSchema = `
CREATE TABLE IF NOT EXISTS notes (
	uri        TEXT PRIMARY KEY,
	cid        TEXT NOT NULL,
	did        TEXT NOT NULL,
	provider   TEXT NOT NULL,
	work_id    TEXT NOT NULL,
	season     INTEGER,
	episode    INTEGER,
	text       TEXT NOT NULL,
	created_at TEXT NOT NULL,
	indexed_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
`

type Note struct {
	URI       string
	DID       string
	Text      string
	CreatedAt string
}

func insertNote(db *sql.DB, uri, cid, did, provider, workID string, season, episode *int, text, createdAt string) error {
	_, err := db.Exec(
		`INSERT INTO notes (uri, cid, did, provider, work_id, season, episode, text, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(uri) DO NOTHING`,
		uri, cid, did, provider, workID, season, episode, text, createdAt,
	)
	return err
}

// listNotes covers both cases with the same query shape: season == nil
// means a whole-work note (matches rows where season/episode are both
// NULL), season != nil means a specific episode (matches an exact pair).
// Season-only notes (a note about a season as a whole, no episode) are
// deliberately not wired up yet — the Lexicon already allows it, the UI
// just doesn't ask for it, same "don't build what wasn't asked for" bar
// as everything else here.
func listNotes(db *sql.DB, provider, workID string, season, episode *int) ([]Note, error) {
	var rows *sql.Rows
	var err error
	if season == nil {
		rows, err = db.Query(
			`SELECT uri, did, text, created_at FROM notes
			 WHERE provider = ? AND work_id = ? AND season IS NULL AND episode IS NULL
			 ORDER BY created_at ASC`,
			provider, workID)
	} else {
		rows, err = db.Query(
			`SELECT uri, did, text, created_at FROM notes
			 WHERE provider = ? AND work_id = ? AND season = ? AND episode = ?
			 ORDER BY created_at ASC`,
			provider, workID, *season, *episode)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.URI, &n.DID, &n.Text, &n.CreatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

func setupNotes(mux *http.ServeMux, db *sql.DB) {
	mux.HandleFunc("POST /notes/add", func(w http.ResponseWriter, r *http.Request) {
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
		text := r.PostFormValue("text")

		record := map[string]any{
			"$type": "social.orbita.note",
			"work": map[string]any{
				"provider": provider,
				"id":       workID,
			},
			"text":      text,
			"createdAt": syntax.DatetimeNow(),
		}
		// Only attach season/episode if this note came from an episode
		// page — the Lexicon leaves both optional for exactly this reason.
		if s := r.PostFormValue("season"); s != "" {
			if n, err := strconv.Atoi(s); err == nil {
				record["season"] = n
			}
		}
		if e := r.PostFormValue("episode"); e != "" {
			if n, err := strconv.Atoi(e); err == nil {
				record["episode"] = n
			}
		}

		c := oauthSess.APIClient()
		body := map[string]any{
			"repo":       c.AccountDID.String(),
			"collection": "social.orbita.note",
			"record":     record,
		}

		log.Printf("writing note via OAuth (DPoP): provider=%s id=%s season=%s episode=%s",
			provider, workID, r.PostFormValue("season"), r.PostFormValue("episode"))
		if err := c.Post(ctx, "com.atproto.repo.createRecord", body, nil); err != nil {
			http.Error(w, fmt.Sprintf("failed to write note: %v", err), http.StatusBadRequest)
			return
		}

		redirectTo := fmt.Sprintf("/works/%s/%s", provider, workID)
		if s := r.PostFormValue("season"); s != "" {
			redirectTo += "/season/" + s
			if e := r.PostFormValue("episode"); e != "" {
				redirectTo += "/episode/" + e
			}
		}
		http.Redirect(w, r, redirectTo, http.StatusFound)
	})
}

// renderNotesSection is called both from works.go's work handler (season
// == nil, a note about the work as a whole) and its episode handler
// (season/episode set). The hidden season/episode inputs are only
// rendered when set — sending literal "0" strings would be
// indistinguishable from a real season/episode 0 to /notes/add's
// "was this field provided at all" check.
func renderNotesSection(w http.ResponseWriter, db *sql.DB, provider, workID string, season, episode *int) {
	notes, err := listNotes(db, provider, workID, season, episode)
	if err != nil {
		fmt.Fprintf(w, "<p>could not load notes: %s</p>", html.EscapeString(err.Error()))
		return
	}

	label := "Notes"
	if season != nil {
		label = fmt.Sprintf("Notes for S%dE%d", *season, *episode)
	}
	fmt.Fprintf(w, "<h2>%s</h2><ul>", label)
	for _, n := range notes {
		fmt.Fprintf(w, "<li>%s — %s (%s)</li>", html.EscapeString(n.Text), n.DID, n.CreatedAt)
	}
	if len(notes) == 0 {
		fmt.Fprint(w, "<li>no notes yet</li>")
	}
	fmt.Fprint(w, "</ul>")

	hiddenFields := ""
	if season != nil {
		hiddenFields = fmt.Sprintf(
			`<input type="hidden" name="season" value="%d"><input type="hidden" name="episode" value="%d">`,
			*season, *episode)
	}
	fmt.Fprintf(w, `<form method="POST" action="/notes/add">
  <input type="hidden" name="provider" value="%s">
  <input type="hidden" name="id" value="%s">
  %s
  <textarea name="text" placeholder="write a note..."></textarea>
  <button type="submit">Add note</button>
</form>`, html.EscapeString(provider), html.EscapeString(workID), hiddenFields)
}
