#!/usr/bin/env python3
"""
Generate ASCII text map from parsed room data.
"""

import json
import sys
from typing import Dict, List, Set

def load_rooms(data_file: str) -> Dict:
    """Load parsed room data from JSON file."""
    with open(data_file, 'r') as f:
        data = json.load(f)
    return data

def create_room_grid(rooms: List[Dict]) -> Dict[int, Dict[str, int]]:
    """Create a simple grid positioning for rooms based on exit connections."""
    # Simple layout algorithm: start from room 100, place rooms in grid
    positions = {}
    visited = set()
    
    # Start with room 100 at position (0, 0)
    queue = [(100, 0, 0)]
    
    # Direction vectors
    dir_vectors = {
        'north': (0, -1),
        'south': (0, 1),
        'east': (1, 0),
        'west': (-1, 0),
        'up': (0, 0, 1),
        'down': (0, 0, -1)
    }
    
    room_dict = {room['vnum']: room for room in rooms}
    
    while queue:
        vnum, x, y = queue.pop(0)
        if vnum in visited:
            continue
            
        visited.add(vnum)
        positions[vnum] = {'x': x, 'y': y}
        
        if vnum not in room_dict:
            continue
            
        room = room_dict[vnum]
        for exit_info in room.get('exits', []):
            direction = exit_info.get('direction', '')
            to_room = exit_info.get('to_room')
            
            if direction in dir_vectors and to_room and to_room not in visited:
                dx, dy = dir_vectors[direction][:2]
                queue.append((to_room, x + dx, y + dy))
    
    return positions

def generate_ascii_map(rooms: List[Dict], positions: Dict[int, Dict[str, int]]) -> str:
    """Generate ASCII representation of the map."""
    if not positions:
        return "No rooms to display."
    
    # Find bounds
    min_x = min(pos['x'] for pos in positions.values())
    max_x = max(pos['x'] for pos in positions.values())
    min_y = min(pos['y'] for pos in positions.values())
    max_y = max(pos['y'] for pos in positions.values())
    
    # Create grid
    width = (max_x - min_x + 1) * 4
    height = (max_y - min_y + 1) * 3
    grid = [[' ' for _ in range(width)] for _ in range(height)]
    
    # Room dictionary for quick lookup
    room_dict = {room['vnum']: room for room in rooms}
    
    # Draw rooms and connections
    for vnum, pos in positions.items():
        grid_x = (pos['x'] - min_x) * 4 + 2
        grid_y = (pos['y'] - min_y) * 3 + 1
        
        # Draw room as a box
        room = room_dict.get(vnum, {})
        room_name = room.get('name', f'Room {vnum}')[:10]
        
        # Room box
        if grid_y - 1 >= 0 and grid_y + 1 < height:
            if grid_x - 1 >= 0:
                grid[grid_y - 1][grid_x - 1] = '+'
                grid[grid_y + 1][grid_x - 1] = '+'
            if grid_x + 1 < width:
                grid[grid_y - 1][grid_x + 1] = '+'
                grid[grid_y + 1][grid_x + 1] = '+'
            if grid_y - 1 >= 0:
                for dx in range(-1, 2):
                    if 0 <= grid_x + dx < width:
                        grid[grid_y - 1][grid_x + dx] = '-'
            if grid_y + 1 < height:
                for dx in range(-1, 2):
                    if 0 <= grid_x + dx < width:
                        grid[grid_y + 1][grid_x + dx] = '-'
            if grid_x - 1 >= 0:
                grid[grid_y][grid_x - 1] = '|'
            if grid_x + 1 < width:
                grid[grid_y][grid_x + 1] = '|'
        
        # Room number/name
        if 0 <= grid_y < height and 0 <= grid_x < width:
            grid[grid_y][grid_x] = str(vnum)[-1]  # Last digit of room number
        
        # Draw exits
        room = room_dict.get(vnum, {})
        for exit_info in room.get('exits', []):
            direction = exit_info.get('direction', '')
            to_room = exit_info.get('to_room')
            
            if to_room in positions:
                target_pos = positions[to_room]
                target_x = (target_pos['x'] - min_x) * 4 + 2
                target_y = (target_pos['y'] - min_y) * 3 + 1
                
                # Draw connection line
                dx = target_x - grid_x
                dy = target_y - grid_y
                
                # Simple line drawing
                steps = max(abs(dx), abs(dy))
                if steps > 0:
                    for i in range(1, steps):
                        x = grid_x + (dx * i) // steps
                        y = grid_y + (dy * i) // steps
                        if 0 <= y < height and 0 <= x < width:
                            if grid[y][x] == ' ':
                                grid[y][x] = '.'
    
    # Convert grid to string
    result = []
    result.append("=" * 60)
    result.append("DARK PAWNS - ASCII MAP")
    result.append("=" * 60)
    result.append("")
    
    for y in range(height):
        line = ''.join(grid[y])
        if line.strip():  # Only add non-empty lines
            result.append(line)
    
    result.append("")
    result.append("=" * 60)
    result.append("LEGEND:")
    result.append("  [0-9] - Room number (last digit)")
    result.append("  +     - Room corner")
    result.append("  -|    - Room walls")
    result.append("  .     - Connection path")
    result.append("=" * 60)
    result.append("")
    
    # Add room list
    result.append("ROOM LIST:")
    result.append("-" * 40)
    for room in sorted(rooms, key=lambda r: r['vnum']):
        exits = ', '.join([f"{e['direction']}→{e['to_room']}" for e in room.get('exits', [])])
        result.append(f"#{room['vnum']:3d} {room['name'][:30]:30} [{exits}]")
    
    return '\n'.join(result)

def main():
    """Main function to generate text map."""
    input_file = "parsed_rooms.json"
    output_file = "text_map.txt"
    
    try:
        print(f"Loading room data from {input_file}...")
        data = load_rooms(input_file)
        rooms = data.get('rooms', [])
        
        print(f"Processing {len(rooms)} rooms...")
        positions = create_room_grid(rooms)
        
        print("Generating ASCII map...")
        ascii_map = generate_ascii_map(rooms, positions)
        
        print(f"Writing map to {output_file}...")
        with open(output_file, 'w') as f:
            f.write(ascii_map)
        
        print("Done!")
        print(f"\nPreview of generated map:\n")
        print(ascii_map[:500] + "..." if len(ascii_map) > 500 else ascii_map)
        
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()