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

  // Beta 7: nooks are managed here (privately, on your own shelf), shown
  // to visitors on the profile — the same reasoning /shelf already
  // follows for the shelf itself (manage here, presented there).
  let nooks = [];
  try {
    const profile = await fetchJSON(`/api/profile/${viewer.handle}`);
    nooks = profile.nooks || [];
  } catch {
    // Non-fatal — the shelf list above already loaded fine; nook
    // management just won't have anything to show yet.
  }
  renderNooksSection(app, items, nooks);
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

const NOOK_THEMES = ["default", "warm", "cool", "midnight"];

// Beta 7: nooks are the primary way a shelf is organized and shown to
// visitors — managed here on your own shelf, presented on your profile.
// "Editing" a nook (add/remove a work) resends its whole works array via
// PUT (com.atproto.repo.putRecord replaces the record), there's no
// separate membership record to patch incrementally.
function renderNooksSection(app, shelfItems, nooks) {
  const section = el("section", {}, [el("h2", { text: "Nooks" })]);
  const nookList = el("div", {});
  section.appendChild(nookList);

  const shelfByKey = new Map(shelfItems.map((it) => [`${it.provider}/${it.id}`, it]));

  const renderExistingNook = (nook) => {
    const box = el("div", { class: `nook nook-${nook.theme}` }, [
      el("h2", { text: nook.name }),
    ]);
    if (nook.description) box.appendChild(el("p", { class: "mono nook-description", text: nook.description }));

    const worksList = el("ul", { class: "plain" });
    const currentWorks = () => nook.works.map((w) => ({ provider: w.provider, id: w.id }));

    const rerenderWorks = () => {
      worksList.innerHTML = "";
      for (const w of nook.works) {
        const removeBtn = el("button", { type: "button", text: "remove" });
        removeBtn.addEventListener("click", async () => {
          nook.works = nook.works.filter((x) => !(x.provider === w.provider && x.id === w.id));
          await saveNook(nook, currentWorks());
          rerenderWorks();
        });
        worksList.appendChild(
          el("li", { class: "account-row" }, [el("span", { text: `${w.title}` }), removeBtn])
        );
      }
      if (nook.works.length === 0) {
        worksList.appendChild(el("li", { class: "empty", text: "no works in this nook yet" }));
      }
    };
    rerenderWorks();
    box.appendChild(worksList);

    const notInNook = () => shelfItems.filter((it) => !nook.works.some((w) => w.provider === it.provider && w.id === it.id));
    const addRow = el("div", {});
    const rebuildAddRow = () => {
      addRow.innerHTML = "";
      const remaining = notInNook();
      if (remaining.length === 0) return;
      const select = el(
        "select",
        {},
        remaining.map((it) => el("option", { value: `${it.provider}/${it.id}`, text: it.title }))
      );
      const addBtn = el("button", { type: "button", text: "+ add work" });
      addBtn.addEventListener("click", async () => {
        const [provider, id] = select.value.split("/");
        const item = shelfByKey.get(`${provider}/${id}`);
        nook.works.push({ provider, id, title: item ? item.title : id });
        await saveNook(nook, currentWorks());
        rerenderWorks();
        rebuildAddRow();
      });
      addRow.appendChild(select);
      addRow.appendChild(addBtn);
    };
    rebuildAddRow();
    box.appendChild(addRow);

    const deleteBtn = el("button", { type: "button", text: "delete nook" });
    deleteBtn.addEventListener("click", async () => {
      if (!confirm(`Delete "${nook.name}"? The works stay on your shelf, unsorted.`)) return;
      try {
        await fetchJSON("/api/nooks/delete", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ uri: nook.uri }),
        });
        box.remove();
      } catch (err) {
        alert(`failed to delete nook: ${err}`);
      }
    });
    box.appendChild(el("p", {}, [deleteBtn]));

    return box;
  };

  async function saveNook(nook, works) {
    try {
      const updated = await fetchJSON(`/api/nooks/${rkeyOf(nook.uri)}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: nook.name, description: nook.description, theme: nook.theme, works }),
      });
      nook.works = updated.works;
    } catch (err) {
      alert(`failed to update nook: ${err}`);
    }
  }

  for (const nook of nooks) {
    nookList.appendChild(renderExistingNook(nook));
  }
  if (nooks.length === 0) {
    nookList.appendChild(el("p", { class: "empty", text: "no nooks yet" }));
  }

  // ---- new nook form ----
  const nameInput = el("input", { type: "text", placeholder: "name (e.g. cold days)" });
  const descInput = el("input", { type: "text", placeholder: "description (optional)" });
  const themeSelect = el(
    "select",
    {},
    NOOK_THEMES.map((t) => el("option", { value: t, text: t }))
  );
  const checkboxes = shelfItems.map((it) => {
    const cb = el("input", { type: "checkbox", value: `${it.provider}/${it.id}` });
    return { item: it, cb };
  });
  const pickList = el(
    "div",
    {},
    checkboxes.map(({ item, cb }) => el("label", { class: "nook-pick" }, [cb, el("span", { text: ` ${item.title}` })]))
  );
  const createBtn = el("button", { type: "button", text: "Create nook" });
  createBtn.addEventListener("click", async () => {
    const name = nameInput.value.trim();
    if (!name) return;
    const works = checkboxes
      .filter(({ cb }) => cb.checked)
      .map(({ item }) => ({ provider: item.provider, id: item.id }));
    createBtn.disabled = true;
    try {
      const created = await fetchJSON("/api/nooks", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name, description: descInput.value.trim(), theme: themeSelect.value, works }),
      });
      nameInput.value = "";
      descInput.value = "";
      for (const { cb } of checkboxes) cb.checked = false;
      const emptyMsg = nookList.querySelector(".empty");
      if (emptyMsg) emptyMsg.remove();
      nookList.appendChild(renderExistingNook(created));
    } catch (err) {
      alert(`failed to create nook: ${err}`);
    } finally {
      createBtn.disabled = false;
    }
  });

  section.appendChild(
    el("div", { class: "new-nook" }, [
      el("h2", { text: "New nook" }),
      nameInput,
      descInput,
      themeSelect,
      pickList,
      createBtn,
    ])
  );

  app.appendChild(section);
}

init();
