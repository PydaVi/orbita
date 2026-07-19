package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
)

// Beta 8: the constellation and its archetype, reimagined for this
// product's own shape rather than ported wholesale from earlier work. The
// key difference: this appview never gained a genre/tag pipeline (the
// catalog only ever cached title/poster/year/overview — see tmdb.go), so
// there's no shared vocabulary to hash into cross-profile anchor
// positions the way tags did elsewhere. What this product *does* have,
// uniquely, is nooks — a small, curated, shared vocabulary of themes
// (style.theme's own knownValues) that's the primary way a shelf gets
// organized here (Beta 7). So the constellation is anchored on **theme**,
// not genre: every "warm" nook across every account lands in the same
// region of everyone's sky, which is the entire point of an anchor system
// — comparable taste lights up the same place, regardless of what any
// individual nook happens to be named. Provider (medium) and decade ride
// along as secondary, weaker pulls. All the actual physics/layout math
// lives client-side in constellation.js, matching how this project's
// force layout has always been client-rendered elsewhere on this page —
// this handler only assembles the raw graph: one node per work, tagged
// with its nook's theme (or none, for Unsorted) and how many notes exist
// for it.
type constellationNode struct {
	Provider  string `json:"provider"`
	ID        string `json:"id"`
	Title     string `json:"title"`
	Poster    string `json:"poster,omitempty"`
	Year      string `json:"year,omitempty"`
	NookURI   string `json:"nookUri,omitempty"`
	NookName  string `json:"nookName,omitempty"`
	Theme     string `json:"theme,omitempty"` // empty = not in any nook
	NoteCount int    `json:"noteCount"`
}

type constellationResponse struct {
	Handle string              `json:"handle"`
	Nodes  []constellationNode `json:"nodes"`
}

func setupConstellation(mux *http.ServeMux, db *sql.DB) {
	mux.HandleFunc("GET /api/profile/{handle}/constellation", func(w http.ResponseWriter, r *http.Request) {
		handle := r.PathValue("handle")
		did, err := resolveHandleToDID(r.Context(), handle)
		if err != nil {
			http.Error(w, fmt.Sprintf("could not resolve handle: %v", err), http.StatusNotFound)
			return
		}

		noteCounts, err := noteCountsByWork(db, did)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		rawNooks, err := listNooksByAccount(db, did)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var nodes []constellationNode
		seen := make(map[string]bool)
		for _, n := range rawNooks {
			for _, work := range n.Works {
				key := work.Provider + "/" + work.WorkID
				if seen[key] {
					continue // a work in more than one nook still lights up as a single star
				}
				seen[key] = true
				title, poster, year, _ := displayWorkFull(db, work.Provider, work.WorkID)
				nodes = append(nodes, constellationNode{
					Provider: work.Provider, ID: work.WorkID, Title: title, Poster: poster, Year: year,
					NookURI: n.URI, NookName: n.Name, Theme: n.Theme,
					NoteCount: noteCounts[key],
				})
			}
		}

		unsorted, err := listUnsortedShelfItems(db, did)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, it := range unsorted {
			key := it.Provider + "/" + it.WorkID
			if seen[key] {
				continue
			}
			seen[key] = true
			title, poster, year, _ := displayWorkFull(db, it.Provider, it.WorkID)
			nodes = append(nodes, constellationNode{
				Provider: it.Provider, ID: it.WorkID, Title: title, Poster: poster, Year: year,
				NoteCount: noteCounts[key],
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(constellationResponse{Handle: handle, Nodes: nodes})
	})
}

// noteCountsByWork collapses this account's notes (thread replies and all)
// down to "how many notes exist for this work" — the constellation only
// cares about a work having a voice attached to it, not the notes
// themselves.
func noteCountsByWork(db *sql.DB, did string) (map[string]int, error) {
	notes, err := listNotesByAccount(db, did)
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int)
	for _, n := range notes {
		counts[n.Provider+"/"+n.WorkID]++
	}
	return counts, nil
}
