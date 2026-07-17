# Órbita

> Every social network has a center. For most, that center is you.
> Here, the center is the work: the movie, the show, the album, the book you love.

This repository is the beginning of an Órbita AppView on top of the **AT Protocol** — the open protocol behind Bluesky. Portable identity via DID, data in the PDS the person already controls, record types defined in Lexicon. No server owns your cultural taste.

## Where this comes from

Órbita started as a distributed systems lab, a real product built to work through problems like resilient distributed systems, persistent state, cache, observability, horizontal scaling, and so on.

In the end, I built a product for that lab that excited me enough that this is the natural continuation: migrating the same product idea to an architecture where nobody — not even Órbita itself — owns the data of the people who use it. `orbita` is built from day one with more than just its author in mind: a public AppView, built in the open, within the AT Protocol community.

## What makes Órbita different

- **The central node is the work, not the person.** Cover, title, and type of the work come before any username.
- **No algorithmic engagement.** Chronological feed. No "trending," no like-based ranking.
- **No public popularity metrics.** Follower counts exist only on your own profile, never as public status data on someone else's.
- **Affinity isn't a number, it's a shape.** Each person's shelf draws a constellation; affinity happens when two constellations resemble each other, with no compatibility score on display.
- **Not a space for content creators.** It's a space for community to find each other through what they genuinely love.

## Current state

**Beta 0, Beta 1, and Beta 2 — done ✅** (2026-07-15, 2026-07-16, and 2026-07-17). End to end, against the real network: OAuth login, authenticated write and delete, sync via Tap against the production relay, cross-account aggregation, a real TMDB-backed search, season/episode-level catalog browsing, notes anchored to a work or to a specific episode — see [`docs/BETA0-PLAN.md`](docs/BETA0-PLAN.md), [`docs/BETA1-PLAN.md`](docs/BETA1-PLAN.md), and [`docs/BETA2-PLAN.md`](docs/BETA2-PLAN.md), which document the whole process, not just the outcome.

What exists:
- [`lexicons/social/orbita/shelf/item.json`](lexicons/social/orbita/shelf/item.json) — the first Lexicon, the schema for the gesture of adding a work to the shelf
- [`lexicons/social/orbita/note.json`](lexicons/social/orbita/note.json) — the second Lexicon, a note about a work as a whole or about one specific season/episode
- [`cmd/appview/`](cmd/appview/) — a complete Go server: real OAuth login, search-then-write (`/search`, `/shelf/add`), delete, season/episode navigation (`/works/{provider}/{id}/season/{n}/episode/{n}`) with real TMDB synopses, notes writable from the work page or from any episode page, a Tap webhook indexing both collections into a local SQLite database, `/shelf` and `/works/{provider}/{id}` (cross-account "who has this") both showing real titles and posters via a TMDB cache
- **Órbita's first real pieces of data on the AT Protocol**: `social.orbita.shelf.item` and `social.orbita.note` records written via OAuth (full PAR/PKCE/DPoP, no shortcuts) on the author's real account, synced by Tap against the **real production relay** (`relay1.us-east.bsky.network`) — same code, same binary that runs against the local sandbox, only the config URL changes
- [`scripts/dev-pds/`](scripts/dev-pds/) — a disposable local PDS + PLC, no Postgres, no TLS, for studying and testing without depending on a real account
- Pipeline validated end to end, repeatedly, against both the local sandbox and the production network — full architecture documented in [`docs/architecture-beta0-local.md`](docs/architecture-beta0-local.md)

Next up: no beta scoped yet — affinity computation (comparing shelves across accounts) is the likely candidate, but nothing is planned until it's discussed.

This is a hobby turning into an idea, documented in public. Progress and decisions also go out on the [@orbita.bsky.social](https://bsky.app/profile/orbita.bsky.social) profile *(coming soon)*.

## Why AT Protocol

If Órbita's server shut down today, someone's cultural shelf would disappear with it. AT Protocol solves exactly that:

- **DID** — portable identity, independent of any specific server
- **PDS** — data lives in a repository the person themselves controls (the same one they already use on Bluesky, or a self-hosted one)
- **Lexicon** — the record format (`social.orbita.shelf.item`, `social.orbita.note`, …) is a public contract, not an internal database detail
- **AppView** — Órbita becomes a lens over data that lives scattered across the network, not the owner of it

## License

[AGPL-3.0](LICENSE). Same choice as Mastodon, for the same reason: the network-use clause closes the loophole plain GPL leaves open — without it, someone could take the code, modify it, and run it as a hosted service without ever having to give the modifications back to the community, since users only interact over the network and never receive a copy of the software. Open to study, use, and contribute to; protected against becoming someone else's closed product.

## AI use in development

This project is developed with active use of AI assistants, as a research, implementation, and documentation partner, under my direction and review at every decision. No product principle, architecture decision, or line of code goes in here without me understanding and validating the why first; that's the actual reason everything stays documented this closely (`docs/BETA0-PLAN.md`, the architecture diagrams) — including mistakes I made and corrected along the way, which stay on the record instead of being hidden.

I disclose this openly because transparency is already a non-negotiable principle of Órbita as a product; it would be inconsistent to ask that of the social network and hide it from the process that builds it.

## Contributing

There's no formal process yet — Beta 0 is close to done, but it's still one person's work. If the idea resonates with you, open an issue with a question, a criticism, or interest in helping. The goal remains finding out, in public, whether and how this becomes more than one person's work.
