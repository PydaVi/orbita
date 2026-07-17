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
