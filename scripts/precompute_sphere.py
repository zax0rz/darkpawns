#!/usr/bin/env python3
"""
Force-directed layout for Dark Pawns world graph.
Phase 1: Force-direct zone centroids on 2D plane.
Phase 2: Place rooms in clusters around zone centroids, then project to sphere.

Input:  website/static/map/world-map.json
Output: website/static/map/world-sphere.json
"""

import json, os, math, random
from collections import defaultdict, Counter

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
REPO_DIR = os.path.dirname(SCRIPT_DIR)
WORLD_JSON = os.path.join(REPO_DIR, "website/static/map/world-map.json")
OVERVIEW_JSON = os.path.join(REPO_DIR, "website/static/map/world-overview.json")
OUT_JSON = os.path.join(REPO_DIR, "website/static/map/world-sphere.json")

REPEL = 750.0
ATTRACT = 0.08
CENTER = 0.002


def main():
    with open(WORLD_JSON) as f:
        world = json.load(f)
    with open(OVERVIEW_JSON) as f:
        overview = json.load(f)

    rooms = world['rooms']
    links = world['links']
    sector_map = {r['id']: r.get('sector', 0) for r in rooms}
    zone_id_map = {r['id']: r.get('zone_id', 0) for r in rooms}
    zone_names = {z['id']: z['name'] for z in overview['nodes']}

    zone_rooms = defaultdict(list)
    for r in rooms:
        zone_rooms[r.get('zone_id', 0)].append(r['id'])

    zone_ids = sorted(zone_rooms.keys())
    zi = {zid: i for i, zid in enumerate(zone_ids)}
    nz = len(zone_ids)

    zone_adj = defaultdict(set)
    for lk in overview['links']:
        zone_adj[lk['s']].add(lk['t'])
        zone_adj[lk['t']].add(lk['s'])

    # ── Phase 1: Force-direct zone centroids ───────────────────────────
    print(f"Phase 1: {nz} zone centroids...")
    zx = [math.cos(2*math.pi*i/nz) * 1000 for i in range(nz)]
    zy = [math.sin(2*math.pi*i/nz) * 1000 for i in range(nz)]

    for it in range(400):
        cooling = 0.5 * (1 - it/400)
        fx, fy = [0.0]*nz, [0.0]*nz

        for i in range(nz):
            for j in range(i+1, nz):
                dx, dy = zx[i]-zx[j], zy[i]-zy[j]
                d = max(math.hypot(dx, dy), 1)
                f = REPEL / (d*d)
                fx[i] += dx/d*f; fy[i] += dy/d*f
                fx[j] -= dx/d*f; fy[j] -= dy/d*f

        for z1, conns in zone_adj.items():
            if z1 not in zi: continue
            i = zi[z1]
            for z2 in conns:
                if z2 not in zi: continue
                j = zi[z2]
                dx, dy = zx[j]-zx[i], zy[j]-zy[i]
                d = max(math.hypot(dx, dy), 1)
                f = ATTRACT * d
                fx[i] += dx/d*f; fy[i] += dy/d*f
                fx[j] -= dx/d*f; fy[j] -= dy/d*f

        for i in range(nz):
            fx[i] -= zx[i]*CENTER
            fy[i] -= zy[i]*CENTER

        for i in range(nz):
            zx[i] += fx[i]*cooling
            zy[i] += fy[i]*cooling

    # ── Phase 2: Scatter rooms around zone centroids ───────────────────
    print("Phase 2: Placing rooms...")

    # Build intra-zone link map for room placement
    room_links = defaultdict(set)
    for lk in links:
        if zone_id_map.get(lk['s']) == zone_id_map.get(lk['t']):
            room_links[lk['s']].add(lk['t'])
            room_links[lk['t']].add(lk['s'])

    room_pos = {}
    for zid, rids in zone_rooms.items():
        if zid not in zi:
            continue
        ci = zi[zid]
        cx, cy = zx[ci], zy[ci]
        n = len(rids)

        # Give each zone a radius proportional to room count
        zone_radius = 20 + math.sqrt(n) * 6

        # Place rooms on a spiral within the zone disc
        for idx, rid in enumerate(rids):
            # Golden angle spiral for even distribution
            angle = idx * 2.399963  # golden angle
            r = zone_radius * math.sqrt(idx / max(n, 1))
            room_pos[rid] = (
                cx + math.cos(angle) * r,
                cy + math.sin(angle) * r,
            )

    # ── Phase 3: Project to sphere ─────────────────────────────────────
    print("Projecting to sphere...")

    xs = [p[0] for p in room_pos.values()]
    ys = [p[1] for p in room_pos.values()]
    minx, maxx = min(xs), max(xs)
    miny, maxy = min(ys), max(ys)
    bcx, bcy = (minx+maxx)/2, (miny+maxy)/2
    scale = max(maxx-minx, maxy-miny) / 2 * 1.05  # slight padding

    def to_sphere(x, y):
        nx = (x - bcx) / scale
        ny = (y - bcy) / scale
        r = min(math.hypot(nx, ny), 1.0)
        if r < 0.0001:
            return (0.0, 0.0, 1.0)
        theta = math.atan2(ny, nx)
        phi = r * (math.pi / 2 - 0.05)  # don't go quite to poles
        return (
            round(math.cos(phi)*math.cos(theta), 6),
            round(math.cos(phi)*math.sin(theta), 6),
            round(math.sin(phi), 6),
        )

    out_rooms = []
    for rid, (x, y) in room_pos.items():
        sx, sy, sz = to_sphere(x, y)
        out_rooms.append({
            "id": rid, "x": sx, "y": sy, "z": sz,
            "sector": sector_map.get(rid, 0),
            "zone": zone_id_map.get(rid, 0),
        })

    out_zones = []
    for zid in zone_ids:
        if zid not in zi: continue
        ci = zi[zid]
        sx, sy, sz = to_sphere(zx[ci], zy[ci])
        secs = [sector_map.get(rid, 0) for rid in zone_rooms[zid]]
        dominant = Counter(secs).most_common(1)[0][0] if secs else 0
        out_zones.append({
            "id": zid, "name": zone_names.get(zid, f"Zone {zid}"),
            "x": sx, "y": sy, "z": sz,
            "sector": dominant, "count": len(zone_rooms[zid]),
        })

    with open(OUT_JSON, "w") as f:
        json.dump({"rooms": out_rooms, "zones": out_zones}, f)

    # Verify clustering
    room_sphere = {r['id']: r for r in out_rooms}
    for zid in [30, 49, 80, 110]:
        zrooms = [room_sphere[rid] for rid in zone_rooms.get(zid, []) if rid in room_sphere]
        if not zrooms: continue
        cx = sum(r['x'] for r in zrooms)/len(zrooms)
        cy = sum(r['y'] for r in zrooms)/len(zrooms)
        cz = sum(r['z'] for r in zrooms)/len(zrooms)
        dists = [math.hypot(r['x']-cx, r['y']-cy, r['z']-cz) for r in zrooms]
        print(f"  Zone {zid}: {len(zrooms)} rooms, spread={sum(dists)/len(dists):.4f}")

    print(f"\nDone! {len(out_rooms)} rooms → {OUT_JSON}")


if __name__ == "__main__":
    random.seed(42)
    main()
