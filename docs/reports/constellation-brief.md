# Constellation View — Implementation Brief

## What This Is

A toggle between the current grid map ("World") and a dark-mode graph render ("Constellation") on the `/map` page. The constellation view renders 9,590 rooms as glowing dots connected by ~15,000 edges, zoom-gated so you see different levels of detail at different zoom levels.

This is the "wow" feature — the grid view says "MUD map," the constellation view says "10,000 rooms in a living graph."

---

## Source Files

- **Map JS:** `website/static/js/map.js` (1,057 lines) — all rendering logic
- **Map HTML:** `website/layouts/_default/map.html` — toolbar, layout, CSS
- **World data:** `website/static/map/world-map.json` (827KB) — already has everything needed:
  - `rooms[]` — `{id, x, y, sector, zone_id}` (9,590 rooms)
  - `links[]` — `{s, t}` (14,958 edges)

**No data precompute needed.** DP-321 is effectively done — `world-map.json` already contains the edge list in `links[]`. Skip DP-321, go straight to the render.

---

## Current Map Architecture

The world map is rendered on a `<canvas>` element using D3 for pan/zoom (`d3.zoom`). The render pipeline in `drawWorldMap()` (line 500) draws in this order:

1. Zone territory hulls (zoom > 0.08)
2. Inter-zone links (zoom > 0.12)
3. Intra-zone links (zoom > 0.1)
4. Rooms as sector-colored dots (viewport-culled)
5. Selected room highlight
6. Zone name labels (zoom >= 0.2)

Key state:
- `cvTransform` — D3 zoom identity `{x, y, k}`
- `wmRoomPosMap` — O(1) room lookup by ID
- `zoneCentroids` — zone center positions
- `zoneHulls` — convex hulls per zone
- `worldZoneLinks` — inter-zone connections

Canvas element: `canvasEl`, 2D context: `ctx`. Both are global.

---

## Implementation Spec

### 1. Toggle Button

Add a toggle button to the toolbar (next to `btn-path`):

```html
<button class="map-btn" id="btn-view" title="Toggle view mode" aria-label="Toggle view mode">✦</button>
```

Button text alternates between "Grid" and "✦" (or "Graph").
Click switches `viewMode` between `'grid'` and `'constellation'`.

### 2. View Mode State

Add a global:

```javascript
let viewMode = 'grid'; // 'grid' | 'constellation'
```

The `drawWorldMap()` function should branch on `viewMode`. Grid mode uses existing render code. Constellation mode uses new render code below.

### 3. Constellation Render (`drawConstellation()`)

Separate function called from the same `requestAnimationFrame` loop. Uses same `ctx`, same `cvTransform`, same viewport culling logic.

#### Background

```javascript
ctx.fillStyle = '#0d0f15';
ctx.fillRect(0, 0, W, H);
```

#### Edges (zoom-gated)

```
zoom < 0.08:   no edges (just dot cloud — the "10K nodes" moment)
zoom 0.08–0.4: inter-zone edges only (faint silver threads between regions)
zoom > 0.4:    all edges (trace corridors room by room)
```

- Inter-zone edges: `rgba(90, 90, 140, 0.25)`, 1px (scaled by `/k`)
- Intra-zone edges: `rgba(90, 90, 140, 0.12)`, 0.5px
- Viewport-cull all lines (same `vx0/vx1/vy0/vy1` bounds check)

#### Rooms (glowing dots)

- Sector-colored (reuse `SECTOR_COLOR`)
- `ctx.shadowBlur = 6` (set once before the batch, restored after)
- `ctx.shadowColor` = same as fill
- Node radius: degree-based
  - Count exits per room at load time (build a `degreeMap` from `links[]`)
  - degree 1: `r = 2`
  - degree 2–4: `r = 3`
  - degree 5+: `r = 4.5`
- Scale by `1/k` so dots stay same visual size on screen

#### Selected Room

Same oxblood ring as grid mode.

#### Zone Labels

Show zone names at zoom >= 0.15 (earlier than grid mode since there are no hulls to identify regions).

### 4. Mode Switch Behavior

When switching modes:
- Keep the same pan/zoom transform (don't reset view)
- Hide zone hulls in constellation mode
- Toggle `btn-path` visibility (pathfinding makes less sense in graph mode — optional, can keep it)
- Store mode in URL hash: `#graph` or no hash = grid

### 5. CSS Changes

Minimal — the dark background is drawn on canvas. Add a class to `#map-page`:

```css
#map-page.constellation { background: #0d0f15; }
#map-page.constellation .map-sidebar { background: #13151f; color: #c8cad8; }
```

Or just draw the background on canvas and skip the CSS (canvas covers the whole page in world view).

### 6. Degree Precomputation

At load time (after `worldMapData` is fetched), compute room degrees:

```javascript
const roomDegrees = {};
for (const lk of worldMapData.links) {
  roomDegrees[lk.s] = (roomDegrees[lk.s] || 0) + 1;
  roomDegrees[lk.t] = (roomDegrees[lk.t] || 0) + 1;
}
```

Store as a module-level map. Used for dot radius in constellation render.

---

## Zoom Progression (the "wow" moments)

1. **Full zoom out (k < 0.08):** Glowing globe of 9,590 colored dots. No edges. Sector colors create a nebula effect. This is the screenshot moment.
2. **Medium zoom (0.08–0.4):** Zone clusters become visible. Faint silver threads connect regions. "Obsidian graph" feel.
3. **Close zoom (0.4+):** Individual room connections visible. Trace corridors. See dead ends vs hubs.

---

## What NOT to Change

- Grid mode rendering — untouched, stays exactly as is
- Zone detail view (SVG) — unaffected, only world map canvas changes
- Pathfinding — works in both modes or grid-only (your call)
- Sidebar, zone list, room search — all unaffected
- D3 zoom handler — reused as-is

---

## Build & Deploy

```bash
cd darkpawns_repo/website && hugo --minify
make deploy-site
```

Or just: `make deploy-site` from repo root (runs full pipeline including world data parse).

## Linear Issues

- **DP-321:** Precompute edge list — **SKIP** (data already exists in world-map.json as `links[]`)
- **DP-322:** Constellation View — this is the implementation target
