#!/usr/bin/env python3
"""
Dark Pawns Globe — Sphere Precomputation Script (Tuned for Starburst Layout)
Force-directs 9,590 rooms and ~93 zones onto a 3D unit sphere.
Mathematical tweaks to match the organic, dendritic starburst of the Obsidian Graph View.
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
        print(f"Loading zone names from {index_path}...")
        with open(index_path, "r") as f:
            idx_data = json.load(f)
            for z in idx_data.get("zones", []):
                zone_names[int(z["id"])] = z["name"]
                
    return world_data, zone_names

def run_precompute():
    world_data, zone_names = load_world_map()
    
    rooms_list = world_data["rooms"]
    links_list = world_data["links"]
    
    print(f"Total Rooms: {len(rooms_list)}")
    print(f"Total Links: {len(links_list)}")
    
    # 1. Map rooms and extract zone groupings
    room_map = {int(r["id"]): r for r in rooms_list}
    zone_rooms = {}
    for r in rooms_list:
        zid = int(r.get("zone_id", 0))
        zone_rooms.setdefault(zid, []).append(int(r["id"]))
        
    zone_ids = sorted(list(zone_rooms.keys()))
    nz = len(zone_ids)
    print(f"Total Zones: {nz}")
    
    # Zone index mapping
    z_to_idx = {zid: idx for idx, zid in enumerate(zone_ids)}
    
    # ── STEP 1: Compute Zone Adjacency & Centroid Sizing ──
    zone_adj = np.zeros((nz, nz))
    for lk in links_list:
        s_room = room_map.get(int(lk["s"]))
        t_room = room_map.get(int(lk["t"]))
        if s_room and t_room:
            sz = int(s_room.get("zone_id", 0))
            tz = int(t_room.get("zone_id", 0))
            if sz != tz:
                s_idx = z_to_idx[sz]
                t_idx = z_to_idx[tz]
                zone_adj[s_idx, t_idx] += 1.5  # Boost inter-zone attraction weight
                zone_adj[t_idx, s_idx] += 1.5
                
    # ── STEP 2: Zone-Level Force-Directed Layout on Sphere ──
    print("\nRunning Zone Centroid Layout (250 iterations)...")
    # Initialize random positions on the unit sphere
    centroids = np.random.normal(size=(nz, 3))
    centroids /= np.linalg.norm(centroids, axis=1, keepdims=True)
    
    # Force coefficients - tuned for wider global distribution
    c_repel = 0.08
    c_attract = 0.22
    c_gravity = 0.015  # Core gravity to lock isolated components
    dt = 0.06
    
    for iteration in range(250):
        # Temp cooling
        temp = 1.0 - (iteration / 250.0)
        
        # Pairwise differences
        diffs = centroids[:, np.newaxis, :] - centroids[np.newaxis, :, :]  # shape (nz, nz, 3)
        dists = np.linalg.norm(diffs, axis=2)  # shape (nz, nz)
        np.fill_diagonal(dists, 1.0)  # avoid division by zero
        
        # Repulsion: repel every zone from every other zone
        repel_mag = c_repel / (dists**2 + 1e-4)
        np.fill_diagonal(repel_mag, 0)
        repel_forces = np.sum(repel_mag[:, :, np.newaxis] * (diffs / dists[:, :, np.newaxis]), axis=1)
        
        # Attraction: pull connected zones closer
        attract_mag = c_attract * dists * zone_adj
        attract_forces = np.sum(attract_mag[:, :, np.newaxis] * (-diffs / dists[:, :, np.newaxis]), axis=1)
        
        # Core central gravity
        gravity_forces = -centroids * c_gravity
        
        # Step centroids
        centroids += (repel_forces + attract_forces + gravity_forces) * dt * temp
        centroids /= np.linalg.norm(centroids, axis=1, keepdims=True)
        
    print("Zone Centroids Layout complete.")
    
    # ── STEP 3: Relaxed Circular Patch Allocation (Obsidian Starburst tweak) ──
    # We increase the patch boundaries (up to 0.65 rad ~37°) so rooms can expand,
    # repel, and form elegant dendritic fibers rather than flat compressed bars.
    zone_radii = {}
    for idx, zid in enumerate(zone_ids):
        count = len(zone_rooms[zid])
        # Relaxed bounds: min 0.12, max 0.65 radians
        theta = min(0.65, max(0.12, 0.055 * np.sqrt(count)))
        zone_radii[zid] = theta
        
    # ── STEP 4: Global/Local Room Layout in Patches (Holes 2 & 5) ──
    print("\nRunning Global/Local Room Layout (200 iterations)...")
    nr = len(rooms_list)
    r_to_idx = {int(r["id"]): idx for idx, r in enumerate(rooms_list)}
    
    # Initialize room positions at their parent zone centroid + tiny random offset
    pos = np.zeros((nr, 3))
    for idx, r in enumerate(rooms_list):
        zid = int(r.get("zone_id", 0))
        z_idx = z_to_idx[zid]
        pos[idx] = centroids[z_idx] + np.random.normal(scale=0.02, size=3)
    pos /= np.linalg.norm(pos, axis=1, keepdims=True)
    
    # Pre-map connections as indices
    s_indices = []
    t_indices = []
    for lk in links_list:
        s_id = int(lk["s"])
        t_id = int(lk["t"])
        if s_id in r_to_idx and t_id in r_to_idx:
            s_indices.append(r_to_idx[s_id])
            t_indices.append(r_to_idx[t_id])
                
    s_arr = np.array(s_indices)
    t_arr = np.array(t_indices)
    
    # Tuned room layout physics coefficients
    # Increased repulsion and attraction forces to create strong hubs and fine dendritic branches
    c_room_repel = 0.008   # Boosted to expand compressed lines into starbursts
    c_room_attract = 0.18 # Boosted to pull exit pathways into tight, readable fibers
    dt_room = 0.05
    
    for iteration in range(200):
        temp = 1.0 - (iteration / 200.0)
        forces = np.zeros((nr, 3))
        
        # 4a. Attraction: Exits pull connected rooms together
        diff = pos[t_arr] - pos[s_arr]
        dists = np.linalg.norm(diff, axis=1, keepdims=True)
        dists = np.maximum(dists, 1e-4)
        
        # Attract along exit edges
        att_force = c_room_attract * dists * (diff / dists)
        np.add.at(forces, s_arr, att_force)
        np.add.at(forces, t_arr, -att_force)
        
        # 4b. Local Repulsion: Rooms in the SAME zone repel each other strongly
        for zid, rids in zone_rooms.items():
            if len(rids) < 2:
                continue
            r_indices = [r_to_idx[rid] for rid in rids]
            z_pos = pos[r_indices]
            
            diffs_z = z_pos[:, np.newaxis, :] - z_pos[np.newaxis, :, :]  # (M, M, 3)
            dists_z = np.linalg.norm(diffs_z, axis=2)  # (M, M)
            np.fill_diagonal(dists_z, 1.0)
            
            rep_mag = c_room_repel / (dists_z**2 + 1e-4)
            np.fill_diagonal(rep_mag, 0)
            
            rep_force = np.sum(rep_mag[:, :, np.newaxis] * (diffs_z / dists_z[:, :, np.newaxis]), axis=1)
            forces[r_indices] += rep_force
            
        # 4c. Apply Forces
        pos += forces * dt_room * temp
        pos /= np.linalg.norm(pos, axis=1, keepdims=True)
        
        # 4d. Strict Patch Constraining
        for idx, r in enumerate(rooms_list):
            zid = int(r.get("zone_id", 0))
            z_idx = z_to_idx[zid]
            z_centroid = centroids[z_idx]
            theta_max = zone_radii[zid]
            
            r_pos = pos[idx]
            dot = np.dot(r_pos, z_centroid)
            cos_max = np.cos(theta_max)
            
            if dot < cos_max:
                parallel_comp = dot * z_centroid
                perp_comp = r_pos - parallel_comp
                perp_norm = np.linalg.norm(perp_comp)
                if perp_norm > 1e-6:
                    perp_dir = perp_comp / perp_norm
                    sin_max = np.sin(theta_max)
                    pos[idx] = cos_max * z_centroid + sin_max * perp_dir
                    pos[idx] /= np.linalg.norm(pos[idx])
                    
    print("Room Layout complete.")
    
    # ── STEP 5: Collision Resolution (Centroid Spacing) ──
    print("\nVerifying centroid spacing...")
    centroid_dists = []
    for i in range(nz):
        for j in range(i+1, nz):
            dist = np.arccos(np.clip(np.dot(centroids[i], centroids[j]), -1.0, 1.0))
            centroid_dists.append(dist)
    print(f"Minimum distance: {np.min(centroid_dists):.4f} rad, Average: {np.mean(centroid_dists):.4f} rad")
    
    # ── STEP 6: Compile Output JSON ──
    output_rooms = []
    for idx, r in enumerate(rooms_list):
        output_rooms.append({
            "id": int(r["id"]),
            "x": float(pos[idx, 0]),
            "y": float(pos[idx, 1]),
            "z": float(pos[idx, 2]),
            "sector": int(r.get("sector", 0)),
            "zone": int(r.get("zone_id", 0))
        })
        
    output_zones = []
    for idx, zid in enumerate(zone_ids):
        zname = zone_names.get(zid, f"Zone {zid}")
        output_zones.append({
            "id": zid,
            "name": zname,
            "x": float(centroids[idx, 0]),
            "y": float(centroids[idx, 1]),
            "z": float(centroids[idx, 2]),
            "count": len(zone_rooms[zid])
        })
        
    output_data = {
        "rooms": output_rooms,
        "zones": output_zones
    }
    
    output_path = "website/static/map/world-sphere.json"
    print(f"\nWriting results to {output_path}...")
    with open(output_path, "w") as f:
        json.dump(output_data, f, indent=2)
    print("Spherical map precomputation completed successfully!")

if __name__ == "__main__":
    run_precompute()
