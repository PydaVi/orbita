// Beta 6: three tabs, matching this product's own established shape —
// Shelf (the main one: notes from anyone about works on your own shelf,
// obra-first), Following (notes from people you follow, reusing the
// existing Bluesky follow graph), and Affinity (needs the Jaccard
// computation from Beta 13 — an honest placeholder, not a fake response).
// Chronological, no ranking, in every real tab.

const FEED_TABS = [
  { key: "shelf", label: "Shelf" },
  { key: "following", label: "Following" },
  { key: "affinity", label: "Affinity" },
];

async function init() {
  const app = renderShell("feed");
  app.innerHTML = "";
  app.appendChild(el("h1", { class: "work-title", text: "Feed" }));

  const viewer = await currentViewer();
  if (!viewer) {
    app.appendChild(el("p", {}, [el("a", { href: "/oauth/login", text: "Sign in to see your feed" })]));
    return;
  }

  const tabBar = el("div", { class: "tab-bar" });
  const content = el("div", {});
  app.appendChild(tabBar);
  app.appendChild(content);

  let active = "shelf";
  const renderTabBar = () => {
    tabBar.innerHTML = "";
    for (const t of FEED_TABS) {
      const btn = el("button", {
        type: "button",
        class: t.key === active ? "tab active" : "tab",
        text: t.label,
      });
      btn.addEventListener("click", () => {
        if (active === t.key) return;
        active = t.key;
        renderTabBar();
        loadTab(content, active);
      });
      tabBar.appendChild(btn);
    }
  };

  renderTabBar();
  loadTab(content, active);
}

async function loadTab(content, tab) {
  content.innerHTML = "";

  if (tab === "affinity") {
    content.appendChild(
      el("p", { class: "mono", text: "Not built yet — needs the Jaccard affinity computation (Beta 13)." })
    );
    return;
  }

  content.appendChild(el("p", { class: "mono", text: "loading…" }));
  try {
    const data = await fetchJSON(`/api/feed?tab=${tab}`);
    content.innerHTML = "";
    renderFeedList(content, data.notes);
  } catch (err) {
    content.innerHTML = "";
    content.appendChild(el("p", { text: `could not load this feed: ${err}` }));
  }
}

function renderFeedList(content, notes) {
  const list = el("ul", { class: "plain" });
  for (const n of notes || []) {
    const workLabel = n.season != null ? `${n.title} — S${n.season}E${n.episode}` : n.title;
    const workHref =
      n.season != null
        ? `/works/${n.provider}/${n.id}/season/${n.season}/episode/${n.episode}`
        : `/works/${n.provider}/${n.id}`;

    const children = [];
    if (n.poster) {
      children.push(el("img", { src: n.poster, class: "episode-still", alt: "" }));
    }
    const body = el("div", { class: "note-main" }, [
      el("div", { class: "note-byline" }, [
        el("a", { href: `/profile/${n.handle}`, class: "note-byline" }, [
          avatarEl(n.handle, n.avatarUrl),
          el("span", { class: "mono", text: `@${displayHandle(n.handle)}` }),
        ]),
        el("span", { class: "mono", text: n.createdAt }),
      ]),
      el("a", { href: workHref, text: workLabel }),
      el("p", { class: "note-text", text: n.text }),
    ]);
    children.push(body);

    list.appendChild(el("li", { class: "note" }, children));
  }
  if (!notes || notes.length === 0) {
    list.appendChild(el("li", { class: "empty", text: "nothing here yet" }));
  }
  content.appendChild(list);
}

init();
