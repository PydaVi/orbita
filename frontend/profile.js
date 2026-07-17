// Beta 4: a real slot for profile, even though the profile itself (a
// person's shelf and notes in one place) is Beta 5 — deliberately just a
// placeholder, not a fake preview of content that doesn't exist yet.

async function init() {
  const app = renderShell("profile");
  app.innerHTML = "";
  app.appendChild(el("h1", { class: "work-title", text: "Profile" }));

  const viewer = await currentViewer();
  if (!viewer) {
    app.appendChild(el("p", {}, [el("a", { href: "/oauth/login", text: "Sign in" })]));
    return;
  }
  app.appendChild(el("p", { class: "mono", text: `Signed in as @${viewer.handle}` }));
  app.appendChild(
    el("p", { class: "mono", text: "Not built yet — your shelf and notes in one place will live here." })
  );
}

init();
