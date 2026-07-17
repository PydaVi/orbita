# Roadmap — macro view

This is a sketch, not a plan. Each beta below still gets its own `BETA{N}-PLAN.md`
with real scope, open questions, and a status once we actually start it — same
process as Beta 0/1/2. This file exists so the order and the "why this, why
now" of the whole arc is visible in one place, before any of it is detailed.

Beta 0, 1, and 2 are done — see [`BETA0-PLAN.md`](BETA0-PLAN.md),
[`BETA1-PLAN.md`](BETA1-PLAN.md), [`BETA2-PLAN.md`](BETA2-PLAN.md). What
follows is what's left before there's a beta worth actually presenting to
someone outside this project.

Real UI comes next, ahead of everything else on this list — not last, as
originally sketched. Every beta after it is built with a real interface from
the start instead of plain HTML, which is worth the reorder on its own: it's
a lot easier to reason about a feature — profile especially — once you can
actually look at it, not just curl it.

## Beta 3 — real UI

**Problem:** everything so far is plain-HTML "prove the mechanism" pages —
fine for proving OAuth writes and Tap indexing work, not fine for reasoning
about anything presentation-shaped, like a profile.

**Rough shape:** to be planned in detail next — see the conversation this
roadmap came out of. Likely direction: the same vanilla stack already
validated in earlier product work (self-hosted fonts, hand-written CSS, no
JS framework, no build step) applied to what already exists here (shelf,
work pages, notes) as the first pass, so every beta after this one inherits
it instead of starting from scratch.

## Beta 4 — profile pages

**Problem:** there's no page yet that's about a *person* rather than a
*work* — even though the shelf/notes data to build one already exists by
this point, and now so does a real interface to build it in. Moved ahead of
forum/events on purpose: a profile is exactly the kind of feature that's
hard to reason about without a screen to look at, so it belongs right after
the UI beta, not several betas later.

**Rough shape:** a profile surfacing one account's shelf and notes to start.
Forum and event activity get added to it once those betas land (5 and 6,
below). Constellation-style visualization and the geometric archetype from
earlier product work are explicitly a stretch goal here, not a requirement
for the first pass.

## Beta 5 — forum

**Problem:** a work's page today only ever shows shelf items and notes.
Longer-form discussion per work — posts and comments, not single notes —
exists in earlier product work and hasn't been brought over yet.

**Rough shape:** one or two more Lexicons (`social.orbita.forum.post`,
maybe a separate `...comment`, or comments as a self-referencing record —
undecided), same write-handler-plus-webhook-case shape as notes.

## Beta 6 — events

**Problem:** an ephemeral, per-work group chat tied to a live/upcoming/ended
window — also real in earlier product work, not yet here.

**Rough shape:** the tricky part isn't the Lexicon, it's the ephemerality.
Earlier product work computes live/upcoming/ended state on read, never
stores it — but AT Protocol repo records don't expire on their own, so a
"live chat" written as ordinary records stays sitting in the PDS as public
history after the event "ends." Worth an explicit decision here, not an
assumption: is that acceptable (state is computed, data just persists
quietly), or does this need real deletion/expiry logic added on top?

## Beta 7 — real fan-out (relay/firehose beyond your own account)

**Problem:** Tap today only ever tracks a repo after *you* log into it via
OAuth. Everything through Beta 6 still only aggregates across accounts that
happened to log into this appview by hand. The actual AT Protocol problem —
discovering and indexing records from accounts that never touched this
appview's OAuth flow — hasn't been faced yet.

**Rough shape:** consume `subscribeRepos` (or extend Tap's own config) for
the collections that matter across more than one manually-known DID first —
probably a short seeded list (you + a few volunteers) rather than attempting
open firehose discovery on day one, since the raw firehose is enormous and
almost none of it is these collections.

## Beta 8 — observability

**Problem:** a different shape of observability problem than earlier product
work's own version of this (which was about correlating a request across
many services you wrote yourself). Here there's still one binary — the real
unknowns are dependencies on infrastructure nobody on this project operates:
is Tap keeping up with the firehose or falling behind, are XRPC calls to
other people's PDSs failing, is OAuth/DPoP session refresh failing quietly,
and — the one that matters most once Beta 7 is real — is the relay actually
handing over every record it should, or are some silently missed. Placed
after fan-out on purpose: before that point, a handful of manually-tested
accounts are still small enough to check by hand with `curl` and `sqlite3`,
same as every beta so far.

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

## Beta 9 — affinity across shelves

**Problem:** "who has similar taste" only becomes a real question once
Beta 7 makes more than a couple of hand-tested accounts' data available.

**Rough shape:** the same Jaccard-similarity idea already proven in earlier
lab work, recomputed against federated data instead of rows in one database.

## Beta 10 — feed

**Problem:** right now there's no page that's actually useful to check
day-to-day — everything lives on a single work's or person's page. A feed is
what turns this from "a place to look something up" into "a place to come
back to."

**Rough shape:** reuse the existing Bluesky follow graph
(`app.bsky.graph.follow`) instead of inventing a parallel "follow" concept —
chronological, deterministic, no ranking, same non-negotiable shape the
product has always used.

## Beta 11 — direct messages

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

## Beta 12 — close known gaps

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
