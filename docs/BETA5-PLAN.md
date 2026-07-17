# Beta 5 — planning draft

**Status: Beta 5 done ✅** (2026-07-17). Profile pages exist for any account, reachable by handle — not scoped to the viewer's own, per the author's call before building.

## Goal

There was no page yet about a *person* rather than a *work* — even though the shelf/notes data to build one already existed, and Beta 4 had already given the site a real frame (topbar, nav) for it to live in.

## Scope

1. **Profile for any account, not just the viewer's own.** ✅ **Built.** Confirmed with the author before writing any code: `/profile/{handle}` shows anyone's profile, not a self-only view. `/profile` with no handle redirects to the signed-in viewer's own (`/profile/{their handle}`), or prompts sign-in if there's no session.
2. **Handle → DID reverse lookup.** ✅ **Built.** [`identity.go`](../cmd/appview/identity.go)'s `resolveHandleToDID` uses the same `identity.CacheDirectory` already in place for avatar/handle resolution — bi-directionally verified, so a handle that doesn't resolve (or resolves to a different DID than expected) is a real 404, not a silent wrong page.
3. **`GET /api/profile/{handle}`.** ✅ **Built.** Combines `listShelfItemsByAccount` (already existed for `/shelf`) with a new `listNotesByAccount` (every note an account has written, across every work) — both scoped to whatever this appview already has locally. An account that's never logged into this appview (real fan-out is Beta 8) comes back with empty shelf/notes, not an error — it may just not have shown up yet, same honesty as everywhere else in this codebase.
4. **A real page, not a placeholder anymore.** ✅ **Built.** Avatar (large), handle, DID, a poster grid for the shelf (a person's shelf is a visual thing to browse, not a text list — same instrument framing as the work page's poster), and a list of every note the account has written, each linking back to the work (and the specific episode, if that's what it was about).
5. **Handles become links wherever they already appeared as plain text.** ✅ **Done.** The work page's "who has this on their shelf" list and each note's byline now link to `/profile/{handle}` instead of showing inert text.

## Decisions made in planning conversation

- **Any account's profile, decided before writing code.** The alternative (self-only, deferring "anyone's profile" to Beta 8 when fan-out exists) was raised explicitly and rejected — profile pages are more useful now even at small scale, and the plumbing (handle resolution, local-index lookups) is the same either way.
- **Empty is not an error.** An unindexed account's profile renders normally with "nothing here yet" in each section, rather than treating the absence of local data as a failure state.

## Explicitly not in this beta

- Constellation-style visualization and the geometric archetype from earlier product work — still an explicit stretch goal, not required here.
- Forum/events activity on the profile — those betas haven't landed yet.
- Anything that depends on real fan-out (seeing an account that's never logged into this appview at all) — Beta 8.

See [`docs/ROADMAP.md`](ROADMAP.md) for where this sits in the overall arc.
