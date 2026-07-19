# Beta 8 — constellation and archetype

**Status: built ✅** (2026-07-19), first version — explicitly not calibrated, revisited as real shelves accumulate.

## Goal

[`docs/ROADMAP.md`](ROADMAP.md) flagged this right after Beta 7 shipped: profile pages don't yet have a constellation-style visualization or a derived archetype, both named as stretch goals back in Beta 5 and deliberately left until the shelf's own shape (nooks, Beta 7) had settled first.

The author was explicit about the bar and the approach: same concept, same ambition, but **not a port** of earlier product work — reimagined around everything this project has actually built differently, at the same level of design care the nook/duotone work (Beta 7's later commits) already established. Archetype specifically: redo everything, names and meanings included, informed by whatever the constellation itself ends up being here. Carte blanche for this first version, to be adjusted once real data exists to look at.

## Why this isn't a port

Earlier product work anchored its constellation on **genre/tag** — a shared, catalog-controlled vocabulary, hashed into a fixed position so the same tag always lands in the same region of every profile's own sky. That's the entire mechanism behind "similar taste lights up the same place across different people."

This appview never built a tag pipeline. The catalog only ever caches `title`/`poster_url`/`year`/`overview` (see `tmdb.go`) — there's no genre data to anchor on. What this product *does* have, uniquely, that's small and curated enough to serve the same purpose, is a **nook's own theme** — `style.theme`'s handful of `knownValues` (Beta 7). So the constellation here is anchored on theme, not genre: every account's "riso" nook lands in the same region of their own sky, and the same region across everyone else's, regardless of what either person actually named the nook. Provider (medium: `tmdb-movie`/`tmdb-tv`/`musicbrainz`/`open-library`) and decade ride along as secondary, weaker pulls — the same relative structure earlier work used (one dominant axis, two minor ones), rebuilt on this project's own data shape.

One more real structural difference: nook membership here is **already a deliberate, real grouping** — a person put a work in a nook on purpose (Beta 7's whole premise). Earlier work had to *infer* connectivity from tag overlap to draw any meaningful edges between nodes. This constellation draws an edge between any two works in the same nook directly — no inference needed, the grouping already exists.

## What shipped

1. **`GET /api/profile/{handle}/constellation`** ([`constellation.go`](../cmd/appview/constellation.go)) — assembles one node per work (deduplicated if a work sits in more than one nook — it still lights up as a single star, keyed to whichever nook is found first) with its nook's theme (or none, for Unsorted), provider, year, and how many notes exist for it. No physics here — this handler only serves the raw graph, matching how this project's client already renders everything else it fetches.
2. **The layout and rendering** ([`constellation.js`](../frontend/constellation.js)) — a from-scratch, synchronous (not per-frame) force simulation: theme anchors at fixed angles around a circle (weight 4), provider sub-position within each theme region (weight 1), decade sub-position via a deterministic string hash (weight 1), pairwise repulsion, spring pull toward each node's computed target, 160 iterations, rendered once to a plain Canvas2D `<canvas>` — no library, matching this frontend's rule everywhere else. Dot color is the work's theme (`THEME_COLORS`, matching `--duo-*-hi` in `styles.css` by hand, the same manual-sync constraint `duotoneFilter()` in `common.js` already lives with); dot size grows with note count; a ring marks any work with at least one note; Unsorted works render fainter, not hidden. Hover shows a tooltip, click goes straight to that work's page.
3. **The archetype**, computed client-side from the same node data (so there's exactly one source of truth for the shape, not a duplicated computation in two languages):
   - **Spread** — mass-weighted inverse Simpson index across the 8 possible regions (7 curated themes + Unsorted): how many of them a shelf actually *reaches*, not just brushes past. Same index earlier work used and for the same reason (Shannon entropy was explicitly rejected there as too generous toward a shelf that leaks a little mass into extra sectors) — reused as a technique, recomputed against this project's own 8 regions instead of tag sectors.
   - **Cohesion**, genuinely redefined, not just re-skinned: earlier work needed a union-find over an inferred tag-overlap graph, because nothing there was already a real grouping. Here, a nook already *is* one — so cohesion is just the fraction of the whole shelf living inside its single biggest nook. No inference, no graph traversal, because the data already carries the structure.
   - **Nine new names**, none reused from anywhere else, each with a one-line "voice" and a real-data evidence sentence (never a bare percentage alone) — see the `ARCHETYPE_NAMES` matrix in `constellation.js`: Luz Cinzenta, Par Próximo, Estrela Fixa, Campo Difuso, Trajeto Orbital (a small pun on the product's own name, landing on the most balanced/common case, mirroring how earlier work's own median case reused its feature's own name), Estrela-Guia, Campo Profundo, Mapa Estelar, Centro de Massa.
   - The archetype's own "symbol" is that person's real layout, recomputed at a small fixed size (120×120) — same mechanic as the interactive canvas, not a separate icon system.

## Explicitly not calibrated

The three-way spread/cohesion thresholds (`levelOf`, cutoffs at 0.4/0.7 and 0.3/0.6) are a reasonable first split, not tuned against a real distribution — this appview doesn't have enough accounts yet for that to mean anything (earlier work calibrated its own thresholds against 20 mock users; there's no equivalent sample here). Worth revisiting once more real shelves exist to look at, same spirit as the author's own "vamos ajustando depois."

## Incidental fix

Verifying the new endpoint against a copy of the real database surfaced a real, pre-existing bug unrelated to this beta: `work_cache` never got a migration for its `overview` column (the same class of gap already fixed for `notes` and `nooks` in earlier commits) — `setCachedWork` had been silently failing on every cache miss, meaning every not-yet-cached work was being re-fetched from TMDB on every single lookup instead of caching after the first. Fixed in `db.go` with the same `ensureColumn` pattern already established.

## Explicitly not in this beta

- Any persistence of the archetype (recomputed fresh on every profile load, same as earlier work's own choice — no confirm/hide UI, no stored history).
- Genre/tag data — still doesn't exist in this catalog. If it's ever added, the anchor system here would likely want a fourth, richer axis alongside theme rather than replacing it, since theme's cross-profile comparability property depends specifically on it being a small curated vocabulary, which genre tags typically aren't.
- Touch interaction for the canvas (hover/click assume a mouse) — same category of gap as the shelf's own drag-and-drop, flagged but deferred.
