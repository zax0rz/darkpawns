---
tags: [active, feature, website, map]
last_updated: 2026-05-25
---
# World Map Enhancement Brief

**Status:** A, B, D implemented (DP-312/313/315). C needs rework (DP-314). E proposed.
**File:** `website/static/js/map.js` (1,043 lines)
**Data:** `website/static/map/world-overview.json` (91 zones + inter-zone links), `website/static/map/zone-*.json` (91 files), `website/static/map/world-map.json` (9,590 rooms + 14,958 links), `website/static/map/map-index.json` (zone list + room→zone lookup)
**Linear:** DP-312 through DP-316

---

## Current State (as of 2026-05-25)

Features A, B, and D are live. Feature C is implemented but broken (see below). Feature E is unstarted.

The world map renders all 9,590 rooms as sector-colored dots on a Canvas with D3 pan/pinch/zoom (scale 0.04–12). Clicking a room lazy-loads the zone file and enters SVG zone view with a room detail panel. Architecture: Canvas for the world view with viewport culling and sector-batched arc rendering, SVG for per-zone detail with pre-baked BFS positions. Zone positions are pre-computed at build time via Fruchterman-Reingold spring layout (`scripts/precompute_map.py`) — no runtime force simulation.

**Implemented client-side (no precompute script changes):**
- Zone centroids computed at runtime by averaging room x/y per `zone_id` from `worldMapData`
- Inter-zone link weights computed at runtime from `worldMapData.links`
- BFS reachability computed at runtime from room 0 — verified correct: room 0 has 4 links, sits in the 8,190-room main component, 1,400 rooms correctly identified as isolated

**What still reads as a scatter plot:** Feature C's global-sector convex hulls are visually broken (see Feature C section). Fix this before the map ships as "complete."

## The Goal

Add enough visual context that the dot field reads as geography — without faking it. No fake elevation, no decorative 3D. Everything rendered must be derived from actual game data.

---

## Feature A: Zone Name Labels ✅ DONE (DP-312)

**What:** Zone names float above centroids at medium zoom; hidden at world-overview zoom.

**Implemented:** Canvas `fillText`, `fontSize = baseSize / k` for constant screen size. Bold oxblood at `k ≥ 1.0`, muted charcoal at `0.2 ≤ k < 1.0`. Centroids computed at runtime from `worldMapData` room averages. Labels positioned 12 world-units above centroid.

**Gate:** `k ≥ 0.2`.

---

## Feature B: Unreachable Room Dimming ✅ DONE (DP-313)

**What:** ~1,400 rooms with no path to the main world render dimmed; the 8,190 connected rooms render at full opacity.

**Implemented:** BFS at load time from room 0 using all links in `worldMapData`. Unreachable rooms drawn as a separate pass at `rgba(74,69,64,0.18)`. BFS from room 0 is correct: room 0 has 4 links into the main component (rooms 8004, 14400, 19563, 21800) and the BFS reaches exactly the 8,190-room main connected component.

**Not done:** `(isolated)` badge in the zone detail panel. Low priority.

---

## Feature C: Zone Territory Shading ⚠️ NEEDS REWORK (DP-314)

**What:** Color-shade each zone's footprint using the zone's dominant sector type, so clusters of zones read as terrain regions.

**Current implementation (broken):** Groups all visible rooms by sector across the entire viewport, computes one convex hull per sector. This produces hulls that cover 85–99% of the world map for common sectors (field 99.3%, forest 96.8%, mountain 97.4%, deep water 96.0%). Multiple 6%-opacity full-canvas rectangles stack into a muddy wash over everything. Not visually useful.

**Correct approach — per-zone dominant-sector hulls:**
- At build time, compute dominant sector per zone: `argmax(count of each sector across all rooms in the zone)`
- Store as `"ds": N` (dominant sector, compact) on each node in `world-overview.json`
- Also store zone bbox: `"bx0", "by0", "bx1", "by1"` (min/max of zone room positions in world coords) for viewport culling
- In `drawWorldMap()` at `k > 0.08`: for each zone node in viewport, draw convex hull of that zone's rooms in `worldMapData` filled with `SECTOR_COLOR[ds]` at 6% opacity
- Zone-level hulls will cover 1–5% of the map each — they produce real territory shapes, not full-canvas blobs
- Render before links and dots so dots sit on top

**Why not global-sector hulls:** Sectors like forest, field, and mountain are distributed in every corner of the world (bounding boxes span 97–99% of the map). Their hulls cover everything and convey nothing.

**Implementation:**
- `precompute_map.py`: ~30 LOC — dominant sector per zone + bbox computation, emit `"ds"` and bbox fields to `world-overview.json` nodes
- `map.js` `drawWorldMap()`: replace the existing per-sector hull block (~26 LOC) with per-zone hull block (~50 LOC); requires `world-overview.json` nodes to be available at render time
- **Fetch change:** `world-overview.json` needs to be loaded alongside `world-map.json`. Add a parallel `fetch('/map/world-overview.json')` in `selectWorldOverview()`.

**Effort:** S-M total — ~30 LOC precompute, ~50 LOC renderer, one additional fetch. The convex hull function already exists in `map.js` (monotone chain, lines 452–474).

---

## Feature D: Inter-Zone Link Lines ✅ DONE (DP-315)

**What:** Thin oxblood lines between zone centroids, thickness proportional to the number of shared exits.

**Implemented:** Zone-to-zone link counts computed at load time by iterating `worldMapData.links` and bucketing by zone pair. Centroids from the runtime-computed `zoneCentroids` map. Lines drawn at `k > 0.12` with `lineWidth = Math.min(count * 0.4, 3.5) / k`, color `rgba(139,0,0,0.06)`. Viewport culled (both endpoints outside = skip).

---

## Feature E: Weighted Spring Layout (Connection-Density Clustering)

**What:** Re-run the spring layout with edge weights proportional to inter-zone connection count, so heavily-connected zone pairs sit closer together.

**Why:** The current spring layout treats all inter-zone edges equally — one exit between zones pulls them as hard as twenty exits. Weighting edges by connection count would cluster the world's major corridors more tightly, making the map topology read more naturally as a continent.

**How:**
- In `spring_layout()` in `precompute_map.py`: add a `weights` dict argument `{(u,v): w}`
- In the attraction step: `f = d * d / k * weight(u, v)` — stronger pull for more-connected pairs
- Weight: `min(n_exits / 3.0, 4.0)` — soft cap so one hyper-connected zone pair doesn't dominate
- Inter-zone weights computed as part of Feature D's precompute pass (same data)
- Re-run generates new `x, y` values in `world-overview.json`; browser code unchanged

**Note:** This changes the baked positions in `world-overview.json`. The visual result will differ from the current layout — worth previewing before committing. Run the script, open a local Hugo serve, compare before/after.

**Data:** No new files. `world-overview.json` `x, y` values change.

**Effort:** M — ~40 LOC in `precompute_map.py` (weight param + attraction formula change + weight computation). Zero browser changes. Requires a visual review pass after generation.

**Dependency:** Needs the inter-zone weight data from Feature D's precompute pass.

---

## Scope Summary

| Feature | Status | Remaining LOC | Files to touch | Dependencies |
|---------|--------|---------------|----------------|--------------|
| A — Zone name labels | ✅ Done | — | — | — |
| B — Unreachable room dimming | ✅ Done | — | — | — |
| C — Zone territory shading | ⚠️ Rework needed | ~80 | `precompute_map.py`, `world-overview.json` schema, `map.js` | None |
| D — Inter-zone link lines | ✅ Done | — | — | — |
| E — Weighted spring layout | Proposed | ~40 | `precompute_map.py` only | None (weight data computed inline) |

**Remaining work:** Feature C rework (~80 LOC) + optional Feature E (~40 LOC). Feature C is the blocker — the current implementation actively degrades the map.

---

## Build Order (remaining)

**C → E**

1. **C (Territory shading rework)** — fix first; the current global-sector hull implementation makes the map look worse at low zoom. Requires a parallel fetch of `world-overview.json` and a precompute pass to add dominant sector + bbox per zone node.
2. **E (Weighted layout)** — optional polish. Requires visual review after running the precompute script. Do it last so you can evaluate the layout with all other visual layers in place.

---

## What This Is NOT

- **Not 3D.** No Three.js, no WebGL, no elevation. The data doesn't have a third dimension and faking it would be dishonest.
- **Not a force simulation at runtime.** All layout changes (E) run at build time in `precompute_map.py`. The browser renders static positions.
- **Not a rewrite.** All five features are additive to the existing `drawWorldMap()` function. No architectural changes.
- **Not performance-critical.** Canvas handles 9,590 dots at 60fps today. Zone shading adds at most 91 filled rects or hulls per frame — negligible.

---

## Success Criteria

- Zone names are readable at medium zoom without clicking
- You can see which zones connect to which, and roughly how strongly
- Isolated zones are visually distinct from the connected world
- Forest zones feel like forests, ocean zones feel like ocean
- The map looks like a map, not a scatter plot
- No fake geography — everything derived from actual game data
