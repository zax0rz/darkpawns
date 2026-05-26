# Dark Pawns Globe — Implementation Brief

## Vision

A precomputed, force-directed spherical graph of 10,000 rooms. Throw away the room coordinates — the *structure* of the world determines the shape. Zones cluster on a sphere based on their connections. Rotate it, zoom into it, click a room to see data, expand into a flat zone map for actual navigation.

Two layers, two jobs:
- **The Globe** — beautiful, explorable, screenshot-worthy. Not for navigation.
- **The Zone Map** — flat, functional, useful. Already exists. Clean it up, don't rewrite it.

---

## Architecture

### Precompute (Python, build-time)

**Input:** `website/static/map/world-map.json` — 9,590 rooms, 14,958 edges.

**Output:** `website/static/map/world-sphere.json` — unit sphere coordinates for every room.

**Algorithm:**

1. **Zone-level force layout (~93 zones)**
   - Build a zone adjacency graph: zones connected by at least one exit link attract, unconnected zones repel.
   - Force-direct the ~93 zone centroids onto a unit sphere. Use great-circle distance for attraction/repulsion. ~200 iterations.
   - Result: ~93 points on a sphere where Midgaard is near the Drow Caves because players walk between them.

2. **Room-level placement within zone patches**
   - Each zone gets a "patch" on the sphere surface — a circular region centered on its zone centroid.
   - Patch radius proportional to room count (bigger zones get more space).
   - Within each patch, force-direct individual rooms using the zone's local exit graph. Constrain positions to the sphere surface near the zone centroid.
   - Result: rooms cluster together, connected rooms are neighbors, dead ends stick to the edges.

3. **Collision resolution**
   - After room placement, check for zone patches that overlap on the sphere surface.
   - Push apart using repulsion until no overlaps. Zones with many connections should resist moving (they're hubs).

4. **Output format:**

```json
{
  "rooms": [
    {"id": 3001, "x": 0.142, "y": -0.832, "z": 0.534, "sector": 1, "zone": 30},
    ...
  ],
  "zones": [
    {"id": 30, "name": "Midgaard", "x": 0.15, "y": -0.82, "z": 0.54, "count": 214},
    ...
  ]
}
```

All coordinates on the unit sphere (x²+y²+z²=1).

**Dependencies:** `numpy`, `scipy` (for sparse graph operations). No heavy ML deps.

**Script location:** `website/scripts/precompute_sphere.py`
**Build hook:** Add to `make deploy-site` pipeline, runs after `parse_db.py`.

**Estimated time:** ~30 seconds for 93-zone layout + ~10 seconds for room placement. Well within build budget.

---

### Renderer (JavaScript, browser)

**File:** `website/static/js/map.js` — add `drawGlobe()` alongside existing `drawWorldMap()` and `drawConstellation()`.

**State:**
```javascript
let globeData = null;       // loaded from world-sphere.json
let globeRotation = [0, 0]; // [theta, phi] — trackball rotation
let globeZoom = 1.0;        // projection radius multiplier
```

**Rendering pipeline:**

1. **Rotate all room coordinates** by the current rotation matrix (3×3, derived from theta/phi).
2. **Project to 2D:** `screen_x = cx + R * x_rot`, `screen_y = cy + R * y_rot`
3. **Depth sort:** `z_rot` determines draw order. Rooms behind the sphere (`z_rot < 0`) drawn dimmer/smaller. Rooms exactly behind hidden entirely.
4. **Draw dots:** Sector-colored, size scaled by `0.5 + 0.5 * (z_rot + 1)` (closer = bigger).
5. **Draw edges (optional, zoom-gated):** Only draw edges between visible rooms. Great-circle arcs on the sphere surface. Very faint at low zoom.

**Interaction:**
- **Mouse drag:** Rotate the sphere (trackball — update theta/phi).
- **Scroll/pinch:** Zoom (adjust globeZoom).
- **Click:** Ray-sphere hit test → find nearest room to click point → show room card.
- **Hash route:** `#globe` or `/map#globe` opens globe view.

**Performance notes:**
- 9,590 dots on Canvas 2D is trivial — it's just arcs. No WebGL needed.
- Don't draw edges at full zoom-out (10K edges = black smear). Only draw edges when zoomed in enough that <2000 visible edges.
- Sort rooms by z_rot once per frame (9,590 items — `Array.sort` takes <1ms).
- The sphere is static geometry. No animation loop needed unless actively dragging.

---

### Room Card (HTML/CSS, new component)

**Trigger:** Click a room on the globe → card slides in from the right side.

**Content:**
- Room name (h2, DM Serif Display)
- Zone name (mono slug, e.g. "ZONE 30 · MIDGAARD")
- Room description (italic body text)
- VNUM
- Exit list (N, E, S, W, U, D — with links)
- Mobs that spawn here (linked to /database#mob-N)
- Items that load here (linked to /database#item-N)
- **"Expand to Zone Map" button** — switches to the existing flat zone view

**Reuse:** The database cross-reference data is already in `database.json`. Load it once, index by room ID, and the card can show mob/item links without extra fetches.

**Styling:** Match existing detail panels — parchment background, oxblood accents, DM Serif Display headings. Bottom sheet on mobile.

---

### Zone Map (existing, minor cleanup)

The flat zone SVG view already exists in `map.js` (`renderZone()`, line 742). It has:
- BFS-positioned rooms with x,y coordinates
- SVG rendering with room boxes, exit lines, sector colors
- Pathfinding (A*)
- Room click → detail panel

**Changes needed:**
- Make the "Expand from Globe" transition smooth — zone map loads in place, globe fades out.
- Maybe add a breadcrumb: "← Back to Globe" button.
- Clean up the detail panel to match the new room card style.

Don't rewrite this. It works.

---

## What NOT to Change

- Existing grid world map — stays as the "geographic" view
- Database page — untouched
- Help system — untouched
- The flat zone SVG rendering pipeline — reuse, don't rewrite

---

## View Toggle

Three modes on /map:
1. **Grid** — existing geographic world map (current default)
2. **Globe** — new force-directed sphere (this project)
3. **Zone** — flat zone SVG (existing, triggered by clicking a room on globe or grid)

Toggle button cycles or offers a dropdown. URL hash: `#grid`, `#globe`, `#zone-{id}`.

---

## Linear Issues (to create)

| # | Title | Scope | Effort |
|---|-------|-------|--------|
| DP-XXX | Precompute sphere layout | Python script, force-directed zone+room placement on unit sphere | ~1 day |
| DP-XXX | Globe renderer | Canvas 2D sphere render with rotation, zoom, depth shading | ~0.5 day |
| XXX | Room card component | Detail card on room click, cross-reference data, expand-to-zone button | ~2 hours |
| DP-XXX | Globe ↔ Zone transition | Smooth fade, breadcrumb navigation | ~1 hour |
| DP-XXX | Three-mode view toggle | Grid / Globe / Zone routing with URL hash | ~1 hour |

**Total estimated effort:** ~2 days for a focused developer.

---

## Build & Deploy

```bash
cd darkpawns_repo
python3 website/scripts/precompute_sphere.py   # generates world-sphere.json
make deploy-site                                 # Hugo builds + rsyncs
```

No Go code changes. No server changes. Pure frontend + build-time data.
