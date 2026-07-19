// Beta 4 (reconsidered mid-build, 2026-07-17): not "everyone's shelf" — a
// global, unscoped list served no real purpose. This is your own shelf,
// gated by sign-in, same as everywhere else that writes to a PDS.
//
// Beta 7 (reworked, 2026-07-18): organizing into nooks is direct
// manipulation, not a form — drag a poster into a nook, drag within a nook
// to reorder, nooks on top, whatever's still unsorted underneath. Native
// HTML5 drag-and-drop, no library.
//
// Beta 7 (redone again, 2026-07-19): the native drag image alone reads as
// dead — no card visibly follows the cursor, nothing shifts out of the way
// until you release the mouse. Rebuilt on top of the same native
// drag-and-drop (still no library) with two additions: a custom element
// that tracks the cursor 1:1 in place of the browser's own drag image
// (attachDragGhost/positionDragGhost/detachDragGhost below), and a live
// reflow — the dragged card's real DOM node is moved into position on
// every dragover, with FLIP (First-Last-Invert-Play) animating whichever
// siblings had to shift, so the grid visibly reorganizes before you let
// go, the way dragging a file in a folder or a track in a Spotify
// playlist does. dataTransfer's payload is unreadable during dragover by
// design (a cross-origin-drag security restriction) — everything here
// tracks the drag through a module-level dragState object instead, which
// works because source and target are always the same page.

const NOOK_THEMES = ["default", "warm", "cool", "midnight", "riso", "indigo", "manifesto"];

// A nook is a curated gesture, not the whole shelf again — see the
// Lexicon's own works.maxLength and its comment for the reasoning (the
// Sight & Sound top-10 ballot, "best of the year/decade" lists, all
// cluster well under this). Enforced here too, not just at the schema
// level, since this appview doesn't validate incoming records against the
// Lexicon on write — without a client-side check, dragging past the cap
// would silently produce a record the schema itself calls invalid.
const NOOK_WORKS_LIMIT = 50;

// This product has exactly 7 curated nook themes, and the constellation
// (constellation.js) anchors on theme — many nooks sharing few themes
// crowd the same region and stop reading as distinct shapes. 7 echoes
// that same curated count directly, sitting within Miller's classic
// "7±2" estimate for how many categories stay easy to hold in mind at
// once (a more rigorous later revision, Cowan 2001, puts it closer to 4 —
// 7 is the generous end of a real range). Mirrors maxNooksPerAccount in
// nooks.go by hand; enforced there too, not just here.
const MAX_NOOKS_PER_ACCOUNT = 7;

// The local index is a cache of the PDS, not the source of truth — Tap's
// webhook delivery only retries so hard, and restarting the appview mid-
// development (to pick up new code) can leave a gap it never recovers
// from (see resync.go). This re-reads everything straight from the PDS
// and reconciles the local index against it — a manual trigger rather
// than an automatic one, since it's a maintenance action, not something
// that belongs on a timer competing with real traffic.
function resyncButton() {
  const btn = el("button", { type: "button", class: "action-btn-text", text: "resync from PDS" });
  btn.addEventListener("click", async () => {
    const original = btn.textContent;
    btn.disabled = true;
    btn.textContent = "resyncing…";
    try {
      const counts = await fetchJSON("/api/resync", { method: "POST" });
      const total = Object.values(counts).reduce((a, b) => a + b, 0);
      btn.textContent = `resynced (${total} records) — reloading…`;
      window.location.reload();
    } catch (err) {
      btn.textContent = original;
      btn.disabled = false;
      alert(`resync failed: ${err}`);
    }
  });
  return btn;
}

async function init() {
  const app = renderShell("shelf");
  app.innerHTML = "";
  app.appendChild(el("h1", { class: "work-title", text: "Your shelf" }));

  const viewer = await currentViewer();
  if (!viewer) {
    app.appendChild(el("p", {}, [el("a", { href: "/oauth/login", text: "Sign in to see your shelf" })]));
    return;
  }

  const toolbar = el("p", {}, [el("a", { href: "/search", text: "+ Add to shelf" })]);
  toolbar.appendChild(document.createTextNode(" "));
  toolbar.appendChild(resyncButton());
  app.appendChild(toolbar);

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
    // First line of defense against a duplicate work landing in a nook's
    // works array — the server (buildNookRecord/insertNook) also collapses
    // duplicates, but there's no reason to even send one.
    const seenKeys = new Set();
    const dedupedWorks = nook.works.filter((w) => {
      const k = workKey(w);
      if (seenKeys.has(k)) return false;
      seenKeys.add(k);
      return true;
    });
    const body = {
      name: nook.name,
      description: nook.description || "",
      theme: nook.theme,
      order: nook.order,
      works: dedupedWorks.map((w) => ({ provider: w.provider, id: w.id })),
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

// ---- custom drag ghost: a real element that tracks the cursor, replacing
// the browser's own (barely visible, unstyled) drag image. ----

let dragGhostEl = null;
let dragGhostOffset = { x: 0, y: 0 };
const blankDragImage = document.createElement("canvas");
blankDragImage.width = blankDragImage.height = 1;

function attachDragGhost(e, sourceEl) {
  e.dataTransfer.setDragImage(blankDragImage, 0, 0);

  const rect = sourceEl.getBoundingClientRect();
  const ghost = sourceEl.cloneNode(true);
  ghost.classList.add("drag-ghost");
  ghost.style.width = `${rect.width}px`;
  ghost.style.height = `${rect.height}px`;
  dragGhostOffset = { x: e.clientX - rect.left, y: e.clientY - rect.top };
  document.body.appendChild(ghost);
  dragGhostEl = ghost;
  positionDragGhost(e.clientX, e.clientY);
}

function positionDragGhost(x, y) {
  if (!dragGhostEl) return;
  const tx = x - dragGhostOffset.x;
  const ty = y - dragGhostOffset.y;
  dragGhostEl.style.transform = `translate(${tx}px, ${ty}px) scale(1.03) rotate(-1.2deg)`;
}

function detachDragGhost() {
  if (dragGhostEl) {
    dragGhostEl.remove();
    dragGhostEl = null;
  }
}

document.addEventListener("dragover", (e) => positionDragGhost(e.clientX, e.clientY));

// ---- FLIP: snapshot a grid's children positions before a DOM mutation,
// then animate each one from its old position to its new one — this is
// the actual mechanism behind "cards visibly sliding out of the way." ----

function flipSnapshot(container) {
  const before = new Map();
  for (const child of container.children) {
    before.set(child, child.getBoundingClientRect());
  }
  return before;
}

function flipPlay(container, before) {
  for (const child of container.children) {
    const b = before.get(child);
    if (!b) continue;
    const a = child.getBoundingClientRect();
    const dx = b.left - a.left;
    const dy = b.top - a.top;
    if (!dx && !dy) continue;
    child.style.transition = "none";
    child.style.transform = `translate(${dx}px, ${dy}px)`;
    child.getBoundingClientRect(); // force reflow so the line above takes effect before the next one
    requestAnimationFrame(() => {
      child.style.transition = "transform 0.18s ease";
      child.style.transform = "";
    });
  }
}

// Nearest-cell heuristic for a multi-column grid: find whichever existing
// card's center is geometrically closest to the cursor, then insert before
// or after it depending on which side of its center the cursor is on.
function getDragAfterElement(grid, dragged, x, y) {
  const cells = [...grid.querySelectorAll(".shelf-grid-item")].filter((c) => c !== dragged);
  let best = null;
  let bestDist = Infinity;
  for (const cell of cells) {
    const box = cell.getBoundingClientRect();
    const cx = box.left + box.width / 2;
    const cy = box.top + box.height / 2;
    const dist = (x - cx) ** 2 + (y - cy) ** 2;
    if (dist < bestDist) {
      bestDist = dist;
      best = { cell, cx };
    }
  }
  if (!best) return null;
  return x < best.cx ? best.cell : best.cell.nextElementSibling;
}

// The one piece of state a drag needs across events — dataTransfer's own
// payload can't be read until drop, so everything live (the reflow, the
// ghost, the capacity check) reads this instead.
let dragState = null; // { key, fromNookUri }

// justLanded marks the one work that should play the themed "arrived"
// animation on its very next render — consumed once so it never replays.
let justLanded = null; // { key, nookUri }

function attachSortableGrid(grid, state, rerender) {
  grid.addEventListener("dragover", (e) => {
    if (!dragState) return;
    e.preventDefault();

    const dragged = grid.querySelector(`[data-work-key="${cssEscape(dragState.key)}"]`) || document.querySelector(`[data-work-key="${cssEscape(dragState.key)}"]`);
    if (!dragged) return;

    const nookUri = grid.dataset.nookUri || null;
    if (nookUri && dragged.parentElement !== grid) {
      const count = grid.querySelectorAll(".shelf-grid-item").length;
      if (count >= NOOK_WORKS_LIMIT) return; // full — this grid just isn't a valid target
    }

    const after = getDragAfterElement(grid, dragged, e.clientX, e.clientY);
    if (after === dragged) return;
    if (dragged.parentElement === grid && dragged.nextElementSibling === after) return;

    const emptyMsg = grid.querySelector(".empty");
    if (emptyMsg) emptyMsg.remove();

    const sourceGrid = dragged.parentElement;
    const grids = sourceGrid && sourceGrid !== grid ? [grid, sourceGrid] : [grid];
    const snaps = grids.map((g) => [g, flipSnapshot(g)]);

    if (after) grid.insertBefore(dragged, after);
    else grid.appendChild(dragged);

    for (const [g, snap] of snaps) flipPlay(g, snap);
  });

  grid.addEventListener("drop", (e) => {
    e.preventDefault();
    commitDrag(state, rerender);
  });
}

function cssEscape(s) {
  return window.CSS && CSS.escape ? CSS.escape(s) : s.replace(/["\\]/g, "\\$&");
}

// Reads back whatever the live reflow already arranged (the DOM is the
// source of truth by the time drop fires) and persists it — no separate
// "insert at index N" math needed, since the cards are already exactly
// where they visually ended up.
async function commitDrag(state, rerender) {
  if (!dragState) return;
  const { key, fromNookUri } = dragState;
  dragState = null;

  const dragged = document.querySelector(`[data-work-key="${cssEscape(key)}"]`);
  detachDragGhost();
  if (!dragged) {
    rerender();
    return;
  }

  const destGrid = dragged.closest(".shelf-grid");
  const destNookUri = destGrid ? destGrid.dataset.nookUri || null : null;

  const worksByKey = new Map(state.items.map((it) => [workKey(it), it]));
  const readGrid = (grid) =>
    [...grid.querySelectorAll(".shelf-grid-item")]
      .map((c) => worksByKey.get(c.dataset.workKey))
      .filter(Boolean);

  const toSave = [];
  if (destNookUri) {
    const nook = state.nooks.find((n) => n.uri === destNookUri);
    if (nook) {
      nook.works = readGrid(destGrid);
      toSave.push(nook);
    }
  }
  if (fromNookUri && fromNookUri !== destNookUri) {
    const srcNook = state.nooks.find((n) => n.uri === fromNookUri);
    const srcGrid = document.querySelector(`.nook[data-nook-uri="${cssEscape(fromNookUri)}"] .shelf-grid`);
    if (srcNook && srcGrid) {
      srcNook.works = readGrid(srcGrid);
      toSave.push(srcNook);
    }
  }

  if (destNookUri && destNookUri !== fromNookUri) {
    justLanded = { key, nookUri: destNookUri };
  }

  for (const nook of toSave) await saveNook(nook);
  rerender();
}

// A poster is the one draggable unit everywhere it appears — in a nook or
// in the unsorted grid.
function renderPoster(work, { fromNookUri, onRemove }) {
  const key = workKey(work);
  const cell = el("a", {
    href: `/works/${work.provider}/${work.id}`,
    class: "shelf-grid-item draggable-work",
    "data-work-key": key,
  });
  if (justLanded && justLanded.key === key && justLanded.nookUri === (fromNookUri || null)) {
    cell.classList.add("just-landed");
    justLanded = null;
  }
  cell.setAttribute("draggable", "true");
  cell.title = work.title;
  if (work.poster) {
    cell.appendChild(el("img", { src: work.poster, alt: work.title }));
  } else {
    cell.appendChild(el("span", { class: "mono", text: work.title }));
  }
  const removeBtn = el("button", { type: "button", class: "poster-remove", title: "Remove from shelf", text: "×" });
  removeBtn.addEventListener("click", (e) => {
    // The cell is now a real <a href> (so a work is clickable straight to
    // its page) — this button sits inside it, so a click needs
    // preventDefault too, not just stopPropagation, or it navigates to the
    // work page instead of removing it.
    e.preventDefault();
    e.stopPropagation();
    onRemove(work);
  });
  cell.appendChild(removeBtn);

  cell.addEventListener("dragstart", (e) => {
    e.dataTransfer.effectAllowed = "move";
    e.dataTransfer.setData("application/json", JSON.stringify({ provider: work.provider, id: work.id }));
    dragState = { key, fromNookUri: fromNookUri || null };
    attachDragGhost(e, cell);
    requestAnimationFrame(() => cell.classList.add("dragging"));
  });
  cell.addEventListener("dragend", () => {
    cell.classList.remove("dragging");
    detachDragGhost();
    // If a real drop already ran, commitDrag already cleared dragState —
    // this only fires when the card was released outside any valid grid,
    // and just needs the last live-preview reflow discarded.
    if (dragState) {
      dragState = null;
      cell.dispatchEvent(new CustomEvent("shelf:discard-drag", { bubbles: true }));
    }
  });
  return cell;
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
  root.addEventListener("shelf:discard-drag", () => renderOrganizer(root, state), { once: true });

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

  // ---- nooks, on top ----
  const nooksArea = el("div", { class: "nooks-area" });
  for (const nook of state.nooks) {
    nooksArea.appendChild(renderNookBox(nook, state, { removeFromShelf, rerender }));
  }
  if (state.nooks.length < MAX_NOOKS_PER_ACCOUNT) {
    nooksArea.appendChild(renderNewNookTrigger(state, rerender));
  } else {
    nooksArea.appendChild(
      el("p", { class: "empty", text: `${MAX_NOOKS_PER_ACCOUNT} nooks is the most one account holds — delete one to make room for another` })
    );
  }
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
  attachSortableGrid(unsortedGrid, state, rerender);
  for (const item of unsorted) {
    unsortedGrid.appendChild(renderPoster(item, { fromNookUri: null, onRemove: removeFromShelf }));
  }
  if (unsorted.length === 0) {
    unsortedGrid.appendChild(el("p", { class: "empty", text: "everything is organized into a nook" }));
  }
  unsortedSection.appendChild(unsortedGrid);
  root.appendChild(unsortedSection);
}

function renderNookBox(nook, state, { removeFromShelf, rerender }) {
  const title = el("h2", { text: nook.name, class: "nook-title", draggable: "true" });
  title.addEventListener("dragstart", (e) => {
    e.dataTransfer.effectAllowed = "move";
    e.dataTransfer.setData("application/json", JSON.stringify({ nookReorder: true, uri: nook.uri }));
    attachDragGhost(e, title);
  });
  title.addEventListener("dragend", () => detachDragGhost());

  const box = el("div", { class: `nook nook-${nook.theme}`, "data-nook-uri": nook.uri }, [title]);
  if (nook.description) box.appendChild(el("p", { class: "mono nook-description", text: nook.description }));

  const grid = el("div", { class: "shelf-grid", "data-nook-uri": nook.uri });
  attachSortableGrid(grid, state, rerender);
  for (const work of nook.works) {
    grid.appendChild(renderPoster(work, { fromNookUri: nook.uri, onRemove: removeFromShelf }));
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

  // Enter and blur both call create() — and the DOM removal that
  // rerender() does at the end of a successful create() fires a native
  // blur on this very input (it's about to be detached), re-entering
  // create() a second time before the first call's await ever resolves.
  // Clearing the value synchronously, before the first await, is what
  // actually prevents the duplicate: the re-entrant call reads an empty
  // string and takes the early-return path instead of submitting again.
  const create = async () => {
    const name = input.value.trim();
    input.value = "";
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
      input.value = name;
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
