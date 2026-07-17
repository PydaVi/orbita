# Beta 2 — planning draft

**Status: Beta 2 done ✅** (2026-07-17). All four items built and confirmed with real data, including an extension beyond the original ask: whole-work notes, not just episode-anchored ones.

## Goal

Beta 1 gets Órbita to show a real title and poster for a work. It still treats a work as one flat thing, with no way to attach anything to a specific part of it. Beta 2's problem: `comum` already solved this exact problem on the Postgres side (season/episode/track/chapter granularity, ADR-006) two years before any AT Protocol code existed. Beta 2 brings that same idea to the AT Protocol side for the first time.

**Scope correction from the original draft:** granularity in `comum` was never a property of the shelf — the shelf is always "this whole work is mine." Granularity belongs to the **note**, which `comum` anchors optionally to a season/episode/track/chapter. Beta 0/1 deliberately deferred introducing a second Lexicon; Beta 2 is that deferral coming due. Confirmed directly with the author: the shelf stays whole-work only (a series on the shelf, not an episode of it), and what's wanted is a work's page where you can browse season → episode and write a note anchored to one specific episode.

## Scope (draft)

1. **A second Lexicon, `social.orbita.note`.** ✅ **Written and tested.** [`lexicons/social/orbita/note.json`](../lexicons/social/orbita/note.json) — text + a `work` reference (reusing `social.orbita.shelf.item#work` via a cross-NSID `ref`, confirmed valid syntax against the spec before writing — no duplicated def) + optional `season`/`episode` integers. TV-only for this pass — no `track`/`chapter` yet for albums/books, noted as a future extension, not built now. Confirmed against the local sandbox PDS: wrote a note anchored to `tmdb-tv/1396` season 1 episode 1, read it back intact.
2. **Fetch and cache episode-level catalog data.** ✅ **Built and confirmed.** [`cmd/appview/episodes.go`](../cmd/appview/episodes.go) (`fetchSeasons`/`fetchEpisodes`/`displaySeasons`/`displayEpisodes`) + `season_cache`/`episode_cache` tables in `db.go`, same cache-first/fail-open shape as `work_cache`.
3. **Extend `GET /works/{provider}/{id}`.** ✅ **Built and confirmed.** For `tmdb-tv` works it now lists seasons, `GET /works/{provider}/{id}/season/{n}` lists episodes, and `GET /works/{provider}/{id}/season/{n}/episode/{n}` shows the real synopsis — verified against Breaking Bad S1E1 (Pilot), synopsis rendered correctly. Notes placeholder ("not built yet") shown honestly instead of pretending the feature is done.
4. **Note write path.** ✅ **Built and confirmed.** New OAuth scope (`repo:social.orbita.note?action=create`), [`cmd/appview/notes.go`](../cmd/appview/notes.go) mirroring `shelf.go`'s write pattern, and `webhook.go` extended to index the second collection alongside the first (both Tap instances — local sandbox and real relay — restarted with a comma-separated `TAP_COLLECTION_FILTERS` to pick up both collections). Went beyond the original scope during testing: the author asked for a note on the work as a whole too, not just episode-anchored ones ("acho que é bom ter nota da obra em si tbm, além da nota dos episódios"), so `listNotes`/`renderNotesSection` take `season, episode *int` and treat both-nil as "whole work" — verified with three real records: two episode-level (`tmdb-tv/1396`, season 0/episode 4 — TMDB's real "Specials" season, confirmed not a bug) and one work-level with `season`/`episode` genuinely absent from the JSON (not sent as zero), checked against both the local SQLite index and the real PDS's `com.atproto.repo.listRecords`.

## Open questions — resolved or still open

- **`season`-without-`episode` / `episode`-without-`season`** — **resolved, application-level.** The Lexicon still allows either field independently (no JSON-Schema-style conditional constructs exist in Lexicon), but the Go write handler only ever sends both together or neither, and `listNotes`/`renderNotesSection` only ever query for both-nil or both-set. A genuine "note about season 2 as a whole, no specific episode" isn't wired up in the UI — the schema permits it, the app doesn't ask for it, matching the "don't build what wasn't asked for" bar.
- **Cache storage cost** — still open. Some shows run 500+ episodes; caching full episode lists for every show that shows up on a shelf is a real (if small-scale) storage question, not addressed this beta.
- **Tap signal-collection is a single NSID, not a list** — turned out not to matter for correctness: `TAP_COLLECTION_FILTERS` (what gets indexed) does take a comma-separated list and both collections are filtered correctly. `TAP_SIGNAL_COLLECTION` (what triggers "start tracking this repo") is still a single NSID, still set to the shelf collection — in practice a non-issue since the product's own flow always puts a shelf item first, but the theoretical gap (a note with no prior shelf item on that account never gets noticed) is unresolved, just deprioritized.
- **Does this beta need real UI** — resolved by not needing it: plain HTML with forms, same bar as Beta 0/Beta 1, proved the mechanism end to end (real OAuth write, real Tap indexing, real PDS read-back).

## Explicitly not in this beta

- Affinity computation — still not scoped in any beta yet, likely comes after this one
- Anything for albums/books beyond noting that they'll eventually need their own granularity too

See `docs/BETA1-PLAN.md` for what this beta builds on, and `comum`'s ADR-006 (season/episode/track/chapter granularity — referenced generically here since `comum` is a private repo, see `docs/BETA0-PLAN.md`'s note on that) for the product precedent this is bringing over.
