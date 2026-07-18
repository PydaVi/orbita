# Beta 6 — planning draft

**Status: Beta 6 done ✅** (2026-07-18). Feed exists with its real three-tab shape — Shelf (main), Following, Affinity (honest placeholder) — not the single "Following" tab first sketched. Extended after review with real reply threads and reposts on notes (see below) — a bigger addition than the "redesign this card" request that started it.

## Goal

There was no page yet that was actually useful to check day-to-day — everything lived on a single work's or person's page. A feed turns this from "a place to look something up" into "a place to come back to."

Corrected mid-build (2026-07-18): the roadmap sketch only named a single "notes from people you follow" feed. The author caught this — this product's real feed shape (already established in earlier product work) is **three tabs**: Shelf (notes from *anyone* about works on your own shelf — obra-first, the main tab), Following (notes from people you follow), and Affinity (people with similar taste, via Jaccard). Built to that shape instead.

## Scope

1. **Shelf tab — the main one.** ✅ **Built.** `GET /api/feed?tab=shelf` (also the default with no `tab` param) pulls notes from *anyone* about works on the viewer's own shelf — [`notes.go`](../cmd/appview/notes.go)'s new `listNotesByWorks`, matched against `listShelfItemsByAccount` (already existed for `/shelf`). Obra-first: this tab isn't organized around who wrote something, it's organized around what you already care about.
2. **Following tab.** ✅ **Built.** `GET /api/feed?tab=following` reuses the existing Bluesky follow graph (`app.bsky.graph.follow`) instead of inventing a parallel "follow" concept — read straight from the viewer's own PDS ([`follows.go`](../cmd/appview/follows.go)'s `fetchFollowedDIDs`, paginated), the same "read a public collection from someone's own repo" shape already used for avatar/bio. `identity.go` gained `resolvePDSURL` to get there.
3. **Affinity tab.** ✅ **Built as an honest placeholder.** No Jaccard computation exists yet (Beta 13) — the frontend shows "not built yet" and never calls the API for this tab, rather than the backend faking an empty/fake response.
4. **Chronological, deterministic, no ranking.** ✅ **Built.** Every real tab orders by `created_at DESC` with no scoring — this product's own non-negotiable shape for any feed, restated here so it's not accidentally violated later.
5. **Only pulls from `social.orbita.note`.** ✅ **Built**, confirmed again while planning: forum comments, whenever they exist, are deliberately not feed material — a note is a voice meant to circulate, a forum comment is confined to the work's own space.

6. **Reply threads on notes.** ✅ **Built**, added after the feed card redesign surfaced the question directly (2026-07-18): "does this need a comment button?" Corrected an assumption made earlier in this beta's own planning — notes replying to notes isn't new ground; earlier product work already did this (ADR-011, root/parent mirroring AT Protocol's own reply shape, the same one `app.bsky.feed.post` uses). `social.orbita.note` gained an optional `reply` field (`{root, parent}`, both `com.atproto.repo.strongRef`s) rather than treating conversation as forum-only. A whole thread shares one root — replying to a reply reuses the original root, computed server-side (`notes.go`'s `noteRootRef`) so the client never has to track it. One level of nesting is fetched and rendered for this pass (`listReplies`); a reply's own replies are stored correctly at the data layer but not surfaced yet.
7. **Reposts.** ✅ **Built**, explicitly **not** a popularity metric: a new Lexicon (`social.orbita.repost`, a `subject` strongRef pointing at the note being reposted — same shape as `app.bsky.feed.repost`) exists solely so a note a followed account reposts shows up in the viewer's Following tab, attributed ("reposted by @handle"). No count is computed or shown anywhere, on this or any other repost — that would violate this product's own non-negotiable against public popularity metrics. A repost sorts into the feed by when it was shared, not when the note was originally written.

## Decisions made in planning conversation

- **Three tabs, not one** — caught by the author after the first pass already had "Following" built; corrected before "Shelf" and "Affinity" were added, not as a rewrite afterward.
- **Small-scale for now.** Both real tabs are scoped to accounts (or works) already indexed locally — real fan-out (seeing a followed account, or a shelf-mate, that's never logged into this appview) is Beta 11.
- **`listNotesByWorks` uses an OR chain, not SQLite row-value `IN`.** A short list (one person's shelf) doesn't need the more compact syntax, and the OR chain stays unambiguous across `modernc.org/sqlite` versions without checking what row-value support it carries.
- **Notes can have real conversation, forum still exists for something else.** Corrected mid-planning: the author pointed out earlier product work already threads replies under notes, and this is "a social space, of interaction, of getting to know people through affinities" — conversation belongs on notes, not only in the not-yet-built forum. Forum (Beta 9) still stays a separate, longer-form surface; this doesn't collapse the two together, it just stops pretending notes were meant to be a monologue.
- **RT was floated, dropped, then brought back with a specific job.** First cut included an RT button with no clear purpose beyond "posts have this on other apps." Revisited once framed narrowly: RT exists only as the mechanism that makes a note travel into a follower's feed, attributed, never as a count — that's the version that got built.
- **An existing `notes` table needed a real migration**, not just a wider `CREATE TABLE IF NOT EXISTS` — `db.go` gained `ensureColumn`, adding the four new reply columns to a table that already existed from earlier betas.

## Explicitly not in this beta

- The actual affinity computation (Beta 13) — this beta only makes room for its tab.
- Real fan-out (Beta 11) — both working tabs are scoped to what's already indexed.
- Deeper reply nesting (replies-to-replies rendered in the UI) — stored correctly, not surfaced yet.
- Any public count of reposts, replies, or anything else — against this product's own non-negotiables, not just unbuilt.

See [`docs/ROADMAP.md`](ROADMAP.md) for where this sits in the overall arc.
