#!/usr/bin/env python3
"""
Generate an Obsidian vault from Dark Pawns world data.
Each room becomes a markdown file with [[wikilinks]] to connected rooms.

Usage:  python3 scripts/generate_obsidian_vault.py
Output: /tmp/dp-obsidian-vault/
"""

import json, os, shutil, re

REPO_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
WORLD_JSON = os.path.join(REPO_DIR, "website/static/map/world-map.json")
OVERVIEW_JSON = os.path.join(REPO_DIR, "website/static/map/world-overview.json")
DB_JSON = os.path.join(REPO_DIR, "website/static/data/database.json")
VAULT_DIR = "/tmp/dp-obsidian-vault"

SECTOR = {
    0: 'inside', 1: 'city', 2: 'field', 3: 'forest', 4: 'hills',
    5: 'mountain', 6: 'water', 7: 'deep water', 8: 'underwater',
    9: 'flying', 10: 'desert', 11: 'fire', 12: 'earth',
    13: 'wind', 14: 'water', 15: 'swamp',
}

def safe_name(s):
    s = re.sub(r'[<>:"/\\|?*]', '', s).strip('. ')
    return s[:80] if s else "unnamed"

def main():
    print("Loading data...")
    with open(WORLD_JSON) as f:
        world = json.load(f)
    with open(OVERVIEW_JSON) as f:
        overview = json.load(f)
    with open(DB_JSON) as f:
        db = json.load(f)

    # Zone names from overview
    zone_names = {z['id']: safe_name(z['name']) for z in overview['nodes']}

    # Index rooms
    room_map = {r['id']: r for r in world['rooms']}
    link_map = {}
    for lk in world['links']:
        link_map.setdefault(lk['s'], []).append(lk['t'])
        link_map.setdefault(lk['t'], []).append(lk['s'])

    # Index mobs/items by room
    mobs_by_room = {}
    for vnum, m in db.get('mobs', {}).items():
        for sp in m.get('spw', []):
            rid = sp.get('room')
            if rid:
                mobs_by_room.setdefault(rid, []).append(m)

    items_by_room = {}
    for vnum, o in db.get('items', {}).items():
        for rm in o.get('rms', []):
            rid = rm.get('room')
            if rid:
                items_by_room.setdefault(rid, []).append(o)

    # Group rooms by zone
    zones = {}
    for r in world['rooms']:
        zones.setdefault(r.get('zone_id', 0), []).append(r)

    # Clean and create vault
    if os.path.exists(VAULT_DIR):
        shutil.rmtree(VAULT_DIR)
    os.makedirs(VAULT_DIR)

    # .obsidian config
    obs = os.path.join(VAULT_DIR, ".obsidian")
    os.makedirs(obs)
    with open(os.path.join(obs, "app.json"), "w") as f:
        json.dump({"strictLineBreaks": True}, f)

    with open(os.path.join(obs, "graph.json"), "w") as f:
        json.dump({
            "collapse-filter": False,
            "search": "",
            "showTags": False,
            "showAttachments": False,
            "hideUnresolved": False,
            "showOrphans": True,
            "collapse-color-groups": False,
            "colorGroups": [],
            "collapse-display": False,
            "showArrow": True,
            "textFadeMultiplier": 0,
            "nodeSizeMultiplier": 1.2,
            "lineSizeMultiplier": 1,
            "collapse-forces": False,
            "centerStrength": 0.3,
            "repelStrength": 15,
            "linkStrength": 0.5,
            "linkDistance": 120,
            "scale": 1,
            "close": False,
        }, f)

    total_rooms = 0
    total_links = 0

    for zid, rooms in sorted(zones.items()):
        zname = zone_names.get(zid, f"zone-{zid}")
        zone_dir = os.path.join(VAULT_DIR, "zones", zname)
        os.makedirs(zone_dir, exist_ok=True)

        for room in rooms:
            rid = room['id']
            neighbors = sorted(set(link_map.get(rid, [])))

            lines = [
                f"# {room.get('name', 'An Unnamed Room')}",
                "",
                f"VNUM: {rid}  ",
                f"Zone: {zname}  ",
                f"Sector: {SECTOR.get(room.get('sector', 0), 'unknown')}  ",
                "",
                "---",
                "",
                "## Exits",
                "",
            ]

            for nrid in neighbors:
                n = room_map.get(nrid)
                if n:
                    n_zid = n.get('zone_id', 0)
                    if n_zid == zid:
                        lines.append(f"- [[{nrid}]]")
                    else:
                        n_zname = zone_names.get(n_zid, f"zone-{n_zid}")
                        lines.append(f"- [[{nrid}]] → *{n_zname}*")

            lines.append("")

            mobs = mobs_by_room.get(rid, [])
            if mobs:
                lines.append("## Mobs")
                lines.append("")
                for m in mobs[:10]:
                    lines.append(f"- {m.get('s', '?')} (Lvl {m.get('lvl', '?')})")
                lines.append("")

            items = items_by_room.get(rid, [])
            if items:
                lines.append("## Items")
                lines.append("")
                for it in items[:10]:
                    lines.append(f"- {it.get('s', '?')}")
                lines.append("")

            with open(os.path.join(zone_dir, f"{rid}.md"), "w") as f:
                f.write("\n".join(lines))

            total_rooms += 1
            total_links += len(neighbors)

    # INDEX file
    with open(os.path.join(VAULT_DIR, "INDEX.md"), "w") as f:
        f.write(f"# Dark Pawns — World Index\n\n{total_rooms} rooms across {len(zones)} zones.\n\n")
        for zid in sorted(zones.keys()):
            zname = zone_names.get(zid, f"zone-{zid}")
            f.write(f"- {zname} ({len(zones[zid])} rooms)\n")

    print(f"Done! {total_rooms} rooms, {total_links} links, {len(zones)} zones")
    print(f"Vault: {VAULT_DIR}")
    print(f"\n1. Open {VAULT_DIR} as a vault in Obsidian")
    print(f"2. Open graph view (Cmd+G)")
    print(f"3. Let it settle, then screenshot")

if __name__ == "__main__":
    main()
