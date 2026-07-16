# Beta 1 — planning draft

**Status:** in progress — item 3 (cross-account grouping) built and confirmed with real overlapping data. Living document, same spirit as `BETA0-PLAN.md`.

## Goal

Beta 0 proved the pipeline end to end for **one account, one Lexicon, one row at a time**. It deliberately never touched the thing that actually defines an AppView: aggregating data that belongs to *different, independent* people into something useful. Beta 1's problem is that gap.

Concretely: if two different real accounts both have the same work on their shelf, can Órbita answer "who else has this," instead of just listing two unrelated rows? And can adding something to your shelf feel like using a real product (search by title) instead of typing a raw TMDB id by hand?

## Scope — three items, agreed in discussion

1. **Search at write time.** Inspired by Popfeed's UX, applied differently: Popfeed embeds the full resolved metadata into the record itself; we keep the minimal `{provider, id}` shape in the actual Lexicon record (no schema change — that decision from Beta 0 stands). Search happens *before* writing, purely to figure out the right `{provider, id}` pair — the PDS record stays exactly as lean as it is today. — *not started*
2. **Lightweight read-time cache.** To show "The Matrix" with a poster instead of `tmdb-movie/603`. Not `comum`'s full growing public catalog (`works` table, community fallback, moderated sources) — just enough local caching of whatever `{provider, id}` pairs actually show up in our indexed `shelf_items`, resolved once, reused after. — *not started, TMDB key now available (see below)*
3. **Cross-account grouping — the actual new problem.** ✅ **Built and confirmed.** [`cmd/appview/works.go`](../cmd/appview/works.go), `GET /works/{provider}/{id}`, backed by `listShelfItemsByWork` in [`db.go`](../cmd/appview/db.go) — groups `shelf_items` by `(provider, work_id)` instead of listing by event. Tested with real overlapping data: added `tmdb-movie/603` (The Matrix) to the sandbox test account (which already had it on the author's real `pydavi.bsky.social` account) — `GET /works/tmdb-movie/603` correctly returned **both** DIDs. This is also the mechanical building block affinity will eventually need (comparing two whole shelves is the same kind of grouped query, one level up) — but affinity itself is explicitly **not** in this beta.

## Explicitly out of scope for Beta 1

- Episode/season-level catalog richness (synopses, per-episode data) — see `docs/BETA2-PLAN.md`
- Affinity computation between shelves
- A second Lexicon (`social.orbita.note` or similar)
- Any real visual design — still bare, unstyled HTML

## Open questions — not decided yet

- **Search backend:** does "search at write time" need a new server-side endpoint (e.g. `GET /search?q=...` proxying TMDB) so the API key never reaches the browser, or is there a simpler shape? Leaning toward a backend proxy for the obvious reason (don't leak API credentials client-side), but not settled.
- ~~**TMDB credentials**~~ — resolved: the author has a TMDB API key. Loaded via `.env` (gitignored) + `github.com/joho/godotenv/autoload` in `main.go`, read as `os.Getenv("TMDB_API_KEY")` — not written into any committed file.
- **Cache table shape:** one flat table keyed by `(provider, id)` with `title`/`poster`/`year`? Movies and TV shows fit that shape; MusicBrainz (albums) and Open Library (books) have different natural fields (artist vs. author, etc.) — worth deciding whether one generic shape is enough for Beta 1 or if that's premature.
- **Test data:** cross-account grouping only means something if at least two *real* accounts have overlapping shelf data. Need to figure out how we'll actually populate that — a second real test account, or is the author's own account plus the sandbox enough to demonstrate the mechanism honestly?

## Completion criterion (draft, not final)

At least two different real accounts have a shelf item for the *same* work, added through the new search UI (not a raw `curl`/manually-typed id). The read-time cache resolves and displays title + poster on both `/shelf` and a new grouped view. A page (route not yet named — candidate: `GET /works/{provider}/{id}`) correctly lists every account that has that specific work.

See `docs/BETA0-PLAN.md` for the pattern this document follows, and `docs/architecture-beta0-local.md` for the AppView-vs-PDS reasoning this beta builds directly on top of.
