// Shared across every page (work, search, and whatever comes next) —
// loaded before the page-specific script. No bundler, no module system,
// just a plain global script tag, matching the rest of this frontend.

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

async function fetchJSON(url, options) {
  const res = await fetch(url, options);
  if (!res.ok) throw new Error(`API returned ${res.status}`);
  return res.json();
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

function rkeyOf(uri) {
  const parts = uri.split("/");
  return parts[parts.length - 1];
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

// The mark: "a body orbiting another — the metric of the relationship, not
// of the user." A thin ring (the orbit) with one amber point offset near
// the edge (the affinity), never centered. Static, no user data — safe as
// a literal SVG string.
function orbitalMark() {
  const wrap = document.createElement("span");
  wrap.className = "logo-mark";
  wrap.innerHTML = `<svg width="26" height="26" viewBox="0 0 24 24" aria-hidden="true">
    <circle cx="12" cy="12" r="9" fill="none" stroke="currentColor" stroke-width="1"></circle>
    <circle cx="18.4" cy="8.4" r="2" fill="var(--signal)"></circle>
  </svg>`;
  return wrap;
}

// Reply and repost icons — plain line glyphs, no text label, matching the
// same restrained visual language as the orbital mark. currentColor so
// they pick up whatever color the surrounding button state gives them
// (muted normally, signal-colored once "reposted" via :disabled).
function replyIcon() {
  const wrap = document.createElement("span");
  wrap.className = "action-icon";
  wrap.innerHTML = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
    <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path>
  </svg>`;
  return wrap;
}

function repostIcon() {
  const wrap = document.createElement("span");
  wrap.className = "action-icon";
  wrap.innerHTML = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
    <path d="M17 1l4 4-4 4"></path>
    <path d="M3 11V9a4 4 0 0 1 4-4h14"></path>
    <path d="M7 23l-4-4 4-4"></path>
    <path d="M21 13v2a4 4 0 0 1-4 4H3"></path>
  </svg>`;
  return wrap;
}

// renderReplyItem is one level of nesting, read-only for this first pass —
// no reply/RT button on a reply itself (see notes.go's own comment on
// why: replying-to-a-reply is stored correctly at the data layer, it just
// isn't surfaced in the UI yet, so offering the control here would look
// like it worked and then the reply would never actually be visible
// anywhere).
function renderReplyItem(rep) {
  return el("li", {}, [
    el("div", { class: "note-byline" }, [
      el("a", { href: `/profile/${rep.handle}`, class: "note-byline" }, [
        avatarEl(rep.handle, rep.avatarUrl),
        el("span", { class: "mono", text: `@${displayHandle(rep.handle)}` }),
      ]),
      el("span", { class: "mono", text: rep.createdAt }),
    ]),
    el("p", { class: "note-text", text: rep.text }),
  ]);
}

// The RT + reply row under a note's text. No count anywhere — RT only
// ever surfaces as "reposted by @handle" in someone's Following feed
// (see api.go's buildFeedEntry), never a number. onReplyAdded gets the
// newly created reply so the caller can render it into its own nested
// list, since where that list lives differs between the work page and
// the feed.
function noteActionRow(n, provider, id, season, episode, onReplyAdded) {
  const row = el("div", { class: "note-actions" });

  const rtBtn = el("button", { type: "button", class: "action-btn", "aria-label": "Repost", title: "Repost" }, [
    repostIcon(),
  ]);
  rtBtn.addEventListener("click", async () => {
    rtBtn.disabled = true;
    try {
      await fetchJSON("/api/notes/repost", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ uri: n.uri, cid: n.cid }),
      });
      rtBtn.title = "Reposted";
      rtBtn.setAttribute("aria-label", "Reposted");
    } catch (err) {
      alert(`failed to repost: ${err}`);
      rtBtn.disabled = false;
    }
  });
  row.appendChild(rtBtn);

  const replyBtn = el("button", { type: "button", class: "action-btn", "aria-label": "Reply", title: "Reply" }, [
    replyIcon(),
  ]);
  const replyBox = el("div", { class: "reply-box", style: "display:none" });
  const textarea = el("textarea", { placeholder: "write a reply..." });
  const submitBtn = el("button", { type: "button", text: "Reply" });
  submitBtn.addEventListener("click", async () => {
    const text = textarea.value.trim();
    if (!text) return;
    submitBtn.disabled = true;
    try {
      const payload = { provider, id, text, replyTo: { uri: n.uri, cid: n.cid } };
      if (season != null) {
        payload.season = season;
        payload.episode = episode;
      }
      const created = await fetchJSON("/api/notes/add", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      textarea.value = "";
      replyBox.style.display = "none";
      if (onReplyAdded) onReplyAdded(created);
    } catch (err) {
      alert(`failed to reply: ${err}`);
    } finally {
      submitBtn.disabled = false;
    }
  });
  replyBox.appendChild(textarea);
  replyBox.appendChild(submitBtn);

  replyBtn.addEventListener("click", () => {
    replyBox.style.display = replyBox.style.display === "none" ? "block" : "none";
  });
  row.appendChild(replyBtn);
  row.appendChild(replyBox);

  return row;
}

// Every page calls this first, with which nav item (if any) is current.
// Builds the persistent topbar (mark + wordmark only) and the 3-column
// layout — sidebar (text nav) / center / right column — into #shell-mount,
// then moves the page's own #app element into the center column and
// returns it. A DOM node handed to appendChild() that's already attached
// elsewhere gets moved, not duplicated, so #app keeps whatever the page's
// own script has already put into it.
function renderShell(active) {
  const mount = document.getElementById("shell-mount");
  const app = document.getElementById("app");
  if (!mount || !app) return app;

  mount.innerHTML = "";

  const topbar = el("header", { class: "topbar" }, [
    el("a", { href: "/shelf", class: "brand" }, [orbitalMark(), el("span", { class: "wordmark", text: "ÓRBITA" })]),
  ]);

  const navItem = (label, href, key) =>
    el("a", { href, class: key === active ? "nav-link active" : "nav-link", text: label });
  const sidebar = el("nav", { class: "sidebar" }, [
    navItem("Shelf", "/shelf", "shelf"),
    navItem("Feed", "/feed", "feed"),
    navItem("Profile", "/profile", "profile"),
  ]);

  const rightcol = el("aside", { class: "rightcol" }, [
    el("p", { class: "mono", text: "— nothing here yet —" }),
  ]);

  const layout = el("div", { class: "layout" }, [sidebar, app, rightcol]);

  mount.appendChild(topbar);
  mount.appendChild(layout);
  return app;
}
