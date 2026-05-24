#!/usr/bin/env python3
"""Parse CircleMUD world files → static/map/world.json for the Dark Pawns map."""

import json
import re
import sys
from pathlib import Path

WORLD_DIR = Path.home() / "darkpawns/lib/world"
if not WORLD_DIR.exists():
    WORLD_DIR = Path(__file__).parent.parent.parent / "lib/world"
OUTPUT = Path(__file__).parent.parent / "static/map/world.json"

DIRECTIONS = {0: "n", 1: "e", 2: "s", 3: "w", 4: "u", 5: "d"}

SECTOR_NAMES = {
    0: "inside", 1: "city", 2: "field", 3: "forest", 4: "hills",
    5: "mountain", 6: "water", 7: "water", 8: "underwater",
    9: "flying", 10: "desert", 11: "swamp",
}


def read_string(lines, i):
    """Read a tilde-terminated string. Handles both inline ~ and bare ~ line."""
    parts = []
    while i < len(lines):
        line = lines[i]
        i += 1
        stripped = line.rstrip()
        if stripped == '~':
            break
        if stripped.endswith('~'):
            parts.append(stripped[:-1])
            break
        parts.append(line.rstrip('\n'))
    return '\n'.join(parts).strip(), i


def parse_zones():
    zones = {}
    zon_dir = WORLD_DIR / "zon"
    for f in sorted(zon_dir.glob("*.zon")):
        try:
            lines = f.read_text(errors='replace').splitlines()
            i = 0
            while i < len(lines):
                line = lines[i].strip()
                i += 1
                if not line.startswith('#') or line == '#$':
                    continue
                try:
                    vnum = int(line[1:])
                except ValueError:
                    continue
                name, i = read_string(lines, i)
                # Line: <top_room> <lifespan> <reset_mode>
                if i < len(lines):
                    parts = lines[i].split()
                    top = int(parts[0]) if parts else vnum * 100 + 99
                    i += 1
                else:
                    top = vnum * 100 + 99
                zones[vnum] = {"name": name, "top": top}
        except Exception as e:
            print(f"Warning: {f.name}: {e}", file=sys.stderr)
    return zones


def parse_rooms():
    rooms = {}
    wld_dir = WORLD_DIR / "wld"
    for f in sorted(wld_dir.glob("*.wld")):
        try:
            lines = f.read_text(errors='replace').splitlines()
            i = 0
            while i < len(lines):
                line = lines[i].strip()
                i += 1
                if line == '$~' or line == '#$':
                    break
                if not line.startswith('#'):
                    continue
                try:
                    vnum = int(line[1:])
                except ValueError:
                    continue

                # Name
                name, i = read_string(lines, i)
                # Description
                desc, i = read_string(lines, i)
                # Flags: zone_num room_flags[4] sector_type [extras…]
                flags_line = lines[i] if i < len(lines) else ""
                i += 1
                parts = flags_line.split()
                zone_num = int(parts[0]) if len(parts) > 0 else -1
                sector = int(parts[5]) if len(parts) > 5 else 0

                exits = []
                while i < len(lines):
                    tok = lines[i].strip()
                    i += 1
                    if tok == 'S':
                        break
                    if re.match(r'^D\d+$', tok):
                        direction = int(tok[1:])
                        _, i = read_string(lines, i)   # exit desc
                        _, i = read_string(lines, i)   # exit keywords
                        nav = lines[i] if i < len(lines) else ""
                        i += 1
                        nav_parts = nav.split()
                        if len(nav_parts) >= 3:
                            try:
                                dest = int(nav_parts[2])
                                if dest >= 0:
                                    exits.append({"d": DIRECTIONS.get(direction, str(direction)), "t": dest})
                            except ValueError:
                                pass
                    elif tok == 'E':
                        # Extra description block: keywords~ then description~
                        _, i = read_string(lines, i)
                        _, i = read_string(lines, i)

                rooms[vnum] = {
                    "id": vnum,
                    "name": name,
                    "desc": desc,
                    "sector": sector,
                    "zone": zone_num,
                    "exits": exits,
                }
        except Exception as e:
            print(f"Warning: {f.name} (near line {i}): {e}", file=sys.stderr)
    return rooms


def main():
    print("Parsing zones...", file=sys.stderr)
    zones = parse_zones()
    print(f"  {len(zones)} zones found", file=sys.stderr)

    print("Parsing rooms...", file=sys.stderr)
    rooms = parse_rooms()
    print(f"  {len(rooms)} rooms found", file=sys.stderr)

    valid_vnums = set(rooms.keys())

    # Group rooms by zone
    zone_buckets = {zid: [] for zid in zones}
    orphans = []
    for vnum, room in rooms.items():
        zid = room["zone"]
        if zid in zone_buckets:
            zone_buckets[zid].append(room)
        else:
            orphans.append(room)

    if orphans:
        print(f"  {len(orphans)} orphan rooms (zone not in .zon files)", file=sys.stderr)

    zone_list = []
    for zid, zone_meta in sorted(zones.items()):
        bucket = sorted(zone_buckets.get(zid, []), key=lambda r: r["id"])
        if not bucket:
            continue
        zone_list.append({
            "id": zid,
            "name": zone_meta["name"],
            "top": zone_meta["top"],
            "rooms": [
                {
                    "id": r["id"],
                    "name": r["name"],
                    "desc": r["desc"],
                    "sector": r["sector"],
                    "exits": [e for e in r["exits"] if e["t"] in valid_vnums],
                }
                for r in bucket
            ],
        })

    print(f"  {len(zone_list)} zones with rooms", file=sys.stderr)

    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    with open(OUTPUT, "w") as f:
        json.dump({"zones": zone_list}, f, separators=(",", ":"))

    size_kb = OUTPUT.stat().st_size / 1024
    print(f"Written: {OUTPUT} ({size_kb:.0f} KB)", file=sys.stderr)


if __name__ == "__main__":
    main()
