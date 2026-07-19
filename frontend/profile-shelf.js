// The complete counterpart to /profile/{handle}'s compact nook-card
// summary: every nook rendered in full (its whole works grid, not a
// six-poster preview), plus the complete Unsorted grid. Same
// GET /api/profile/{handle} data as the summary page — this just renders
// all of it instead of a glance, reached via "see full shelf" there.

async function init() {
  const app = renderShell(null);
  const match = window.location.pathname.match(/^\/profile\/([^/]+)\/shelf$/);
  if (!match) return;
  const handle = match[1];

  let profile;
  try {
    profile = await fetchJSON(`/api/profile/${handle}`);
  } catch (err) {
    app.appendChild(el("h1", { class: "work-title", text: handle }));
    app.appendChild(el("p", { text: `could not find this account: ${err}` }));
    return;
  }

  renderFullShelf(app, profile);
}

// A nook's theme is one of a small curated set (see the Lexicon's own
// style def) — never a free color, so every profile stays visually
// coherent with the rest of the product while still feeling distinct.
function renderNookSection(nook, handle) {
  const header = [el("h2", { text: nook.name })];
  if (nook.description) {
    header.push(el("p", { class: "mono nook-description", text: nook.description }));
  }
  const section = el("section", { class: `nook nook-${nook.theme || "default"}` }, header);
  section.appendChild(renderWorkGrid(nook.works));
  const shareURL = `${window.location.origin}/profile/${handle}/nook/${rkeyOf(nook.uri)}`;
  section.appendChild(shareButton(nook.name, shareURL));
  return section;
}

function renderFullShelf(app, profile) {
  const hero = el("div", { class: "hero" });
  const avatar = avatarEl(profile.handle, profile.avatarUrl);
  avatar.classList.add("avatar-lg");
  hero.appendChild(el("div", {}, [avatar]));
  const heroBody = el("div", {}, [
    el("h1", { class: "work-title", text: `@${displayHandle(profile.handle)}` }),
    el("hr", { class: "hero-rule" }),
    el("p", {}, [el("a", { href: `/profile/${profile.handle}`, text: "← back to profile" })]),
  ]);
  hero.appendChild(heroBody);
  app.appendChild(hero);

  for (const nook of profile.nooks || []) {
    app.appendChild(renderNookSection(nook, profile.handle));
  }

  const unsorted = profile.unsorted || [];
  const unsortedSection = el("section", {}, [el("h2", { text: `Unsorted (${unsorted.length})` })]);
  unsortedSection.appendChild(renderWorkGrid(unsorted));
  app.appendChild(unsortedSection);
}

init();
