# Beta 2 — planning draft

**Status:** scope agreed in conversation, not yet started, and further out than `BETA1-PLAN.md` — expect this to shift more than Beta 1's plan before work actually begins.

## Goal

Beta 1 gets Órbita to show a real title and poster for a work. It still treats a work as one flat thing — a whole movie, a whole series — with no way to point at a specific part of it. Beta 2's problem: `comum` already solved this exact problem on the Postgres side (season/episode/track/chapter granularity, ADR-006) two years before any AT Protocol code existed. Beta 2 is bringing that same idea to the AT Protocol side for the first time — and it isn't a trivial port, because Lexicon fields are much less forgiving to change after real accounts start using them than a Postgres column is.

Concretely, inspired directly by Popfeed: a series shouldn't just be "Breaking Bad" — it should be able to show season → episode → synopsis, and let a shelf item (or whatever record ends up representing this) point at one specific episode, not just the show as a whole.

## Scope (draft)

1. **Extend the catalog cache to store episode-level structure** for TV shows specifically — season number, episode number, title, synopsis, air date. Movies, albums, and books don't need this same shape (though they have their own granularity per `comum`'s precedent: track for albums, chapter for books) — likely out of scope for this beta's first pass, TV-only to start.
2. **Fetch episode-level data from TMDB** (`/tv/{id}/season/{n}`), cached locally the same way Beta 1's work-level cache works.
3. **Decide how a record references a specific episode** — this is the real design question of this beta, not a detail.

## Open questions — genuinely undecided, need real discussion before building

- **Schema evolution or new record type?** Does `social.orbita.shelf.item`'s `work` object gain optional `season`/`episode` fields, or does granular anchoring deserve its own record type entirely? Beta 0 already learned (the hard way, watching Popfeed's leftover `app.popsky.post` namespace) that Lexicon decisions are close to permanent once real accounts write real records. This needs the same care Beta 0 gave the `work` field itself, not a quick bolt-on.
- **Which granularity per work type** — season/episode for TV, track for albums, chapter for books, mirroring `comum`'s nullable-column pattern (ADR-006) — but that was a Postgres answer; the Lexicon-shaped answer isn't obviously the same shape.
- **Cache storage cost** — some shows run 500+ episodes; caching full episode lists for every show that shows up on a shelf is a real (if small-scale) storage question, not just a formality.
- **Does this beta need real UI**, or is a plain-text listing of episodes still an acceptable "prove the mechanism" bar, same spirit as Beta 0 and Beta 1's bare HTML?

## Explicitly not in this beta

- Affinity computation — still not scoped in any beta yet, likely comes after this one
- Anything for albums/books beyond noting that they'll eventually need their own granularity too

See `docs/BETA1-PLAN.md` for what this beta builds on, and `comum`'s ADR-006 (season/episode/track/chapter granularity — referenced generically here since `comum` is a private repo, see `docs/BETA0-PLAN.md`'s note on that) for the product precedent this is bringing over.
