#!/usr/bin/env python3
"""
Simple visualization of Dark Pawns room graph.
Generates a simplified graph visualization for a sample of rooms.
"""

import json
import networkx as nx
import matplotlib.pyplot as plt
from pathlib import Path

def load_graph_data(json_path: str):
    """Load room graph data from JSON file."""
    with open(json_path, 'r', encoding='utf-8') as f:
        data = json.load(f)
    return data

def create_sample_graph(data, max_nodes=100):
    """Create a sample graph with limited nodes for visualization."""
    G = nx.Graph()
    
    # Take first max_nodes nodes
    nodes = data['nodes'][:max_nodes]
    edges = []
    
    # Add edges only between nodes in our sample
    node_ids = {node['id'] for node in nodes}
    for edge in data['edges']:
        if edge['from'] in node_ids and edge['to'] in node_ids:
            edges.append(edge)
    
    # Add nodes with attributes
    for node in nodes:
        G.add_node(node['id'], 
                   name=node['name'][:20],  # Truncate long names
                   zone=node['zone'],
                   flags=node['flags'])
    
    # Add edges with attributes
    for edge in edges:
        G.add_edge(edge['from'], edge['to'], 
                   direction=edge['direction'],
                   door_state=edge['door_state'])
    
    return G, nodes, edges

def visualize_graph(G, output_path: str):
    """Visualize the graph and save to file."""
    plt.figure(figsize=(20, 15))
    
    # Use spring layout for better visualization
    pos = nx.spring_layout(G, k=0.5, iterations=50)
    
    # Color nodes by zone
    zones = [G.nodes[node]['zone'] for node in G.nodes()]
    unique_zones = list(set(zones))
    zone_colors = plt.cm.tab20(range(len(unique_zones)))
    zone_color_map = {zone: zone_colors[i] for i, zone in enumerate(unique_zones)}
    node_colors = [zone_color_map[G.nodes[node]['zone']] for node in G.nodes()]
    
    # Draw nodes
    nx.draw_networkx_nodes(G, pos, node_color=node_colors, node_size=100, alpha=0.8)
    
    # Draw edges
    nx.draw_networkx_edges(G, pos, alpha=0.3, edge_color='gray')
    
    # Draw labels (room vnums)
    labels = {node: str(node) for node in G.nodes()}
    nx.draw_networkx_labels(G, pos, labels, font_size=8)
    
    # Create legend for zones
    import matplotlib.patches as mpatches
    legend_patches = []
    for zone in sorted(unique_zones)[:10]:  # Show first 10 zones
        patch = mpatches.Patch(color=zone_color_map[zone], label=f'Zone {zone}')
        legend_patches.append(patch)
    
    plt.legend(handles=legend_patches, loc='upper right', fontsize=8)
    plt.title(f'Dark Pawns Room Graph (Sample: {len(G.nodes())} rooms, {len(G.edges())} connections)')
    plt.axis('off')
    plt.tight_layout()
    
    # Save figure
    plt.savefig(output_path, dpi=150, bbox_inches='tight')
    print(f"Graph visualization saved to: {output_path}")
    
    # Also save a text summary
    summary_path = output_path.replace('.png', '_summary.txt')
    with open(summary_path, 'w', encoding='utf-8') as f:
        f.write(f"Graph Summary:\n")
        f.write(f"==============\n")
        f.write(f"Total nodes: {len(G.nodes())}\n")
        f.write(f"Total edges: {len(G.edges())}\n")
        f.write(f"Average degree: {sum(dict(G.degree()).values()) / len(G.nodes()):.2f}\n")
        f.write(f"Connected components: {nx.number_connected_components(G)}\n")
        
        # Zone distribution
        f.write(f"\nZone distribution in sample:\n")
        zone_counts = {}
        for node in G.nodes():
            zone = G.nodes[node]['zone']
            zone_counts[zone] = zone_counts.get(zone, 0) + 1
        
        for zone, count in sorted(zone_counts.items(), key=lambda x: x[1], reverse=True):
            f.write(f"  Zone {zone}: {count} rooms\n")
    
    print(f"Graph summary saved to: {summary_path}")

def generate_zone_report(data, output_path: str):
    """Generate a detailed zone report."""
    zones = {}
    
    # Group rooms by zone
    for node in data['nodes']:
        zone = node['zone']
        if zone not in zones:
            zones[zone] = {
                'room_count': 0,
                'rooms': [],
                'exits': 0,
                'flags': {}
            }
        zones[zone]['room_count'] += 1
        zones[zone]['rooms'].append(node['id'])
        
        # Count flags
        for flag in node['flags']:
            zones[zone]['flags'][flag] = zones[zone]['flags'].get(flag, 0) + 1
    
    # Count exits per zone
    for edge in data['edges']:
        # Find zone of source room
        for node in data['nodes']:
            if node['id'] == edge['from']:
                zone = node['zone']
                if zone in zones:
                    zones[zone]['exits'] += 1
                break
    
    # Write report
    with open(output_path, 'w', encoding='utf-8') as f:
        f.write("DARK PAWNS ZONE ANALYSIS REPORT\n")
        f.write("=" * 60 + "\n\n")
        
        for zone in sorted(zones.keys()):
            zone_data = zones[zone]
            f.write(f"ZONE {zone}\n")
            f.write(f"-" * 40 + "\n")
            f.write(f"Rooms: {zone_data['room_count']}\n")
            f.write(f"Exits: {zone_data['exits']}\n")
            f.write(f"Exits per room: {zone_data['exits']/zone_data['room_count']:.2f}\n")
            
            if zone_data['flags']:
                f.write(f"Flags:\n")
                for flag, count in sorted(zone_data['flags'].items(), key=lambda x: x[1], reverse=True):
                    percentage = count / zone_data['room_count'] * 100
                    f.write(f"  {flag}: {count} ({percentage:.1f}%)\n")
            
            # Sample rooms
            f.write(f"Sample rooms (first 5): {', '.join(map(str, zone_data['rooms'][:5]))}\n")
            f.write("\n")
    
    print(f"Zone analysis report saved to: {output_path}")

def main():
    """Main function for graph visualization."""
    data_dir = Path(__file__).parent
    graph_json = data_dir / "room_graph.json"
    
    print("Loading graph data...")
    data = load_graph_data(graph_json)
    
    print(f"Loaded {len(data['nodes'])} rooms and {len(data['edges'])} exits")
    
    # Create sample graph for visualization
    print("\nCreating sample graph (first 200 rooms)...")
    G, nodes, edges = create_sample_graph(data, max_nodes=200)
    
    # Visualize
    output_image = data_dir / "room_graph_sample.png"
    visualize_graph(G, output_image)
    
    # Generate zone report
    zone_report = data_dir / "zone_analysis.txt"
    generate_zone_report(data, zone_report)
    
    # Generate simple connectivity analysis
    connectivity_path = data_dir / "connectivity_analysis.txt"
    with open(connectivity_path, 'w', encoding='utf-8') as f:
        f.write("CONNECTIVITY ANALYSIS\n")
        f.write("=" * 60 + "\n\n")
        
        # Count rooms by exit count
        exit_counts = {}
        for node in data['nodes']:
            # Count exits for this room
            room_exits = 0
            for edge in data['edges']:
                if edge['from'] == node['id']:
                    room_exits += 1
            
            exit_counts[room_exits] = exit_counts.get(room_exits, 0) + 1
        
        f.write("Rooms by number of exits:\n")
        for exit_count in sorted(exit_counts.keys()):
            room_count = exit_counts[exit_count]
            percentage = room_count / len(data['nodes']) * 100
            f.write(f"  {exit_count} exits: {room_count} rooms ({percentage:.1f}%)\n")
        
        # Door states
        door_states = {0: 'open', 1: 'closed', 2: 'locked'}
        door_counts = {0: 0, 1: 0, 2: 0}
        for edge in data['edges']:
            state = edge.get('door_state', 0)
            if state in door_counts:
                door_counts[state] += 1
        
        f.write("\nDoor states:\n")
        for state, count in door_counts.items():
            if count > 0:
                percentage = count / len(data['edges']) * 100
                f.write(f"  {door_states[state]}: {count} exits ({percentage:.1f}%)\n")
    
    print(f"Connectivity analysis saved to: {connectivity_path}")
    
    print("\n" + "=" * 60)
    print("VISUALIZATION COMPLETE")
    print("=" * 60)
    print(f"Output files in: {data_dir}")
    print(f"1. Graph visualization: {output_image}")
    print(f"2. Zone analysis: {zone_report}")
    print(f"3. Connectivity analysis: {connectivity_path}")

if __name__ == "__main__":
    main()