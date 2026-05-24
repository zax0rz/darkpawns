---
title: "Interactive Map"
description: "Interactive map of the Dark Pawns world with zoom, search, and room details"
date: 2026-04-22
draft: false
section: "docs"
weight: 30
tags: ["game", "world", "map", "interactive"]
categories: ["Game"]
---

# Interactive Dark Pawns Map

Explore the world of Dark Pawns with this interactive map. The map shows all areas, rooms, and connections in the game world.

## Features

- **Zoom & Pan**: Navigate the map with mouse or touch controls
- **Room Search**: Find specific rooms by name or description
- **Area Overview**: Switch between different game areas
- **Room Details**: Click on any room to see detailed information
- **Pathfinding**: Find routes between rooms
- **Export Options**: Download map as PNG or SVG

## Map Interface

<div id="map-container" class="map-interface-container">
  <div id="map-loading" class="map-interface-loading">
    Loading interactive map...
  </div>
  <canvas id="map-canvas"></canvas>
</div>

<div id="map-controls" class="map-interface-controls">
  <div class="map-interface-row">
    <button id="zoom-in" class="btn btn-secondary btn-small">Zoom In</button>
    <button id="zoom-out" class="btn btn-secondary btn-small">Zoom Out</button>
    <button id="reset-view" class="btn btn-secondary btn-small">Reset View</button>
    <div class="map-interface-spacer"></div>
    <input id="room-search" type="text" class="map-interface-input" placeholder="Search rooms...">
    <button id="search-btn" class="btn btn-primary btn-small">Search</button>
  </div>
  
  <div id="search-results" class="map-interface-results">
    <div class="card">
      <h4>Search Results</h4>
      <div id="results-list"></div>
    </div>
  </div>
</div>

<div id="room-details" class="card map-interface-details">
  <h3 id="room-title">Room Details</h3>
  <div id="room-content">
    <p>Select a room on the map to see details.</p>
  </div>
</div>

## Areas

<div id="areas-list" style="display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 20px; margin-top: 20px;">
  <!-- Areas will be populated by JavaScript -->
</div>

## Using the Map

### For Players
- **Navigation**: Use the map to plan your journey through the game world
- **Discovery**: Find hidden areas and secret connections
- **Quest Planning**: Identify key locations for quest objectives

### For Agent Developers
- **Programmatic Access**: The map data is available via API at `/api/map.json`
- **Pathfinding**: Use the map for automated navigation planning
- **Room Analysis**: Access detailed room information for agent decision-making

## Map Data API

The map data is available in multiple formats:

```bash
# Get map data as JSON
curl https://darkpawns.labz0rz.com/docs/api/map.json

# Get specific area data
curl https://darkpawns.labz0rz.com/docs/api/map/areas.json

# Get room details
curl https://darkpawns.labz0rz.com/docs/api/map/rooms.json
```

## Keyboard Navigation

- **Arrow Keys**: Pan the map
- **+/-**: Zoom in/out
- **Space**: Reset view
- **F**: Fit map to screen
- **Esc**: Clear search/selection

## Accessibility Features

- **Screen Reader Support**: All map controls are properly labeled
- **Keyboard Navigation**: Full keyboard support for all features
- **High Contrast Mode**: Map supports high contrast themes
- **Zoom Compatibility**: Works with browser zoom up to 400%

## Troubleshooting

If the map doesn't load:
1. Check your internet connection
2. Ensure JavaScript is enabled
3. Try refreshing the page
4. Clear your browser cache

For persistent issues, please report them on [GitHub](https://github.com/zax0rz/darkpawns/issues).

---

*Note: This is an interactive prototype. The full Mudlet map integration will be deployed separately at `https://darkpawns.labz0rz.com/map/`*
