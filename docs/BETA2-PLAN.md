# Beta 2 — planning draft

**Status:** scope clarified in conversation (2026-07-16), not yet started.

## Goal

Beta 1 gets Órbita to show a real title and poster for a work. It still treats a work as one flat thing, with no way to attach anything to a specific part of it. Beta 2's problem: `comum` already solved this exact problem on the Postgres side (season/episode/track/chapter granularity, ADR-006) two years before any AT Protocol code existed. Beta 2 brings that same idea to the AT Protocol side for the first time.

**Scope correction from the original draft:** granularity in `comum` was never a property of the shelf — the shelf is always "this whole work is mine." Granularity belongs to the **note**, which `comum` anchors optionally to a season/episode/track/chapter. Beta 0/1 deliberately deferred introducing a second Lexicon; Beta 2 is that deferral coming due. Confirmed directly with the author: the shelf stays whole-work only (a series on the shelf, not an episode of it), and what's wanted is a work's page where you can browse season → episode and write a note anchored to one specific episode.

## Scope (draft)

1. **A second Lexicon, `social.orbita.note`.** ✅ **Written and tested.** [`lexicons/social/orbita/note.json`](../lexicons/social/orbita/note.json) — text + a `work` reference (reusing `social.orbita.shelf.item#work` via a cross-NSID `ref`, confirmed valid syntax against the spec before writing — no duplicated def) + optional `season`/`episode` integers. TV-only for this pass — no `track`/`chapter` yet for albums/books, noted as a future extension, not built now. Confirmed against the local sandbox PDS: wrote a note anchored to `tmdb-tv/1396` season 1 episode 1, read it back intact.
2. **Fetch and cache episode-level catalog data** for TV shows — season list, episode list per season, synopsis, air date — from TMDB (`/tv/{id}` for seasons, `/tv/{id}/season/{n}` for episodes per season).
3. **Extend `GET /works/{provider}/{id}`** (already exists, Beta 1) so that, for `tmdb-tv` works, it lets you browse season → episode, and each episode can have a note attached.
4. **Note write path** — new OAuth scope (`repo:social.orbita.note?action=create`), a write handler mirroring `shelf.go`'s pattern, and Tap/webhook extended to index the second collection alongside the first.

## Open questions — still real, need discussion before/while building

- **`season`-without-`episode` / `episode`-without-`season`** — does the Lexicon schema need to enforce "episode requires season," or is that an application-level check in the Go write handler (Lexicon doesn't have JSON-Schema-style conditional-field constructs as far as confirmed)? Leaning toward application-level, not settled.
- **Cache storage cost** — some shows run 500+ episodes; caching full episode lists for every show that shows up on a shelf is a real (if small-scale) storage question, not just a formality.
- **Tap signal-collection is a single NSID, not a list** — `TAP_SIGNAL_COLLECTION` (what triggers "start tracking this repo") only takes one collection per the docs. If someone writes a note before ever using the shelf, would Tap notice them? Practically low-risk (shelf comes first in the product's own flow) but worth naming, not assuming away.
- **Does this beta need real UI**, or is a plain-text season/episode listing still an acceptable "prove the mechanism" bar, same spirit as Beta 0 and Beta 1's bare HTML?

## Explicitly not in this beta

- Affinity computation — still not scoped in any beta yet, likely comes after this one
- Anything for albums/books beyond noting that they'll eventually need their own granularity too

See `docs/BETA1-PLAN.md` for what this beta builds on, and `comum`'s ADR-006 (season/episode/track/chapter granularity — referenced generically here since `comum` is a private repo, see `docs/BETA0-PLAN.md`'s note on that) for the product precedent this is bringing over.
