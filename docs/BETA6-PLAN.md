# Beta 6 — planning draft

**Status: Beta 6 done ✅** (2026-07-18). Feed exists with its real three-tab shape — Shelf (main), Following, Affinity (honest placeholder) — not the single "Following" tab first sketched.

## Goal

There was no page yet that was actually useful to check day-to-day — everything lived on a single work's or person's page. A feed turns this from "a place to look something up" into "a place to come back to."

Corrected mid-build (2026-07-18): the roadmap sketch only named a single "notes from people you follow" feed. The author caught this — this product's real feed shape (already established in earlier product work) is **three tabs**: Shelf (notes from *anyone* about works on your own shelf — obra-first, the main tab), Following (notes from people you follow), and Affinity (people with similar taste, via Jaccard). Built to that shape instead.

## Scope

1. **Shelf tab — the main one.** ✅ **Built.** `GET /api/feed?tab=shelf` (also the default with no `tab` param) pulls notes from *anyone* about works on the viewer's own shelf — [`notes.go`](../cmd/appview/notes.go)'s new `listNotesByWorks`, matched against `listShelfItemsByAccount` (already existed for `/shelf`). Obra-first: this tab isn't organized around who wrote something, it's organized around what you already care about.
2. **Following tab.** ✅ **Built.** `GET /api/feed?tab=following` reuses the existing Bluesky follow graph (`app.bsky.graph.follow`) instead of inventing a parallel "follow" concept — read straight from the viewer's own PDS ([`follows.go`](../cmd/appview/follows.go)'s `fetchFollowedDIDs`, paginated), the same "read a public collection from someone's own repo" shape already used for avatar/bio. `identity.go` gained `resolvePDSURL` to get there.
3. **Affinity tab.** ✅ **Built as an honest placeholder.** No Jaccard computation exists yet (Beta 13) — the frontend shows "not built yet" and never calls the API for this tab, rather than the backend faking an empty/fake response.
4. **Chronological, deterministic, no ranking.** ✅ **Built.** Every real tab orders by `created_at DESC` with no scoring — this product's own non-negotiable shape for any feed, restated here so it's not accidentally violated later.
5. **Only pulls from `social.orbita.note`.** ✅ **Built**, confirmed again while planning: forum comments, whenever they exist, are deliberately not feed material — a note is a voice meant to circulate, a forum comment is confined to the work's own space.

## Decisions made in planning conversation

- **Three tabs, not one** — caught by the author after the first pass already had "Following" built; corrected before "Shelf" and "Affinity" were added, not as a rewrite afterward.
- **Small-scale for now.** Both real tabs are scoped to accounts (or works) already indexed locally — real fan-out (seeing a followed account, or a shelf-mate, that's never logged into this appview) is Beta 11.
- **`listNotesByWorks` uses an OR chain, not SQLite row-value `IN`.** A short list (one person's shelf) doesn't need the more compact syntax, and the OR chain stays unambiguous across `modernc.org/sqlite` versions without checking what row-value support it carries.

## Explicitly not in this beta

- The actual affinity computation (Beta 13) — this beta only makes room for its tab.
- Real fan-out (Beta 11) — both working tabs are scoped to what's already indexed.
- Forum comments as feed material — explicitly excluded, not deferred.

See [`docs/ROADMAP.md`](ROADMAP.md) for where this sits in the overall arc.
