// Beta 4: the site now has a real slot for the feed, even though the feed
// itself is Beta 6 — this is deliberately just a placeholder, not a fake
// preview of content that doesn't exist yet.

function init() {
  const app = renderShell("feed");
  app.innerHTML = "";
  app.appendChild(el("h1", { class: "work-title", text: "Feed" }));
  app.appendChild(
    el("p", { class: "mono", text: "Not built yet — chronological notes from people you follow will live here." })
  );
}

init();
