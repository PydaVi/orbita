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

// Every page calls this first, with which nav item (if any) is current —
// builds the persistent topbar into #shell-mount and returns the #app
// element the page renders its own content into. Kept as a mount-point
// injection rather than replacing document.body wholesale, so each page's
// own <script> tags are untouched.
function renderShell(active) {
  const mount = document.getElementById("shell-mount");
  if (mount) {
    mount.innerHTML = "";
    const navItem = (label, href, key) =>
      el("a", { href, class: key === active ? "nav-link active" : "nav-link", text: label });
    mount.appendChild(
      el("header", { class: "topbar" }, [
        el("a", { href: "/search", class: "brand" }, [orbitalMark(), el("span", { class: "wordmark", text: "ÓRBITA" })]),
        el("nav", { class: "topnav" }, [
          navItem("Shelf", "/search", "shelf"),
          navItem("Feed", "/feed", "feed"),
          navItem("Profile", "/profile", "profile"),
        ]),
      ])
    );
  }
  return document.getElementById("app");
}
