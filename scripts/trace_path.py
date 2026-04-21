#!/usr/bin/env python3
"""Trace direction paths through Dark Pawns world files."""
import re, os, sys

WORLD_DIR = "/home/zach/.openclaw/workspace/darkpawns/lib/world/wld/"

DIR_MAP = {'n': 0, 'e': 1, 's': 2, 'w': 3, 'u': 4, 'd': 5}
DIR_NAME = {0: 'N', 1: 'E', 2: 'S', 3: 'W', 4: 'U', 5: 'D'}

def load_world():
    rooms = {}
    for fname in os.listdir(WORLD_DIR):
        if not fname.endswith('.wld') and not fname.endswith('.new'):
            continue
        path = os.path.join(WORLD_DIR, fname)
        try:
            with open(path, 'r', errors='replace') as f:
                content = f.read()
        except:
            continue
        blocks = re.split(r'\n(?=#\d)', content)
        for block in blocks:
            m = re.match(r'#(\d+)\n(.*?)~', block, re.DOTALL)
            if not m:
                continue
            vnum = int(m.group(1))
            name = m.group(2).strip()
            exits = {}
            for dm in re.finditer(r'D(\d)\n.*?~\n.*?~\n(\d+) (-?\d+) (\d+)', block, re.DOTALL):
                direction = int(dm.group(1))
                to_room = int(dm.group(4))
                if to_room > 0:
                    exits[direction] = to_room
            rooms[vnum] = {'name': name, 'exits': exits}
    return rooms

def parse_path(path_str):
    steps = []
    path_str = path_str.lower().replace(' ', '').replace(',', '').replace(';', '')
    i = 0
    while i < len(path_str):
        count = 0
        while i < len(path_str) and path_str[i].isdigit():
            count = count * 10 + int(path_str[i])
            i += 1
        if count == 0:
            count = 1
        if i < len(path_str) and path_str[i] in DIR_MAP:
            steps.append((DIR_MAP[path_str[i]], count))
            i += 1
        else:
            i += 1
    return steps

def trace(rooms, start_vnum, path_str):
    steps = parse_path(path_str)
    current = start_vnum
    for direction, count in steps:
        for _ in range(count):
            if current not in rooms:
                return current, f"DEAD END — room {current} not in world"
            exits = rooms[current].get('exits', {})
            if direction not in exits:
                return current, f"NO EXIT {DIR_NAME[direction]} from {current} ({rooms[current]['name']})"
            current = exits[direction]
    return current, rooms.get(current, {}).get('name', 'UNKNOWN')

print("Loading world files...")
rooms = load_world()
print(f"Loaded {len(rooms)} rooms\n")

MARKET_SQUARE = 8046  # Market Square, Kir Drax'in (zone 80)

paths = [
    ("Crystal Temple", "2n4e2n"),
    ("Ender Village",  "14s4wd"),
    ("Zoo",            "2n2e2n"),
    ("Cold Village",   "11w7n"),
    ("Slums",          "4se"),
    ("King Seilon's Keep", "22w2n"),
    ("Orcs",           "32w5sd"),
    ("Ogres",          "16e7nw2s"),
    ("Hell",           "15w4nw3n3e"),
    ("Bhyroga",        "15w4nw3nw5n2e4n"),
    ("Shax'in Brown Dragon", "18s30w4s8ws2wd"),
]

for label, path in paths:
    end_vnum, result = trace(rooms, MARKET_SQUARE, path)
    status = "OK" if "NO EXIT" not in result and "DEAD END" not in result else "FAIL"
    print(f"[{status}] {label}")
    print(f"       -> room {end_vnum}: {result}\n")
