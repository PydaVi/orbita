// Beta 4 (reconsidered mid-build, 2026-07-17): not "everyone's shelf" — a
// global, unscoped list served no real purpose. This is your own shelf,
// gated by sign-in, same as everywhere else that writes to a PDS.
//
// Beta 7 (reworked, 2026-07-18): organizing into nooks is direct
// manipulation, not a form — drag a poster into a nook, drag within a nook
// to reorder, nooks on top, whatever's still unsorted underneath. Native
// HTML5 drag-and-drop, no library.

const NOOK_THEMES = ["default", "warm", "cool", "midnight", "riso", "indigo", "manifesto"];

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

  let nooks = [];
  try {
    const profile = await fetchJSON(`/api/profile/${viewer.handle}`);
    nooks = profile.nooks || [];
  } catch {
    // Non-fatal — the shelf still loaded; nooks just won't have anything
    // to show yet.
  }

  const state = { items, nooks };
  const root = el("div", {});
  app.appendChild(root);
  renderOrganizer(root, state);
}

function workKey(w) {
  return `${w.provider}/${w.id}`;
}

async function saveNook(nook) {
  try {
    const body = {
      name: nook.name,
      description: nook.description || "",
      theme: nook.theme,
      order: nook.order,
      works: nook.works.map((w) => ({ provider: w.provider, id: w.id })),
    };
    const updated = await fetchJSON(`/api/nooks/${rkeyOf(nook.uri)}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    nook.works = updated.works;
  } catch (err) {
    alert(`failed to save nook: ${err}`);
  }
}

// A poster is the one draggable unit everywhere it appears — in a nook or
// in the unsorted grid. Dropping ON one inserts before/after it (whichever
// half of it you dropped on); dropping on empty space in a container
// appends at the end (handled by the container's own drop listener, which
// only fires when a child's listener didn't already stop it).
function renderPoster(work, { fromNookUri, onMove, onRemove }) {
  const cell = el("div", { class: "shelf-grid-item draggable-work" });
  cell.setAttribute("draggable", "true");
  cell.title = work.title;
  if (work.poster) {
    cell.appendChild(el("img", { src: work.poster, alt: work.title }));
  } else {
    cell.appendChild(el("span", { class: "mono", text: work.title }));
  }
  const removeBtn = el("button", { type: "button", class: "poster-remove", title: "Remove from shelf", text: "×" });
  removeBtn.addEventListener("click", (e) => {
    e.stopPropagation();
    onRemove(work);
  });
  cell.appendChild(removeBtn);

  cell.addEventListener("dragstart", (e) => {
    e.dataTransfer.effectAllowed = "move";
    e.dataTransfer.setData("application/json", JSON.stringify({ provider: work.provider, id: work.id, fromNook: fromNookUri || null }));
    cell.classList.add("dragging");
  });
  cell.addEventListener("dragend", () => cell.classList.remove("dragging"));
  cell.addEventListener("dragover", (e) => {
    e.preventDefault();
    e.stopPropagation();
  });
  cell.addEventListener("drop", (e) => {
    e.preventDefault();
    e.stopPropagation();
    const data = JSON.parse(e.dataTransfer.getData("application/json"));
    const rect = cell.getBoundingClientRect();
    const before = e.clientX - rect.left < rect.width / 2;
    onMove(data, work, before);
  });
  return cell;
}

function makeDropContainer(grid, onDropEmpty) {
  grid.addEventListener("dragover", (e) => e.preventDefault());
  grid.addEventListener("drop", (e) => {
    e.preventDefault();
    e.stopPropagation();
    const data = JSON.parse(e.dataTransfer.getData("application/json"));
    if (data.nookReorder) return; // not for this drop zone — see reorderNooksArea
    onDropEmpty(data);
  });
}

// Repositioning a nook among its siblings: gapped integer positions
// (1000, 2000, 3000…) so moving one nook only ever rewrites that one
// record — recomputed as a full sequence only the first time (some nooks
// predate this field and have no order yet), or if two neighbors are too
// close together to fit a value between them.
function reorderNooks(nooks, draggedUri, newIndex) {
  const fromIndex = nooks.findIndex((n) => n.uri === draggedUri);
  if (fromIndex < 0) return [];
  const [moved] = nooks.splice(fromIndex, 1);
  nooks.splice(newIndex > fromIndex ? newIndex - 1 : newIndex, 0, moved);

  const renumberAll = () => {
    nooks.forEach((n, i) => (n.order = (i + 1) * 1000));
    return nooks.slice();
  };

  if (!nooks.every((n) => typeof n.order === "number")) return renumberAll();

  const idx = nooks.indexOf(moved);
  const before = idx > 0 ? nooks[idx - 1].order : null;
  const after = idx < nooks.length - 1 ? nooks[idx + 1].order : null;
  let newOrder;
  if (before == null && after != null) newOrder = after - 1000;
  else if (before != null && after == null) newOrder = before + 1000;
  else if (before != null && after != null) {
    newOrder = Math.floor((before + after) / 2);
    if (newOrder === before || newOrder === after) return renumberAll();
  } else {
    newOrder = 1000;
  }
  moved.order = newOrder;
  return [moved];
}

// The drop target for reordering nooks themselves: a nook's title is the
// drag handle, dropped anywhere in this area repositions it relative to
// its siblings based on vertical position — the works-inside-a-nook drop
// zones already stop propagation, so this only ever fires for drops in the
// gaps between/around nook boxes (headers, margins), not inside a grid.
function makeNookReorderArea(container, getBoxes, onReorder) {
  container.addEventListener("dragover", (e) => e.preventDefault());
  container.addEventListener("drop", (e) => {
    const data = JSON.parse(e.dataTransfer.getData("application/json") || "{}");
    if (!data.nookReorder) return;
    e.preventDefault();
    const boxes = getBoxes().filter((b) => b.dataset.nookUri !== data.uri);
    let targetIndex = boxes.length;
    for (let i = 0; i < boxes.length; i++) {
      const rect = boxes[i].getBoundingClientRect();
      if (e.clientY < rect.top + rect.height / 2) {
        targetIndex = i;
        break;
      }
    }
    onReorder(data.uri, targetIndex);
  });
}

function renderOrganizer(root, state) {
  root.innerHTML = "";

  const rerender = () => renderOrganizer(root, state);

  const removeFromShelf = async (work) => {
    const key = workKey(work);
    for (const nook of state.nooks) {
      if (nook.works.some((w) => workKey(w) === key)) {
        nook.works = nook.works.filter((w) => workKey(w) !== key);
        await saveNook(nook);
      }
    }
    const item = state.items.find((it) => workKey(it) === key);
    try {
      await fetchJSON("/api/shelf/delete", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ uri: item.uri }),
      });
      state.items = state.items.filter((it) => workKey(it) !== key);
      rerender();
    } catch (err) {
      alert(`failed to remove from shelf: ${err}`);
    }
  };

  // The one place a dragged poster actually gets moved: pulled out of
  // wherever it came from, inserted at the target position (or appended,
  // if dropped on empty space), then whichever nook(s) changed get saved.
  const moveWork = async (data, targetNookUri, beforeWork, before) => {
    const key = `${data.provider}/${data.id}`;
    const touched = new Set();

    if (data.fromNook) {
      const src = state.nooks.find((n) => n.uri === data.fromNook);
      if (src) {
        src.works = src.works.filter((w) => workKey(w) !== key);
        touched.add(src.uri);
      }
    }

    if (targetNookUri) {
      const target = state.nooks.find((n) => n.uri === targetNookUri);
      if (target) {
        const source = state.items.find((it) => workKey(it) === key) || { ...data, title: data.id };
        const entry = { provider: source.provider, id: source.id, title: source.title, poster: source.poster };
        if (beforeWork) {
          const idx = target.works.findIndex((w) => workKey(w) === workKey(beforeWork));
          target.works.splice(idx < 0 ? target.works.length : before ? idx : idx + 1, 0, entry);
        } else {
          target.works.push(entry);
        }
        touched.add(target.uri);
      }
    }

    for (const uri of touched) {
      const nook = state.nooks.find((n) => n.uri === uri);
      if (nook) await saveNook(nook);
    }
    rerender();
  };

  // ---- nooks, on top ----
  const nooksArea = el("div", { class: "nooks-area" });
  for (const nook of state.nooks) {
    nooksArea.appendChild(renderNookBox(nook, state, { moveWork, removeFromShelf, rerender }));
  }
  nooksArea.appendChild(renderNewNookTrigger(state, rerender));
  makeNookReorderArea(
    nooksArea,
    () => Array.from(nooksArea.querySelectorAll(".nook[data-nook-uri]")),
    async (draggedUri, targetIndex) => {
      const toSave = reorderNooks(state.nooks, draggedUri, targetIndex);
      for (const nook of toSave) await saveNook(nook);
      rerender();
    }
  );
  root.appendChild(nooksArea);

  // ---- unsorted, underneath ----
  const nookedKeys = new Set(state.nooks.flatMap((n) => n.works.map(workKey)));
  const unsorted = state.items.filter((it) => !nookedKeys.has(workKey(it)));

  const unsortedSection = el("section", {}, [el("h2", { text: `Unsorted (${unsorted.length})` })]);
  const unsortedGrid = el("div", { class: "shelf-grid" });
  makeDropContainer(unsortedGrid, (data) => moveWork(data, null, null, false));
  for (const item of unsorted) {
    unsortedGrid.appendChild(renderPoster(item, { fromNookUri: null, onMove: (data, w, before) => moveWork(data, null, w, before), onRemove: removeFromShelf }));
  }
  if (unsorted.length === 0) {
    unsortedGrid.appendChild(el("p", { class: "empty", text: "everything is organized into a nook" }));
  }
  unsortedSection.appendChild(unsortedGrid);
  root.appendChild(unsortedSection);
}

function renderNookBox(nook, state, { moveWork, removeFromShelf, rerender }) {
  const title = el("h2", { text: nook.name, class: "nook-title", draggable: "true" });
  title.addEventListener("dragstart", (e) => {
    e.dataTransfer.effectAllowed = "move";
    e.dataTransfer.setData("application/json", JSON.stringify({ nookReorder: true, uri: nook.uri }));
  });

  const box = el("div", { class: `nook nook-${nook.theme}`, "data-nook-uri": nook.uri }, [title]);
  if (nook.description) box.appendChild(el("p", { class: "mono nook-description", text: nook.description }));

  const grid = el("div", { class: "shelf-grid" });
  makeDropContainer(grid, (data) => moveWork(data, nook.uri, null, false));
  for (const work of nook.works) {
    grid.appendChild(
      renderPoster(work, {
        fromNookUri: nook.uri,
        onMove: (data, w, before) => moveWork(data, nook.uri, w, before),
        onRemove: removeFromShelf,
      })
    );
  }
  if (nook.works.length === 0) {
    grid.appendChild(el("p", { class: "empty", text: "drag works here" }));
  }
  box.appendChild(grid);

  const controls = el("div", { class: "nook-controls" });

  const settingsBtn = el("button", { type: "button", class: "action-btn-text", text: "edit" });
  const settingsBox = el("div", { class: "nook-settings", style: "display:none" });
  const descInput = el("input", { type: "text", value: nook.description || "", placeholder: "description" });
  const themeSelect = el("select", {}, NOOK_THEMES.map((t) => el("option", { value: t, text: t, ...(t === nook.theme ? { selected: "selected" } : {}) })));
  const saveBtn = el("button", { type: "button", text: "Save" });
  saveBtn.addEventListener("click", async () => {
    nook.description = descInput.value.trim();
    nook.theme = themeSelect.value;
    await saveNook(nook);
    rerender();
  });
  settingsBox.appendChild(descInput);
  settingsBox.appendChild(themeSelect);
  settingsBox.appendChild(saveBtn);
  settingsBtn.addEventListener("click", () => {
    settingsBox.style.display = settingsBox.style.display === "none" ? "flex" : "none";
  });

  const deleteBtn = el("button", { type: "button", class: "action-btn-text", text: "delete" });
  deleteBtn.addEventListener("click", async () => {
    if (!confirm(`Delete "${nook.name}"? The works stay on your shelf, unsorted.`)) return;
    try {
      await fetchJSON("/api/nooks/delete", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ uri: nook.uri }),
      });
      state.nooks = state.nooks.filter((n) => n.uri !== nook.uri);
      rerender();
    } catch (err) {
      alert(`failed to delete nook: ${err}`);
    }
  });

  controls.appendChild(settingsBtn);
  controls.appendChild(deleteBtn);
  box.appendChild(controls);
  box.appendChild(settingsBox);

  return box;
}

// Lightweight creation, no form: click, type a name, done — filling it is
// a drag away, not a follow-up step in the same dialog.
function renderNewNookTrigger(state, rerender) {
  const wrap = el("div", { class: "nook nook-new" });
  const trigger = el("button", { type: "button", class: "action-btn-text", text: "+ New nook" });
  const input = el("input", { type: "text", placeholder: "name…", style: "display:none" });

  const create = async () => {
    const name = input.value.trim();
    if (!name) {
      input.style.display = "none";
      trigger.style.display = "inline-flex";
      return;
    }
    try {
      const created = await fetchJSON("/api/nooks", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name, description: "", theme: "default", works: [] }),
      });
      state.nooks.push(created);
      rerender();
    } catch (err) {
      alert(`failed to create nook: ${err}`);
    }
  };

  trigger.addEventListener("click", () => {
    trigger.style.display = "none";
    input.style.display = "block";
    input.focus();
  });
  input.addEventListener("keydown", (e) => {
    if (e.key === "Enter") create();
    if (e.key === "Escape") {
      input.value = "";
      input.style.display = "none";
      trigger.style.display = "inline-flex";
    }
  });
  input.addEventListener("blur", create);

  wrap.appendChild(trigger);
  wrap.appendChild(input);
  return wrap;
}

init();
