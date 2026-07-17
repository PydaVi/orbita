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
    el("a", { href: "/search", class: "brand" }, [orbitalMark(), el("span", { class: "wordmark", text: "ÓRBITA" })]),
  ]);

  const navItem = (label, href, key) =>
    el("a", { href, class: key === active ? "nav-link active" : "nav-link", text: label });
  const sidebar = el("nav", { class: "sidebar" }, [
    navItem("Shelf", "/search", "shelf"),
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
