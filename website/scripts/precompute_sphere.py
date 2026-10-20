#!/usr/bin/env python3
"""
Dark Pawns 3D Obsidian World — Volumetric 3D Force-Directed Precomputation
Places all 91 zone clusters and 9,590 rooms in a true 3D volumetric constellation
(with no hard circular clamp rings), mirror of website/scripts/precompute_graph.py
in R^3.

Output: website/static/map/world-sphere.json
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
    np.random.seed(1997)  # deterministic 3D constellation
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

    # ── STEP 2: 3D Volumetric Zone Centroid Layout ────────────────────────
    zone_radii = {zid: 9.0 * np.sqrt(len(zone_rooms[zid])) + 6.0 for zid in zone_ids}
    max_r = max(zone_radii.values())

    print("Running 3D Zone Centroid Layout (400 iterations)...")
    centroids = np.random.normal(size=(nz, 3)) * max_r * 1.6

    c_repel = 2.4e6     # pairwise repulsion in 3D so clusters have natural voids
    c_attract = 0.0045  # pull along inter-zone connections
    c_gravity = 0.0025  # gentle center pull
    dt = 0.05

    for iteration in range(400):
        temp = 1.0 - iteration / 400.0
        diffs = centroids[:, np.newaxis, :] - centroids[np.newaxis, :, :]
        dists = np.linalg.norm(diffs, axis=2)
        np.fill_diagonal(dists, 1.0)

        repel_mag = c_repel / (dists ** 2 + 1.0)
        np.fill_diagonal(repel_mag, 0)
        repel_forces = np.sum(repel_mag[:, :, np.newaxis] * (diffs / dists[:, :, np.newaxis]), axis=1)

        attract_mag = c_attract * dists * zone_adj
        attract_forces = np.sum(attract_mag[:, :, np.newaxis] * (-diffs / dists[:, :, np.newaxis]), axis=1)

        gravity_forces = -centroids * c_gravity
        centroids += (repel_forces + attract_forces + gravity_forces) * dt * temp

    print("3D Zone Centroids complete.")

    # ── STEP 3: 3D Room Layout inside zone clusters (NO hard circular clamp!) ──
    print("Running 3D Room Layout (250 iterations)...")
    nr = len(rooms_list)
    r_to_idx = {int(r["id"]): i for i, r in enumerate(rooms_list)}

    pos = np.zeros((nr, 3))
    room_zone_idx = np.zeros(nr, dtype=int)
    for i, r in enumerate(rooms_list):
        zid = int(r.get("zone_id", 0))
        zi = z_to_idx[zid]
        room_zone_idx[i] = zi
        pos[i] = centroids[zi] + np.random.normal(scale=zone_radii[zid] * 0.25, size=3)

    s_arr = np.array([r_to_idx[int(lk["s"])] for lk in links_list
                      if int(lk["s"]) in r_to_idx and int(lk["t"]) in r_to_idx])
    t_arr = np.array([r_to_idx[int(lk["t"])] for lk in links_list
                      if int(lk["s"]) in r_to_idx and int(lk["t"]) in r_to_idx])

    c_room_repel = 950.0     # 3D repulsion inside each zone
    c_room_attract = 0.055   # spring along exit edges
    c_cohesion = 0.024       # soft spring toward zone centroid (no hard boundary!)
    dt_room = 0.04

    for iteration in range(250):
        temp = 1.0 - iteration / 250.0
        forces = np.zeros((nr, 3))

        # Edge springs
        diff = pos[t_arr] - pos[s_arr]
        dists = np.linalg.norm(diff, axis=1, keepdims=True)
        dists = np.maximum(dists, 1.0)
        att = c_room_attract * dists * (diff / dists)
        np.add.at(forces, s_arr, att)
        np.add.at(forces, t_arr, -att)

        # Intra-zone 3D repulsion
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

        # Soft zone cohesion spring
        cvec = centroids[room_zone_idx] - pos
        forces += c_cohesion * cvec

        pos += forces * dt_room * temp

    print("3D Room Layout complete.")

    # ── STEP 4: Recompute true zone centroids from final room positions & normalize ──
    for i, zid in enumerate(zone_ids):
        ri = [r_to_idx[rid] for rid in zone_rooms[zid]]
        if ri:
            centroids[i] = pos[ri].mean(axis=0)

    # Center at the weighted center of mass of all rooms, then normalize radius
    # using the 98th percentile distance so compact core clusters fill the 3D view nicely.
    center = pos.mean(axis=0)
    pos -= center
    centroids -= center
    radii = np.linalg.norm(pos, axis=1)
    p98 = float(np.percentile(radii, 98))
    scale = 1.0 / max(p98, 1.0)
    pos *= scale
    centroids *= scale

    print(f"Normalized 3D volume: 98th-percentile radius={p98:.1f} -> 1.0 (max={np.linalg.norm(pos, axis=1).max():.2f})")

    # ── STEP 5: Emit ──────────────────────────────────────────────────────
    output = {
        "rooms": [
            {
                "id": int(r["id"]),
                "x": round(float(pos[i, 0]), 5),
                "y": round(float(pos[i, 1]), 5),
                "z": round(float(pos[i, 2]), 5),
                "sector": int(r.get("sector", 0)),
                "zone": int(r.get("zone_id", 0)),
            }
            for i, r in enumerate(rooms_list)
        ],
        "zones": [
            {
                "id": zid,
                "name": zone_names.get(zid, f"Zone {zid}"),
                "x": round(float(centroids[i, 0]), 5),
                "y": round(float(centroids[i, 1]), 5),
                "z": round(float(centroids[i, 2]), 5),
                "count": len(zone_rooms[zid]),
            }
            for i, zid in enumerate(zone_ids)
        ],
    }

    out_path = "website/static/map/world-sphere.json"
    print(f"Writing {out_path}...")
    with open(out_path, "w") as f:
        json.dump(output, f, separators=(",", ":"))
    print("3D Volumetric Obsidian World precomputation completed successfully!")


if __name__ == "__main__":
    run_precompute()
