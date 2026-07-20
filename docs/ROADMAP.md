# Roadmap — macro view

This is a sketch, not a plan. Each beta below still gets its own `BETA{N}-PLAN.md`
with real scope, open questions, and a status once we actually start it — same
process as Beta 0/1/2. This file exists so the order and the "why this, why
now" of the whole arc is visible in one place, before any of it is detailed.

Beta 0 through 7 are done — see [`BETA0-PLAN.md`](BETA0-PLAN.md),
[`BETA1-PLAN.md`](BETA1-PLAN.md), [`BETA2-PLAN.md`](BETA2-PLAN.md),
[`BETA3-PLAN.md`](BETA3-PLAN.md), [`BETA4-PLAN.md`](BETA4-PLAN.md),
[`BETA5-PLAN.md`](BETA5-PLAN.md), [`BETA6-PLAN.md`](BETA6-PLAN.md),
[`BETA7-PLAN.md`](BETA7-PLAN.md). Real UI
(Beta 3) landed ahead of everything else on this list — not last, as
originally sketched — precisely because a profile is hard to reason about
without a screen to look at, and Beta 4 gave every page from here on a
persistent shell (topbar, nav) to live inside instead of being its own
island. What follows is what's left before there's a beta worth actually
presenting to someone outside this project.

## Beta 6 — feed ✅

Three tabs — **Shelf** (main: notes from anyone about works on your own
shelf, obra-first), **Following** (notes from people you follow, via the
existing Bluesky follow graph, plus reposts by anyone followed, attributed
— never counted), **Affinity** (an honest placeholder — needs the Jaccard
computation from Beta 13). Chronological, no ranking, only
`social.orbita.note`. Notes also gained real reply threads (root/parent,
same shape earlier product work already established) — see
[`BETA6-PLAN.md`](BETA6-PLAN.md) for the full account, including two
corrections caught mid-build: the original sketch below only named a
single "Following" tab before the real three-tab shape was restored, and
conversation on notes was initially assumed to be forum-only before the
author pointed out that was never actually true.

<details>
<summary>original sketch (superseded, kept for history)</summary>

**Problem:** right now there's no page that's actually useful to check
day-to-day — everything lives on a single work's or person's page. A feed
is what turns this from "a place to look something up" into "a place to
come back to." Reprioritized ahead of forum/events/fan-out (2026-07-17) as
one of the product's four core surfaces (work, shelf, feed, profile).

**Rough shape:** reuse the existing Bluesky follow graph
(`app.bsky.graph.follow`) instead of inventing a parallel "follow" concept
— chronological, deterministic, no ranking, same non-negotiable shape the
product has always used. Only pulls from `social.orbita.note` — forum
comments, whenever they exist, are deliberately **not** feed material
(confirmed 2026-07-17): a note is a voice meant to circulate to people who
follow you, a forum comment is a conversation confined to the work's own
space. Open question carried over from before this reprioritization: at
what scale is this actually meaningful before real fan-out (below) exists —
likely starts scoped to accounts that have already logged into this
appview, same small-scale aggregation every beta so far has used, with
true cross-network discovery arriving whenever fan-out does.

</details>

## Beta 7 — nooks: the shelf as a creative space ✅

The shelf used to be a flat, chronological list — no room for a person's
own curatorial voice. Flagged early as potentially **the apex of the whole
product**, and given its own real planning conversation across several
rounds before any code, since it was deliberately left under-scoped in the
original sketch. What shipped: **nooks** (`social.orbita.shelf.nook`) — a
person's own, freely-ordered, cross-media groupings of works already on
their shelf, each with an optional description and one of a few curated
presentation themes (never a free color picker). A nook is now the
*primary* way a shelf is organized and shown to visitors, not a side
layer — the profile groups by nook, with an honest "unsorted" catch-all
for anything not yet placed.

Named "nook," not "playlist" — rejected for presuming music and sequence
when the goal was to stay as open as possible; the author's own reference
point was how personal an old Tumblr theme used to feel. See
[`BETA7-PLAN.md`](BETA7-PLAN.md) for the full account, including the
"whole-record-replacement" editing model this introduced (the first
record in this project that gets edited after creation, which meant the
webhook had to start handling `update` events, not just `create`).

## Beta 8 — constellation and archetype on the profile ✅

**Problem:** flagged in review after Beta 5 shipped (2026-07-17): profile
pages don't yet have the constellation-style visualization or the
geometric archetype from earlier product work — both were named as
explicit stretch goals in Beta 5, not gaps to fix retroactively. Placed
right after the shelf-as-creative-space beta on purpose: what a
constellation actually visualizes depended on how Beta 7 resolved the
shelf's own shape — building this first would have risked visualizing
something about to change underneath it.

**What shipped:** a reimagining, not a port — see
[`BETA8-PLAN.md`](BETA8-PLAN.md) for the full account. This catalog never
built a genre/tag pipeline, so there's no shared vocabulary to hash into
cross-profile anchor positions the way earlier work did. What this
product has instead, uniquely, is a nook's own **theme** — a small,
curated, shared vocabulary (Beta 7) — so the constellation is anchored on
theme, not genre, with provider and decade as secondary pulls. Nook
membership is already a real, deliberate grouping here, so edges between
works in the same nook are drawn directly, not inferred from overlap.
Archetype is fully redone: two metrics recomputed against this project's
own shape (spread across theme regions; cohesion as the share of a shelf
inside its single biggest nook — no union-find needed, since a nook
already *is* the grouping), and nine entirely new names, none reused from
anywhere else.

## Beta 9 — forum

**Problem:** a work's page today only ever shows shelf items and notes.
Longer-form discussion per work — posts and comments, not single notes —
exists in earlier product work and hasn't been brought over yet. Demoted
below the four core surfaces (2026-07-17): still real, just less central
to the product than work/shelf/feed/profile.

**Rough shape:** one or two more Lexicons (`social.orbita.forum.post`,
maybe a separate `...comment`, or comments as a self-referencing record —
undecided), same write-handler-plus-webhook-case shape as notes. Like
notes vs. forum comments (see Beta 6): confirmed **not** feed material —
a forum conversation is confined to the work's own space, it doesn't
circulate to followers the way a note does.

## Beta 10 — events

**Problem:** an ephemeral, per-work group chat tied to a live/upcoming/ended
window — also real in earlier product work, not yet here. Demoted
alongside forum for the same reason (2026-07-17).

**Rough shape:** the tricky part isn't the Lexicon, it's the ephemerality.
Earlier product work computes live/upcoming/ended state on read, never
stores it — but AT Protocol repo records don't expire on their own, so a
"live chat" written as ordinary records stays sitting in the PDS as public
history after the event "ends." Worth an explicit decision here, not an
assumption: is that acceptable (state is computed, data just persists
quietly), or does this need real deletion/expiry logic added on top?

## Beta 11 — real fan-out (relay/firehose beyond your own account)

**Problem:** Tap today only ever tracks a repo after *you* log into it via
OAuth. Everything through Beta 10 still only aggregates across accounts that
happened to log into this appview by hand. The actual AT Protocol problem —
discovering and indexing records from accounts that never touched this
appview's OAuth flow — hasn't been faced yet.

**Rough shape:** consume `subscribeRepos` (or extend Tap's own config) for
the collections that matter across more than one manually-known DID first —
probably a short seeded list (you + a few volunteers) rather than attempting
open firehose discovery on day one, since the raw firehose is enormous and
almost none of it is these collections.

## Beta 12 — observability

**Problem:** a different shape of observability problem than earlier product
work's own version of this (which was about correlating a request across
many services you wrote yourself). Here there's still one binary — the real
unknowns are dependencies on infrastructure nobody on this project operates:
is Tap keeping up with the firehose or falling behind, are XRPC calls to
other people's PDSs failing, is OAuth/DPoP session refresh failing quietly,
and — the one that matters most once Beta 11 is real — is the relay
actually handing over every record it should, or are some silently missed.
Placed after fan-out on purpose: before that point, a handful of
manually-tested accounts are still small enough to check by hand with
`curl` and `sqlite3`, same as every beta so far.

**Rough shape:** at least one well-structured Grafana is a firm requirement,
not just raw metrics nobody looks at. Two categories of thing to show:
infra health (firehose lag, XRPC error rate against external PDSs, fan-out
completeness) and business metrics in the same spirit as this product's own
definition of what matters (works added to shelves, notes/posts created,
catalog diversity) — never session time, DAU/MAU, or anything that reads as
an engagement metric, consistent with the product's own non-negotiables.
Which exact stack (full OpenTelemetry+Jaeger+Prometheus+Alertmanager like
earlier product work, or something lighter given this is one binary, not
nine services) is explicitly left for when this beta actually starts.

## Beta 13 — affinity across shelves

**Problem:** "who has similar taste" only becomes a real question once
Beta 11 makes more than a couple of hand-tested accounts' data available.

**Rough shape:** the same Jaccard-similarity idea already proven in earlier
lab work, recomputed against federated data instead of rows in one database.

## Beta 14 — direct messages

**Problem:** the odd one out on this whole list, flagged rather than
scoped. Every other collection here is meant to be public repo data — that's
the entire point of AT Protocol repos. Private 1:1 messages are the opposite
of that by definition, so they likely *can't* just be another
`social.orbita.*` Lexicon record sitting in someone's public PDS the way
shelf items and notes do. Bluesky's own DMs work this way for exactly this
reason: they live in a separate, non-federated service behind the same
account, not as ordinary repo records. Whether Órbita needs (or wants) an
equivalent side-service, or whether DMs simply stay a feature that doesn't
cross over to the federated product, is a real open question — worth a full
planning conversation on its own before this beta gets scoped for real.

## Beta 15 — close known gaps

**Problem:** loose ends already named and deliberately deferred: update/delete
for notes (only create is wired, same gap already named for shelf items),
and `track`/`chapter` granularity for albums/books once Beta 16 (below) gives
them real resolution to build that granularity on top of.

**Rough shape:** mostly mechanical extensions of patterns already
established — lower risk and lower novelty than what's around it.

## Beta 16 — real catalog support for books and albums

**Problem:** flagged directly (2026-07-19), and more foundational than it
first looked. `musicbrainz` and `open-library` are declared as valid
`work.provider` values in the Lexicon since Beta 0, but `resolveWork`
(`tmdb.go`) has no case for either — only `tmdb-movie`/`tmdb-tv` actually
resolve to a title/poster/year. `GET /api/search` (`search.go`) only ever
queries TMDB too. In practice this means a book or an album can't be found
by search, and even shelved by a hand-typed provider/id it would only ever
show as a raw, unresolved string — this product's own four-medium promise
(série/filme/álbum/livro) only half holds today.

**Rough shape:** a resolver for each — MusicBrainz's own API for
`musicbrainz` (releases/recordings), Open Library's API for `open-library`
(editions/works) — mirroring `fetchFromTMDB`'s existing shape (title,
cover/poster URL, year, overview), plus extending search to query all three
sources rather than just TMDB. Genuinely two different APIs with two
different response shapes to learn, not a copy-paste of the TMDB resolver —
likely the reason this got declared in the Lexicon early but never actually
built.

## Not on this list yet

Anything for production concerns (hosting, uptime, backups) — this whole
roadmap is still "prove it works," not "run it for real users." That's a
different, later conversation.
