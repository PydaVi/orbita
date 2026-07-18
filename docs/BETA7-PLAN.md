# Beta 7 — planning draft

**Status: Beta 7 done ✅** (2026-07-18). Nooks exist — a person's own shelf is now organized and presented through them, not a single flat grid.

## Goal

Flagged early (2026-07-17) as potentially **the apex of the whole product**: the shelf is the one surface entirely about a person's own taste and judgment, and it didn't reflect that at all — just a flat, chronological list. Deliberately left under-scoped in the roadmap sketch until this beta got its own real planning conversation, across several rounds:

- Confirmed a **nook is the primary way a shelf is organized and shown to visitors** — not an optional side list layered on top. What a visitor sees on someone's profile *is* their nooks (plus an honest "unsorted" catch-all for anything not yet placed).
- Named after rejecting "playlist" — too suggestive of music and of sequence/reproduction, when the point is to be completely open to whatever the person wants it to mean. The author's own reference point: how personal and expressive an old Tumblr blog's theme used to feel, before feeds homogenized everything.
- Confirmed a nook needs its **own Lexicon record** (not a label field on `shelf.item`) — free ordering and a single shareable thing both require it.
- Confirmed a curated, non-free set of visual presentation choices (a handful of themes) rather than an open color picker — personalization without breaking the product's own visual coherence.

## Scope

1. **`social.orbita.shelf.nook`, a new Lexicon.** ✅ **Built.** [`lexicons/social/orbita/shelf/nook.json`](../lexicons/social/orbita/shelf/nook.json) — `name`, optional `description`, an ordered `works` array (same `{provider, id}` shape as `shelf.item#work`, reused rather than duplicated), and an optional `style.theme` from a small known set (`default`/`warm`/`cool`/`midnight`). Order is just array order — entirely the author's choice, nothing derived.
2. **Only works already on the shelf can go in a nook.** ✅ **Built.** Enforced by the appview at write time (`shelfItemExists` in [`nooks.go`](../cmd/appview/nooks.go)) — the Lexicon itself has no way to validate against another collection's contents.
3. **Editing is a whole-record replacement, not incremental.** ✅ **Built.** A nook has no separate membership record type — renaming, reordering, or adding/removing a work all just resend the full `works` array via `com.atproto.repo.putRecord`. This is the first record in this project that gets edited after creation, which meant the webhook had to start handling `action: "update"` events, not just `"create"` — scoped specifically to nooks for now, since nothing else has an edit path yet.
4. **The profile groups by nook.** ✅ **Built.** `GET /api/profile/{handle}` returns `nooks` (each with its own works resolved to title/poster) and `unsorted` (shelf items not in any nook) instead of one flat `shelf` array. Unsorted is never hidden or folded into a default grouping — a freshly shelved work sits there honestly until the person places it somewhere.
5. **Curated presentation themes.** ✅ **Built.** Four themes (`default`/`warm`/`cool`/`midnight`), implemented as CSS classes derived from the existing Observatório tokens (`color-mix` against `--signal`/`--surface`, no new hex values introduced) — personalization stays inside the product's own palette rather than opening a free color picker.
6. **Nook management lives on `/shelf`, presentation lives on `/profile`.** ✅ **Built.** Matches the split `/shelf`/`/profile` already established in Beta 4/5 — manage your own things privately in one place, see how they present publicly in another.
7. **Direct manipulation, reworked after first review (2026-07-18).** ✅ **Built.** The first pass used a form (name/description/theme fields, a checkbox list to pick works) — reviewed and replaced: creating a nook is now a single click plus a name, no other fields up front; populating and reordering one is native HTML5 drag-and-drop (no library) — drag a poster from Unsorted into a nook, drag within a nook to reposition, drag back out to un-sort it. Posters render everywhere works are picked or arranged, never plain text names. Nooks render above the page, the Unsorted grid below — the visual order matches "this is organized, this isn't" directly. A small "remove from shelf" control and a lightweight settings toggle (description/theme) stay available, just secondary to the drag interaction itself.

## Decisions made in planning conversation

- **"Nook," not "playlist."** Named after the author explicitly rejected "playlist" for presuming music/sequence, and wanted something as open as an old Tumblr theme. English chosen deliberately for this one (unlike most of this product's still-Portuguese vocabulary) — a call made mid-conversation, not a default.
- **A dedicated Lexicon record, decided before writing it** — a label field on `shelf.item` was considered and rejected: it can't give free ordering or a single shareable object, both explicitly wanted.
- **`nook_items` is fully rebuilt on every (re)index**, matching `putRecord`'s "whole record replaced" semantics, rather than tracked as incremental diffs.
- **Webhook now handles `"update"` actions — scoped to this need, not applied broadly.** Every other collection here still only ever arrives as `"create"` in practice (no edit path exists for them), so this isn't a blanket policy change, just closing the specific gap nooks introduced.
- **Same class of bug as Beta 1's "Titanic" incident, caught again.** The OAuth scope list in `oauth.go` never gained entries for `social.orbita.repost` or `social.orbita.shelf.nook` when those collections were added — nook creation failed against the real PDS with the same `ScopeMissingError` shape as the original delete-scope bug. Fixed by adding both, with nook explicitly requesting `create`, `update`, *and* `delete` — the first collection in this project needing all three from day one. Existing sessions can't be retroactively upgraded; signing out and back in is required to pick up the new scope, same as the original incident.

## Explicitly not in this beta

- Constellation/archetype awareness of nooks (Beta 8, next) — how those visualize a shelf organized this way is an open question for that beta, not resolved here.
- Any change to how `/shelf` (the private management list) itself looks — nooks organize the *public presentation*, not the private list.

See [`docs/ROADMAP.md`](ROADMAP.md) for where this sits in the overall arc.
