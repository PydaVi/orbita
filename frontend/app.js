// Beta 3: the work page is the hub for a whole series/movie/album/book —
// poster, who has it on their shelf, and (for TV) a season list you expand
// in place. Clicking a specific episode navigates to its own dedicated
// page: that's where its community actually gathers, so it gets real
// space (a big still, the synopsis, and that episode's notes below) —
// not a nested accordion three levels deep.
//
// Anything that came from another account (title, handle, note text) goes
// in via textContent, never innerHTML — nothing written by a stranger's
// PDS can inject markup here.

function el(tag, props, children) {
  const node = document.createElement(tag);
  if (props) {
    for (const [k, v] of Object.entries(props)) {
      if (k === "class") node.className = v;
      else if (k === "text") node.textContent = v;
      else node.setAttribute(k, v);
    }
  }
  for (const child of children || []) {
    node.appendChild(child);
  }
  return node;
}

async function fetchJSON(url) {
  const res = await fetch(url);
  if (!res.ok) throw new Error(`API returned ${res.status}`);
  return res.json();
}

// Purely decorative, deterministic per work id, no backend data involved —
// the "catalog coordinate" motif from this product's own visual language.
function pseudoCoord(seed) {
  let h = 0;
  for (let i = 0; i < seed.length; i++) {
    h = (h * 31 + seed.charCodeAt(i)) >>> 0;
  }
  const ra = h % 24;
  const min = (h >>> 5) % 60;
  const dec = (h >>> 11) % 90;
  return `α ${String(ra).padStart(2, "0")}h${String(min).padStart(2, "0")}m · δ +${String(dec).padStart(2, "0")}°`;
}

// The API falls back to the raw DID as "handle" when it genuinely can't be
// resolved (an account that doesn't exist on the real network — a local
// sandbox test account, for instance). Truncated here rather than shown in
// full: still honest that it's unresolved, without a long did:plc:... string
// breaking the layout.
function displayHandle(handle) {
  if (handle && handle.startsWith("did:") && handle.length > 20) {
    return handle.slice(0, 16) + "…";
  }
  return handle;
}

// Handle/avatar come resolved from the API (see identity.go) — this just
// renders them: a real image if one resolved, otherwise a plain initial in
// a circle, never a broken <img> and never a raw DID.
function avatarEl(handle, avatarUrl) {
  if (avatarUrl) {
    return el("img", { class: "avatar", src: avatarUrl, alt: "" });
  }
  const initial = (handle || "?").replace("did:", "").charAt(0).toUpperCase();
  return el("span", { class: "avatar avatar-fallback", text: initial || "?" });
}

function rkeyOf(uri) {
  const parts = uri.split("/");
  return parts[parts.length - 1];
}

// A shared link to a specific note (#note-{rkey}) scrolls to and
// highlights it, once the page it lives on has actually rendered.
function scrollToHash() {
  if (!window.location.hash) return;
  const target = document.getElementById(window.location.hash.slice(1));
  if (target) {
    target.classList.add("highlight");
    target.scrollIntoView({ block: "center" });
  }
}

async function currentViewer() {
  try {
    const res = await fetch("/api/me");
    if (!res.ok) return null;
    return res.json();
  } catch {
    return null;
  }
}

async function loadPage() {
  const app = document.getElementById("app");
  const match = window.location.pathname.match(
    /^\/works\/([^/]+)\/([^/]+)(?:\/season\/(\d+)(?:\/episode\/(\d+))?)?$/
  );
  if (!match) {
    app.textContent = "not a work page";
    return;
  }
  const [, provider, id, seasonParam, episodeParam] = match;

  try {
    if (episodeParam) {
      const data = await fetchJSON(
        `/api/works/${provider}/${id}/season/${seasonParam}/episode/${episodeParam}`
      );
      renderEpisodePage(app, provider, id, data);
    } else {
      const [work, viewer] = await Promise.all([fetchJSON(`/api/works/${provider}/${id}`), currentViewer()]);
      renderWorkPage(app, provider, id, work, viewer, seasonParam ? Number(seasonParam) : null);
    }
    scrollToHash();
  } catch (err) {
    app.innerHTML = "";
    app.appendChild(el("p", { text: `could not load this page: ${err}` }));
  }
}

function renderWorkPage(app, provider, id, work, viewer, deepSeason) {
  app.innerHTML = "";

  // Principle 1: the work comes before anything about who posted.
  const hero = el("div", { class: "hero" });
  if (work.poster) {
    hero.appendChild(
      el("div", { class: "poster-wrap" }, [
        el("img", { src: work.poster, alt: work.title }),
        el("span", { class: "mono coord", text: pseudoCoord(`${provider}/${id}`) }),
      ])
    );
  } else {
    hero.appendChild(el("span", { class: "mono coord", text: pseudoCoord(`${provider}/${id}`) }));
  }
  hero.appendChild(
    el("div", {}, [
      el("h1", { class: "work-title", text: work.title || `${provider}/${id}` }),
      el("hr", { class: "hero-rule" }),
    ])
  );
  app.appendChild(hero);

  // Order: poster > who has this on their shelf (with your own add/remove
  // control) > seasons to browse > notes about the work as a whole.
  renderAccountsSection(app, provider, id, work.accounts, viewer);

  if (work.seasons && work.seasons.length > 0) {
    renderSeasonsSection(app, provider, id, work.seasons, deepSeason);
  }

  renderNotesSection(app, provider, id, work.notes, null, null, work.poster);
}

function renderAccountsSection(app, provider, id, accounts, viewer) {
  const section = el("section", {}, [el("h2", { text: `${accounts.length} account(s) have this on their shelf` })]);

  const shelfControl = el("div", { class: "shelf-control" });
  section.appendChild(shelfControl);

  const list = el("ul", { class: "plain" });
  const renderAccountRow = (acc) =>
    list.appendChild(
      el("li", { class: "account-row", id: `shelf-${rkeyOf(acc.uri)}` }, [
        avatarEl(acc.handle, acc.avatarUrl),
        el("span", { class: "mono", text: `@${displayHandle(acc.handle)} — ${acc.addedAt}` }),
      ])
    );
  for (const acc of accounts) renderAccountRow(acc);
  if (accounts.length === 0) {
    list.appendChild(el("li", { class: "empty", text: "nobody has this yet" }));
  }
  section.appendChild(list);
  app.appendChild(section);

  if (!viewer) {
    shelfControl.appendChild(el("a", { href: "/oauth/login", text: "Sign in to add this to your shelf" }));
    return;
  }

  const mine = accounts.find((a) => a.did === viewer.did);
  const renderAdd = () => {
    shelfControl.innerHTML = "";
    const button = el("button", { type: "button", text: "+ Add to shelf" });
    button.addEventListener("click", async () => {
      button.disabled = true;
      try {
        const res = await fetch("/api/shelf/add", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ provider, id }),
        });
        if (!res.ok) throw new Error(await res.text());
        const created = await res.json();
        const empty = list.querySelector(".empty");
        if (empty) empty.remove();
        renderAccountRow(created);
        renderRemove(created.uri);
      } catch (err) {
        alert(`failed to add to shelf: ${err}`);
      } finally {
        button.disabled = false;
      }
    });
    shelfControl.appendChild(button);
  };
  const renderRemove = (uri) => {
    shelfControl.innerHTML = "";
    const label = el("span", { class: "mono", text: "✓ on your shelf — " });
    const button = el("button", { type: "button", text: "remove" });
    button.addEventListener("click", async () => {
      button.disabled = true;
      try {
        const res = await fetch("/api/shelf/delete", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ uri }),
        });
        if (!res.ok) throw new Error(await res.text());
        const row = document.getElementById(`shelf-${rkeyOf(uri)}`);
        if (row) row.remove();
        if (!list.querySelector("li")) {
          list.appendChild(el("li", { class: "empty", text: "nobody has this yet" }));
        }
        renderAdd();
      } catch (err) {
        alert(`failed to remove from shelf: ${err}`);
      } finally {
        button.disabled = false;
      }
    });
    shelfControl.appendChild(label);
    shelfControl.appendChild(button);
  };

  if (mine) {
    renderRemove(mine.uri);
  } else {
    renderAdd();
  }
}

function renderSeasonsSection(app, provider, id, seasons, deepSeason) {
  const section = el("section", {}, [el("h2", { text: "Seasons" })]);
  let deepDetails = null;

  for (const s of seasons) {
    const body = el("div", { class: "season-body mono", text: "…" });
    const details = el("details", { class: "season" }, [
      el("summary", { text: `${s.name} · ${s.episodeCount} episodes` }),
      body,
    ]);

    let loadPromise = null;
    const ensureLoaded = () => {
      if (!loadPromise) {
        loadPromise = fetchJSON(`/api/works/${provider}/${id}/season/${s.number}`)
          .then((data) => {
            body.className = "season-body";
            body.innerHTML = "";
            body.appendChild(renderEpisodeLinks(provider, id, data));
          })
          .catch((err) => {
            body.textContent = `could not load episodes: ${err}`;
          });
      }
      return loadPromise;
    };
    details.addEventListener("toggle", () => {
      if (details.open) ensureLoaded();
    });

    section.appendChild(details);
    if (s.number === deepSeason) {
      details.open = true;
      ensureLoaded();
      deepDetails = details;
    }
  }

  app.appendChild(section);
  if (deepDetails) deepDetails.scrollIntoView({ block: "start" });
}

// Episodes are plain links now, not a further nested <details> — clicking
// one navigates to that episode's own page, where its discussion lives.
function renderEpisodeLinks(provider, id, seasonData) {
  const list = el("ul", { class: "plain" });
  for (const e of seasonData.episodes || []) {
    const rowChildren = [];
    if (e.stillUrl) {
      rowChildren.push(el("img", { class: "episode-still", src: e.stillUrl, alt: "" }));
    }
    rowChildren.push(
      el("span", { class: "episode-summary-text" }, [
        el("span", { text: `Episode ${e.number} — ${e.name} ` }),
        el("span", { class: "mono", text: e.airDate }),
      ])
    );
    const link = el("a", { href: `/works/${provider}/${id}/season/${seasonData.season}/episode/${e.number}`, class: "episode-link" }, rowChildren);
    list.appendChild(el("li", {}, [link]));
  }
  if (!seasonData.episodes || seasonData.episodes.length === 0) {
    list.appendChild(el("li", { class: "empty", text: "could not load episodes" }));
  }
  return list;
}

function renderEpisodePage(app, provider, id, data) {
  app.innerHTML = "";

  app.appendChild(
    el("p", {}, [
      el("a", { href: `/works/${provider}/${id}/season/${data.season}`, text: `« ${data.title} · Season ${data.season}` }),
    ])
  );

  const hero = el("div", { class: "hero" });
  if (data.stillUrl) {
    hero.appendChild(el("div", { class: "poster-wrap" }, [el("img", { src: data.stillUrl, alt: data.name })]));
  }
  hero.appendChild(
    el("div", {}, [
      el("h1", { class: "work-title", text: `S${data.season}E${data.episode} — ${data.name}` }),
      el("hr", { class: "hero-rule" }),
      el("p", { text: data.overview }),
      el("p", { class: "mono", text: data.airDate }),
    ])
  );
  app.appendChild(hero);

  renderNotesSection(app, provider, id, data.notes, data.season, data.episode, data.stillUrl);
}

// mediaUrl is the work's poster for a work-level notes section, or that
// episode's still for an episode-level one — shown beside each note so a
// shared note carries the work with it, per principle 1.
function renderNotesSection(container, provider, id, notes, season, episode, mediaUrl) {
  const label = season != null ? `Notes for S${season}E${episode}` : "Notes";
  const section = el("section", {}, [el("h2", { text: label })]);
  const list = el("ul", { class: "plain" });

  const shareURLFor = (rkey) => {
    const base = season != null
      ? `${window.location.origin}/works/${provider}/${id}/season/${season}/episode/${episode}`
      : `${window.location.origin}/works/${provider}/${id}`;
    return `${base}#note-${rkey}`;
  };

  const renderNote = (n) => {
    const rkey = rkeyOf(n.uri);
    const byline = el("div", { class: "note-byline" }, [
      avatarEl(n.handle, n.avatarUrl),
      el("span", { class: "mono", text: `@${displayHandle(n.handle)} · ${n.createdAt}` }),
    ]);
    const shareBtn = el("button", { class: "share-btn", type: "button", text: "share ⤴" });
    shareBtn.addEventListener("click", async () => {
      const url = shareURLFor(rkey);
      if (navigator.share) {
        navigator.share({ title: "Órbita", url }).catch(() => {});
      } else {
        await navigator.clipboard.writeText(url);
        const original = shareBtn.textContent;
        shareBtn.textContent = "copied!";
        setTimeout(() => (shareBtn.textContent = original), 1500);
      }
    });
    byline.appendChild(shareBtn);

    const main = el("div", { class: "note-main" }, [byline, el("p", { class: "note-text", text: n.text })]);
    const children = [main];
    if (mediaUrl) {
      children.unshift(el("div", { class: "note-media" }, [el("img", { src: mediaUrl, alt: "" })]));
    }
    list.appendChild(el("li", { class: "note", id: `note-${rkey}` }, children));
  };

  for (const n of notes || []) renderNote(n);
  if (!notes || notes.length === 0) {
    list.appendChild(el("li", { class: "empty", text: "no notes yet" }));
  }
  section.appendChild(list);

  const textarea = el("textarea", { placeholder: "write a note..." });
  const button = el("button", { type: "button", text: "Add note" });
  button.addEventListener("click", async () => {
    const text = textarea.value.trim();
    if (!text) return;
    button.disabled = true;
    try {
      const payload = { provider, id, text };
      if (season != null) {
        payload.season = season;
        payload.episode = episode;
      }
      const res = await fetch("/api/notes/add", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      if (!res.ok) throw new Error(await res.text());
      const created = await res.json();
      // Appended straight from the write response instead of re-fetching:
      // the PDS write already succeeded, but the local index only catches
      // up once Tap delivers the event — re-fetching here would race that
      // lag and could show nothing for a moment.
      const empty = list.querySelector(".empty");
      if (empty) empty.remove();
      renderNote(created);
      textarea.value = "";
    } catch (err) {
      alert(`failed to add note: ${err}`);
    } finally {
      button.disabled = false;
    }
  });

  section.appendChild(el("div", {}, [textarea, el("br", {}), button]));
  container.appendChild(section);
}

loadPage();
