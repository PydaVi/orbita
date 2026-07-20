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

## First real use, two fixes

Screenshotted right after shipping: every dot was pinned to the canvas edges/corners, with the region labels floating in empty space in the middle — nothing about it read as a constellation.

**The physics were broken, not just untuned.** Repulsion used a bare constant (`900 / d²`) with no relationship to canvas size, and every node started jittered around one shared center point regardless of its target region — meaning most pairs began almost on top of each other, where `900 / d²` is enormous. The anchor pull (`× 0.03`) never stood a chance against it; the first few of 160 iterations flung everything straight into the wall clamp, and it never recovered. Fixed two ways: repulsion is now scaled by canvas area (`width × height × 0.00004`, the same kind of scaling earlier product work used for the same reason) so it stays a gentle spacing force regardless of render size, and each node now starts jittered around its *own* target instead of a shared center, so repulsion only ever has to locally sort out nodes that genuinely belong near each other. A hard per-iteration force cap was added as a second line of defense against the same class of blow-up, independent of tuning.

**Placement moved to match the ask directly.** The constellation now sits where a cover photo would traditionally go — full-width, wide-and-shallow (`.constellation-cover`), above the identity block, not a section further down the page. The archetype card moved to just after the bio, inside the hero body itself, not bundled below the big canvas. This meant splitting what was one `renderConstellationSection()` function into a small public surface (`fetchConstellationNodes`, `mountConstellationCanvas`, `mountArchetypeSymbol`, `buildArchetypeCard`) so `profile.js` could place each piece independently while still fetching the graph once — and fetching it earlier, alongside the profile itself in `init()`, since a cover has to be ready before the page's first paint, not popped in after.

## Second look: it read as a generic scatter plot, not this product

Clustering correctly wasn't enough — the visuals still clashed with everything else on the site, flagged directly. Reworked to reuse this product's own established visual language instead of inventing a new one: the exact same corner-crosshair "instrument" framing every poster already uses (`.constellation-frame`/`.archetype-symbol-frame`, mirroring `.poster-wrap`/`.shelf-grid-item`); nook connections rebuilt as a greedy nearest-neighbor chain instead of a full pairwise mesh (a real star chart connects a handful of stars into one legible line, not every possible pair — the old version drew O(n²) crossing lines per nook and read as a hairball past a handful of works), colored by the nook's own theme instead of generic gray; the "you wrote about this" ring recolored to `--signal` specifically, since that ring is a fact about a person's own voice, not about which nook something's in, and `--signal` is reserved for exactly that meaning everywhere else on this site; and a faint dashed orbit guide circle at the anchor radius — a compass rose, and a quiet pun on the product's own name.

## Comparison: the promise the anchor system makes, actually kept

Raised directly after the redesign: the entire reason a shelf's constellation anchors on a small shared vocabulary (theme) instead of a free one is so two different people's shapes can be compared — "encontrar afinidades a partir da comparação de desenhos." Before this, nothing did that: each profile only ever rendered its own shape in isolation.

**Built:** visiting someone else's profile while signed in now fetches the viewer's *own* constellation graph too (skipped entirely on your own profile — nothing to compare against) and renders it as a ghosted overlay on the same canvas — hollow rings only, no connecting lines, no labels, so it never competes with the profile actually being viewed for attention. Because both shapes are computed at the same canvas dimensions and theme anchors are a pure function of (theme, canvas size), the two shapes' regions land in exactly the same places — where both people have real presence in the same theme, the hollow rings and the solid dots visually cluster together, which is the affinity itself, shown, not summarized into a number. A small caption ("hollow rings — your own shelf, overlaid") names what's on screen rather than leaving it to be guessed, matching this product's own standing rule that a person should always know why they're seeing what they're seeing.

## Third look: still didn't feel like this product, even after the redesign

The corner-crosshair box from the second pass borrowed the poster's own framing wholesale — reasonable-sounding, but wrong on reflection: a poster earns a hard rectangular frame because it's a bounded, physical object. A sky is the opposite of that. Boxing it the same way read as "specimen in a container," which is exactly the wrong feeling for something meant to feel boundless — flagged directly, with a nudge toward researching the concept properly rather than iterating on the same box.

Looked at H.A. Rey's *The Stars: A New Way to See Them* (1952) for how an actual constellation earns its lines: a small number of deliberate connections forming one legible shape, never a dense mesh — reinforces the nearest-neighbor-chain decision from the second pass rather than replacing it, but confirms restraint is the right instinct, not more structure.

**Rebuilt again:**
- No frame, no border, no background of its own. The canvas (`.constellation-cover`, `.archetype-symbol`) sits directly on the page's own `--bg`, transparent. A CSS radial `mask-image` fades its own content to nothing near the edges instead of cutting it off with a hard boundary — an actual horizon, not a box with a line around it.
- Dots got a real glow: a soft radial-gradient bloom underneath a small precise core circle, the way a photograph of an actual star always shows some halo around the point of light — a flat filled circle reads as a data-viz bubble, not a star.
- The orbit guide circle and nearest-neighbor connections from the previous pass are unchanged — those were never the problem, the box around all of it was.

## Incidental fix: DID-shaped path segments

Found while testing the comparison feature: a client-side redirect (or a pasted/bookmarked link) landing a DID in the `/profile/{handle}` slot instead of a real handle 404'd outright — `resolveHandleToDID` only ever tried `syntax.ParseHandle`, which correctly rejects a DID string, and there was no fallback. Fixed by trying `syntax.ParseDID` first and returning it directly when it parses — a DID is already a fully resolved identifier, every other route on this site already knows how to work with one, and requiring specifically an unresolved handle bought nothing.

## Related fix, filed under Beta 7

Looking at the constellation surfaced a real gap in the nook system itself: nothing capped how many nooks one account could have, and with only 7 theme regions to anchor on, many nooks piling into a handful of themes crowds those regions past legibility regardless of any single nook's own size. Fixed with a 7-nook-per-account cap — see [`BETA7-PLAN.md`](BETA7-PLAN.md) item 20 for the full reasoning (Miller's "7±2," and why 7 specifically). Filed there rather than here since it's a change to the nook system, not the constellation — the constellation just surfaced why it mattered.

## Layout, prompted by looking at the profile with a cover on it for the first time

Adding a cover-sized element to the top of the profile made an existing, unrelated problem obvious: the sidebar was just nav links floating at the top of an otherwise-empty column, and the profile's avatar sat in its own disconnected block with a visible gap below the cover — "meio desordenado," in the author's own words, pointed at with a real reference screenshot rather than left abstract.

Two structural fixes, not a visual reskin (this product's own type/color system is untouched):

- **Sidebar** (`renderShell`, `common.js`) gained a grounded top: a small identity snippet (avatar + handle, linking to your own profile) above the nav list, filled in asynchronously via `currentViewer()` once it resolves rather than blocking the rest of the shell — every page already calls `renderShell()` synchronously and uses its return value immediately, so this couldn't become a second async entry point. Nav links themselves gained real block-level padding and a hover background instead of being plain inline text with only a color change — structure, not just a list of words. Icons were deliberately *not* added — this site's sidebar being text-only nav (not Twitter's icon+label rows) was already a considered choice, and copying that specific piece of the reference screenshot's structure would have undone it.
- **Profile hero** (`profile.js`) — the avatar now overlaps the cover's lower-left edge (`.profile-avatar-overlap`, `position: absolute` against `.profile-cover-frame`) instead of living in a separate side-by-side block below it, closing the gap and reading as one continuous piece rather than two stacked sections. Only applies when there's a real cover to overlap (`hasConstellation`); falls back to the plain side-by-side pairing `.hero` already uses elsewhere (the work page's own poster + title) when a shelf is too small for a shape yet. Deliberately did *not* copy the reference's follower/following/post-count row — this product doesn't show public popularity metrics anywhere, and that's a considered absence, not a gap.

## Explicitly not in this beta

- Any persistence of the archetype (recomputed fresh on every profile load, same as earlier work's own choice — no confirm/hide UI, no stored history).
- Genre/tag data — still doesn't exist in this catalog. If it's ever added, the anchor system here would likely want a fourth, richer axis alongside theme rather than replacing it, since theme's cross-profile comparability property depends specifically on it being a small curated vocabulary, which genre tags typically aren't.
- Touch interaction for the canvas (hover/click assume a mouse) — same category of gap as the shelf's own drag-and-drop, flagged but deferred.

## Unplanned, same session: the rest of the sidebar, and a real "Saved"

Not part of the constellation work, but landed right after it while the sidebar was already open: the reference screenshot named three more menu items this site didn't have yet (Messages, Saved, Settings).

- **Messages and Settings are honest placeholders**, not hidden or dead links. `frontend/placeholder.html`/`.js` renders a short, specific explanation of what's actually missing and why (for Messages: the same "can't just be a public repo record" question already open for Beta 14's direct messages) rather than a generic "coming soon."
- **Saved is real**, not a placeholder, and deliberately **not** an AT Protocol record (`saved.go`) — every other collection this appview writes is meant to be public, that's the point of a repo on the network; a private bookmark list is the opposite kind of data, the same category of problem flagged for DMs, just with an obvious answer here: every real platform that ships a save/bookmark feature (Bluesky's own included) stores it privately, server-side. Lives only in this appview's own SQLite (`saved_notes` table) — a real, accepted limitation (it won't follow you to a different AppView the way your shelf does).
  - `POST /api/notes/save` / `/unsave`, `GET /api/saved/uris` (cheap, just URIs, for painting a button's state) and `GET /api/saved` (full hydrated notes, for the `/saved` page itself).
  - A save button (`saveIcon`, `common.js`) joined the existing repost/reply row on every note — filled vs outline, no count anywhere, matching how repost already works here (attributed or private, never tallied). `renderFeedList` moved from `feed.js` into `common.js` so `/saved` could reuse the exact same note-card rendering the feed already has, rather than a second copy of it.

## Filed on the roadmap, not built yet

Raised in passing: `musicbrainz` and `open-library` are declared valid providers in the Lexicon since Beta 0 but were never actually wired up — no resolver, no search. See [`ROADMAP.md`](ROADMAP.md)'s new Beta 16 for the real scope; books and albums don't resolve at all today, not just missing fine-grained track/chapter detail.

## Two more layout fixes, and a real bug

- **A discreet vertical rule between all three layout columns.** `.layout` moved from `align-items: start` to `stretch` so every column's box spans the full row height regardless of which one has the most content — a `border-right` on the sidebar and the center `.page` (removed on the sub-800px collapse, where there's only one column left) wouldn't have looked right otherwise, stopping short wherever its own column's content happened to end rather than running the whole way down.
- **The brand mark now sits above the center column, not the sidebar.** `.topbar` was a plain flex row, independent of `.layout`'s own grid — nothing tied its logo's horizontal position to any column below it, which is what made the new divider lines read as slightly misaligned with it. `.topbar` now shares `.layout`'s *exact* grid definition (same columns, same max-width, same centering, no horizontal padding of its own) and places `.brand` in the second column specifically, so it lines up with the center column on every page, not just by accident of a matching left margin.
- **Real bug, not a layout one: saving a note threw `"Unexpected end of JSON input"`.** `POST /api/notes/save`/`/unsave` (`saved.go`) returned a bare `200` with no body, but `fetchJSON` (`common.js`) always calls `res.json()` on the response — an empty body isn't valid JSON, so it throws. Every other write endpoint on this site already returns `{"ok": true}` for exactly this reason; `saved.go`'s handlers just hadn't followed that convention yet. Fixed there, and hardened `fetchJSON` itself to treat an empty-but-successful response as `null` rather than a parse error, so the same class of mistake can't silently reintroduce this bug from some other handler later.
