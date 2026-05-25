#!/usr/bin/env python3
"""
precompute_map.py — Build-time map position precomputation for Dark Pawns.

Reads website/static/map/world.json and emits:
  website/static/map/map-index.json       — zone list + room→zone lookup (~111KB)
  website/static/map/world-overview.json  — 91 zone-centroid nodes + inter-zone links (~9KB)
  website/static/map/zone-{id}.json       — per-zone rooms with baked x,y positions (lazy)
  website/static/map/world-map.json       — all rooms global positions for full world map (~827KB)

Note: map-index.json (not index.json) avoids conflict with Hugo's generated section JSON
output at /map/index.json.

Run from repo root:
  python3 scripts/precompute_map.py

The world is static — these files are generated once at build time and shipped.
The browser becomes a renderer, not a physics engine.
"""

import json
import math
import random
import sys
from collections import deque
from pathlib import Path

REPO_ROOT = Path(__file__).parent.parent
WORLD_JSON = REPO_ROOT / "website/static/map/world.json"
OUT_DIR    = REPO_ROOT / "website/static/map"

CELL = 38          # pixels per grid cell in zone view
OV_W = 9000        # world-overview canvas coordinate space
OV_H = 7000
SPRING_ITERS = 800 # force spring iterations for world overview
GRAVITY      = 0.09 # center-pull per iteration (creates globe shape)
WEIGHT_CAP   = 4.0  # max edge-weight multiplier (prevents dominant corridors collapsing)

DIR_DELTA = {"n": (0, -1), "s": (0, 1), "e": (1, 0), "w": (-1, 0)}
# U/D intentionally skipped — 2D map; connected rooms appear as separate nodes


# ── BFS cardinal-direction grid layout ──────────────────────────────────────

def bfs_layout(rooms: list, cell: int = CELL) -> dict:
    """
    Assign pixel x,y to each room using BFS from the first room, following
    N/S/E/W exits. Collisions resolved by spiral outward search. Rooms
    unreachable via cardinal exits are placed below the main grid.

    Returns: {room_id: {"x": px, "y": py}}
    """
    if not rooms:
        return {}

    room_by_id = {r["id"]: r for r in rooms}
    pos   = {}          # room_id → (col, row)
    grid  = {}          # (col, row) → room_id

    start = rooms[0]["id"]
    pos[start]    = (0, 0)
    grid[(0, 0)]  = start
    queue = deque([start])

    while queue:
        rid  = queue.popleft()
        room = room_by_id.get(rid)
        if not room:
            continue
        col, row = pos[rid]

        for ex in room.get("exits", []):
            tid   = ex.get("t")
            d     = ex.get("d", "")
            delta = DIR_DELTA.get(d)
            if not delta or tid not in room_by_id or tid in pos:
                continue

            tc, tr = col + delta[0], row + delta[1]

            # Resolve collision with spiral search
            if (tc, tr) in grid and grid[(tc, tr)] != tid:
                placed = False
                for radius in range(1, 10):
                    for dc in range(-radius, radius + 1):
                        for dr in range(-radius, radius + 1):
                            if abs(dc) != radius and abs(dr) != radius:
                                continue
                            nc = col + delta[0] + dc
                            nr = row + delta[1] + dr
                            if (nc, nr) not in grid:
                                tc, tr = nc, nr
                                placed = True
                                break
                        if placed:
                            break
                    if placed:
                        break
                if not placed:
                    continue  # give up on this exit; room may arrive via another path

            pos[tid]      = (tc, tr)
            grid[(tc, tr)] = tid
            queue.append(tid)

    # Fallback: rooms unreachable via cardinal exits → row below main grid
    max_row = max((r for _, r in pos.values()), default=0)
    col_cursor = 0
    for room in rooms:
        if room["id"] not in pos:
            pos[room["id"]]               = (col_cursor, max_row + 3)
            grid[(col_cursor, max_row + 3)] = room["id"]
            col_cursor += 1

    # Normalize to (0,0) origin and convert to pixels
    min_col = min(c for c, _ in pos.values())
    min_row = min(r for _, r in pos.values())
    return {
        rid: {"x": (c - min_col) * cell, "y": (r - min_row) * cell}
        for rid, (c, r) in pos.items()
    }


# ── Fruchterman–Reingold spring layout (91 nodes, fast) ─────────────────────

def spring_layout(node_ids: set, links: list, weights: dict = None,
                  width: int = OV_W, height: int = OV_H,
                  iterations: int = SPRING_ITERS, seed: int = 42,
                  gravity: float = GRAVITY) -> dict:
    """
    Fruchterman–Reingold spring layout for the world-overview (91 zone nodes).
    Runs at build time — no browser physics needed.

    weights: {(min_id, max_id): count} — heavier edges pull zones tighter together.
    gravity: constant pull toward canvas center each iteration (produces globe shape).

    Returns: {node_id: {"x": float, "y": float}}
    """
    n = len(node_ids)
    if n == 0:
        return {}

    rng = random.Random(seed)
    xs  = {nid: rng.uniform(width * 0.15, width * 0.85)  for nid in node_ids}
    ys  = {nid: rng.uniform(height * 0.15, height * 0.85) for nid in node_ids}

    adj = {nid: set() for nid in node_ids}
    for s, t in links:
        if s in adj and t in adj:
            adj[s].add(t)
            adj[t].add(s)

    k      = math.sqrt(width * height / n) * 0.75
    temp   = width * 0.08
    cool   = temp / (iterations + 1)
    cx, cy = width / 2.0, height / 2.0

    node_list = list(node_ids)

    for _ in range(iterations):
        disp_x = {nid: 0.0 for nid in node_ids}
        disp_y = {nid: 0.0 for nid in node_ids}

        # Repulsion (all pairs — 91 nodes, O(n²) is fine)
        for i, u in enumerate(node_list):
            for v in node_list[i + 1:]:
                dx = xs[u] - xs[v]
                dy = ys[u] - ys[v]
                d  = math.hypot(dx, dy) + 0.01
                f  = k * k / d
                disp_x[u] += dx / d * f;  disp_x[v] -= dx / d * f
                disp_y[u] += dy / d * f;  disp_y[v] -= dy / d * f

        # Attraction along edges (weighted: more exits = stronger pull)
        for u in node_ids:
            for v in adj[u]:
                if v < u:
                    continue
                dx = xs[u] - xs[v]
                dy = ys[u] - ys[v]
                d  = math.hypot(dx, dy) + 0.01
                key = (min(u, v), max(u, v))
                w   = min((weights[key] / 3.0), WEIGHT_CAP) if (weights and key in weights) else 1.0
                f   = d * d / k * w
                disp_x[u] -= dx / d * f;  disp_x[v] += dx / d * f
                disp_y[u] -= dy / d * f;  disp_y[v] += dy / d * f

        # Center gravity — pulls all nodes toward canvas center, producing globe shape.
        # Isolated components orbit the periphery rather than flying off to corners.
        for nid in node_ids:
            disp_x[nid] += (cx - xs[nid]) * gravity
            disp_y[nid] += (cy - ys[nid]) * gravity

        # Apply displacements with temperature cooling
        for nid in node_ids:
            d = math.hypot(disp_x[nid], disp_y[nid]) + 0.01
            step = min(d, temp)
            xs[nid] += disp_x[nid] / d * step
            ys[nid] += disp_y[nid] / d * step
            xs[nid] = max(60.0, min(float(width  - 60), xs[nid]))
            ys[nid] = max(60.0, min(float(height - 60), ys[nid]))

        temp -= cool

    return {nid: {"x": round(xs[nid], 1), "y": round(ys[nid], 1)} for nid in node_ids}


# ── Main ─────────────────────────────────────────────────────────────────────

def main():
    print(f"Reading {WORLD_JSON} …")
    if not WORLD_JSON.exists():
        print(f"ERROR: {WORLD_JSON} not found. Run from repo root.", file=sys.stderr)
        sys.exit(1)

    with open(WORLD_JSON) as f:
        world = json.load(f)

    zones = world["zones"]
    total_rooms = sum(len(z["rooms"]) for z in zones)
    print(f"  {len(zones)} zones · {total_rooms} rooms total\n")

    OUT_DIR.mkdir(parents=True, exist_ok=True)

    # ── 1. Per-zone JSON with baked positions ────────────────────────────────
    zone_of_room   = {}  # room_id → zone_id (for inter-zone link detection)
    zone_centroids = {}  # zone_id → (cx, cy) in zone-local pixel space
    zone_positions = {}  # zone_id → {room_id: {"x": px, "y": py}}

    for zone in zones:
        rooms    = zone["rooms"]
        zone_id  = zone["id"]
        zone_of_room.update({r["id"]: zone_id for r in rooms})

        positions = bfs_layout(rooms)
        zone_positions[zone_id] = positions

        rooms_out = []
        for room in rooms:
            r   = {k: room[k] for k in ("id", "name", "desc", "sector", "exits")}
            p   = positions.get(room["id"], {"x": 0, "y": 0})
            r["x"] = p["x"]
            r["y"] = p["y"]
            rooms_out.append(r)

        # Intra-zone links (deduplicated undirected pairs)
        room_ids_set = {r["id"] for r in rooms}
        link_set     = set()
        links_out    = []
        for room in rooms:
            for ex in room.get("exits", []):
                t = ex.get("t")
                if t in room_ids_set:
                    a, b = min(room["id"], t), max(room["id"], t)
                    if (a, b) not in link_set:
                        link_set.add((a, b))
                        links_out.append({"s": a, "t": b})

        # Zone centroid in local pixel space
        if positions:
            xs_vals = [p["x"] for p in positions.values()]
            ys_vals = [p["y"] for p in positions.values()]
            zone_centroids[zone_id] = (
                sum(xs_vals) / len(xs_vals),
                sum(ys_vals) / len(ys_vals),
            )
        else:
            zone_centroids[zone_id] = (0.0, 0.0)

        out_file = OUT_DIR / f"zone-{zone_id}.json"
        with open(out_file, "w") as f:
            json.dump(
                {"id": zone_id, "name": zone["name"],
                 "rooms": rooms_out, "links": links_out},
                f, separators=(",", ":"),
            )
        kb = out_file.stat().st_size / 1024
        print(f"  zone-{zone_id:>4}.json  {kb:7.1f} KB  "
              f"{len(rooms_out):4d} rooms  {len(links_out):4d} links  "
              f"{zone['name'][:40]}")

    # ── 2. World overview: spring layout on 91 zone-centroid nodes ───────────
    print("\nComputing world-overview spring layout …")

    # Inter-zone links with exit counts — heavier corridors pull zones tighter
    inter_counts = {}
    for zone in zones:
        for room in zone["rooms"]:
            for ex in room.get("exits", []):
                t_zone = zone_of_room.get(ex.get("t"))
                if t_zone and t_zone != zone["id"]:
                    a, b = min(zone["id"], t_zone), max(zone["id"], t_zone)
                    inter_counts[(a, b)] = inter_counts.get((a, b), 0) + 1

    inter_links = list(inter_counts.keys())
    node_ids    = {z["id"] for z in zones}
    ov_positions = spring_layout(node_ids, inter_links, weights=inter_counts)

    overview_nodes = [
        {
            "id":    z["id"],
            "name":  z["name"],
            "rooms": len(z["rooms"]),
            "x":     ov_positions[z["id"]]["x"],
            "y":     ov_positions[z["id"]]["y"],
        }
        for z in zones
    ]
    ov_links_out = [{"s": a, "t": b} for a, b in inter_links]

    ov_file = OUT_DIR / "world-overview.json"
    with open(ov_file, "w") as f:
        json.dump({"nodes": overview_nodes, "links": ov_links_out},
                  f, separators=(",", ":"))
    print(f"\n  world-overview.json  {ov_file.stat().st_size / 1024:.1f} KB  "
          f"{len(overview_nodes)} zone nodes  {len(ov_links_out)} links")

    # ── 3. Index: zone list + room→zone lookup ───────────────────────────────
    # room_zones maps room_id (string key for JSON) → zone_id
    # Used by the browser for fast ?room=N deep-link resolution without
    # iterating through all zone files.
    room_zones = {str(rid): zid for rid, zid in zone_of_room.items()}

    # Named map-index.json (not index.json) to avoid conflict with Hugo's
    # generated section JSON output at /map/index.json.
    idx_file = OUT_DIR / "map-index.json"
    with open(idx_file, "w") as f:
        json.dump(
            {
                "zones":      [{"id": z["id"], "name": z["name"],
                                "rooms": len(z["rooms"])} for z in zones],
                "room_zones": room_zones,
            },
            f, separators=(",", ":"),
        )
    print(f"  map-index.json       {idx_file.stat().st_size / 1024:.1f} KB")

    # ── 4. World map: global room positions (stitched zones) ─────────────────
    # Each zone's local BFS positions are scaled to world units and translated
    # so the zone centroid lands at its spring-layout position.
    print("\nComputing world-map stitched layout …")

    TARGET_SIZE  = 400.0       # max zone bounding-box extent in world units
    GLOBAL_SCALE = 5.0 / CELL  # ≈ 0.132  (world units per local pixel)

    wm_rooms_out = []
    wm_links_out = []
    wm_link_set  = set()

    # All room IDs (for link target validation)
    all_room_ids = {r["id"] for z in zones for r in z["rooms"]}

    for zone in zones:
        zone_id   = zone["id"]
        rooms     = zone["rooms"]
        positions = zone_positions.get(zone_id, {})
        sx = ov_positions[zone_id]["x"]
        sy = ov_positions[zone_id]["y"]

        if not positions:
            for room in rooms:
                wm_rooms_out.append({
                    "id": room["id"], "x": round(sx), "y": round(sy),
                    "sector": room["sector"], "zone_id": zone_id,
                })
            continue

        xs_vals  = [p["x"] for p in positions.values()]
        ys_vals  = [p["y"] for p in positions.values()]
        bbox_w   = max(xs_vals) - min(xs_vals)
        bbox_h   = max(ys_vals) - min(ys_vals)
        cx_local = (max(xs_vals) + min(xs_vals)) / 2.0
        cy_local = (max(ys_vals) + min(ys_vals)) / 2.0
        max_dim  = max(bbox_w, bbox_h, float(CELL))
        scale    = min(TARGET_SIZE / max_dim, GLOBAL_SCALE)

        for room in rooms:
            p  = positions.get(room["id"], {"x": cx_local, "y": cy_local})
            gx = round((p["x"] - cx_local) * scale + sx)
            gy = round((p["y"] - cy_local) * scale + sy)
            wm_rooms_out.append({
                "id": room["id"], "x": gx, "y": gy,
                "sector": room["sector"], "zone_id": zone_id,
            })

    # Links: all room exits (all 6 directions), undirected dedup
    for zone in zones:
        for room in zone["rooms"]:
            for ex in room.get("exits", []):
                t = ex.get("t")
                if t not in all_room_ids:
                    continue
                a, b = min(room["id"], t), max(room["id"], t)
                if (a, b) not in wm_link_set:
                    wm_link_set.add((a, b))
                    wm_links_out.append({"s": a, "t": b})

    wm_file = OUT_DIR / "world-map.json"
    with open(wm_file, "w") as f:
        json.dump({"rooms": wm_rooms_out, "links": wm_links_out},
                  f, separators=(",", ":"))
    print(f"  world-map.json       {wm_file.stat().st_size / 1024:.1f} KB  "
          f"{len(wm_rooms_out)} rooms  {len(wm_links_out)} links")

    print("\nDone. Run hugo to rebuild the site.")


if __name__ == "__main__":
    main()
