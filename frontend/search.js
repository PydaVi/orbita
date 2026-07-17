// Beta 1's search-before-write, Beta 3's shape: a static shell + JSON API
// (GET /api/search) instead of a server-rendered results page, adding via
// the same POST /api/shelf/add the work page already uses.
//
// el(), fetchJSON(), and currentViewer() come from common.js.

async function init() {
  const app = renderShell(null);
  app.innerHTML = "";

  const viewer = await currentViewer();
  if (!viewer) {
    app.appendChild(el("h1", { class: "work-title", text: "Search" }));
    app.appendChild(el("p", {}, [el("a", { href: "/oauth/login", text: "Sign in to search and add to your shelf" })]));
    return;
  }

  app.appendChild(el("h1", { class: "work-title", text: "Search" }));

  const params = new URLSearchParams(window.location.search);
  const initialQuery = params.get("q") || "";

  const input = el("input", { type: "text", value: initialQuery, placeholder: "search a title..." });
  const button = el("button", { type: "button", text: "Search" });
  const form = el("div", {}, [input, button]);
  app.appendChild(form);

  const results = el("ul", { class: "plain" });
  app.appendChild(results);

  const runSearch = async () => {
    const q = input.value.trim();
    history.replaceState(null, "", q ? `/search?q=${encodeURIComponent(q)}` : "/search");
    if (!q) {
      results.innerHTML = "";
      return;
    }
    results.innerHTML = "";
    results.appendChild(el("li", { class: "empty", text: "searching…" }));
    try {
      const items = await fetchJSON(`/api/search?q=${encodeURIComponent(q)}`);
      results.innerHTML = "";
      for (const item of items) {
        results.appendChild(renderResult(item));
      }
      if (items.length === 0) {
        results.appendChild(el("li", { class: "empty", text: "no results" }));
      }
    } catch (err) {
      results.innerHTML = "";
      results.appendChild(el("li", { class: "empty", text: `search failed: ${err}` }));
    }
  };

  button.addEventListener("click", runSearch);
  input.addEventListener("keydown", (e) => {
    if (e.key === "Enter") runSearch();
  });

  if (initialQuery) runSearch();
}

function renderResult(item) {
  const children = [];
  if (item.posterUrl) {
    children.push(el("img", { src: item.posterUrl, class: "episode-still", alt: "" }));
  }
  children.push(
    el("a", {
      href: `/works/${item.provider}/${item.id}`,
      class: "episode-summary-text",
      text: item.year ? `${item.title} (${item.year})` : item.title,
    })
  );

  const addButton = el("button", { type: "button", text: "+ Add to shelf" });
  addButton.addEventListener("click", async () => {
    addButton.disabled = true;
    try {
      await fetchJSON("/api/shelf/add", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ provider: item.provider, id: item.id }),
      });
      addButton.textContent = "✓ Added";
    } catch (err) {
      alert(`failed to add to shelf: ${err}`);
      addButton.disabled = false;
    }
  });
  children.push(addButton);

  return el("li", { class: "account-row" }, children);
}

init();
