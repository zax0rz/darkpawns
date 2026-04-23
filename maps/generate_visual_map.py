#!/usr/bin/env python3
"""
Generate visual map (SVG/PNG) using Graphviz from parsed room data.
"""

import json
import subprocess
import sys
import os
from typing import Dict, List

def load_rooms(data_file: str) -> Dict:
    """Load parsed room data from JSON file."""
    with open(data_file, 'r') as f:
        data = json.load(f)
    return data

def generate_graphviz_dot(rooms: List[Dict]) -> str:
    """Generate Graphviz DOT format for the map."""
    dot_lines = []
    dot_lines.append('digraph DarkPawnsMap {')
    dot_lines.append('  rankdir=TB;')
    dot_lines.append('  node [shape=box, style=filled, fillcolor=lightblue, fontname="Helvetica"];')
    dot_lines.append('  edge [fontname="Helvetica", fontsize=10];')
    dot_lines.append('  graph [bgcolor="transparent"];')
    dot_lines.append('')
    
    # Add nodes (rooms)
    for room in rooms:
        vnum = room['vnum']
        name = room['name']
        area = room.get('area', 'Unknown')
        mobs_count = len(room.get('mobs', []))
        items_count = len(room.get('items', []))
        
        # Determine node color based on area/type
        fillcolor = 'lightblue'
        if 'secret' in name.lower() or 'laboratory' in name.lower():
            fillcolor = 'lightcoral'
        elif 'garden' in name.lower() or 'courtyard' in name.lower():
            fillcolor = 'lightgreen'
        elif 'cavern' in name.lower() or 'underground' in name.lower():
            fillcolor = 'wheat'
        elif 'morgue' in name.lower():
            fillcolor = 'gray'
        
        # Create node label
        label_lines = []
        label_lines.append(f'<b>#{vnum}: {name}</b>')
        label_lines.append(f'Area: {area}')
        if mobs_count > 0:
            label_lines.append(f'Mobs: {mobs_count}')
        if items_count > 0:
            label_lines.append(f'Items: {items_count}')
        
        label = '\\n'.join(label_lines)
        
        dot_lines.append(f'  room{vnum} [label="{label}", fillcolor="{fillcolor}"];')
    
    dot_lines.append('')
    
    # Add edges (exits)
    for room in rooms:
        vnum = room['vnum']
        for exit_info in room.get('exits', []):
            direction = exit_info.get('direction', '')
            to_room = exit_info.get('to_room')
            flags = exit_info.get('flags', [])
            
            if to_room:
                # Check if this is a two-way connection
                is_two_way = False
                target_room = next((r for r in rooms if r['vnum'] == to_room), None)
                if target_room:
                    for target_exit in target_room.get('exits', []):
                        if target_exit.get('to_room') == vnum:
                            is_two_way = True
                            break
                
                # Edge styling
                edge_attrs = []
                edge_attrs.append(f'label="{direction}"')
                
                if 'locked' in flags:
                    edge_attrs.append('color="red"')
                    edge_attrs.append('style="dashed"')
                elif 'door' in flags or 'gate' in flags:
                    edge_attrs.append('color="brown"')
                elif 'water' in flags:
                    edge_attrs.append('color="blue"')
                elif 'trapdoor' in flags:
                    edge_attrs.append('color="purple"')
                    edge_attrs.append('style="dotted"')
                else:
                    edge_attrs.append('color="black"')
                
                if not is_two_way:
                    edge_attrs.append('dir="forward"')
                
                edge_attr_str = ', '.join(edge_attrs)
                dot_lines.append(f'  room{vnum} -> room{to_room} [{edge_attr_str}];')
    
    dot_lines.append('}')
    return '\n'.join(dot_lines)

def generate_svg(dot_content: str, output_file: str) -> bool:
    """Generate SVG file from DOT content using Graphviz."""
    try:
        # Write DOT file
        dot_file = output_file.replace('.svg', '.dot')
        with open(dot_file, 'w') as f:
            f.write(dot_content)
        
        # Run Graphviz
        cmd = ['dot', '-Tsvg', dot_file, '-o', output_file]
        result = subprocess.run(cmd, capture_output=True, text=True)
        
        if result.returncode != 0:
            print(f"Graphviz error: {result.stderr}", file=sys.stderr)
            return False
        
        print(f"SVG generated: {output_file}")
        return True
        
    except FileNotFoundError:
        print("Error: Graphviz 'dot' command not found. Please install Graphviz.", file=sys.stderr)
        return False
    except Exception as e:
        print(f"Error generating SVG: {e}", file=sys.stderr)
        return False

def generate_png(dot_content: str, output_file: str) -> bool:
    """Generate PNG file from DOT content using Graphviz."""
    try:
        # Write DOT file
        dot_file = output_file.replace('.png', '.dot')
        with open(dot_file, 'w') as f:
            f.write(dot_content)
        
        # Run Graphviz
        cmd = ['dot', '-Tpng', dot_file, '-o', output_file]
        result = subprocess.run(cmd, capture_output=True, text=True)
        
        if result.returncode != 0:
            print(f"Graphviz error: {result.stderr}", file=sys.stderr)
            return False
        
        print(f"PNG generated: {output_file}")
        return True
        
    except FileNotFoundError:
        print("Error: Graphviz 'dot' command not found. Please install Graphviz.", file=sys.stderr)
        return False
    except Exception as e:
        print(f"Error generating PNG: {e}", file=sys.stderr)
        return False

def main():
    """Main function to generate visual maps."""
    input_file = "parsed_rooms.json"
    svg_output = "visual_map.svg"
    png_output = "visual_map.png"
    
    try:
        print(f"Loading room data from {input_file}...")
        data = load_rooms(input_file)
        rooms = data.get('rooms', [])
        
        print(f"Processing {len(rooms)} rooms...")
        dot_content = generate_graphviz_dot(rooms)
        
        # Save DOT file
        dot_file = "map_graph.dot"
        with open(dot_file, 'w') as f:
            f.write(dot_content)
        print(f"DOT file saved: {dot_file}")
        
        # Generate SVG
        print("Generating SVG map...")
        if generate_svg(dot_content, svg_output):
            print(f"SVG map generated: {svg_output}")
        else:
            print("Failed to generate SVG map")
        
        # Generate PNG
        print("Generating PNG map...")
        if generate_png(dot_content, png_output):
            print(f"PNG map generated: {png_output}")
        else:
            print("Failed to generate PNG map")
        
        print("\nMap generation complete!")
        print(f"\nFiles created:")
        print(f"  - {dot_file} (Graphviz source)")
        print(f"  - {svg_output} (Visual map - SVG)")
        print(f"  - {png_output} (Visual map - PNG)")
        
        # Show preview of DOT file
        print(f"\nDOT file preview (first 20 lines):")
        print("-" * 60)
        lines = dot_content.split('\n')
        for i, line in enumerate(lines[:20]):
            print(f"{i+1:3d}: {line}")
        if len(lines) > 20:
            print("...")
        print("-" * 60)
        
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()