# Beta 0 architecture — local environment

> Educational document: explains how the pieces fit together in the local development environment, not a new decision (decisions already live in `docs/BETA0-PLAN.md`). Written after validating each hop by hand, via `curl`, not just in theory.

## Overview — from the local PDS to our appview

```
┌──────────────────────────────────────────┐
│  scripts/dev-pds/run.mjs (Node process)   │
│                                            │
│   ┌──────────┐        ┌────────────────┐ │
│   │   PLC    │        │      PDS       │ │
│   │ :33195   │◄───────┤     :2583      │ │
│   │ (fake,   │  DID   │  (real         │ │
│   │ in       │  ops   │   @atproto/pds,│ │
│   │ memory)  │        │   same code as │ │
│   └──────────┘        │   Bluesky's)   │ │
│                        └───────┬────────┘ │
└────────────────────────────────┼──────────┘
                                  │ raw WebSocket
                                  │ com.atproto.sync.subscribeRepos
                                  ▼
                        ┌──────────────────┐
                        │    Tap (local)    │
                        │  points at :2583   │
                        │  instead of the    │
                        │  real production   │
                        │  relay              │
                        │                    │
                        │  filters:          │
                        │  social.orbita.    │
                        │  shelf.item        │
                        └─────────┬──────────┘
                                  │ webhook (HTTP POST)
                                  ▼
                        ┌──────────────────┐
                        │  cmd/appview (Go)  │
                        │  /webhook handler  │
                        │  → local database   │
                        └──────────────────┘
```

## Why this works without any relay

A real relay aggregates the firehose of many PDSes and re-exposes it as a single stream. Tap doesn't distinguish a relay from an individual PDS — the code (`cmd/tap/firehose.go`) takes the configured URL, swaps the scheme for `ws`/`wss`, and appends `xrpc/com.atproto.sync.subscribeRepos` to it, always. Since every PDS already exposes that same path (it's its raw firehose, before any aggregation), pointing Tap straight at our local PDS works — it's the degenerate case where "the whole network" and "a single source" coincide, because there's only one PDS in our sandbox.

**Important consequence:** the same Tap binary, with no code change at all, serves both scenarios — only the config URL changes:

| Scenario | Tap URL | What it sees |
|---|---|---|
| Local dev (this document) | `http://localhost:2583` (our PDS) | only the records we ourselves created in the sandbox |
| Real Beta 0 (real Bluesky account) | `https://relay1.us-east.bsky.network` (default, nothing configured) | any `social.orbita.shelf.item` record written by any real account on the network |

## What we already validated by hand (no Go, no Tap yet)

Real sequence, run via `curl` against the local PDS:

1. `POST /xrpc/com.atproto.server.createAccount` → created `did:plc:nuftb5ux5jsmfsitowhsu4ab`, with a full DID document (`alsoKnownAs`, `verificationMethod`, `service` pointing at `:2583`)
2. The access token received has header `{"typ":"at+jwt","alg":"HS256"}` — confirming the domain separation from the XRPC spec
3. `POST /xrpc/com.atproto.repo.createRecord` (`collection: social.orbita.shelf.item`) → returned `uri` (`at://did:plc:.../social.orbita.shelf.item/3mqgdrhodjk2i`) and `cid` — the pair that becomes a strongRef whenever another record needs to point at this one
4. `GET /xrpc/com.atproto.repo.getRecord` → read the same record back, intact

Detail noted during the test: the response carried `"validationStatus": "unknown"` — the PDS accepts any NSID without validating it against the Lexicon, because it has no way to know our schema exists. Schema validation is the client's responsibility, not the server's.

## Pipeline validated end to end

We ran the real Tap (binary built via `go install github.com/bluesky-social/indigo/cmd/tap`), configured like this:

```
TAP_PLC_URL=http://localhost:33195
TAP_RELAY_URL=http://localhost:2583        # our local PDS, not the real relay
TAP_SIGNAL_COLLECTION=social.orbita.shelf.item
TAP_COLLECTION_FILTERS=social.orbita.shelf.item
TAP_WEBHOOK_URL=http://localhost:8092/webhook
TAP_NO_REPLAY=true
```

Writing a second record (`workSlug: duna-parte-2`) with Tap already connected, the log showed the real **backfill** mechanism at work: `"fetching repo from PDS"` → `"parsing repo CAR"` → `"iterating repo records"`. Tap didn't just deliver the new event — it went and fetched the whole repository (exported as a CAR) because this was the first time it saw this DID, and reprocessed everything. Result: our `cmd/appview` received **three** events on `/webhook`, not one:

```json
{"id":1,"type":"identity","identity":{"did":"did:plc:...","handle":"handle.invalid","is_active":true,"status":"active"}}
{"id":2,"type":"record","record":{"collection":"social.orbita.shelf.item","action":"create","record":{"workSlug":"matrix",...}}}
{"id":3,"type":"record","record":{"collection":"social.orbita.shelf.item","action":"create","record":{"workSlug":"duna-parte-2",...}}}
```

`id:2` is the `matrix` record, written **before** Tap even existed — it only came through because of the backfill.

**`"handle": "handle.invalid"` is not a bug.** Our test handle (`alice.test`) isn't a real domain, so the bidirectional handle↔DID resolution we studied (DNS TXT / `.well-known`, against `alsoKnownAs` in the DID document) has no way to confirm it — Tap honestly marks it invalid instead of pretending everything's fine. It's the same security check from the spec, working.

## The `work` schema changed — repipeline confirmed

After the ecosystem research (see `docs/BETA0-PLAN.md`), `workSlug` (free string) became `work: {provider, id}` (minimal external reference, e.g. `{"provider": "tmdb-movie", "id": "603"}`). We wrote a new record with the updated schema and confirmed the whole pipeline again, this time with Tap already running (no backfill needed):

```json
{"id":4,"type":"record","record":{"live":true,"collection":"social.orbita.shelf.item","action":"create","record":{"work":{"id":"603","provider":"tmdb-movie"},"createdAt":"2026-07-14T02:25:47.000Z"}}}
```

`"live": true` this time — a genuine live event, not a backfill, confirming that the pipeline reacts to new writes in real time, not just on first discovery of the repository. The old records (`workSlug: "matrix"`, `workSlug: "duna-parte-2"`) remain on the local PDS as orphaned data from the previous schema — the sandbox is disposable, no migration needed.

## Real OAuth — why the local PDS doesn't work for this, and the networking saga to test with a real account

### The local PDS was never going to work here, by design

The `Resolver` in the `atproto/auth/oauth` package (`resolver.go`) requires `https://` and forbids an explicit port in three methods (`ResolveAuthServerURL`, `ResolveAuthServerMetadata`, `ResolveClientMetadata`) — no configurable exception, it's fixed logic in the code, not a swappable field (the type is concrete, not an interface). This isn't about the `client_id` (which can be `http://localhost`, the dev exception we already use) — it's about the **authorization server itself** never being resolvable over plain HTTP with a port. Makes sense: allowing that in general would open a real SSRF hole. Conclusion: OAuth login can only be tested against a **real PDS** — exactly the role the hybrid-identities decision already anticipated for this situation.

### The saga to reach the callback (WSL2 + browser)

Running the appview here (the assistant's environment) wasn't enough — the `127.0.0.1:8092` here isn't the `127.0.0.1:8092` the author's browser sees, even on the same machine/WSL2 (confirmed by a test: `bind: address already in use` proved the network *is* shared at that level, so the problem was further along, between WSL2 and the browser itself).

Step by step of what happened:
1. `http://127.0.0.1:8092/oauth/callback` as the redirect_uri → `ERR_CONNECTION_REFUSED` in the browser, even with the appview running in the author's own terminal (not just here). An isolated test (`http://127.0.0.1:8092/health` directly, no OAuth) gave the same error — confirming the problem was purely networking, nothing to do with OAuth.
2. The author's empirical finding: `http://localhost:8092/health` **worked**, `127.0.0.1` didn't — exact cause not identified (hypothesis: a local proxy/VPN with a bypass rule for the name "localhost" but not the literal IP).
3. We switched the redirect_uri to `http://localhost:8092/oauth/callback` → PAR was **rejected by Bluesky's real server** (`HTTP 400 invalid_request`) — the spec only accepts the literal forms `127.0.0.1`/`[::1]`, "localhost" as text isn't one of them, and the server genuinely validates this.
4. Next hypothesis: if "localhost" resolves and "127.0.0.1" doesn't, maybe the environment prefers IPv6 — we tried `http://[::1]:8092/oauth/callback`. **Worked on both sides**: PAR accepted by Bluesky (a valid literal form) *and* reachable by the author's browser.

Full login, end to end, against `pydavi.bsky.social` for real: `did:plc:kpsswg4vfyzjvxp577wsqh3t` (confirmed matching `com.atproto.identity.resolveHandle` against Bluesky's public API).

**Lesson for anyone repeating this on another machine:** if `127.0.0.1` can't reach the callback, try `[::1]` before touching WSL2/Windows network configuration (`.wslconfig`, `netsh portproxy`) — it might be just that.

A periodic error also showed up (`"failed to enumerate network"`, HTTP 401) — a separate attempt by Tap to enumerate pre-existing repos by collection, which requires auth we hadn't configured; it doesn't affect the live firehose, which connected and delivered normally.

## Real write via OAuth — confirmed on the production network

`cmd/appview/oauth.go` + `cmd/appview/shelf.go` replace the manual `curl`: real login (`StartAuthFlow`/`ProcessCallback`, PAR+PKCE+DPoP handled inside the library) and authenticated write (`oauthSess.APIClient().Post(ctx, "com.atproto.repo.createRecord", ...)`). Tested against the author's real account (`pydavi.bsky.social`, not the local PDS — reason in the section above), and **confirmed on the network**, not just by the success screen:

```
GET .../xrpc/com.atproto.repo.listRecords?repo=did:plc:kpsswg4vfyzjvxp577wsqh3t&collection=social.orbita.shelf.item
→ at://did:plc:kpsswg4vfyzjvxp577wsqh3t/social.orbita.shelf.item/3mqlbnf4e7m2e
```

Órbita's first real piece of data on the AT Protocol — not sandbox, not backfill, written by our own Go code.

## Local database and listing — full pipeline confirmed in the sandbox

`cmd/appview/db.go` (pure-Go SQLite, `modernc.org/sqlite`, `shelf_items` table) + `webhook.go` rewritten to parse the real Tap event (`type: "record"`, `action: "create"`, collection `social.orbita.shelf.item`) and index it, instead of just logging it. `list.go` exposes `GET /shelf`, reading straight from the database.

Tested end to end against the local PDS (a new record, TMDB tv id `1396` — Breaking Bad): write → Tap (already tracking the test DID) → webhook → `INSERT` into SQLite → `GET /shelf` showing the item, all in the same cycle, no manual intervention at any intermediate hop.

## Tap against the real relay — the last item, confirmed

We started a second Tap instance, **without configuring `TAP_RELAY_URL` or `TAP_PLC_URL`** — the defaults are already the production relay (`https://relay1.us-east.bsky.network`) and the real `plc.directory`. Only `TAP_SIGNAL_COLLECTION`, `TAP_COLLECTION_FILTERS`, and `TAP_WEBHOOK_URL` (pointed at the same `:8092`), plus a different `TAP_BIND` to avoid colliding with the local instance.

The log showed something the local instance couldn't do (there, "enumeration" failed with 401 — our fake PLC doesn't implement that endpoint): against the real network, enumeration **worked**:

```
"enumerated repos by collection batch" collection=social.orbita.shelf.item count=1
"finished enumerating network, sleeping for 1 day"
"starting resync" did=did:plc:kpsswg4vfyzjvxp577wsqh3t
"fetching repo from PDS" pds=https://agaric.us-west.host.bsky.network
```

Found the author's real account (the only one with a record in our collection, count 1), fetched the real repository from the production PDS, ran a backfill — no new write needed. `GET /shelf` started showing both records side by side, sandbox and production:

```
tmdb-tv/1396    — did:plc:nuftb5ux5jsmfsitowhsu4ab   (local sandbox)
tmdb-movie/603  — did:plc:kpsswg4vfyzjvxp577wsqh3t   (real network, pydavi.bsky.social)
```

**Same binary, same Go code, zero change in logic** — only the config URL told apart "playing in the sandbox" from "working for real against the whole Bluesky network." It's the most concrete proof we have that the design (AppView as a derived, rebuildable view, never owning the data) works the same way in both worlds.

## Beta 0 — done

- [x] Run the real Tap, pointed at the local `:2583`, and confirm it delivers a webhook when a new `social.orbita.shelf.item` is written
- [x] `cmd/appview` gets a `/webhook` handler that receives this and indexes it into a local database
- [x] Replace the manual `curl` with real Go code — full OAuth, write confirmed on the real network
- [x] Local database — SQLite, `shelf_items` table, automatic indexing via webhook
- [x] Simple page listing what's been synced — `GET /shelf`
- [x] Tap pointed at the real relay — confirmed, enumeration + backfill working against the production network

See the full checklist and decisions in [`docs/BETA0-PLAN.md`](BETA0-PLAN.md).
