# Beta 0 — planning draft

**Status: Beta 0 done ✅** (2026-07-15). All 7 items of the completion criteria ready and confirmed on the real network, not just in a sandbox — see "Progress" below. Still a living document. Developed with active AI use under direct review — see "AI use in development" in [`README.md`](../README.md).

## Goal

In the same spirit as a classic Beta 0 ("product before infrastructure"), the goal here is to prove the smallest possible slice of Órbita running end to end over real AT Protocol — before any bigger ambition (own PDS, relay, firehose, multiple record types, real federation between AppViews).

Feel the minimal problem first: authenticate against an identity that isn't ours, write a record into a repository we don't control, and read that data back.

## Progress

- [x] Lexicon `social.orbita.shelf.item` — [`lexicons/social/orbita/shelf/item.json`](../lexicons/social/orbita/shelf/item.json)
- [x] Go module skeleton (single module) — [`cmd/appview/main.go`](../cmd/appview/main.go). `go 1.26` now (bump forced by `indigo`, which requires >=1.26 — no longer `1.25.0` like in the initial commit)
- [x] OAuth against a real account (`atproto/auth/oauth`) — full login, end to end, against `pydavi.bsky.social` for real. See the networking saga (WSL2/localhost/IPv6) in [`docs/architecture-beta0-local.md`](architecture-beta0-local.md)
- [x] Writing the record to the PDS via an authenticated session — `oauthSess.APIClient().Post(...)` for real, against the real account. Record confirmed via `listRecords` on the production PDS (`agaric.us-west.host.bsky.network`): `at://did:plc:kpsswg4vfyzjvxp577wsqh3t/social.orbita.shelf.item/3mqlbnf4e7m2e` — Órbita's first real piece of data on the AT Protocol
- [x] Local development PDS — [`scripts/dev-pds/run.mjs`](../scripts/dev-pds/run.mjs), via `@atproto/dev-env` (`TestNetworkNoAppView`: just PLC + PDS, no Bsky AppView/Ozone/Postgres)
- [x] Full manual cycle validated via `curl` (create account → write `shelf.item` → read it back) — see [`docs/architecture-beta0-local.md`](architecture-beta0-local.md)
- [x] Webhook + Tap consumption, filtered for `social.orbita.shelf.item` — run for real, backfill confirmed, see [`docs/architecture-beta0-local.md`](architecture-beta0-local.md)
- [x] Local database — [`cmd/appview/db.go`](../cmd/appview/db.go), pure-Go SQLite (`modernc.org/sqlite`, no CGO), `shelf_items` table. [`webhook.go`](../cmd/appview/webhook.go) now parses the real Tap event and indexes it, not just logs it
- [x] Simple page listing what's been synced — [`cmd/appview/list.go`](../cmd/appview/list.go), `GET /shelf`. Tested end to end against the local PDS: write → Tap → webhook → SQLite → page, every hop confirmed
- [x] Tap pointed at the **real relay** (`https://relay1.us-east.bsky.network`, default, zero URL configuration) — confirmed: it enumerated real repos by collection, found the author's account, backfilled via CAR from the production PDS, delivered via webhook. `GET /shelf` showing the real record (`tmdb-movie/603`, `did:plc:kpsswg4vfyzjvxp577wsqh3t`) side by side with the sandbox one — same binary, same code, zero change, only the config URL

## Where each piece of data lives

A distinction worth keeping fixed in code, not just in conversation: **the PDS is the source of truth** (the record the person authored, in their own repository); **the AppView is a derived view, disposable and rebuildable** (our indexed copy, never authoritative — the same role a Redis cache already plays in any traditional backend, except now the "source of truth" is also out of our control). Even the logged-in user's own write goes through Tap before it shows up in our database — there's no shortcut for direct local writes, not even for the data of the person currently using it.

## Study reference

The official **Statusphere** tutorial (`atproto.com/guides/statusphere-tutorial`) is the closest thing to a "Beta 0" that the AT Protocol documentation itself offers. Architecture confirmed (verified against two sources — the tutorial page and the example repository):

- **OAuth** against the PDS the person already has (we don't host our own PDS in this beta) — permission scope restricted to the custom Lexicon
- **Custom Lexicon** — versioned record schema, with type codegen
- **Real-time sync** via **Tap** (`github.com/bluesky-social/indigo/cmd/tap`) — a tool that watches the network stream, filters by collection, and delivers events via webhook; this is what replaces consuming Jetstream/firehose by hand at this stage
- **Local database** (SQLite via Kysely, in the reference tutorial) — the AppView only indexes what's already been synced, never queries the PDS live on a read request
- **Minimal frontend** — just enough to prove the data came back

## What differs from Statusphere

Statusphere uses a single Lexicon (`xyz.statusphere.status`, an emoji as status). For Órbita, the closest equivalent to the product's "fundamental gesture" — the shelf has always been described as Órbita's most important action — is a first Lexicon of our own: `social.orbita.shelf.item`, already written and validated against real examples (`xyz.statusphere.status`, `app.bsky.feed.like`) in [`lexicons/social/orbita/shelf/item.json`](../lexicons/social/orbita/shelf/item.json).

No notes, no affinity, no constellation, no work type — just the gesture of adding something to the shelf.

## Decisions already closed

1. **Stack: Go.** Confirmed there's no technical blocker — `github.com/bluesky-social/indigo` is the official Go monorepo for Bluesky/AT Protocol (the same one Tap comes from) and covers exactly what this beta needs: `atproto/auth/oauth` (OAuth client), `atproto/identity` (DID/handle resolution), `atproto/lexicon` (schema validation), `atproto/repo` (repository structure), `atproto/atcrypto` (signing/crypto). It's not a workaround — it's the reference implementation, the same one that runs Bluesky's real infrastructure.
   - **Risk accepted and documented, not hidden:** Indigo itself declares active development — "features and software interfaces have not stabilized and may break or be removed." In other words: expect some API breakage on dependency updates, and pin the version explicitly from the very first `go.mod`. Same spirit of risk already accepted consciously in earlier dependency decisions (e.g. external API rate-gating, fail-open cache) — named here, not discovered later.

3. **Test identities: hybrid.** Two environments, different purposes:
   - **Local development PDS — already running.** `indigo/cmd/pds` doesn't exist (correction: not a Go tool, a wrong assumption I had made). The real tool is `@atproto/dev-env` (an npm package from the Bluesky team itself) — but the published binary (`bin.js`) boots their entire test network (PDS + Bsky AppView + Ozone + Bsync, requiring Postgres for the AppView schema), too heavy for what we need. We instead use the `TestNetworkNoAppView` class, which only boots PLC + PDS, no Postgres — our own script at [`scripts/dev-pds/run.mjs`](../scripts/dev-pds/run.mjs). Fast, disposable, no rate limits, no polluting the real network with test records. Same role a local Postgres/Redis plays in any traditional backend.
   - **Real Bluesky account(s)** for periodic real interoperability validation — confirming that a `social.orbita.shelf.item` record written by this code survives on a real production PDS, not just in the controlled environment. Technically frictionless: the protocol doesn't require Bluesky's approval to write a custom NSID into someone's repository — that's exactly the point of AT Protocol.
   - Practical criterion: Beta 0 only counts as validated (item 5) once it passes in both environments, not just locally.

4. **License: AGPL-3.0.** Same choice as Mastodon, for the same specific reason: the network-use clause closes the loophole plain GPL leaves open — without it, someone could take the code, modify it, and run it as a hosted service without ever having to distribute the modifications (users only interact over the network, never receive a copy of the software). AGPL requires making modified code available to anyone using the service over the network, not just to whoever receives a binary copy. It's the right protection against "someone closes this and sells it" without blocking legitimate use/study/forking.

5. **"Beta 0 done" criterion** — ✅ **actually reached, on the real network** (2026-07-15): OAuth login working against `pydavi.bsky.social`, `social.orbita.shelf.item` record created on that account's production PDS, Tap (pointed at the real relay, not the sandbox) syncing that record into a local database via webhook, and `GET /shelf` listing the result. No UI beyond that, no second Lexicon, no affinity — exactly the agreed scope, nothing more.

2. **Work identification: migrated to the `{provider, id}` format.** Replaced the free string (`workSlug`) — the original decision was accepted as an intermediate step, revisited after the ecosystem research below. Already implemented and validated end to end against the local PDS (`work: {provider: "tmdb-movie", id: "603"}` — the Matrix's real TMDB ID), including delivery via Tap/webhook with `"live": true`. See the full schema in [`lexicons/social/orbita/shelf/item.json`](../lexicons/social/orbita/shelf/item.json).

## Ecosystem research: what Popfeed and Skylights teach

Before settling on a direction for work identification, we researched two real AT Protocol media apps and queried the real network (not just secondary documentation) — `com.atproto.identity.resolveHandle`, `plc.directory`, `com.atproto.repo.describeRepo`, and `listRecords` against the real `popfeed.social` account, plus Skylights' public lexicons (`github.com/Gregoor/skylights`).

**Three different ways of referencing a work, found in practice:**

1. **Popfeed duplicates everything** — every `social.popfeed.feed.listItem` carries the full title, genres, poster, and release date, tied together only by a loose `identifiers.tmdbId`/`igdbId`. No canonical record.
2. **Skylights uses a minimal external reference** — `{"ref": "tmdb:m", "value": "603"}` inside the user's own record. No metadata duplication, no second record type invented.
3. **What we had planned** (a strongRef to a `social.orbita.work` published by our own service account) — neither real app does this.

**Decision: we follow Skylights' pattern**, not our old plan. It's already nearly identical to the external resolution the original Órbita already validates (TMDB/MusicBrainz/Open Library, search → normalize → stable identifier) — only the exposure format changes, becoming a `{provider, id}` pair inside `shelf.item` itself, instead of a loose free string or a new record type. **Removes entirely the need for a service account, self-hosting our own PDS, and the `social.orbita.work` record type** — none of those three things is necessary. See the revised schema proposal below.

**Extra finding that validates the product's principles, not just the architecture:** an external UX critique of Popfeed (unaffiliated) points out that the app is "caught between two chairs" — a tracker (like Trakt) and a social network (like Letterboxd) at the same time, inheriting infinite scroll and Bluesky's feed logic in a way that gets in the way of people who just want to log what they consumed. It's exactly the problem the original product's principles 2 and 4 (no algorithmic engagement, no infinite scroll, work-before-person hierarchy) avoid by design — concrete validation, not a hypothesis.

**What exists in the ecosystem that we shouldn't copy:** `social.popfeed.challenge.*` is gamification (consumption challenges/goals) — directly contradicts principle 4 ("no addictive design"). We also found that **not even Popfeed self-hosts its own service account** (their PDS is at `*.host.bsky.network`, Bluesky's own infra) — reinforces that self-hosting wasn't necessary even before we dropped that idea for a different reason.

## Work identification — revised schema (replaces the strongRef plan), already implemented

Migrated immediately, not deferred — it didn't require new infrastructure, just a different field format. Following the idiomatic pattern confirmed in Skylights (`{"type": "ref", "ref": "#work"}` pointing to a local def, not an inline object — verified against their `rel.json` before writing, since the Lexicon spec doesn't make clear whether inline nesting without `ref` is valid):

```json
"work": { "type": "ref", "ref": "#work" }
// ...
"work": {
  "type": "object",
  "required": ["provider", "id"],
  "properties": {
    "provider": { "type": "string", "knownValues": ["tmdb-movie", "tmdb-tv", "musicbrainz", "open-library"] },
    "id": { "type": "string", "minLength": 1, "maxLength": 200 }
  }
}
```

The old records on the local PDS (`workSlug: "matrix"`, `workSlug: "duna-parte-2"`) are left behind as orphaned data from the previous schema — the sandbox is disposable on purpose, no migration needed.
