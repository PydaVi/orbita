// Honest stand-ins for menu items that don't exist yet — shown instead of
// a dead link or a nav item quietly hidden until it's real. Each names
// what's actually missing rather than a generic "coming soon."
const PLACEHOLDER_COPY = {
  "/messages": {
    active: "messages",
    title: "Messages",
    text: "Not built yet. Private messages can't just be another public repo record the way everything else on this site is — every other collection here is meant to be public, that's the point of a repo on the network, but a 1:1 conversation is the opposite kind of thing. Whether that needs its own non-federated service, the way Bluesky's own DMs work, is still an open question.",
  },
  "/settings": {
    active: "settings",
    title: "Settings",
    text: "Not built yet — nothing to configure here so far that isn't already a control on the page it belongs to.",
  },
};

function init() {
  const info = PLACEHOLDER_COPY[window.location.pathname];
  const app = renderShell(info ? info.active : null);
  app.appendChild(el("h1", { class: "work-title", text: info ? info.title : "Not built yet" }));
  app.appendChild(el("p", { class: "overview", text: info ? info.text : "" }));
}

init();
