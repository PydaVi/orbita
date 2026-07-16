# Beta 1 — planning draft

**Status: Beta 1 done ✅** (2026-07-16). All three items built and confirmed with real data, plus an unplanned delete feature and a real OAuth scope fix found along the way. Living document, same spirit as `BETA0-PLAN.md`.

## Goal

Beta 0 proved the pipeline end to end for **one account, one Lexicon, one row at a time**. It deliberately never touched the thing that actually defines an AppView: aggregating data that belongs to *different, independent* people into something useful. Beta 1's problem is that gap.

Concretely: if two different real accounts both have the same work on their shelf, can Órbita answer "who else has this," instead of just listing two unrelated rows? And can adding something to your shelf feel like using a real product (search by title) instead of typing a raw TMDB id by hand?

## Scope — three items, agreed in discussion

1. **Search at write time.** ✅ **Built and confirmed.** [`cmd/appview/search.go`](../cmd/appview/search.go), `GET /search?q=...`, backed by `searchTMDB` in [`tmdb.go`](../cmd/appview/tmdb.go) — queries both the movie and TV search endpoints (capped at 5 results each), each result a one-click "Add to shelf" form pre-filled with the resolved `{provider, id}`. Popfeed-inspired UX, applied differently: Popfeed embeds the full resolved metadata into the record itself; we keep the minimal `{provider, id}` shape in the actual Lexicon record (no schema change — that decision from Beta 0 stands), search only exists to find the right pair *before* writing. Confirmed end to end: searched "Inception"-style by title, picked a real result (*Foundation*, `tmdb-tv/93740`), and it appeared on `/shelf` with the correct resolved title — a real, TMDB-recognized id, unlike the `Titanic` incident this same testing loop caught earlier.
2. **Lightweight read-time cache.** ✅ **Built and confirmed.** [`cmd/appview/tmdb.go`](../cmd/appview/tmdb.go) (`resolveWork`, `displayWork`) + a `work_cache` table in [`db.go`](../cmd/appview/db.go), keyed by `(provider, work_id)`. `/shelf` and `/works/{provider}/{id}` both now show real titles and posters via TMDB instead of raw ids, with a poster `<img>`. Verified the cache actually dedupes, not just "runs without error": three indexed shelf items exist (two pointing at the same Matrix, one at Breaking Bad), and `work_cache` has exactly **two** rows — Matrix was resolved once and reused for both accounts, not fetched twice. Fails open by design: an unsupported provider or a TMDB error falls back to showing the raw `provider/id` string instead of breaking the page (`displayWork`'s fallback path) — not `comum`'s full growing public catalog, just enough caching for what's actually been indexed.
3. **Cross-account grouping — the actual new problem.** ✅ **Built and confirmed.** [`cmd/appview/works.go`](../cmd/appview/works.go), `GET /works/{provider}/{id}`, backed by `listShelfItemsByWork` in [`db.go`](../cmd/appview/db.go) — groups `shelf_items` by `(provider, work_id)` instead of listing by event. Tested with real overlapping data: added `tmdb-movie/603` (The Matrix) to the sandbox test account (which already had it on the author's real `pydavi.bsky.social` account) — `GET /works/tmdb-movie/603` correctly returned **both** DIDs. This is also the mechanical building block affinity will eventually need (comparing two whole shelves is the same kind of grouped query, one level up) — but affinity itself is explicitly **not** in this beta.

## Unplanned addition: delete, and a real OAuth scope lesson

Testing item 2 by hand exposed a real gap: the Lexicon's `id` field has no format validation, so typing a movie *title* into the raw id field (there's no search yet — item 1) wrote a nonsense record (`{provider: "tmdb-movie", id: "Titanic"}`) straight into the author's real PDS. The write succeeded; only the *display* step failed (TMDB correctly returns 404 for a non-numeric id), caught by `displayWork`'s existing fail-open fallback. This is itself a point in favor of item 1: search-before-write is what prevents nonsense from reaching the PDS at all, not just a UX nicety.

Fixing it needed a delete feature that wasn't in the original three-item scope — added `cmd/appview/shelf.go`'s `handleShelfDelete` (`POST /shelf/delete`, real `com.atproto.repo.deleteRecord`, ownership enforced by the protocol itself, not custom code) plus a "Delete" button per row on `/shelf`.

First attempt failed with a real, useful error: `ScopeMissingError: Missing required scope "repo:social.orbita.shelf.item?action=delete"` — Beta 0's OAuth scope only ever requested `action=create`. Confirmed against `@atproto/oauth-scopes`' source that `REPO_ACTIONS` is `create`/`update`/`delete`, and multiple actions on one collection repeat the query param (`?action=create&action=delete`), not a comma list. Scope updated to request exactly `create` and `delete` — not `update`, since that feature doesn't exist yet, keeping the same least-privilege discipline as Beta 0. Confirmed fixed: deleted the bad `Titanic` record for real, verified gone from *both* the local index and the real PDS via `listRecords`.

## Explicitly out of scope for Beta 1

- Episode/season-level catalog richness (synopses, per-episode data) — see `docs/BETA2-PLAN.md`
- Affinity computation between shelves
- A second Lexicon (`social.orbita.note` or similar)
- Any real visual design — still bare, unstyled HTML

## Open questions — not decided yet

- ~~**Search backend**~~ — resolved: a server-side `GET /search?q=...` proxying TMDB (`search.go`), exactly the backend-proxy shape that was already the leaning — the API key never reaches the browser.
- ~~**TMDB credentials**~~ — resolved: the author has a TMDB API key. Loaded via `.env` (gitignored) + `github.com/joho/godotenv/autoload` in `main.go`, read as `os.Getenv("TMDB_API_KEY")` — not written into any committed file.
- ~~**Cache table shape**~~ — resolved for now: one flat `(provider, work_id) → title, poster_url, year` table, confirmed working for `tmdb-movie`/`tmdb-tv` (different underlying TMDB field names — `title`/`release_date` vs. `name`/`first_air_date` — normalized into the same three columns). MusicBrainz and Open Library still have no resolver at all (`resolveWork` returns an error for them, `displayWork` falls back to the raw id) — deferred, not a blocker for Beta 1's completion criterion since no test data uses those providers yet.
- **Test data:** cross-account grouping only means something if at least two *real* accounts have overlapping shelf data. Need to figure out how we'll actually populate that — a second real test account, or is the author's own account plus the sandbox enough to demonstrate the mechanism honestly?

## Completion criterion

✅ **Met.** Two different real accounts have a shelf item for the same work (`tmdb-movie/603`, The Matrix — the author's real account and the sandbox account). At least one item (*Foundation*) was added through the real search UI, not a manually-typed id. The read-time cache resolves and displays title + poster on both `/shelf` and `GET /works/{provider}/{id}`, which correctly lists every account that has a given work.

See `docs/BETA0-PLAN.md` for the pattern this document follows, and `docs/architecture-beta0-local.md` for the AppView-vs-PDS reasoning this beta builds directly on top of.
