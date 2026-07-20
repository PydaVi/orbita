// Notes saved privately — never a public collection, never visible to
// anyone else (see saved.go for why: this is the one kind of data here
// that's the opposite of what every other collection on this site is for).
// Renders the exact same feedNoteEntry shape and renderFeedList (common.js)
// the feed itself uses — a saved note is a real note, shown the same way.

async function init() {
  const app = renderShell("saved");
  app.innerHTML = "";
  app.appendChild(el("h1", { class: "work-title", text: "Saved" }));

  const viewer = await currentViewer();
  if (!viewer) {
    app.appendChild(el("p", {}, [el("a", { href: "/oauth/login", text: "Sign in to see what you've saved" })]));
    return;
  }

  const content = el("div", {});
  app.appendChild(content);
  content.appendChild(el("p", { class: "mono", text: "loading…" }));

  try {
    const notes = await fetchJSON("/api/saved");
    const savedURIs = new Set(notes.map((n) => n.uri));
    content.innerHTML = "";
    renderFeedList(content, notes, savedURIs);
  } catch (err) {
    content.innerHTML = "";
    content.appendChild(el("p", { text: `could not load your saved notes: ${err}` }));
  }
}

init();
