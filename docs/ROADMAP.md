# Roadmap — macro view

This is a sketch, not a plan. Each beta below still gets its own `BETA{N}-PLAN.md`
with real scope, open questions, and a status once we actually start it — same
process as Beta 0/1/2. This file exists so the order and the "why this, why
now" of the whole arc is visible in one place, before any of it is detailed.

Beta 0 through 5 are done — see [`BETA0-PLAN.md`](BETA0-PLAN.md),
[`BETA1-PLAN.md`](BETA1-PLAN.md), [`BETA2-PLAN.md`](BETA2-PLAN.md),
[`BETA3-PLAN.md`](BETA3-PLAN.md), [`BETA4-PLAN.md`](BETA4-PLAN.md),
[`BETA5-PLAN.md`](BETA5-PLAN.md). Real UI (Beta 3) landed ahead of
everything else on this list — not last, as originally sketched —
precisely because a profile is hard to reason about without a screen to
look at, and Beta 4 gave every page from here on a persistent shell
(topbar, nav) to live inside instead of being its own island. What follows
is what's left before there's a beta worth actually presenting to someone
outside this project.

## Beta 6 — feed

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

## Beta 7 — the shelf as a creative space

**Problem:** the shelf today is just a flat, chronological list of items —
no room for a person's own curatorial voice in how it's organized or
presented. Flagged by the author as potentially **the apex of the whole
product** (2026-07-17): the shelf is the one surface that's entirely about
the person's own taste and judgment, and right now it doesn't reflect that
at all. This is explicitly under-scoped on purpose. It needs its own
dedicated planning conversation before it becomes a real `BETA{N}-PLAN.md`,
not just a scope item squeezed in here.

**Ideas floated, refined over two conversations (2026-07-17):**
- Custom labels the person defines themselves — not fixed system
  categories — that group works into something closer to a playlist than a
  folder: "cold days" could hold a blues album, a drama series, a horror
  movie, and a novel all together. The label's name is itself the
  explanation for why these things sit together, in the same spirit as
  principle 2 ("the user always knows why they're seeing what they're
  seeing") — except here the person writes that explanation themselves,
  instead of the system inferring or ranking anything. Ordering within a
  label is entirely free-form, chosen by the person, not derived from any
  criteria the system understands.
- These label-playlists are explicitly **a layer over the shelf, not a
  parallel structure** (confirmed 2026-07-17): only works already on the
  person's own shelf can go into one. This keeps "adding to your shelf" as
  the one real commitment/gesture (see the product's own success metrics —
  shelf additions are a real cultural gesture, playlists are recombination
  of that, not a way around it).
- Shareable: a label-playlist is a much richer thing to share than a
  single note — closer to a small manifesto of taste than a list. Crossing
  media types in one playlist (album + series + film + book together) may
  be the one thing here no single-medium competitor (Letterboxd, Goodreads,
  Discogs) can do, since it only works because this catalog already treats
  every type as one thing.
- A size limit, if it exists at all, applies only to the base shelf — the
  scarce, curated set of what's really "yours" — never to playlists built
  from it, which stay free to recombine without limit. Resolves the earlier
  worry that a limit might fight against the creative, expressive side of
  this idea: scarcity where it should force real curatorial choices,
  freedom where the point is combination and re-combination.

**Explicitly not decided:** whether a label-playlist becomes its own
Lexicon record (an ordered list of work references, its own name/title, its
own AT-URI to share) or something simpler layered onto `shelf.item` itself;
whether a shelf size limit survives contact with real use; how (or whether)
any of this interacts with the constellation visualization from earlier
product work. This entry exists so the ambition and the shape it's taking
don't get lost before the rest of the roadmap moves on, not to pre-commit
to an implementation.

## Beta 8 — forum

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

## Beta 9 — events

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

## Beta 10 — real fan-out (relay/firehose beyond your own account)

**Problem:** Tap today only ever tracks a repo after *you* log into it via
OAuth. Everything through Beta 9 still only aggregates across accounts that
happened to log into this appview by hand. The actual AT Protocol problem —
discovering and indexing records from accounts that never touched this
appview's OAuth flow — hasn't been faced yet.

**Rough shape:** consume `subscribeRepos` (or extend Tap's own config) for
the collections that matter across more than one manually-known DID first —
probably a short seeded list (you + a few volunteers) rather than attempting
open firehose discovery on day one, since the raw firehose is enormous and
almost none of it is these collections.

## Beta 11 — observability

**Problem:** a different shape of observability problem than earlier product
work's own version of this (which was about correlating a request across
many services you wrote yourself). Here there's still one binary — the real
unknowns are dependencies on infrastructure nobody on this project operates:
is Tap keeping up with the firehose or falling behind, are XRPC calls to
other people's PDSs failing, is OAuth/DPoP session refresh failing quietly,
and — the one that matters most once Beta 10 is real — is the relay
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

## Beta 12 — affinity across shelves

**Problem:** "who has similar taste" only becomes a real question once
Beta 10 makes more than a couple of hand-tested accounts' data available.

**Rough shape:** the same Jaccard-similarity idea already proven in earlier
lab work, recomputed against federated data instead of rows in one database.

## Beta 13 — direct messages

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

## Beta 14 — close known gaps

**Problem:** loose ends already named and deliberately deferred: update/delete
for notes (only create is wired, same gap already named for shelf items),
and `track`/`chapter` granularity for albums/books (the Lexicon allows it,
nothing reads or writes it yet).

**Rough shape:** mostly mechanical extensions of patterns already
established — lower risk and lower novelty than what's around it.

## Not on this list yet

Anything for production concerns (hosting, uptime, backups) — this whole
roadmap is still "prove it works," not "run it for real users." That's a
different, later conversation.
