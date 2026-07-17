// Beta 4 (reconsidered mid-build, 2026-07-17): not "everyone's shelf" — a
// global, unscoped list served no real purpose. This is your own shelf,
// gated by sign-in, same as everywhere else that writes to a PDS.

async function init() {
  const app = renderShell("shelf");
  app.innerHTML = "";
  app.appendChild(el("h1", { class: "work-title", text: "Your shelf" }));

  const viewer = await currentViewer();
  if (!viewer) {
    app.appendChild(el("p", {}, [el("a", { href: "/oauth/login", text: "Sign in to see your shelf" })]));
    return;
  }

  app.appendChild(el("p", {}, [el("a", { href: "/search", text: "+ Add to shelf" })]));

  let items;
  try {
    items = await fetchJSON("/api/shelf");
  } catch (err) {
    app.appendChild(el("p", { text: `could not load your shelf: ${err}` }));
    return;
  }

  const list = el("ul", { class: "plain" });
  for (const item of items) {
    list.appendChild(renderItem(item, list));
  }
  if (items.length === 0) {
    list.appendChild(el("li", { class: "empty", text: "nothing on your shelf yet" }));
  }
  app.appendChild(list);
}

function renderItem(item, list) {
  const children = [];
  if (item.poster) {
    children.push(el("img", { src: item.poster, class: "episode-still", alt: "" }));
  }
  children.push(
    el("a", {
      href: `/works/${item.provider}/${item.id}`,
      class: "episode-summary-text",
      text: item.title,
    })
  );
  children.push(el("span", { class: "mono", text: item.addedAt }));

  const removeButton = el("button", { type: "button", text: "remove" });
  removeButton.addEventListener("click", async () => {
    removeButton.disabled = true;
    try {
      await fetchJSON("/api/shelf/delete", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ uri: item.uri }),
      });
      row.remove();
      if (!list.querySelector("li")) {
        list.appendChild(el("li", { class: "empty", text: "nothing on your shelf yet" }));
      }
    } catch (err) {
      alert(`failed to remove: ${err}`);
      removeButton.disabled = false;
    }
  });
  children.push(removeButton);

  const row = el("li", { class: "account-row" }, children);
  return row;
}

init();
