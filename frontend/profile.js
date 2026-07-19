// Beta 5: a page about a *person* — any account, not just the viewer's
// own, reachable by handle (/profile/{handle}). /profile with no handle
// redirects to the signed-in viewer's own profile, or prompts sign-in.
//
// Only ever shows what this appview already has locally — an account
// that's never logged in here (real fan-out is Beta 8) comes back empty,
// not broken.

async function init() {
  const app = renderShell("profile");
  const match = window.location.pathname.match(/^\/profile\/([^/]+)$/);

  if (!match) {
    const viewer = await currentViewer();
    if (!viewer) {
      app.innerHTML = "";
      app.appendChild(el("h1", { class: "work-title", text: "Profile" }));
      app.appendChild(el("p", {}, [el("a", { href: "/oauth/login", text: "Sign in to see your profile" })]));
      return;
    }
    window.location.replace(`/profile/${viewer.handle}`);
    return;
  }

  const handle = match[1];
  app.innerHTML = "";
  let profile;
  try {
    profile = await fetchJSON(`/api/profile/${handle}`);
  } catch (err) {
    app.appendChild(el("h1", { class: "work-title", text: handle }));
    app.appendChild(el("p", { text: `could not find this account: ${err}` }));
    return;
  }

  renderProfilePage(app, profile);
}

// Beta 7 (reconsidered again, 2026-07-19): the profile used to render every
// nook's complete works grid inline — with a nook's own cap now 50 works
// (see BETA7-PLAN.md item 13), that's up to 50 posters per nook, stacked
// for every nook on the page. Nothing distinguished "glancing at a
// profile" from "browsing a whole shelf," which is exactly the redundancy
// the author flagged: if the profile always shows everything, there's no
// real reason for the standalone nook page (nookpage.go) to exist. Each
// nook is now a compact card — a handful of its posters, the name, a
// count — and the *entire card* is the link to that nook's own complete
// page. "See it all" is one click away, on the nook that earned it, not a
// default the whole page pays for.
const NOOK_CARD_PREVIEW_COUNT = 6;

// How many nooks the summary shows before pointing at the complete page
// (profile-shelf.js) instead — "a couple," per the author's own words,
// not every nook the person has.
const PROFILE_SUMMARY_NOOK_COUNT = 2;

function renderNookCard(nook, handle) {
  const href = `/profile/${handle}/nook/${rkeyOf(nook.uri)}`;
  const preview = el("div", { class: "nook-card-preview" });
  const shown = (nook.works || []).slice(0, NOOK_CARD_PREVIEW_COUNT);
  for (const w of shown) {
    const item = el("span", { class: "shelf-grid-item nook-mini-item" });
    if (w.poster) {
      item.appendChild(el("img", { src: w.poster, alt: w.title }));
    } else {
      item.appendChild(el("span", { class: "mono", text: w.title }));
    }
    preview.appendChild(item);
  }
  if (shown.length === 0) {
    preview.appendChild(el("p", { class: "empty", text: "empty" }));
  }

  const card = el("a", { href, class: `nook nook-card nook-${nook.theme || "default"}` }, [preview]);
  card.appendChild(el("h3", { class: "nook-card-title", text: nook.name }));
  const count = (nook.works || []).length;
  card.appendChild(el("span", { class: "mono nook-card-count", text: `${count} work${count === 1 ? "" : "s"}` }));
  return card;
}

function renderProfilePage(app, profile) {
  const hero = el("div", { class: "hero" });
  const avatar = avatarEl(profile.handle, profile.avatarUrl);
  avatar.classList.add("avatar-lg");
  hero.appendChild(el("div", {}, [avatar]));
  const heroBody = el("div", {}, [
    el("h1", { class: "work-title", text: `@${displayHandle(profile.handle)}` }),
    el("hr", { class: "hero-rule" }),
  ]);
  if (profile.bio) {
    heroBody.appendChild(el("p", { class: "overview", text: profile.bio }));
  }
  heroBody.appendChild(el("p", { class: "mono", text: profile.did }));
  hero.appendChild(heroBody);
  app.appendChild(hero);

  // Beta 8: a visual signature of this person's own taste, computed from
  // their real nooks/themes — see constellation.js. Fetches its own data
  // and appends itself once ready, same as everything else on this page;
  // not awaited here since there's nothing else worth blocking on it.
  renderConstellationSection(app, profile.handle);

  // Beta 7 (reconsidered once more, 2026-07-19): even a card-per-nook grid
  // is still "the whole shelf" once every nook is in it — a summary means
  // showing *some*, not all, of them. This is a glance at someone's shelf,
  // not the shelf itself: a couple of nooks, then one button to the
  // complete counterpart page (profile-shelf.js) with every nook in full
  // and the whole Unsorted grid, for whoever actually wants that.
  const allNooks = profile.nooks || [];
  const shownNooks = allNooks.slice(0, PROFILE_SUMMARY_NOOK_COUNT);
  if (shownNooks.length > 0) {
    const nooksSection = el("section", {}, [el("h2", { text: "Nooks" })]);
    const grid = el("div", { class: "nook-card-grid" });
    for (const nook of shownNooks) {
      grid.appendChild(renderNookCard(nook, profile.handle));
    }
    nooksSection.appendChild(grid);
    app.appendChild(nooksSection);
  }

  const unsortedCount = (profile.unsorted || []).length;
  if (unsortedCount > 0) {
    app.appendChild(
      el("p", { class: "mono unsorted-note", text: `+${unsortedCount} more, not yet organized into a nook` })
    );
  }

  if (allNooks.length > 0 || unsortedCount > 0) {
    const fullLink = el("a", { href: `/profile/${profile.handle}/shelf`, class: "full-shelf-btn", text: "See full shelf →" });
    app.appendChild(fullLink);
  }

  const notesSection = el("section", {}, [el("h2", { text: `Notes (${profile.notes.length})` })]);
  const list = el("ul", { class: "plain" });
  for (const n of profile.notes) {
    const label = n.season != null ? `${n.title} — S${n.season}E${n.episode}` : n.title;
    const workHref =
      n.season != null
        ? `/works/${n.provider}/${n.id}/season/${n.season}/episode/${n.episode}`
        : `/works/${n.provider}/${n.id}`;
    list.appendChild(
      el("li", {}, [
        el("a", { href: workHref, text: label }),
        el("p", { class: "note-text", text: n.text }),
        el("span", { class: "mono", text: n.createdAt }),
      ])
    );
  }
  if (profile.notes.length === 0) {
    list.appendChild(el("li", { class: "empty", text: "nothing here yet" }));
  }
  notesSection.appendChild(list);
  app.appendChild(notesSection);
}

init();
