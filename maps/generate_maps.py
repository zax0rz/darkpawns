#!/usr/bin/env python3
"""
Generate three map outputs from parsed Dark Pawns world data:
1. ASCII text map
2. SVG zone map
3. Interactive HTML map
"""

import json
import math
import os
import random

DATA_FILE = "/home/zach/.openclaw/workspace/darkpawns/maps/world_data.json"
OUT_DIR = "/home/zach/.openclaw/workspace/darkpawns/maps"

# Dark theme colors
BG_COLOR = "#0a0908"
TEXT_COLOR = "#c8b896"
ACCENT_COLOR = "#8b0000"
GRID_COLOR = "#2a2520"

# Zone color palette (distinct, dark-theme-friendly)
ZONE_COLORS = [
    "#e6194b", "#3cb44b", "#ffe119", "#0082c8", "#f58231",
    "#911eb4", "#46f0f0", "#f032e6", "#d2f53c", "#fabebe",
    "#008080", "#e6beff", "#aa6e28", "#fffac8", "#800000",
    "#aaffc3", "#808000", "#ffd8b1", "#000075", "#808080",
    "#ff6b6b", "#4ecdc4", "#45b7d1", "#96ceb4", "#ffeaa7",
    "#dfe6e9", "#fd79a8", "#a29bfe", "#00b894", "#e17055",
    "#74b9ff", "#55efc4", "#ff7675", "#fab1a0", "#fdcb6e",
    "#6c5ce7", "#a8e6cf", "#ff8b94", "#c7ecee", "#dfe6e9",
    "#b2bec3", "#636e72", "#2d3436", "#e84393", "#0984e3",
    "#00cec9", "#6c5ce7", "#fd79a8", "#e17055", "#00b894",
    "#fdcb6e", "#d63031", "#e84393", "#0984e3", "#00cec9",
    "#6c5ce7", "#fd79a8", "#e17055", "#00b894", "#fdcb6e",
    "#d63031", "#e84393", "#0984e3", "#00cec9", "#6c5ce7",
    "#fd79a8", "#e17055", "#00b894", "#fdcb6e", "#d63031",
    "#e84393", "#0984e3", "#00cec9", "#6c5ce7", "#fd79a8",
    "#e17055", "#00b894", "#fdcb6e", "#d63031", "#e84393",
    "#0984e3", "#00cec9", "#6c5ce7", "#fd79a8", "#e17055",
    "#00b894", "#fdcb6e", "#d63031", "#e84393", "#0984e3",
    "#00cec9", "#6c5ce7", "#fd79a8", "#e17055", "#00b894",
]


def load_data():
    with open(DATA_FILE, "r", encoding="utf-8") as f:
        return json.load(f)


def generate_ascii_map(data):
    """Generate a grid-based ASCII text map."""
    rooms = data['rooms']
    zones = data['zones']

    # Build lookups
    room_map = {r['vnum']: r for r in rooms}
    zone_map = {z['number']: z for z in zones}

    # Group rooms by zone for per-zone maps
    zone_rooms = {}
    for r in rooms:
        znum = r['zone']
        if znum not in zone_rooms:
            zone_rooms[znum] = []
        zone_rooms[znum].append(r)

    lines = []
    lines.append("=" * 120)
    lines.append("DARK PAWNS WORLD MAP")
    lines.append("=" * 120)
    lines.append(f"Total Rooms: {len(rooms)} | Total Zones: {len(zones)}")
    lines.append("")

    # World overview: show zones as a grid of summaries
    lines.append("-" * 120)
    lines.append("ZONE OVERVIEW")
    lines.append("-" * 120)

    # Sort zones by number
    sorted_zones = sorted(zones, key=lambda z: z['number'])

    for z in sorted_zones:
        znum = z['number']
        zname = z['name']
        zrooms = zone_rooms.get(znum, [])
        if not zrooms:
            continue

        # Count exits
        total_exits = sum(len(r['exits']) for r in zrooms)
        connected = sum(1 for r in zrooms for e in r['exits'].values() if e['to_room'] in room_map)

        lines.append(f"Zone {znum:3d}: {zname:<40s} | {len(zrooms):4d} rooms | {total_exits:4d} exits | {connected:4d} internal")

    lines.append("")
    lines.append("=" * 120)
    lines.append("DETAILED ZONE MAPS (showing room connections)")
    lines.append("=" * 120)
    lines.append("")

    # For each zone, render a compact ASCII grid
    for z in sorted_zones:
        znum = z['number']
        zname = z['name']
        zrooms = zone_rooms.get(znum, [])
        if len(zrooms) < 2:
            continue

        lines.append(f"\n{'='*120}")
        lines.append(f"ZONE {znum}: {zname}")
        lines.append(f"{'='*120}")

        # Get coordinate bounds for this zone
        xs = [r['x'] for r in zrooms]
        ys = [r['y'] for r in zrooms]
        min_x, max_x = min(xs), max(xs)
        min_y, max_y = min(ys), max(ys)

        # Create a sparse grid
        grid = {}
        for r in zrooms:
            grid[(r['x'], r['y'])] = r

        # Render grid rows
        width = max_x - min_x + 1
        height = max_y - min_y + 1

        # If too large, skip detailed grid and show room list instead
        if width > 60 or height > 30:
            lines.append(f"  (Zone too large for grid: {width}x{height}, showing room list)")
            for r in sorted(zrooms, key=lambda x: x['vnum'])[:50]:
                exit_str = ", ".join(f"{d}:{e['to_room']}" for d, e in r['exits'].items())
                lines.append(f"    #{r['vnum']:5d} {r['name']:<40s} | Exits: {exit_str}")
            if len(zrooms) > 50:
                lines.append(f"    ... and {len(zrooms) - 50} more rooms")
            continue

        # Build the ASCII grid
        # Each cell is 3 chars wide, with connections shown between cells
        for gy in range(min_y, max_y + 1):
            # Room row
            row_chars = []
            for gx in range(min_x, max_x + 1):
                if (gx, gy) in grid:
                    r = grid[(gx, gy)]
                    # Show first char of room name, or vnum last digit
                    label = r['name'][0].upper() if r['name'] else '?'
                    row_chars.append(f"[{label}]")
                else:
                    row_chars.append("   ")
            lines.append("".join(row_chars))

            # Connection row (south exits)
            conn_chars = []
            for gx in range(min_x, max_x + 1):
                if (gx, gy) in grid:
                    r = grid[(gx, gy)]
                    if 'south' in r['exits'] and r['exits']['south']['to_room'] in room_map:
                        conn_chars.append(" | ")
                    else:
                        conn_chars.append("   ")
                else:
                    conn_chars.append("   ")
            lines.append("".join(conn_chars))

        # Add room legend
        lines.append("")
        lines.append("  Rooms:")
        for r in sorted(zrooms, key=lambda x: x['vnum']):
            label = r['name'][0].upper() if r['name'] else '?'
            exit_dirs = "/".join(d[0].upper() for d in r['exits'].keys())
            lines.append(f"    [{label}] #{r['vnum']:5d} {r['name']:<35s} ({exit_dirs})")

    # Add key landmarks section
    lines.append("")
    lines.append("=" * 120)
    lines.append("KEY LANDMARKS")
    lines.append("=" * 120)

    # Find interesting rooms (those with many exits, special names, or zone transitions)
    landmarks = []
    for r in rooms:
        name_lower = r['name'].lower()
        if any(k in name_lower for k in ['temple', 'shrine', 'castle', 'tower', 'dungeon', 'portal', 'gate', 'inn', 'shop', 'bank', 'guild', 'arena', 'throne', 'crypt', 'cave', 'forest', 'mountain', 'river', 'ocean', 'city', 'village']):
            landmarks.append(r)
        elif len(r['exits']) >= 5:
            landmarks.append(r)

    # Deduplicate and sort
    seen = set()
    unique_landmarks = []
    for r in landmarks:
        if r['vnum'] not in seen:
            seen.add(r['vnum'])
            unique_landmarks.append(r)

    unique_landmarks.sort(key=lambda r: (r['zone'], r['vnum']))

    for r in unique_landmarks[:100]:
        zname = zone_map.get(r['zone'], {}).get('name', f"Zone {r['zone']}")
        exit_str = ", ".join(f"{d}" for d in r['exits'].keys())
        lines.append(f"  Zone {r['zone']:3d} ({zname:<25s}) #{r['vnum']:5d}: {r['name']:<40s} | Exits: {exit_str}")

    if len(unique_landmarks) > 100:
        lines.append(f"  ... and {len(unique_landmarks) - 100} more landmarks")

    # Write output
    out_path = os.path.join(OUT_DIR, "world_ascii.txt")
    with open(out_path, "w", encoding="utf-8") as f:
        f.write("\n".join(lines))

    print(f"Wrote ASCII map: {out_path} ({len(lines)} lines)")


def generate_svg_map(data):
    """Generate an SVG zone map with all rooms and connections."""
    rooms = data['rooms']
    zones = data['zones']

    room_map = {r['vnum']: r for r in rooms}
    zone_map = {z['number']: z for z in zones}

    # Assign colors to zones
    zone_colors = {}
    for i, z in enumerate(sorted(zones, key=lambda z: z['number'])):
        zone_colors[z['number']] = ZONE_COLORS[i % len(ZONE_COLORS)]

    # Get coordinate bounds
    xs = [r['x'] for r in rooms]
    ys = [r['y'] for r in rooms]
    min_x, max_x = min(xs), max(xs)
    min_y, max_y = min(ys), max(ys)

    # SVG dimensions
    svg_size = 2000
    padding = 100

    # Scale coordinates to fit SVG
    width = max_x - min_x + 1
    height = max_y - min_y + 1
    scale = min((svg_size - 2 * padding) / max(width, 1),
                (svg_size - 2 * padding) / max(height, 1))

    def to_svg_x(x):
        return padding + (x - min_x) * scale

    def to_svg_y(y):
        return svg_size - padding - (y - min_y) * scale  # Flip Y

    # Build SVG
    svg_parts = []
    svg_parts.append(f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {svg_size} {svg_size}" width="{svg_size}" height="{svg_size}">')
    svg_parts.append(f'<rect width="{svg_size}" height="{svg_size}" fill="{BG_COLOR}"/>')

    # Draw connections first (so they're behind rooms)
    drawn_connections = set()
    for r in rooms:
        for direction, exit_info in r['exits'].items():
            to_vnum = exit_info['to_room']
            if to_vnum not in room_map:
                continue
            target = room_map[to_vnum]

            # Avoid drawing same connection twice
            conn_key = tuple(sorted([r['vnum'], to_vnum]))
            if conn_key in drawn_connections:
                continue
            drawn_connections.add(conn_key)

            x1, y1 = to_svg_x(r['x']), to_svg_y(r['y'])
            x2, y2 = to_svg_x(target['x']), to_svg_y(target['y'])

            # Color by source zone
            color = zone_colors.get(r['zone'], "#888888")
            opacity = "0.3"

            svg_parts.append(f'<line x1="{x1:.1f}" y1="{y1:.1f}" x2="{x2:.1f}" y2="{y2:.1f}" stroke="{color}" stroke-width="0.5" opacity="{opacity}"/>')

    # Draw rooms
    for r in rooms:
        x, y = to_svg_x(r['x']), to_svg_y(r['y'])
        color = zone_colors.get(r['zone'], "#888888")
        radius = 2.5

        # Highlight rooms with many exits
        if len(r['exits']) >= 4:
            radius = 4

        svg_parts.append(f'<circle cx="{x:.1f}" cy="{y:.1f}" r="{radius}" fill="{color}"/>')

    # Draw legend
    legend_x = svg_size - 380
    legend_y = 30
    legend_items = sorted(zones, key=lambda z: z['number'])[:40]  # Show first 40

    svg_parts.append(f'<rect x="{legend_x - 10}" y="{legend_y - 20}" width="370" height="{len(legend_items) * 22 + 40}" fill="{BG_COLOR}" stroke="{GRID_COLOR}" stroke-width="1" rx="5"/>')
    svg_parts.append(f'<text x="{legend_x}" y="{legend_y}" fill="{TEXT_COLOR}" font-family="monospace" font-size="14" font-weight="bold">ZONES</text>')

    for i, z in enumerate(legend_items):
        cy = legend_y + 25 + i * 22
        color = zone_colors.get(z['number'], "#888888")
        svg_parts.append(f'<circle cx="{legend_x + 8}" cy="{cy}" r="6" fill="{color}"/>')
        svg_parts.append(f'<text x="{legend_x + 22}" y="{cy + 4}" fill="{TEXT_COLOR}" font-family="monospace" font-size="11">{z["number"]:3d}: {z["name"][:35]}</text>')

    # Title
    svg_parts.append(f'<text x="{svg_size // 2}" y="40" fill="{TEXT_COLOR}" font-family="monospace" font-size="24" text-anchor="middle" font-weight="bold">Dark Pawns World Map</text>')
    svg_parts.append(f'<text x="{svg_size // 2}" y="65" fill="{TEXT_COLOR}" font-family="monospace" font-size="14" text-anchor="middle">{len(rooms)} rooms across {len(zones)} zones</text>')

    svg_parts.append('</svg>')

    out_path = os.path.join(OUT_DIR, "world_zones.svg")
    with open(out_path, "w", encoding="utf-8") as f:
        f.write("\n".join(svg_parts))

    print(f"Wrote SVG map: {out_path}")


def generate_html_map(data):
    """Generate an interactive HTML map with zoom, pan, search, and tooltips."""
    rooms = data['rooms']
    zones = data['zones']

    # Embed the data as JSON in the HTML
    # To keep file size reasonable, we'll include all rooms but minimize the data
    min_rooms = []
    for r in rooms:
        exits_dict = {}
        for d, e in r['exits'].items():
            exits_dict[d] = e['to_room']
        min_rooms.append({
            'v': r['vnum'],
            'n': r['name'],
            'z': r['zone'],
            'zn': r.get('zone_name', f"Zone {r['zone']}"),
            'x': r['x'],
            'y': r['y'],
            'e': exits_dict
        })

    min_zones = []
    for z in zones:
        min_zones.append({
            'n': z['number'],
            'name': z['name']
        })

    # Assign colors to zones
    zone_colors = {}
    for i, z in enumerate(sorted(zones, key=lambda z: z['number'])):
        zone_colors[z['number']] = ZONE_COLORS[i % len(ZONE_COLORS)]

    # Get bounds
    xs = [r['x'] for r in rooms]
    ys = [r['y'] for r in rooms]
    min_x, max_x = min(xs), max(xs)
    min_y, max_y = min(ys), max(ys)

    rooms_json = json.dumps(min_rooms)
    zones_json = json.dumps(min_zones)
    colors_json = json.dumps(zone_colors)
    bounds_json = json.dumps({'minX': min_x, 'maxX': max_x, 'minY': min_y, 'maxY': max_y})

    html = f'''<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Dark Pawns — World Map</title>
<style>
  * {{ margin: 0; padding: 0; box-sizing: border-box; }}
  body {{
    background: {BG_COLOR};
    color: {TEXT_COLOR};
    font-family: 'Courier New', monospace;
    overflow: hidden;
    width: 100vw;
    height: 100vh;
  }}
  #map-container {{
    width: 100%;
    height: 100%;
    position: relative;
    cursor: grab;
  }}
  #map-container:active {{ cursor: grabbing; }}
  canvas {{
    display: block;
    width: 100%;
    height: 100%;
  }}
  #ui {{
    position: absolute;
    top: 10px;
    left: 10px;
    z-index: 10;
    display: flex;
    flex-direction: column;
    gap: 8px;
    pointer-events: none;
  }}
  #ui > * {{ pointer-events: auto; }}
  #search-box {{
    background: #1a1510;
    border: 1px solid {GRID_COLOR};
    color: {TEXT_COLOR};
    padding: 8px 12px;
    font-family: inherit;
    font-size: 14px;
    width: 280px;
    border-radius: 4px;
    outline: none;
  }}
  #search-box:focus {{ border-color: {ACCENT_COLOR}; }}
  #search-results {{
    background: #1a1510;
    border: 1px solid {GRID_COLOR};
    max-height: 300px;
    overflow-y: auto;
    width: 280px;
    border-radius: 4px;
    display: none;
  }}
  .search-item {{
    padding: 6px 10px;
    cursor: pointer;
    font-size: 12px;
    border-bottom: 1px solid {GRID_COLOR};
  }}
  .search-item:hover {{ background: #2a2520; }}
  .search-item .vnum {{ color: #888; font-size: 10px; }}
  #controls {{
    display: flex;
    gap: 4px;
  }}
  .btn {{
    background: #1a1510;
    border: 1px solid {GRID_COLOR};
    color: {TEXT_COLOR};
    padding: 6px 12px;
    font-family: inherit;
    font-size: 12px;
    cursor: pointer;
    border-radius: 4px;
  }}
  .btn:hover {{ background: #2a2520; }}
  #tooltip {{
    position: absolute;
    background: #1a1510;
    border: 1px solid {ACCENT_COLOR};
    color: {TEXT_COLOR};
    padding: 10px 14px;
    font-size: 12px;
    border-radius: 4px;
    pointer-events: none;
    display: none;
    z-index: 20;
    max-width: 300px;
    box-shadow: 0 4px 12px rgba(0,0,0,0.5);
  }}
  #tooltip .room-name {{ color: {ACCENT_COLOR}; font-weight: bold; margin-bottom: 4px; }}
  #tooltip .room-info {{ color: #aaa; font-size: 11px; }}
  #tooltip .exits {{ color: #888; margin-top: 4px; }}
  #legend {{
    position: absolute;
    top: 10px;
    right: 10px;
    z-index: 10;
    background: #1a1510;
    border: 1px solid {GRID_COLOR};
    padding: 10px;
    border-radius: 4px;
    max-height: 80vh;
    overflow-y: auto;
    font-size: 11px;
  }}
  #legend h3 {{ margin-bottom: 8px; color: {ACCENT_COLOR}; font-size: 12px; }}
  .legend-item {{ display: flex; align-items: center; gap: 6px; margin-bottom: 4px; cursor: pointer; }}
  .legend-item:hover {{ opacity: 0.8; }}
  .legend-dot {{ width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0; }}
  .legend-item.hidden-zone {{ opacity: 0.3; }}
  #stats {{
    position: absolute;
    bottom: 10px;
    left: 10px;
    z-index: 10;
    background: #1a1510;
    border: 1px solid {GRID_COLOR};
    padding: 8px 12px;
    border-radius: 4px;
    font-size: 11px;
    color: #888;
  }}
  @media (max-width: 600px) {{
    #search-box {{ width: 200px; }}
    #search-results {{ width: 200px; }}
    #legend {{ max-height: 50vh; font-size: 10px; }}
  }}
</style>
</head>
<body>
<div id="map-container">
  <canvas id="map"></canvas>
</div>

<div id="ui">
  <input type="text" id="search-box" placeholder="Search rooms or zones..." autocomplete="off">
  <div id="search-results"></div>
  <div id="controls">
    <button class="btn" id="btn-zoom-in">+</button>
    <button class="btn" id="btn-zoom-out">−</button>
    <button class="btn" id="btn-reset">Reset</button>
    <button class="btn" id="btn-toggle-connections">Lines</button>
  </div>
</div>

<div id="tooltip"></div>

<div id="legend">
  <h3>ZONES</h3>
  <div id="legend-content"></div>
</div>

<div id="stats">
  <span id="stat-zoom">Zoom: 100%</span> |
  <span id="stat-rooms">Rooms: 0</span> |
  <span id="stat-pos">Pos: 0,0</span>
</div>

<script>
(function() {{
  const rooms = {rooms_json};
  const zones = {zones_json};
  const zoneColors = {colors_json};
  const bounds = {bounds_json};

  const canvas = document.getElementById('map');
  const ctx = canvas.getContext('2d');
  const container = document.getElementById('map-container');
  const tooltip = document.getElementById('tooltip');
  const searchBox = document.getElementById('search-box');
  const searchResults = document.getElementById('search-results');
  const legendContent = document.getElementById('legend-content');

  let width, height;
  let scale = 1;
  let offsetX = 0, offsetY = 0;
  let isDragging = false;
  let lastX = 0, lastY = 0;
  let showConnections = true;
  let hoveredRoom = null;
  let hiddenZones = new Set();

  const worldW = bounds.maxX - bounds.minX + 1;
  const worldH = bounds.maxY - bounds.minY + 1;

  // Build room lookup
  const roomMap = {{}};
  for (const r of rooms) roomMap[r.v] = r;

  // Build zone lookup
  const zoneMap = {{}};
  for (const z of zones) zoneMap[z.n] = z;

  function resize() {{
    width = container.clientWidth;
    height = container.clientHeight;
    canvas.width = width * window.devicePixelRatio;
    canvas.height = height * window.devicePixelRatio;
    ctx.scale(window.devicePixelRatio, window.devicePixelRatio);
    fitWorld();
    draw();
  }}

  function fitWorld() {{
    const padding = 60;
    const sx = (width - padding * 2) / worldW;
    const sy = (height - padding * 2) / worldH;
    scale = Math.min(sx, sy);
    offsetX = width / 2 - (bounds.minX + worldW / 2) * scale;
    offsetY = height / 2 - (bounds.minY + worldH / 2) * scale;
  }}

  function worldToScreen(wx, wy) {{
    return {{
      x: wx * scale + offsetX,
      y: wy * scale + offsetY
    }};
  }}

  function screenToWorld(sx, sy) {{
    return {{
      x: (sx - offsetX) / scale,
      y: (sy - offsetY) / scale
    }};
  }}

  function draw() {{
    ctx.clearRect(0, 0, width, height);

    // Draw grid
    ctx.strokeStyle = '{GRID_COLOR}';
    ctx.lineWidth = 0.5;
    const gridSize = Math.max(10, Math.round(50 / scale));
    const startWX = Math.floor(screenToWorld(0, 0).x / gridSize) * gridSize;
    const startWY = Math.floor(screenToWorld(0, 0).y / gridSize) * gridSize;
    const endWX = Math.ceil(screenToWorld(width, 0).x / gridSize) * gridSize;
    const endWY = Math.ceil(screenToWorld(0, height).y / gridSize) * gridSize;

    ctx.beginPath();
    for (let x = startWX; x <= endWX; x += gridSize) {{
      const p1 = worldToScreen(x, startWY);
      const p2 = worldToScreen(x, endWY);
      ctx.moveTo(p1.x, p1.y);
      ctx.lineTo(p2.x, p2.y);
    }}
    for (let y = startWY; y <= endWY; y += gridSize) {{
      const p1 = worldToScreen(startWX, y);
      const p2 = worldToScreen(endWX, y);
      ctx.moveTo(p1.x, p1.y);
      ctx.lineTo(p2.x, p2.y);
    }}
    ctx.stroke();

    // Draw connections
    if (showConnections && scale > 0.3) {{
      const drawn = new Set();
      ctx.lineWidth = 0.5;
      for (const r of rooms) {{
        if (hiddenZones.has(r.z)) continue;
        for (const [dir, toVnum] of Object.entries(r.e)) {{
          if (!roomMap[toVnum]) continue;
          const key = [r.v, toVnum].sort().join('-');
          if (drawn.has(key)) continue;
          drawn.add(key);

          const target = roomMap[toVnum];
          if (hiddenZones.has(target.z)) continue;

          const p1 = worldToScreen(r.x, r.y);
          const p2 = worldToScreen(target.x, target.y);

          ctx.strokeStyle = zoneColors[r.z] || '#888';
          ctx.globalAlpha = 0.2;
          ctx.beginPath();
          ctx.moveTo(p1.x, p1.y);
          ctx.lineTo(p2.x, p2.y);
          ctx.stroke();
          ctx.globalAlpha = 1;
        }}
      }}
    }}

    // Draw rooms
    const roomRadius = Math.max(1.5, Math.min(4, scale * 0.8));
    for (const r of rooms) {{
      if (hiddenZones.has(r.z)) continue;
      const p = worldToScreen(r.x, r.y);
      const color = zoneColors[r.z] || '#888';

      ctx.fillStyle = color;
      ctx.beginPath();
      ctx.arc(p.x, p.y, roomRadius, 0, Math.PI * 2);
      ctx.fill();

      // Highlight hovered room
      if (hoveredRoom && hoveredRoom.v === r.v) {{
        ctx.strokeStyle = '#fff';
        ctx.lineWidth = 2;
        ctx.beginPath();
        ctx.arc(p.x, p.y, roomRadius + 3, 0, Math.PI * 2);
        ctx.stroke();
      }}
    }}

    // Update stats
    document.getElementById('stat-zoom').textContent = 'Zoom: ' + Math.round(scale * 100) + '%';
    document.getElementById('stat-rooms').textContent = 'Rooms: ' + rooms.length;
    const center = screenToWorld(width / 2, height / 2);
    document.getElementById('stat-pos').textContent = 'Pos: ' + Math.round(center.x) + ',' + Math.round(center.y);
  }}

  // Mouse / touch events
  container.addEventListener('mousedown', e => {{
    isDragging = true;
    lastX = e.clientX;
    lastY = e.clientY;
  }});

  window.addEventListener('mousemove', e => {{
    if (isDragging) {{
      const dx = e.clientX - lastX;
      const dy = e.clientY - lastY;
      offsetX += dx;
      offsetY += dy;
      lastX = e.clientX;
      lastY = e.clientY;
      draw();
    }}

    // Find hovered room
    const rect = canvas.getBoundingClientRect();
    const mx = e.clientX - rect.left;
    const my = e.clientY - rect.top;
    const worldPos = screenToWorld(mx, my);

    let closest = null;
    let closestDist = Infinity;
    const threshold = 10 / scale;

    for (const r of rooms) {{
      if (hiddenZones.has(r.z)) continue;
      const dx = r.x - worldPos.x;
      const dy = r.y - worldPos.y;
      const dist = Math.sqrt(dx * dx + dy * dy);
      if (dist < threshold && dist < closestDist) {{
        closest = r;
        closestDist = dist;
      }}
    }}

    if (closest !== hoveredRoom) {{
      hoveredRoom = closest;
      draw();
      if (hoveredRoom) {{
        const exitDirs = Object.keys(hoveredRoom.e).join(', ') || 'none';
        tooltip.innerHTML = `
          <div class="room-name">#${{hoveredRoom.v}} ${{hoveredRoom.n}}</div>
          <div class="room-info">Zone: ${{hoveredRoom.zn}}</div>
          <div class="exits">Exits: ${{exitDirs}}</div>
        `;
        tooltip.style.display = 'block';
        tooltip.style.left = (e.clientX + 15) + 'px';
        tooltip.style.top = (e.clientY + 15) + 'px';
      }} else {{
        tooltip.style.display = 'none';
      }}
    }} else if (hoveredRoom) {{
        tooltip.style.left = (e.clientX + 15) + 'px';
        tooltip.style.top = (e.clientY + 15) + 'px';
      }}
    }}
  }});

  window.addEventListener('mouseup', () => {{
    isDragging = false;
  }});

  // Zoom with scroll wheel
  container.addEventListener('wheel', e => {{
    e.preventDefault();
    const zoomFactor = e.deltaY > 0 ? 0.9 : 1.1;
    const rect = canvas.getBoundingClientRect();
    const mx = e.clientX - rect.left;
    const my = e.clientY - rect.top;
    const worldBefore = screenToWorld(mx, my);

    scale *= zoomFactor;
    scale = Math.max(0.1, Math.min(scale, 50));

    const worldAfter = screenToWorld(mx, my);
    offsetX += (worldAfter.x - worldBefore.x) * scale;
    offsetY += (worldAfter.y - worldBefore.y) * scale;
    draw();
  }}, {{ passive: false }});

  // Touch support for mobile
  let lastTouchDist = 0;
  container.addEventListener('touchstart', e => {{
    if (e.touches.length === 1) {{
      isDragging = true;
      lastX = e.touches[0].clientX;
      lastY = e.touches[0].clientY;
    }} else if (e.touches.length === 2) {{
      const dx = e.touches[0].clientX - e.touches[1].clientX;
      const dy = e.touches[0].clientY - e.touches[1].clientY;
      lastTouchDist = Math.sqrt(dx * dx + dy * dy);
    }}
  }}, {{ passive: false }});

  container.addEventListener('touchmove', e => {{
    e.preventDefault();
    if (e.touches.length === 1 && isDragging) {{
      const dx = e.touches[0].clientX - lastX;
      const dy = e.touches[0].clientY - lastY;
      offsetX += dx;
      offsetY += dy;
      lastX = e.touches[0].clientX;
      lastY = e.touches[0].clientY;
      draw();
    }} else if (e.touches.length === 2) {{
      const dx = e.touches[0].clientX - e.touches[1].clientX;
      const dy = e.touches[0].clientY - e.touches[1].clientY;
      const dist = Math.sqrt(dx * dx + dy * dy);
      if (lastTouchDist > 0) {{
        const zoomFactor = dist / lastTouchDist;
        scale *= zoomFactor;
        scale = Math.max(0.1, Math.min(scale, 50));
        draw();
      }}
      lastTouchDist = dist;
    }}
  }}, {{ passive: false }});

  container.addEventListener('touchend', () => {{
    isDragging = false;
    lastTouchDist = 0;
  }});

  // Button controls
  document.getElementById('btn-zoom-in').addEventListener('click', () => {{
    scale *= 1.3;
    scale = Math.min(scale, 50);
    draw();
  }});

  document.getElementById('btn-zoom-out').addEventListener('click', () => {{
    scale *= 0.7;
    scale = Math.max(0.1, scale);
    draw();
  }});

  document.getElementById('btn-reset').addEventListener('click', () => {{
    fitWorld();
    draw();
  }});

  document.getElementById('btn-toggle-connections').addEventListener('click', () => {{
    showConnections = !showConnections;
    draw();
  }});

  // Search
  searchBox.addEventListener('input', () => {{
    const query = searchBox.value.toLowerCase().trim();
    if (!query) {{
      searchResults.style.display = 'none';
      return;
    }}

    const matches = rooms.filter(r =>
      r.n.toLowerCase().includes(query) ||
      r.zn.toLowerCase().includes(query) ||
      String(r.v).includes(query)
    ).slice(0, 20);

    searchResults.innerHTML = '';
    if (matches.length === 0) {{
      searchResults.style.display = 'none';
      return;
    }}

    for (const r of matches) {{
      const div = document.createElement('div');
      div.className = 'search-item';
      div.innerHTML = `<span class="vnum">#${{r.v}}</span> ${{r.n}} <span style="color:#888">(${{r.zn}})</span>`;
      div.addEventListener('click', () => {{
        const p = worldToScreen(r.x, r.y);
        offsetX = width / 2 - r.x * scale;
        offsetY = height / 2 - r.y * scale;
        scale = Math.max(scale, 3);
        draw();
        searchResults.style.display = 'none';
        searchBox.value = '';
      }});
      searchResults.appendChild(div);
    }}
    searchResults.style.display = 'block';
  }});

  // Hide search results on click outside
  document.addEventListener('click', e => {{
    if (!searchBox.contains(e.target) && !searchResults.contains(e.target)) {{
      searchResults.style.display = 'none';
    }}
  }});

  // Build legend
  function buildLegend() {{
    legendContent.innerHTML = '';
    for (const z of zones) {{
      const item = document.createElement('div');
      item.className = 'legend-item' + (hiddenZones.has(z.n) ? ' hidden-zone' : '');
      item.innerHTML = `<div class="legend-dot" style="background:${{zoneColors[z.n]||'#888'}}"></div><span>${{z.n}}: ${{z.name}}</span>`;
      item.addEventListener('click', () => {{
        if (hiddenZones.has(z.n)) {{
          hiddenZones.delete(z.n);
        }} else {{
          hiddenZones.add(z.n);
        }}
        buildLegend();
        draw();
      }});
      legendContent.appendChild(item);
    }}
  }}
  buildLegend();

  // Initial setup
  window.addEventListener('resize', resize);
  resize();
}})();
</script>
</body>
</html>'''

    out_path = os.path.join(OUT_DIR, "world_interactive.html")
    with open(out_path, "w", encoding="utf-8") as f:
        f.write(html)

    print(f"Wrote HTML map: {out_path}")