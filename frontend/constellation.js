// Beta 8: the constellation, reimagined for this product's own shape
// rather than ported from earlier work. That earlier version anchored
// nodes by genre/tag, hashed into a fixed position so similar taste lit up
// the same region across different people's profiles. This appview never
// built a tag pipeline — the one thing it *does* have, uniquely, that's
// small and curated enough to serve the same purpose, is a nook's own
// theme (style.theme's handful of knownValues, see nook.json). So this
// constellation is anchored on theme, not genre: every "riso" nook across
// every account lands in the same region of everyone's sky. Provider
// (medium) and decade ride along as secondary, weaker pulls. Nook
// membership is real and deliberate here (not inferred from tag overlap
// the way it had to be before), so works in the same nook get an actual
// drawn connection, not a computed one.
//
// All physics/rendering is fresh, hand-rolled Canvas2D — no library,
// matching this frontend's own rule everywhere else.

const THEME_ORDER = ["default", "warm", "cool", "midnight", "riso", "indigo", "manifesto", "unsorted"];

// Must match --duo-*-hi in styles.css (and --signal/--text-muted) by
// hand — same constraint duotoneFilter() already lives with in common.js,
// for the same reason: no way to read a CSS custom property into a canvas
// fill color without a DOM round-trip, so the values are just duplicated.
const THEME_COLORS = {
  default: "#d98a3d",
  warm: "#e2a45c",
  cool: "#8fb4d1",
  midnight: "#4a5578",
  riso: "#ff48b0",
  indigo: "#a8c9d9",
  manifesto: "#d7301f",
  unsorted: "#8a8894",
};

// --signal itself — reserved here for the one thing it means everywhere
// else in this product (a signal/annotation accent, "used with
// restraint," never a stand-in for content color): the ring around a
// work marks that *you wrote about it*, which is a fact about your own
// voice, not about which nook it's in. Theme color stays on the dot fill.
const SIGNAL_COLOR = "#d98a3d";

function hexToRgba(hex, alpha) {
  const r = parseInt(hex.slice(1, 3), 16);
  const g = parseInt(hex.slice(3, 5), 16);
  const b = parseInt(hex.slice(5, 7), 16);
  return `rgba(${r}, ${g}, ${b}, ${alpha})`;
}

// A real star chart connects a handful of stars into a shape — it doesn't
// draw a line between every possible pair. Greedy nearest-neighbor chain
// through a nook's own works gives the same "connect the dots into one
// legible line" reading instead of the dense hairball a full mesh drew
// (O(n²) edges for an n-work nook, illegible past a handful of works).
function nearestNeighborChain(group) {
  if (group.length < 2) return [];
  const remaining = new Set(group);
  const first = group[0];
  remaining.delete(first);
  const order = [first];
  while (remaining.size > 0) {
    const last = order[order.length - 1];
    let best = null;
    let bestDist = Infinity;
    for (const cand of remaining) {
      const dx = last.x - cand.x;
      const dy = last.y - cand.y;
      const d = dx * dx + dy * dy;
      if (d < bestDist) {
        bestDist = d;
        best = cand;
      }
    }
    remaining.delete(best);
    order.push(best);
  }
  const edges = [];
  for (let i = 0; i < order.length - 1; i++) edges.push([order[i], order[i + 1]]);
  return edges;
}

const PROVIDERS = ["tmdb-movie", "tmdb-tv", "musicbrainz", "open-library"];

function hashString(s) {
  let h = 0;
  for (let i = 0; i < s.length; i++) {
    h = (h * 31 + s.charCodeAt(i)) >>> 0;
  }
  return h;
}

function dotRadius(noteCount) {
  return 4 + Math.log(1 + noteCount) * 2.4;
}

// Deterministic anchor for a theme — the same angle on every profile is
// what makes two people's "warm" nooks land in the same region of their
// own skies, regardless of what either of them named the nook itself.
function themeAnchor(theme, cx, cy, radius) {
  const idx = Math.max(0, THEME_ORDER.indexOf(theme || "unsorted"));
  const angle = (idx / THEME_ORDER.length) * Math.PI * 2 - Math.PI / 2;
  return { x: cx + Math.cos(angle) * radius, y: cy + Math.sin(angle) * radius, angle };
}

function providerOffset(provider, radius) {
  const idx = Math.max(0, PROVIDERS.indexOf(provider));
  const angle = (idx / PROVIDERS.length) * Math.PI * 2;
  return { x: Math.cos(angle) * radius, y: Math.sin(angle) * radius };
}

function decadeOffset(year, radius) {
  if (!year) return { x: 0, y: 0 };
  const decade = Math.floor(parseInt(year, 10) / 10) * 10;
  const angle = (hashString(String(decade)) % 360) * (Math.PI / 180);
  return { x: Math.cos(angle) * radius, y: Math.sin(angle) * radius };
}

// Synchronous force simulation (not a per-frame animation loop) — the
// same "settle it once, render the final frame" approach this appview
// already leans toward for anything computed, not streamed. Anchor pull
// (theme, weight 4) dominates provider/decade (weight 1 each) on purpose:
// nooks are the primary way a shelf is organized here (Beta 7's own
// stated principle), so the constellation's *dominant* visible structure
// should be theme, with medium/decade only as fine texture within it.
function computeConstellationLayout(nodes, width, height) {
  const cx = width / 2;
  const cy = height / 2;
  const anchorRadius = Math.min(width, height) * 0.36;
  const themesPresent = new Set();

  // Starting each node near its *own* target (not all of them piled at
  // dead center) matters more than it looks: with everything jittered
  // around one shared point, most pairs start almost on top of each
  // other, and repulsion at near-zero distance is enormous regardless of
  // how it's tuned — the simulation's first few iterations fling
  // everything straight to the walls before the anchor pull ever gets a
  // say. Starting near the real target means repulsion only ever has to
  // gently sort out nodes that genuinely belong in the same neighborhood.
  const items = nodes.map((n) => {
    const theme = n.theme || "unsorted";
    themesPresent.add(theme);
    const anchor = themeAnchor(theme, cx, cy, anchorRadius);
    const pOff = providerOffset(n.provider, anchorRadius * 0.22);
    const dOff = decadeOffset(n.year, anchorRadius * 0.14);
    const target = { x: anchor.x + pOff.x + dOff.x, y: anchor.y + pOff.y + dOff.y };
    return {
      node: n,
      theme,
      x: target.x + (Math.random() - 0.5) * 40,
      y: target.y + (Math.random() - 0.5) * 40,
      vx: 0,
      vy: 0,
      target,
      r: dotRadius(n.noteCount),
    };
  });

  // Repulsion scaled by canvas area (not a bare constant) so it stays a
  // gentle spacing force regardless of how big the canvas is rendered —
  // an unscaled constant here is what caused the first version to
  // explode: at close range (small d2) it dwarfed the anchor pull by two
  // orders of magnitude and threw every node straight into the wall clamp,
  // which is exactly the "dots pinned to the edges" bug this replaces.
  const repStrength = width * height * 0.00004;
  const maxForce = 6;

  for (let iter = 0; iter < 200; iter++) {
    for (let i = 0; i < items.length; i++) {
      const a = items[i];
      let fx = (a.target.x - a.x) * 0.03;
      let fy = (a.target.y - a.y) * 0.03;
      for (let j = 0; j < items.length; j++) {
        if (i === j) continue;
        const b = items[j];
        const dx = a.x - b.x;
        const dy = a.y - b.y;
        const d2 = Math.max(dx * dx + dy * dy, 60);
        const rep = repStrength / d2;
        fx += dx * rep;
        fy += dy * rep;
      }
      const mag = Math.sqrt(fx * fx + fy * fy);
      if (mag > maxForce) {
        fx = (fx / mag) * maxForce;
        fy = (fy / mag) * maxForce;
      }
      a.vx = (a.vx + fx) * 0.72;
      a.vy = (a.vy + fy) * 0.72;
    }
    for (const a of items) {
      a.x = Math.min(width - a.r - 8, Math.max(a.r + 8, a.x + a.vx));
      a.y = Math.min(height - a.r - 8, Math.max(a.r + 8, a.y + a.vy));
    }
  }

  const anchors = {};
  for (const theme of themesPresent) {
    anchors[theme] = themeAnchor(theme, cx, cy, anchorRadius);
  }
  return { items, anchors, cx, cy, anchorRadius };
}

// compareLayout, when given, is the viewer's *own* shape — computed at
// the same canvas dimensions, so its theme anchors land in exactly the
// same places as the profile being viewed (that alignment is the entire
// point of anchoring on a small shared vocabulary rather than a free
// one). Drawn first, as hollow rings only — no connecting lines, no
// labels — so it reads as a ghosted reference behind the real content,
// not a second competing constellation. Where the two shapes cluster in
// the same region, that's the affinity this whole anchor system exists
// to make visible.
function renderConstellationCanvas(canvas, layout, compareLayout) {
  const { items, anchors, cx, cy, anchorRadius } = layout;
  const dpr = window.devicePixelRatio || 1;
  const rect = canvas.getBoundingClientRect();
  canvas.width = rect.width * dpr;
  canvas.height = rect.height * dpr;
  const ctx = canvas.getContext("2d");
  ctx.scale(dpr, dpr);
  ctx.clearRect(0, 0, rect.width, rect.height);

  // The orbit itself: a faint guide circle at the same radius every
  // anchor sits on — a compass rose, not a chart axis, and a quiet nod to
  // the product's own name (things held at a fixed distance from a
  // center). Drawn first, under everything else.
  ctx.strokeStyle = "rgba(43, 43, 51, 0.7)";
  ctx.lineWidth = 1;
  ctx.setLineDash([2, 5]);
  ctx.beginPath();
  ctx.arc(cx, cy, anchorRadius, 0, Math.PI * 2);
  ctx.stroke();
  ctx.setLineDash([]);

  if (compareLayout) {
    for (const it of compareLayout.items) {
      const color = THEME_COLORS[it.theme] || THEME_COLORS.unsorted;
      ctx.globalAlpha = 0.55;
      ctx.strokeStyle = color;
      ctx.lineWidth = 1.5;
      ctx.beginPath();
      ctx.arc(it.x, it.y, it.r + 2, 0, Math.PI * 2);
      ctx.stroke();
    }
    ctx.globalAlpha = 1;
  }

  // Region labels — only for themes actually present, so an account with
  // three nooks doesn't show five empty labels for moods it never touched.
  // Canvas fonts can't reference a CSS custom property — this has to be
  // the literal family name, unlike everywhere else this project sets
  // type via var(--font-data).
  ctx.font = "11px 'Space Mono', monospace";
  ctx.fillStyle = "rgba(138, 136, 148, 0.6)";
  ctx.textAlign = "center";
  for (const [theme, anchor] of Object.entries(anchors)) {
    ctx.fillText(theme, anchor.x, anchor.y);
  }

  // Nook edges: a real, deliberate grouping here (not inferred from tag
  // overlap), so membership earns an actual drawn connection — but a
  // *chart* connection, one clean line through the group (nearest-
  // neighbor chain), not a line between every possible pair. A full mesh
  // on an n-work nook draws O(n²) crossing lines and reads as a hairball,
  // not a constellation; a real star chart never does that either. Each
  // nook's own lines take its own theme color, faint — the region's
  // identity extends to its connections, not just its dots.
  const byNook = new Map();
  for (const it of items) {
    if (!it.node.nookUri) continue;
    if (!byNook.has(it.node.nookUri)) byNook.set(it.node.nookUri, []);
    byNook.get(it.node.nookUri).push(it);
  }
  for (const group of byNook.values()) {
    const color = THEME_COLORS[group[0].theme] || THEME_COLORS.unsorted;
    ctx.strokeStyle = hexToRgba(color, 0.4);
    ctx.lineWidth = 1;
    for (const [a, b] of nearestNeighborChain(group)) {
      ctx.beginPath();
      ctx.moveTo(a.x, a.y);
      ctx.lineTo(b.x, b.y);
      ctx.stroke();
    }
  }

  for (const it of items) {
    const color = THEME_COLORS[it.theme] || THEME_COLORS.unsorted;
    ctx.globalAlpha = it.node.nookUri ? 0.88 : 0.4; // Unsorted reads fainter — not yet decided, not hidden
    ctx.fillStyle = color;
    ctx.beginPath();
    ctx.arc(it.x, it.y, it.r, 0, Math.PI * 2);
    ctx.fill();
    if (it.node.noteCount > 0) {
      ctx.globalAlpha = 1;
      ctx.strokeStyle = SIGNAL_COLOR;
      ctx.lineWidth = 1.5;
      ctx.beginPath();
      ctx.arc(it.x, it.y, it.r + 3, 0, Math.PI * 2);
      ctx.stroke();
    }
  }
  ctx.globalAlpha = 1;
}

function attachConstellationInteractivity(canvas, items, tooltip) {
  const hitTest = (clientX, clientY) => {
    const rect = canvas.getBoundingClientRect();
    const x = clientX - rect.left;
    const y = clientY - rect.top;
    for (let i = items.length - 1; i >= 0; i--) {
      const it = items[i];
      const dx = x - it.x;
      const dy = y - it.y;
      if (dx * dx + dy * dy <= (it.r + 2) * (it.r + 2)) return it;
    }
    return null;
  };

  canvas.addEventListener("mousemove", (e) => {
    const hit = hitTest(e.clientX, e.clientY);
    if (!hit) {
      tooltip.style.display = "none";
      canvas.style.cursor = "default";
      return;
    }
    canvas.style.cursor = "pointer";
    const label = hit.node.year ? `${hit.node.title} (${hit.node.year})` : hit.node.title;
    tooltip.textContent = hit.node.nookName ? `${label} — ${hit.node.nookName}` : label;
    tooltip.style.display = "block";
    tooltip.style.left = `${e.clientX + 12}px`;
    tooltip.style.top = `${e.clientY + 12}px`;
  });
  canvas.addEventListener("mouseleave", () => {
    tooltip.style.display = "none";
  });
  canvas.addEventListener("click", (e) => {
    const hit = hitTest(e.clientX, e.clientY);
    if (hit) window.location.href = `/works/${hit.node.provider}/${hit.node.id}`;
  });
}

// ---- archetype: a signature derived from the same shape, not a separate
// feature bolted on next to it. ----

// spread: mass-weighted inverse Simpson index across the 8 possible
// regions (7 curated themes + Unsorted) — how many of them a shelf
// actually reaches, weighted so a work with notes counts for a bit more
// than one with none (a voice, not just a placement).
function computeSpread(nodes) {
  const mass = {};
  for (const n of nodes) {
    const theme = n.theme || "unsorted";
    mass[theme] = (mass[theme] || 0) + 1 + n.noteCount * 0.5;
  }
  const total = Object.values(mass).reduce((a, b) => a + b, 0);
  const simpson = Object.values(mass).reduce((sum, m) => sum + (m / total) ** 2, 0);
  return { spread: 1 / simpson / THEME_ORDER.length, mass };
}

// cohesion: the fraction of the whole shelf living inside its single
// biggest nook. A nook already *is* a deliberate, real grouping here —
// unlike a tag-overlap graph, there's no connectivity to infer, just a
// share to measure.
function computeCohesion(nodes) {
  const massByNook = {};
  for (const n of nodes) {
    if (!n.nookUri) continue;
    massByNook[n.nookUri] = (massByNook[n.nookUri] || 0) + 1;
  }
  const entries = Object.entries(massByNook).sort((a, b) => b[1] - a[1]);
  if (entries.length === 0) return { cohesion: 0, biggestNookURI: null, biggestNookCount: 0 };
  return { cohesion: entries[0][1] / nodes.length, biggestNookURI: entries[0][0], biggestNookCount: entries[0][1] };
}

// A first pass, not a calibrated one — these three-way cutoffs are a
// reasonable starting split, not tuned against real distribution data
// (this appview doesn't have enough accounts yet for that to mean
// anything). Worth revisiting once more shelves exist to look at.
const ARCHETYPE_NAMES = [
  // spread: low
  [
    { title: "Luz Cinzenta", voice: "Poucas regiões, ainda nada reunido — um sinal em formação." },
    { title: "Par Próximo", voice: "Duas ou três obsessões, perto uma da outra, ainda não uma só." },
    { title: "Estrela Fixa", voice: "Um gosto raro e definido: quase tudo gravita em torno de um único nook." },
  ],
  // spread: mid
  [
    { title: "Campo Difuso", voice: "Gosto plural, mas nada puxou o resto pra perto ainda." },
    { title: "Trajeto Orbital", voice: "Algumas órbitas bem definidas, o resto ainda em trânsito." },
    { title: "Estrela-Guia", voice: "Gosto plural, mas um nook guia todo o resto." },
  ],
  // spread: high
  [
    { title: "Campo Profundo", voice: "Estante ampla, nada domina — cada obra é seu próprio ponto de luz." },
    { title: "Mapa Estelar", voice: "Muitas regiões, parcialmente organizadas em nooks." },
    { title: "Centro de Massa", voice: "Gosto amplo, mas com gravidade real: um nook grande o bastante pra puxar quase tudo." },
  ],
];

function levelOf(value, lowMax, midMax) {
  if (value < lowMax) return 0;
  if (value < midMax) return 1;
  return 2;
}

function buildEvidence(nodes, mass, cohesionInfo) {
  const touched = Object.keys(mass).length;
  if (cohesionInfo.biggestNookURI && cohesionInfo.cohesion >= 0.4) {
    const nook = nodes.find((n) => n.nookUri === cohesionInfo.biggestNookURI);
    const pct = Math.round(cohesionInfo.cohesion * 100);
    const rest = nodes.length - cohesionInfo.biggestNookCount;
    return `${pct}% da sua estante está no nook "${nook.nookName}" — ${rest} obra${rest === 1 ? "" : "s"} em outros lugares ou ainda sem nook.`;
  }
  return `Sua estante toca ${touched} de ${THEME_ORDER.length} climas possíveis.`;
}

function computeArchetype(nodes) {
  if (nodes.length === 0) return null;
  const { spread, mass } = computeSpread(nodes);
  const cohesionInfo = computeCohesion(nodes);
  const spreadLevel = levelOf(spread, 0.4, 0.7);
  const cohesionLevel = levelOf(cohesionInfo.cohesion, 0.3, 0.6);
  const named = ARCHETYPE_NAMES[spreadLevel][cohesionLevel];
  return {
    title: named.title,
    voice: named.voice,
    evidence: buildEvidence(nodes, mass, cohesionInfo),
  };
}

// ---- public surface — profile.js decides where each piece goes (the
// cover canvas and the archetype card end up in different parts of the
// page, not bundled into one section), fetching the graph once and
// handing these pieces the same node array. ----

async function fetchConstellationNodes(handle) {
  try {
    const data = await fetchJSON(`/api/profile/${handle}/constellation`);
    return data.nodes || [];
  } catch {
    return []; // non-fatal — the rest of the profile still renders
  }
}

// Mounts the interactive canvas into an already-appended <canvas> element —
// appended first because sizing reads the element's real, laid-out
// dimensions (getBoundingClientRect), which only exist once it's actually
// in the document with its CSS applied. compareNodes, when given (the
// signed-in viewer's own shelf, looking at someone else's profile), is
// rendered as the ghosted overlay described above — real affinity, shown,
// not just a promise the anchor system makes in the abstract.
function mountConstellationCanvas(canvas, nodes, compareNodes) {
  const rect = canvas.getBoundingClientRect();
  const w = rect.width;
  const h = rect.height || rect.width * 0.4;
  const layout = computeConstellationLayout(nodes, w, h);
  const compareLayout =
    compareNodes && compareNodes.length >= 2 ? computeConstellationLayout(compareNodes, w, h) : null;
  renderConstellationCanvas(canvas, layout, compareLayout);

  const tooltip = el("div", { class: "constellation-tooltip mono", style: "display:none" });
  document.body.appendChild(tooltip);
  attachConstellationInteractivity(canvas, layout.items, tooltip);
}

// The archetype's own "symbol" — that person's real layout, recomputed at
// a small fixed size, same mechanic as the full canvas. Same appended-
// first requirement as above. Labels are skipped (region names would be
// illegible at this size) by clearing anchors after layout, not by a
// separate rendering path.
function mountArchetypeSymbol(canvas, nodes) {
  const layout = computeConstellationLayout(nodes, 120, 120);
  renderConstellationCanvas(canvas, { ...layout, anchors: {} });
}

function buildArchetypeCard(nodes) {
  const archetype = computeArchetype(nodes);
  if (!archetype) return null;
  const symbolCanvas = el("canvas", { class: "archetype-symbol" });
  const symbolFrame = el("div", { class: "archetype-symbol-frame" }, [symbolCanvas]);
  const card = el("div", { class: "archetype-card" }, [
    symbolFrame,
    el("div", { class: "archetype-body" }, [
      el("h3", { class: "archetype-title", text: archetype.title }),
      el("p", { class: "archetype-voice", text: archetype.voice }),
      el("p", { class: "archetype-evidence mono", text: archetype.evidence }),
    ]),
  ]);
  return { card, symbolCanvas };
}
