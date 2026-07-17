# Beta 3 — planning draft

**Status:** scope drafted in conversation (2026-07-17), not yet started.

## Goal

Everything through Beta 2 proves the protocol mechanics work — OAuth writes, Tap indexing, cross-account aggregation, season/episode browsing — but every page is a raw `fmt.Fprintf` HTML string. That was the right call while the point was proving mechanism, but it's the wrong shape for anything presentation-driven going forward (profile especially, per the conversation that reordered this roadmap — you can't really discuss a profile page without being able to look at one). Beta 3 is where `orbita` gets a real interface, applied first to what already exists (shelf, work pages, season/episode navigation, notes), so every beta after this one inherits it instead of starting from raw strings again.

Before drafting scope, we researched how other open-source AT Protocol apps (Bluesky's own `social-app`, Frontpage, Smoke Signal, WhiteWind, Leaflet) approach this. Two findings shaped the plan below:

- **There's no shared visual convention in the Atmosphere ecosystem to align with.** Every app looks like its own independent product; the only thing they share is that the protocol itself (DIDs, PDS hosts) stays invisible to end users. So keeping Órbita's own distinct "Observatório" identity isn't going against ecosystem norms — every app already does this.
- **One thing to explicitly avoid**: Bluesky's web layout has a right-hand column for trending topics/suggested feeds — directly against this project's own principle 2 (no trending, no algorithmic surfaces). Worth naming as a conscious decision, not an oversight.
- What's worth borrowing isn't visual identity but **practices**: menu/nav placement across breakpoints, how a profile and a feed are laid out structurally, responsive behavior at different screen sizes, sensible CSS transition patterns — studied as engineering reference, not copied, same way `comum`'s own Observatório identity was always meant to be its own thing.

## Scope (draft)

1. **Split the appview into a JSON API.** The handlers in `works.go`, `notes.go`, `shelf.go` (list), `search.go` stop emitting HTML directly and return JSON instead. OAuth session handling, DPoP-bound tokens, and the actual `com.atproto.repo.*` write calls stay exactly where they are today (server-side, inside the appview) — a DPoP private key and access/refresh tokens must never reach browser JS, so the appview keeps being the only thing that ever talks to a PDS. The browser only ever calls the appview's own JSON endpoints, same-origin, session via the existing cookie.
2. **Writes become JSON endpoints too, not just reads.** Today `/shelf/add`, `/shelf/delete`, `/notes/add` are classic form POSTs with a server-side redirect. For the frontend to feel like an actual interface (no full-page reloads for every action), these need to become JSON endpoints the frontend calls via `fetch()`, still authenticated by the same session cookie.
3. **A new `frontend/` — static HTML/CSS/JS, no build step.** Mirrors the shape `comum`'s own frontend already validated: self-hosted fonts, hand-written CSS, vanilla JS, no framework. Implements the Observatório identity (dark-first palette, amber accent used sparingly as a signal color, Spectral italic for work titles, Space Grotesk for body, Space Mono for metadata) and a responsive layout that collapses without hiding functionality (sidebar becomes a top bar on mobile, same pattern already proven in `comum`).
4. **Apply it to what already exists.** Shelf list, a work's page (with season → episode navigation), and notes (work-level and episode-level) all get real pages instead of raw strings. No new product surface in this beta — forum, events, and profile are still Beta 4/5/6.

## Decisions made in planning conversation

- **One process, not two.** `comum` splits its frontend (static files via nginx) from its API (api-gateway) as separate services; `orbita`'s appview will serve the static frontend files itself — one Go binary, one process, no nginx. Simpler, and fits a single-developer hobby project better than mirroring `comum`'s topology out of habit. CORS doesn't come up as a result — same-origin throughout.

## Open questions — need a decision before/while building

- **How literally to apply "borrowed practices" from Bluesky's `social-app`.** The research surfaced concrete things worth studying directly against their source before building (their `src/alf/` token system's shape, their breakpoint handling, their compound-component pattern for dialogs/menus adapting mobile vs. desktop) — worth a closer look at actual code once this beta starts, not just the summary from this planning conversation.

## Explicitly not in this beta

- Any new product surface — forum (Beta 4), events (Beta 5), profile (Beta 6) all come after, once the interface this beta builds already exists for them to use.
- Constellation/archetype visualization — explicitly deferred to whenever profile lands (Beta 6) or later.
- Real fan-out, affinity, feed, DMs — unrelated to this beta, still further out on `docs/ROADMAP.md`.

See [`docs/ROADMAP.md`](ROADMAP.md) for where this sits in the overall arc, and `docs/BETA0-PLAN.md`'s note on `comum` being a private repo for why earlier product decisions are referenced generically here rather than by file path.
