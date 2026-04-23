#!/usr/bin/env python3
"""
Parse Dark Pawns CircleMUD world files and export room/zone data as JSON.
"""

import json
import os
import re
import sys
from collections import defaultdict

WLD_DIR = "/home/zach/.openclaw/workspace/rparet-darkpawns/lib/world/wld"
ZON_DIR = "/home/zach/.openclaw/workspace/rparet-darkpawns/lib/world/zon"
OUT_FILE = "/home/zach/.openclaw/workspace/darkpawns/maps/world_data.json"


def parse_wld_file(path):
    """Parse a single .wld file and return list of room dicts."""
    rooms = []
    with open(path, "r", encoding="utf-8", errors="replace") as f:
        content = f.read()

    # Split by room markers like #123
    # Each room starts with #<vnum> and ends with S or next #
    pattern = r'#(\d+)\n(.*?)\nS'
    matches = re.findall(pattern, content, re.DOTALL)

    for vnum_str, body in matches:
        vnum = int(vnum_str)
        lines = body.split('\n')
        if not lines:
            continue

        # Name is first line, ends with ~
        name = lines[0].rstrip('~')

        # Description: collect lines until one ends with ~
        desc_lines = []
        idx = 1
        while idx < len(lines):
            line = lines[idx]
            if line.endswith('~'):
                desc_lines.append(line.rstrip('~'))
                idx += 1
                break
            desc_lines.append(line)
            idx += 1
        description = '\n'.join(desc_lines)

        # Numeric line: zone flags sector 0 0 0
        zone = 0
        sector = 0
        if idx < len(lines):
            nums = lines[idx].strip().split()
            if len(nums) >= 3:
                zone = int(nums[0])
                sector = int(nums[2])
            idx += 1

        # Parse exits
        exits = {}
        dir_map = {'D0': 'north', 'D1': 'east', 'D2': 'south', 'D3': 'west', 'D4': 'up', 'D5': 'down'}
        while idx < len(lines):
            line = lines[idx].strip()
            if line.startswith('D') and len(line) == 2 and line in dir_map:
                direction = dir_map[line]
                idx += 1
                # Exit description (may span multiple lines, ends with ~)
                exit_desc_lines = []
                while idx < len(lines):
                    dl = lines[idx]
                    if dl.endswith('~'):
                        exit_desc_lines.append(dl.rstrip('~'))
                        idx += 1
                        break
                    exit_desc_lines.append(dl)
                    idx += 1
                exit_desc = '\n'.join(exit_desc_lines)

                # Exit keywords (may span multiple lines, ends with ~)
                exit_kw_lines = []
                while idx < len(lines):
                    kl = lines[idx]
                    if kl.endswith('~'):
                        exit_kw_lines.append(kl.rstrip('~'))
                        idx += 1
                        break
                    exit_kw_lines.append(kl)
                    idx += 1
                exit_keywords = '\n'.join(exit_kw_lines)

                # Numeric line: door_state key to_room
                to_room = -1
                door_state = 0
                key_vnum = -1
                if idx < len(lines):
                    nums = lines[idx].strip().split()
                    if len(nums) >= 3:
                        # Validate all three fields are integers
                        try:
                            door_state = int(nums[0])
                            key_vnum = int(nums[1])
                            to_room = int(nums[2])
                        except ValueError:
                            # Malformed numeric line — skip this exit
                            door_state = 0
                            key_vnum = -1
                            to_room = -1
                    idx += 1
                exits[direction] = {
                    'to_room': to_room,
                    'door_state': door_state,
                    'key': key_vnum,
                    'description': exit_desc,
                    'keywords': exit_keywords
                }
            elif line == 'E':
                # Extra description, skip
                idx += 1
                # keywords
                while idx < len(lines):
                    if lines[idx].endswith('~'):
                        idx += 1
                        break
                    idx += 1
                # description
                while idx < len(lines):
                    if lines[idx].endswith('~'):
                        idx += 1
                        break
                    idx += 1
            else:
                idx += 1

        rooms.append({
            'vnum': vnum,
            'name': name,
            'description': description,
            'zone': zone,
            'sector': sector,
            'exits': exits,
            'source_file': os.path.basename(path)
        })

    return rooms


def parse_zon_file(path):
    """Parse a single .zon file and return zone info."""
    with open(path, "r", encoding="utf-8", errors="replace") as f:
        content = f.read()

    lines = content.strip().split('\n')
    if not lines:
        return None

    # First line: #<number>
    if not lines[0].startswith('#'):
        return None
    number = int(lines[0][1:])

    # Second line: zone name~
    name = lines[1].rstrip('~') if len(lines) > 1 else "Unknown"

    # Third line: top_room lifespan reset_mode 0 0
    top_room = 0
    lifespan = 30
    reset_mode = 0
    if len(lines) > 2:
        nums = lines[2].strip().split()
        if len(nums) >= 3:
            top_room = int(nums[0])
            lifespan = int(nums[1])
            reset_mode = int(nums[2])

    return {
        'number': number,
        'name': name,
        'top_room': top_room,
        'lifespan': lifespan,
        'reset_mode': reset_mode,
        'source_file': os.path.basename(path)
    }


def main():
    all_rooms = []
    all_zones = []

    # Parse all .wld files
    wld_files = sorted([f for f in os.listdir(WLD_DIR) if f.endswith('.wld')])
    for fname in wld_files:
        path = os.path.join(WLD_DIR, fname)
        rooms = parse_wld_file(path)
        all_rooms.extend(rooms)
        print(f"  {fname}: {len(rooms)} rooms")

    # Parse all .zon files
    zon_files = sorted([f for f in os.listdir(ZON_DIR) if f.endswith('.zon')])
    for fname in zon_files:
        path = os.path.join(ZON_DIR, fname)
        zone = parse_zon_file(path)
        if zone:
            all_zones.append(zone)

    # Build zone lookup
    zone_map = {z['number']: z for z in all_zones}

    # Add zone names to rooms
    for room in all_rooms:
        znum = room['zone']
        if znum in zone_map:
            room['zone_name'] = zone_map[znum]['name']
        else:
            room['zone_name'] = f"Zone {znum}"

    # Build exit graph for coordinate assignment
    # Only include exits that point to valid rooms
    room_vnums = {r['vnum'] for r in all_rooms}

    # Assign coordinates via BFS from multiple seeds
    coords = {}  # vnum -> (x, y, z)
    visited = set()

    # Group rooms by zone for better layout
    zone_rooms = defaultdict(list)
    for r in all_rooms:
        zone_rooms[r['zone']].append(r)

    # Sort zones and assign base offsets
    sorted_zones = sorted(zone_rooms.keys())
    zone_offsets = {}
    grid_cols = 8
    for i, znum in enumerate(sorted_zones):
        row = i // grid_cols
        col = i % grid_cols
        zone_offsets[znum] = (col * 80, row * 40, 0)

    # BFS within each zone
    dir_deltas = {
        'north': (0, -1, 0),
        'south': (0, 1, 0),
        'east': (1, 0, 0),
        'west': (-1, 0, 0),
        'up': (0, 0, 1),
        'down': (0, 0, -1),
    }

    for znum in sorted_zones:
        rooms_in_zone = zone_rooms[znum]
        if not rooms_in_zone:
            continue
        base_x, base_y, base_z = zone_offsets[znum]

        # Find a seed room (prefer one with most exits)
        seed = max(rooms_in_zone, key=lambda r: len(r['exits']))
        if seed['vnum'] in visited:
            # Find any unvisited room in this zone
            unvisited = [r for r in rooms_in_zone if r['vnum'] not in visited]
            if not unvisited:
                continue
            seed = unvisited[0]

        queue = [(seed['vnum'], base_x, base_y, base_z)]
        visited.add(seed['vnum'])
        coords[seed['vnum']] = (base_x, base_y, base_z)

        while queue:
            vnum, x, y, z = queue.pop(0)
            room = next((r for r in all_rooms if r['vnum'] == vnum), None)
            if not room:
                continue

            for direction, exit_info in room['exits'].items():
                to_room = exit_info['to_room']
                if to_room < 0 or to_room not in room_vnums:
                    continue
                if to_room in visited:
                    continue

                dx, dy, dz = dir_deltas.get(direction, (0, 0, 0))
                nx, ny, nz = x + dx, y + dy, z + dz

                # Check for collision - if occupied, nudge
                attempts = 0
                while (nx, ny, nz) in coords.values() and attempts < 10:
                    nx += 1
                    attempts += 1

                visited.add(to_room)
                coords[to_room] = (nx, ny, nz)
                queue.append((to_room, nx, ny, nz))

    # For any unvisited rooms, place them near their zone base
    for r in all_rooms:
        if r['vnum'] not in coords:
            bx, by, bz = zone_offsets.get(r['zone'], (0, 0, 0))
            # Offset by vnum to spread them out
            offset = r['vnum'] % 20
            coords[r['vnum']] = (bx + offset, by + (r['vnum'] // 20) % 10, bz)

    # Add coordinates to rooms
    for r in all_rooms:
        r['x'], r['y'], r['z'] = coords[r['vnum']]

    # Normalize coordinates to start near 0,0
    min_x = min(c[0] for c in coords.values())
    min_y = min(c[1] for c in coords.values())
    min_z = min(c[2] for c in coords.values())

    for r in all_rooms:
        r['x'] -= min_x
        r['y'] -= min_y
        r['z'] -= min_z

    output = {
        'rooms': all_rooms,
        'zones': all_zones,
        'stats': {
            'total_rooms': len(all_rooms),
            'total_zones': len(all_zones),
            'coordinate_bounds': {
                'x': [min(r['x'] for r in all_rooms), max(r['x'] for r in all_rooms)],
                'y': [min(r['y'] for r in all_rooms), max(r['y'] for r in all_rooms)],
                'z': [min(r['z'] for r in all_rooms), max(r['z'] for r in all_rooms)],
            }
        }
    }

    with open(OUT_FILE, "w", encoding="utf-8") as f:
        json.dump(output, f, indent=2)

    print(f"\nWrote {len(all_rooms)} rooms, {len(all_zones)} zones to {OUT_FILE}")
    print(f"Coordinate bounds: X={output['stats']['coordinate_bounds']['x']}, "
          f"Y={output['stats']['coordinate_bounds']['y']}, "
          f"Z={output['stats']['coordinate_bounds']['z']}")


if __name__ == "__main__":
    main()
