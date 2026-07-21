#!/usr/bin/env python3
"""
Dark Pawns Graph — Planar Obsidian-style Layout Precomputation
Force-directs 9,590 rooms across a 2D plane: zone clusters as luminous
constellations with natural voids between them, every edge drawable.

Unlike the sphere layout (precompute_sphere.py), a plane has unlimited
room — zone patches never have to overlap, so no clamps and no rim
artifacts. Output: website/static/map/world-graph.json
"""

import json
import os
import sys
import numpy as np

def load_world_map():
    map_path = "website/static/map/world-map.json"
    index_path = "website/static/map/map-index.json"

    if not os.path.exists(map_path):
        print(f"Error: {map_path} not found. Are you running from the repo root?")
        sys.exit(1)

    print(f"Loading {map_path}...")
    with open(map_path, "r") as f:
        world_data = json.load(f)

    zone_names = {}
    if os.path.exists(index_path):
        with open(index_path, "r") as f:
            idx_data = json.load(f)
            for z in idx_data.get("zones", []):
                zone_names[int(z["id"])] = z["name"]

    return world_data, zone_names


def run_precompute():
    np.random.seed(1997)  # deterministic graph — same layout on every build
    world_data, zone_names = load_world_map()

    rooms_list = world_data["rooms"]
    links_list = world_data["links"]
    print(f"Total Rooms: {len(rooms_list)}  Total Links: {len(links_list)}")

    room_map = {int(r["id"]): r for r in rooms_list}
    zone_rooms = {}
    for r in rooms_list:
        zone_rooms.setdefault(int(r.get("zone_id", 0)), []).append(int(r["id"]))
    zone_ids = sorted(zone_rooms.keys())
    nz = len(zone_ids)
    z_to_idx = {zid: i for i, zid in enumerate(zone_ids)}
    print(f"Total Zones: {nz}")

    # ── STEP 1: Zone adjacency ────────────────────────────────────────────
    zone_adj = np.zeros((nz, nz))
    for lk in links_list:
        s_room = room_map.get(int(lk["s"]))
        t_room = room_map.get(int(lk["t"]))
        if s_room and t_room:
            sz, tz = int(s_room.get("zone_id", 0)), int(t_room.get("zone_id", 0))
            if sz != tz:
                zone_adj[z_to_idx[sz], z_to_idx[tz]] += 1.5
                zone_adj[z_to_idx[tz], z_to_idx[sz]] += 1.5

    # ── STEP 2: Zone centroid layout on the plane ─────────────────────────
    # Patch radius per zone (world units) — drives how far centroids must sit
    # apart so patches never overlap.
    zone_radii = {zid: 9.0 * np.sqrt(len(zone_rooms[zid])) + 6.0 for zid in zone_ids}
    max_r = max(zone_radii.values())
    print(f"Largest patch radius: {max_r:.0f} units")

    print("Running Zone Centroid Layout (400 iterations)...")
    centroids = np.random.normal(size=(nz, 2)) * max_r * 2.0

    c_repel = 2.0e6     # strong pairwise repulsion — patches keep their voids
    c_attract = 0.004   # gentle pull along inter-zone links
    c_gravity = 0.002   # faint center gravity so the graph stays compact
    dt = 0.05

    for iteration in range(400):
        temp = 1.0 - iteration / 400.0
        diffs = centroids[:, np.newaxis, :] - centroids[np.newaxis, :, :]
        dists = np.linalg.norm(diffs, axis=2)
        np.fill_diagonal(dists, 1.0)

        # Repulsion — but only meaningful within a few patch radii
        repel_mag = c_repel / (dists ** 2 + 1.0)
        np.fill_diagonal(repel_mag, 0)
        repel_forces = np.sum(repel_mag[:, :, np.newaxis] * (diffs / dists[:, :, np.newaxis]), axis=1)

        attract_mag = c_attract * dists * zone_adj
        attract_forces = np.sum(attract_mag[:, :, np.newaxis] * (-diffs / dists[:, :, np.newaxis]), axis=1)

        gravity_forces = -centroids * c_gravity

        centroids += (repel_forces + attract_forces + gravity_forces) * dt * temp

    print("Zone Centroids complete.")

    # ── STEP 3: Room layout inside patches ────────────────────────────────
    print("Running Room Layout (250 iterations)...")
    nr = len(rooms_list)
    r_to_idx = {int(r["id"]): i for i, r in enumerate(rooms_list)}

    pos = np.zeros((nr, 2))
    room_zone_idx = np.zeros(nr, dtype=int)
    for i, r in enumerate(rooms_list):
        zid = int(r.get("zone_id", 0))
        zi = z_to_idx[zid]
        room_zone_idx[i] = zi
        pos[i] = centroids[zi] + np.random.normal(scale=zone_radii[zid] * 0.25, size=2)

    s_arr = np.array([r_to_idx[int(lk["s"])] for lk in links_list
                      if int(lk["s"]) in r_to_idx and int(lk["t"]) in r_to_idx])
    t_arr = np.array([r_to_idx[int(lk["t"])] for lk in links_list
                      if int(lk["s"]) in r_to_idx and int(lk["t"]) in r_to_idx])

    c_room_repel = 900.0     # pairwise repulsion inside a zone (units²)
    c_room_attract = 0.05    # spring along edges
    c_cohesion = 0.02        # spring toward own zone centroid (keeps patches tight)
    dt_room = 0.04

    for iteration in range(250):
        temp = 1.0 - iteration / 250.0
        forces = np.zeros((nr, 2))

        # Edge springs
        diff = pos[t_arr] - pos[s_arr]
        dists = np.linalg.norm(diff, axis=1, keepdims=True)
        dists = np.maximum(dists, 1.0)
        att = c_room_attract * dists * (diff / dists)
        np.add.at(forces, s_arr, att)
        np.add.at(forces, t_arr, -att)

        # Intra-zone repulsion
        for zid, rids in zone_rooms.items():
            if len(rids) < 2:
                continue
            ri = [r_to_idx[rid] for rid in rids]
            zp = pos[ri]
            dz = zp[:, np.newaxis, :] - zp[np.newaxis, :, :]
            dd = np.linalg.norm(dz, axis=2)
            np.fill_diagonal(dd, 1.0)
            rm = c_room_repel / (dd ** 2 + 1.0)
            np.fill_diagonal(rm, 0)
            forces[ri] += np.sum(rm[:, :, np.newaxis] * (dz / dd[:, :, np.newaxis]), axis=1)

        # Zone cohesion — quadratic spring toward the centroid, so outliers
        # feel increasing pull (no hard boundary, no rim pile-up)
        cvec = centroids[room_zone_idx] - pos
        forces += c_cohesion * cvec

        pos += forces * dt_room * temp

    print("Room Layout complete.")

    # ── STEP 4: Normalize to a stable world frame ─────────────────────────
    # Center on the densest zone (the world's hub) at (0,0), scale so the
    # graph spans roughly [-1000, 1000].
    hub_idx = max(range(nz), key=lambda i: len(zone_rooms[zone_ids[i]]))
    pos -= centroids[hub_idx]
    centroids -= centroids[hub_idx]
    span = np.abs(pos).max()
    scale = 1000.0 / span
    pos *= scale
    centroids *= scale
    print(f"Normalized: span {span:.0f} -> 1000 units around zone {zone_ids[hub_idx]}")

    # ── STEP 5: Emit ──────────────────────────────────────────────────────
    output = {
        "rooms": [
            {"id": int(r["id"]), "x": float(pos[i, 0]), "y": float(pos[i, 1]),
             "sector": int(r.get("sector", 0)), "zone": int(r.get("zone_id", 0))}
            for i, r in enumerate(rooms_list)
        ],
        "zones": [
            {"id": zid, "name": zone_names.get(zid, f"Zone {zid}"),
             "x": float(centroids[i, 0]), "y": float(centroids[i, 1]),
             "count": len(zone_rooms[zid])}
            for i, zid in enumerate(zone_ids)
        ],
    }

    out_path = "website/static/map/world-graph.json"
    print(f"Writing {out_path}...")
    with open(out_path, "w") as f:
        json.dump(output, f)
    print("Graph precomputation completed successfully!")


if __name__ == "__main__":
    run_precompute()
