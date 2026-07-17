# Beta 4 — planning draft

**Status: Beta 4 done ✅** (2026-07-17). A persistent site shell now exists, with real Feed and Profile pages (placeholder content, real navigation) — every page built from here on inherits it.

## Goal

Reprioritized mid-conversation (2026-07-17): forum and events, originally next, were called less central to the product than four core surfaces — the work, the shelf, feeds, and profile. Before building feed (Beta 6) or profile (Beta 5) for real, the site needed an actual frame for them to live in — until this beta, the work page and the search page were each their own island, with no persistent navigation between them and nowhere for a feed or profile to go once built.

## Scope

1. **A persistent topbar and a real 3-column layout.** ✅ **Built.** [`frontend/common.js`](../frontend/common.js)'s `renderShell(active)` builds the mark (a thin ring with one amber point offset near the edge, never centered — "a body orbiting another, the metric of the relationship, not of the user") and the "ÓRBITA" wordmark into the topbar, and — refined after the first pass — a real sidebar/center/right-column layout: navigation (Shelf/Feed/Profile, highlighting whichever is current) moved into a text sidebar rather than sitting in the topbar, matching the split earlier product work already validated (topbar carries identity, sidebar carries the nav list). Injected into a `#shell-mount` element each page's HTML already has, rather than replacing `document.body` wholesale — keeps each page's own `<script>` tags untouched; the page's existing `#app` element gets moved (not duplicated) into the layout's center column. The right column is an honest "nothing here yet" for now — no fake content. Collapses for real under 800px: sidebar becomes a horizontal bar (same items, just reflowed), right column hides, matching the "never hide a page or action that exists on desktop" rule from earlier product work.
2. **Every existing page adopts it.** ✅ **Done.** The work page and search page each gained one line (`renderShell(...)` instead of `document.getElementById("app")`) — no other logic changed.
3. **Feed and Profile become real, reachable pages.** ✅ **Done**, deliberately as placeholders — [`frontend/feed.html`](../frontend/feed.html)/`feed.js` and [`frontend/profile.html`](../frontend/profile.html)/`profile.js` say plainly "not built yet" rather than faking content. Profile does check `/api/me` and greets the signed-in account by handle, since that's real and free to show; nothing about shelf/notes is faked.

## Decisions made in planning conversation

- **Reprioritization**: forum and events (previously Beta 4/5) moved to Beta 8/9. The four core surfaces — work (done), shelf (done, creative-space beta still ahead), feeds, profile — take priority. See `docs/ROADMAP.md` for the full reordering.
- **Shelf nav item points at `/search` for now.** There's no "my own shelf as a list" view yet in the new UI (that's Beta 5, profile) — `/search` is the real, working page closest to "interact with your shelf" today, so the nav is honest about what exists rather than linking somewhere that doesn't.
- **Mount-point injection, not a `document.body` rewrite.** Considered clearing and rebuilding the whole page body from a shared shell function, but injecting into a dedicated `#shell-mount` element is less invasive and keeps each page's existing script-loading order intact.

## Explicitly not in this beta

- Feed's real content (Beta 6) and profile's real content (Beta 5) — this beta only builds the frame they'll go in.
- Any visual redesign of the topbar for mobile beyond basic flex-wrap — revisit once there's enough real content on narrow screens to judge against.
- Forum, events, fan-out, observability, affinity, DMs, closing gaps — unrelated, see `docs/ROADMAP.md`.

See [`docs/ROADMAP.md`](ROADMAP.md) for where this sits in the overall arc.
