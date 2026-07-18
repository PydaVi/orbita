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

// A nook's theme is one of a small curated set (see the Lexicon's own
// style def) — never a free color, so every profile stays visually
// coherent with the rest of the product while still feeling distinct.
// handle is the owner's — needed to build the standalone share URL
// (/profile/{handle}/nook/{rkey}), which is what gets a real Open Graph
// preview when pasted elsewhere (see nookpage.go), unlike the profile page
// itself.
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

  // Beta 7: the shelf is organized by nook, not one flat grid — a nook is
  // the primary way this person chose to present their own shelf, not a
  // side list. Unsorted works (not yet placed in any nook) get their own
  // honest section at the end, never folded silently into a default.
  for (const nook of profile.nooks || []) {
    app.appendChild(renderNookSection(nook, profile.handle));
  }

  const unsortedSection = el("section", {}, [el("h2", { text: `Unsorted (${(profile.unsorted || []).length})` })]);
  unsortedSection.appendChild(renderWorkGrid(profile.unsorted));
  app.appendChild(unsortedSection);

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
