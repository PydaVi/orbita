// Beta 7: a single nook, addressable and shareable on its own — the URL a
// person hands to someone else when they want to show off one shelf of
// taste rather than their whole profile. nookpage.go server-renders real
// Open Graph tags for this same URL (crawlers never run this script); this
// file is the same data rendered interactively for a person who actually
// opens the link.

async function init() {
  const app = renderShell(null);
  const match = window.location.pathname.match(/^\/profile\/([^/]+)\/nook\/([^/]+)$/);
  if (!match) return;
  const [, handle, rkey] = match;

  let nook;
  try {
    nook = await fetchJSON(`/api/profile/${handle}/nook/${rkey}`);
  } catch (err) {
    app.innerHTML = "";
    app.appendChild(el("h1", { class: "work-title", text: "Nook not found" }));
    app.appendChild(el("p", { text: `${err}` }));
    return;
  }

  renderNookPage(app, nook, handle);
}

function renderNookPage(app, nook, handle) {
  app.innerHTML = "";

  const owner = el("a", { href: `/profile/${handle}`, class: "note-byline" }, [
    avatarEl(handle, nook.ownerAvatar),
    el("span", { class: "mono", text: `@${displayHandle(nook.ownerHandle || handle)}` }),
  ]);
  app.appendChild(owner);

  const header = [el("h1", { class: "work-title", text: nook.name })];
  if (nook.description) {
    header.push(el("p", { class: "overview", text: nook.description }));
  }
  const section = el("section", { class: `nook nook-${nook.theme || "default"}` }, header);
  section.appendChild(renderWorkGrid(nook.works));
  app.appendChild(section);

  app.appendChild(shareButton(nook.name, window.location.href));
}

init();
