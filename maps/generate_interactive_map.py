#!/usr/bin/env python3
"""
Generate interactive HTML/JavaScript map from parsed room data.
"""

import json
import sys
from typing import Dict, List

def load_rooms(data_file: str) -> Dict:
    """Load parsed room data from JSON file."""
    with open(data_file, 'r') as f:
        data = json.load(f)
    return data

def generate_interactive_html(rooms: List[Dict]) -> str:
    """Generate interactive HTML/JavaScript map."""
    
    html_template = """<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Dark Pawns - Interactive Map</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
            color: #e6e6e6;
            min-height: 100vh;
            padding: 20px;
        }
        
        .container {
            max-width: 1400px;
            margin: 0 auto;
        }
        
        header {
            text-align: center;
            margin-bottom: 30px;
            padding: 20px;
            background: rgba(0, 0, 0, 0.3);
            border-radius: 10px;
            border: 1px solid #2d4059;
        }
        
        h1 {
            color: #ffd166;
            font-size: 2.5em;
            margin-bottom: 10px;
            text-shadow: 0 2px 4px rgba(0, 0, 0, 0.5);
        }
        
        .subtitle {
            color: #8ac6d1;
            font-size: 1.2em;
        }
        
        .main-content {
            display: flex;
            gap: 30px;
            flex-wrap: wrap;
        }
        
        .map-container {
            flex: 1;
            min-width: 300px;
            background: rgba(0, 0, 0, 0.4);
            border-radius: 10px;
            padding: 20px;
            border: 1px solid #2d4059;
        }
        
        .info-panel {
            flex: 1;
            min-width: 300px;
            background: rgba(0, 0, 0, 0.4);
            border-radius: 10px;
            padding: 20px;
            border: 1px solid #2d4059;
        }
        
        .map-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
            gap: 15px;
            margin-top: 20px;
        }
        
        .room-node {
            background: #2d4059;
            border-radius: 8px;
            padding: 15px;
            cursor: pointer;
            transition: all 0.3s ease;
            border: 2px solid transparent;
            position: relative;
            overflow: hidden;
        }
        
        .room-node:hover {
            transform: translateY(-3px);
            box-shadow: 0 5px 15px rgba(0, 0, 0, 0.3);
            border-color: #ffd166;
        }
        
        .room-node.active {
            border-color: #ff9a76;
            background: #3a506b;
        }
        
        .room-number {
            font-weight: bold;
            color: #ffd166;
            font-size: 1.1em;
            margin-bottom: 5px;
        }
        
        .room-name {
            font-size: 0.9em;
            margin-bottom: 8px;
            line-height: 1.3;
        }
        
        .room-stats {
            font-size: 0.8em;
            color: #8ac6d1;
            display: flex;
            justify-content: space-between;
        }
        
        .room-details {
            margin-top: 20px;
            padding: 20px;
            background: rgba(0, 0, 0, 0.3);
            border-radius: 8px;
            display: none;
        }
        
        .room-details.active {
            display: block;
            animation: fadeIn 0.3s ease;
        }
        
        @keyframes fadeIn {
            from { opacity: 0; transform: translateY(10px); }
            to { opacity: 1; transform: translateY(0); }
        }
        
        .detail-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 15px;
            padding-bottom: 10px;
            border-bottom: 1px solid #2d4059;
        }
        
        .detail-title {
            font-size: 1.3em;
            color: #ffd166;
        }
        
        .detail-area {
            color: #8ac6d1;
            font-style: italic;
        }
        
        .detail-description {
            margin-bottom: 20px;
            line-height: 1.6;
        }
        
        .detail-section {
            margin-bottom: 20px;
        }
        
        .section-title {
            color: #ff9a76;
            margin-bottom: 10px;
            font-size: 1.1em;
        }
        
        .exits-list, .mobs-list, .items-list {
            list-style: none;
        }
        
        .exits-list li, .mobs-list li, .items-list li {
            padding: 8px 12px;
            margin-bottom: 5px;
            background: rgba(255, 255, 255, 0.05);
            border-radius: 5px;
            border-left: 3px solid #ffd166;
        }
        
        .exits-list li {
            border-left-color: #8ac6d1;
        }
        
        .mobs-list li {
            border-left-color: #ff9a76;
        }
        
        .items-list li {
            border-left-color: #a3de83;
        }
        
        .legend {
            display: flex;
            flex-wrap: wrap;
            gap: 15px;
            margin-top: 20px;
            padding: 15px;
            background: rgba(0, 0, 0, 0.3);
            border-radius: 8px;
        }
        
        .legend-item {
            display: flex;
            align-items: center;
            gap: 8px;
            font-size: 0.9em;
        }
        
        .legend-color {
            width: 15px;
            height: 15px;
            border-radius: 3px;
        }
        
        .controls {
            display: flex;
            gap: 10px;
            margin-bottom: 20px;
            flex-wrap: wrap;
        }
        
        .control-btn {
            padding: 8px 16px;
            background: #2d4059;
            border: none;
            border-radius: 5px;
            color: #e6e6e6;
            cursor: pointer;
            transition: background 0.3s ease;
        }
        
        .control-btn:hover {
            background: #3a506b;
        }
        
        .control-btn.active {
            background: #ff9a76;
            color: #1a1a2e;
        }
        
        footer {
            text-align: center;
            margin-top: 30px;
            padding: 20px;
            color: #8ac6d1;
            font-size: 0.9em;
            border-top: 1px solid #2d4059;
        }
        
        @media (max-width: 768px) {
            .main-content {
                flex-direction: column;
            }
            
            .map-grid {
                grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>Dark Pawns - Interactive Map</h1>
            <p class="subtitle">Explore the world of Dark Pawns MUD</p>
        </header>
        
        <div class="controls">
            <button class="control-btn active" onclick="filterRooms('all')">All Rooms</button>
            <button class="control-btn" onclick="filterRooms('indoors')">Indoors</button>
            <button class="control-btn" onclick="filterRooms('outdoors')">Outdoors</button>
            <button class="control-btn" onclick="filterRooms('dangerous')">Dangerous</button>
            <button class="control-btn" onclick="filterRooms('safe')">Safe</button>
        </div>
        
        <div class="main-content">
            <div class="map-container">
                <h2>World Map</h2>
                <div class="map-grid" id="mapGrid">
                    <!-- Room nodes will be inserted here by JavaScript -->
                </div>
                
                <div class="legend">
                    <div class="legend-item">
                        <div class="legend-color" style="background: #2d4059;"></div>
                        <span>Standard Room</span>
                    </div>
                    <div class="legend-item">
                        <div class="legend-color" style="background: #ff9a76;"></div>
                        <span>Selected Room</span>
                    </div>
                    <div class="legend-item">
                        <div class="legend-color" style="background: #8ac6d1;"></div>
                        <span>Exit Connection</span>
                    </div>
                    <div class="legend-item">
                        <div class="legend-color" style="background: #ffd166;"></div>
                        <span>Room Number</span>
                    </div>
                </div>
            </div>
            
            <div class="info-panel">
                <h2>Room Details</h2>
                <div class="room-details" id="roomDetails">
                    <!-- Room details will be inserted here by JavaScript -->
                </div>
                
                <div class="room-details active" id="welcomeMessage">
                    <div class="detail-header">
                        <div class="detail-title">Welcome to Dark Pawns Map</div>
                    </div>
                    <div class="detail-description">
                        <p>Click on any room in the map to view its details, including description, exits, mobs, and items.</p>
                        <p>Use the filter buttons above to show specific types of rooms.</p>
                    </div>
                    <div class="detail-section">
                        <div class="section-title">Map Statistics</div>
                        <p>Total Rooms: <span id="totalRooms">0</span></p>
                        <p>Areas: <span id="areasList">None</span></p>
                        <p>Total Mobs: <span id="totalMobs">0</span></p>
                        <p>Total Items: <span id="totalItems">0</span></p>
                    </div>
                </div>
            </div>
        </div>
        
        <footer>
            <p>Dark Pawns MUD Interactive Map | Generated from parsed room data</p>
            <p>Use this map for navigation, planning, and exploration of the game world.</p>
        </footer>
    </div>

    <script>
        // Room data will be injected here
        const rooms = %ROOMS_DATA%;
        
        let currentRoom = null;
        let currentFilter = 'all';
        
        // Initialize the map
        function initMap() {
            const mapGrid = document.getElementById('mapGrid');
            const totalRooms = document.getElementById('totalRooms');
            const areasList = document.getElementById('areasList');
            const totalMobs = document.getElementById('totalMobs');
            const totalItems = document.getElementById('totalItems');
            
            // Calculate statistics
            let areas = new Set();
            let mobsCount = 0;
            let itemsCount = 0;
            
            rooms.forEach(room => {
                areas.add(room.area);
                mobsCount += room.mobs.length;
                itemsCount += room.items.length;
            });
            
            totalRooms.textContent = rooms.length;
            areasList.textContent = Array.from(areas).join(', ');
            totalMobs.textContent = mobsCount;
            totalItems.textContent = itemsCount;
            
            // Render all rooms initially
            renderRooms(rooms);
        }
        
        // Render rooms to the grid
        function renderRooms(roomsToRender) {
            const mapGrid = document.getElementById('mapGrid');
            mapGrid.innerHTML = '';
            
            roomsToRender.forEach(room => {
                const roomNode = document.createElement('div');
                roomNode.className = 'room-node';
                if (currentRoom && currentRoom.vnum === room.vnum) {
                    roomNode.classList.add('active');
                }
                
                roomNode.innerHTML = `
                    <div class="room-number">#${room.vnum}</div>
                    <div class="room-name">${room.name}</div>
                    <div class="room-stats">
                        <span>${room.mobs.length} mobs</span>
                        <span>${room.items.length} items</span>
                    </div>
                `;
                
                roomNode.addEventListener('click', () => showRoomDetails(room));
                mapGrid.appendChild(roomNode);
            });
        }
        
        // Show room details
        function showRoomDetails(room) {
            currentRoom = room;
            
            // Update room nodes
            document.querySelectorAll('.room-node').forEach(node => {
                node.classList.remove('active');
            });
            
            // Show details panel
            const roomDetails = document.getElementById('roomDetails');
            const welcomeMessage = document.getElementById('welcomeMessage');
            
            welcomeMessage.classList.remove('active');
            roomDetails.classList.add('active');
            
            // Generate exits HTML
            const exitsHtml = room.exits.length > 0 
                ? room.exits.map(exit => `
                    <li>
                        <strong>${exit.direction}</strong> → Room #${exit.to_room}
                        ${exit.flags.length > 0 ? `<br><small>Flags: ${exit.flags.join(', ')}</small>` : ''}
                    </li>
                `).join('')
                : '<li>No exits</li>';
            
            // Generate mobs HTML
            const mobsHtml = room.mobs.length > 0
                ? room.mobs.map(mob => `
                    <li>
                        <strong>${mob.name}</strong> (Level ${mob.level})
                        <br><small>Type: ${mob.type}</small>
                    </li>
                `).join('')
                : '<li>No mobs</li>';
            
            // Generate items HTML
            const itemsHtml = room.items.length > 0
                ? room.items.map(item => `
                    <li>
                        <strong>${item.name}</strong>
                        <br><small>Type: ${item.type}</small>
                    </li>
                `).join('')
                : '<li>No items</li>';
            
            // Update room details
            roomDetails.innerHTML = `
                <div class="detail-header">
                    <div class="detail-title">#${room.vnum}: ${room.name}</div>
                    <div class="detail-area">${room.area}</div>
                </div>
                
                <div class="detail-description">
                    ${room.description}
                </div>
                
                <div class="detail-section">
                    <div class="section-title">Environment</div>
                    <p>Terrain: ${room.terrain} | Light: ${room.light}</p>
                </div>
                
                <div class="detail-section">
                    <div class="section-title">Exits (${room.exits.length})</div>
                    <ul class="exits-list">
                        ${exitsHtml}
                    </ul>
                </div>
                
                <div class="detail-section">
                    <div class="section-title">Mobs (${room.mobs.length})</div>
                    <ul class="mobs-list">
                        ${mobsHtml}
                    </ul>
                </div>
                
                <div class="detail-section">
                    <div class="section-title">Items (${room.items.length})</div>
                    <ul class="items-list">
                        ${itemsHtml}
                    </ul>
                </div>
            `;
            
            // Re-render rooms to update active state
            renderRooms(currentFilter === 'all' ? rooms : filterRooms(currentFilter, true));
        }
        
        // Filter rooms
        function filterRooms(filterType, internalCall = false) {
            if (!internalCall) {
                currentFilter = filterType;
                
                // Update button states
                document.querySelectorAll('.control-btn').forEach(btn => {
                    btn.classList.remove('active');
                });
                event.target.classList.add('active');
            }
            
            let filteredRooms = rooms;
            
            switch(filterType) {
                case 'indoors':
                    filteredRooms = rooms.filter(room => 
                        room.terrain === 'indoors' || 
                        room.name.toLowerCase().includes('room') ||
                        room.name.toLowerCase().includes('chamber') ||
                        room.name.toLowerCase().includes('morgue')
                    );
                    break;
                    
                case 'outdoors':
                    filteredRooms = rooms.filter(room => 
                        room.terrain === 'outdoors' ||
                        room.name.toLowerCase().includes('garden') ||
                        room.name.toLowerCase().includes('courtyard') ||
                        room.name.toLowerCase().includes('gate')
                    );
                    break;
                    
                case 'dangerous':
                    filteredRooms = rooms.filter(room => 
                        room.mobs.length > 0 && 
                        room.mobs.some(mob => mob.level > 5)
                    );
                    break;
                    
                case 'safe':
                    filteredRooms = rooms.filter(room => 
                        room.mobs.length === 0 || 
                        room.mobs.every(mob => mob.level <= 3)
                    );
                    break;
                    
                default:
                    // 'all' - show all rooms
                    break;
            }
            
            renderRooms(filteredRooms);
            
            // If current room is not in filtered list, clear details
            if (currentRoom && !filteredRooms.some(room => room.vnum === currentRoom.vnum)) {
                const roomDetails = document.getElementById('roomDetails');
                const welcomeMessage = document.getElementById('welcomeMessage');
                
                roomDetails.classList.remove('active');
                welcomeMessage.classList.add('active');
                currentRoom = null;
            }
            
            return filteredRooms;
        }
        
        // Initialize when page loads
        document.addEventListener('DOMContentLoaded', initMap);
    </script>
</body>
</html>"""
    
    # Convert rooms to JSON string for JavaScript
    rooms_json = json.dumps(rooms, indent=2)
    
    # Replace placeholder with actual data
    html_content = html_template.replace('%ROOMS_DATA%', rooms_json)
    
    return html_content

def main():
    """Main function to generate interactive map."""
    input_file = "parsed_rooms.json"
    output_file = "interactive_map.html"
    
    try:
        print(f"Loading room data from {input_file}...")
        data = load_rooms(input_file)
        rooms = data.get('rooms', [])
        
        print(f"Processing {len(rooms)} rooms...")
        html_content = generate_interactive_html(rooms)
        
        print(f"Writing interactive map to {output_file}...")
        with open(output_file, 'w') as f:
            f.write(html_content)
        
        print("Done!")
        print(f"\nInteractive map generated: {output_file}")
        print(f"Open this file in a web browser to explore the map interactively.")
        
        # Show preview
        print(f"\nHTML preview (first 10 lines):")
        print("-" * 60)
        lines = html_content.split('\n')
        for i, line in enumerate(lines[:10]):
            print(f"{i+1:3d}: {line[:80]}{'...' if len(line) > 80 else ''}")
        print("...")
        print("-" * 60)
        
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()