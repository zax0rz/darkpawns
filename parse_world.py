#!/usr/bin/env python3
"""
Dark Pawns World Parser
Parses .wld files to extract room data and build a room graph.
"""

import os
import json
import csv
from pathlib import Path
from typing import Dict, List, Any, Optional
import re

class Room:
    """Represents a parsed room from a .wld file."""
    def __init__(self, vnum: int, name: str, description: str, zone: int, 
                 flags: List[str], sector: int, exits: Dict[str, Dict]):
        self.vnum = vnum
        self.name = name
        self.description = description
        self.zone = zone
        self.flags = flags
        self.sector = sector
        self.exits = exits  # direction -> {to_room, door_state, key, keywords, description}
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert room to dictionary for JSON serialization."""
        return {
            "vnum": self.vnum,
            "name": self.name,
            "description": self.description,
            "zone": self.zone,
            "flags": self.flags,
            "sector": self.sector,
            "exits": self.exits
        }

class WorldParser:
    """Parser for Dark Pawns .wld files."""
    
    # Direction mapping from D0-D5 to direction names
    DIRECTION_MAP = {
        0: "north",
        1: "east", 
        2: "south",
        3: "west",
        4: "up",
        5: "down"
    }
    
    # Room flag bitmask mapping (from parser/wld.go)
    ROOM_FLAGS = {
        0: "dark",
        1: "death",
        2: "nomob",
        3: "indoors",
        4: "peaceful",
        5: "soundproof",
        6: "notrack",
        7: "nomagic",
        8: "tunnel",
        9: "private",
        10: "godroom",
        11: "house",
        12: "house_crash",
        13: "atrium",
        14: "olc",
        15: "bspace"
    }
    
    def __init__(self, world_dir: str):
        self.world_dir = Path(world_dir)
        self.rooms: Dict[int, Room] = {}
        self.zones: Dict[int, List[int]] = {}  # zone -> list of room vnums
        
    def parse_all_wld_files(self) -> None:
        """Parse all .wld files in the world directory."""
        wld_dir = self.world_dir / "wld"
        if not wld_dir.exists():
            raise FileNotFoundError(f"World directory not found: {wld_dir}")
        
        for wld_file in wld_dir.glob("*.wld"):
            print(f"Parsing {wld_file.name}...")
            self.parse_wld_file(wld_file)
        
        print(f"Parsed {len(self.rooms)} rooms from {len(list(wld_dir.glob('*.wld')))} files")
    
    def parse_wld_file(self, file_path: Path) -> None:
        """Parse a single .wld file."""
        with open(file_path, 'r', encoding='utf-8', errors='ignore') as f:
            lines = f.readlines()
        
        i = 0
        while i < len(lines):
            line = lines[i].strip()
            
            # Skip empty lines and comments
            if not line or line.startswith('*'):
                i += 1
                continue
            
            # Room starts with #<vnum>
            if line.startswith('#'):
                vnum_str = line[1:].strip()
                
                # Special case: #99999 is end-of-world marker
                if vnum_str == '99999':
                    break
                
                if vnum_str == '$':
                    i += 1
                    continue
                
                try:
                    vnum = int(vnum_str)
                except ValueError:
                    i += 1
                    continue
                
                # Parse the room
                room, new_i = self._parse_room(lines, i, vnum)
                if room:
                    self.rooms[vnum] = room
                    # Track zone membership
                    if room.zone not in self.zones:
                        self.zones[room.zone] = []
                    self.zones[room.zone].append(vnum)
                
                i = new_i
            else:
                i += 1
    
    def _parse_room(self, lines: List[str], start_idx: int, vnum: int) -> tuple[Optional[Room], int]:
        """Parse a single room starting at start_idx."""
        i = start_idx + 1
        
        # Parse name (ends with ~)
        if i >= len(lines):
            return None, i
        name = lines[i].rstrip('~\n').strip()
        i += 1
        
        # Parse description (ends with ~)
        desc_lines = []
        while i < len(lines):
            line = lines[i]
            if '~' in line:
                # Handle case where ~ might be in the middle of line
                if line.strip().endswith('~'):
                    desc_lines.append(line.rstrip('~\n'))
                    i += 1
                    break
                else:
                    # Find the first ~
                    tilde_idx = line.find('~')
                    desc_lines.append(line[:tilde_idx])
                    # Check if there's more on this line after ~
                    rest = line[tilde_idx + 1:].strip()
                    if rest:
                        # This shouldn't happen in valid files, but handle it
                        pass
                    i += 1
                    break
            else:
                desc_lines.append(line.rstrip('\n'))
                i += 1
        
        description = '\n'.join(desc_lines).strip()
        
        # Parse numeric line: zone flags sector 0 0 0
        if i >= len(lines):
            return None, i
        
        nums_line = lines[i].strip()
        nums = nums_line.split()
        if len(nums) < 6:
            # Try to recover
            i += 1
            return None, i
        
        try:
            zone = int(nums[0])
            flags_bitmask = int(nums[1])
            sector = int(nums[2])
        except ValueError:
            i += 1
            return None, i
        
        # Parse room flags
        flags = self._parse_room_flags(flags_bitmask)
        
        i += 1
        
        # Parse exits and other sections until 'S' or next room
        exits = {}
        while i < len(lines):
            line = lines[i].strip()
            
            if line == 'S':
                i += 1
                break
            
            if line.startswith('#'):
                # Next room
                break
            
            if line.startswith('D') and len(line) == 2:
                try:
                    dir_num = int(line[1])
                    if 0 <= dir_num <= 5:
                        direction = self.DIRECTION_MAP[dir_num]
                        exit_data, new_i = self._parse_exit(lines, i, direction)
                        if exit_data:
                            exits[direction] = exit_data
                        i = new_i
                        continue
                except ValueError:
                    pass
            
            i += 1
        
        room = Room(vnum, name, description, zone, flags, sector, exits)
        return room, i
    
    def _parse_room_flags(self, bitmask: int) -> List[str]:
        """Convert bitmask to list of flag names."""
        flags = []
        for bit, name in self.ROOM_FLAGS.items():
            if bitmask & (1 << bit):
                flags.append(name)
        return flags
    
    def _parse_exit(self, lines: List[str], start_idx: int, direction: str) -> tuple[Optional[Dict], int]:
        """Parse an exit section."""
        i = start_idx + 1
        
        # Exit description (ends with ~)
        if i >= len(lines):
            return None, i
        
        exit_desc = lines[i].rstrip('~\n').strip()
        i += 1
        
        # Keywords (ends with ~)
        if i >= len(lines):
            return None, i
        
        keywords = lines[i].rstrip('~\n').strip()
        i += 1
        
        # Numeric line: door_state key to_room
        if i >= len(lines):
            return None, i
        
        nums_line = lines[i].strip()
        nums = nums_line.split()
        if len(nums) < 3:
            return None, i
        
        try:
            door_state = int(nums[0])
            key = int(nums[1])
            to_room = int(nums[2])
        except ValueError:
            return None, i
        
        i += 1
        
        exit_data = {
            "direction": direction,
            "to_room": to_room,
            "door_state": door_state,
            "key": key,
            "keywords": keywords,
            "description": exit_desc
        }
        
        return exit_data, i
    
    def build_room_graph(self) -> Dict[str, Any]:
        """Build a graph representation of room connections."""
        nodes = []
        edges = []
        
        for vnum, room in self.rooms.items():
            nodes.append({
                "id": vnum,
                "name": room.name,
                "zone": room.zone,
                "flags": room.flags,
                "sector": room.sector
            })
            
            for direction, exit_data in room.exits.items():
                edges.append({
                    "from": vnum,
                    "to": exit_data["to_room"],
                    "direction": direction,
                    "door_state": exit_data["door_state"],
                    "key": exit_data["key"]
                })
        
        return {
            "nodes": nodes,
            "edges": edges,
            "zones": {zone: room_list for zone, room_list in self.zones.items()}
        }
    
    def export_json(self, output_path: str) -> None:
        """Export room data as JSON."""
        data = {
            "rooms": {vnum: room.to_dict() for vnum, room in self.rooms.items()},
            "statistics": {
                "total_rooms": len(self.rooms),
                "rooms_with_exits": sum(1 for room in self.rooms.values() if room.exits),
                "total_exits": sum(len(room.exits) for room in self.rooms.values()),
                "zones": len(self.zones),
                "rooms_per_zone": {zone: len(rooms) for zone, rooms in self.zones.items()}
            }
        }
        
        with open(output_path, 'w', encoding='utf-8') as f:
            json.dump(data, f, indent=2, ensure_ascii=False)
    
    def export_csv(self, output_dir: str) -> None:
        """Export room data as CSV files."""
        output_dir = Path(output_dir)
        output_dir.mkdir(exist_ok=True)
        
        # Rooms CSV
        rooms_csv = output_dir / "rooms.csv"
        with open(rooms_csv, 'w', newline='', encoding='utf-8') as f:
            writer = csv.writer(f)
            writer.writerow(["vnum", "name", "zone", "flags", "sector", "description_preview"])
            
            for vnum, room in self.rooms.items():
                desc_preview = room.description[:100] + "..." if len(room.description) > 100 else room.description
                writer.writerow([vnum, room.name, room.zone, ";".join(room.flags), room.sector, desc_preview])
        
        # Exits CSV
        exits_csv = output_dir / "exits.csv"
        with open(exits_csv, 'w', newline='', encoding='utf-8') as f:
            writer = csv.writer(f)
            writer.writerow(["from_vnum", "to_vnum", "direction", "door_state", "key", "keywords"])
            
            for vnum, room in self.rooms.items():
                for direction, exit_data in room.exits.items():
                    writer.writerow([
                        vnum, 
                        exit_data["to_room"], 
                        direction, 
                        exit_data["door_state"], 
                        exit_data["key"],
                        exit_data["keywords"]
                    ])
        
        # Zones CSV
        zones_csv = output_dir / "zones.csv"
        with open(zones_csv, 'w', newline='', encoding='utf-8') as f:
            writer = csv.writer(f)
            writer.writerow(["zone", "room_count", "room_vnums"])
            
            for zone, room_vnums in self.zones.items():
                writer.writerow([zone, len(room_vnums), ";".join(map(str, room_vnums))])
    
    def generate_report(self) -> str:
        """Generate a parsing report."""
        report_lines = []
        report_lines.append("=" * 80)
        report_lines.append("DARK PAWNS WORLD PARSING REPORT")
        report_lines.append("=" * 80)
        report_lines.append(f"World directory: {self.world_dir}")
        report_lines.append(f"Total rooms parsed: {len(self.rooms)}")
        report_lines.append(f"Total zones: {len(self.zones)}")
        
        # Room statistics
        rooms_with_exits = sum(1 for room in self.rooms.values() if room.exits)
        total_exits = sum(len(room.exits) for room in self.rooms.values())
        report_lines.append(f"Rooms with exits: {rooms_with_exits} ({rooms_with_exits/len(self.rooms)*100:.1f}%)")
        report_lines.append(f"Total exits: {total_exits}")
        report_lines.append(f"Average exits per room: {total_exits/len(self.rooms):.2f}")
        
        # Zone statistics
        report_lines.append("\nZone Statistics:")
        report_lines.append("-" * 40)
        for zone in sorted(self.zones.keys()):
            room_count = len(self.zones[zone])
            report_lines.append(f"Zone {zone}: {room_count} rooms")
        
        # Flag statistics
        flag_counts = {}
        for room in self.rooms.values():
            for flag in room.flags:
                flag_counts[flag] = flag_counts.get(flag, 0) + 1
        
        if flag_counts:
            report_lines.append("\nRoom Flag Statistics:")
            report_lines.append("-" * 40)
            for flag, count in sorted(flag_counts.items(), key=lambda x: x[1], reverse=True):
                percentage = count / len(self.rooms) * 100
                report_lines.append(f"{flag}: {count} rooms ({percentage:.1f}%)")
        
        # Sector statistics
        sector_counts = {}
        for room in self.rooms.values():
            sector_counts[room.sector] = sector_counts.get(room.sector, 0) + 1
        
        if sector_counts:
            report_lines.append("\nSector Type Statistics:")
            report_lines.append("-" * 40)
            for sector, count in sorted(sector_counts.items(), key=lambda x: x[1], reverse=True):
                percentage = count / len(self.rooms) * 100
                report_lines.append(f"Sector {sector}: {count} rooms ({percentage:.1f}%)")
        
        # Top 10 largest rooms by description length
        sorted_rooms = sorted(self.rooms.values(), key=lambda r: len(r.description), reverse=True)[:10]
        report_lines.append("\nTop 10 Rooms by Description Length:")
        report_lines.append("-" * 40)
        for room in sorted_rooms:
            report_lines.append(f"#{room.vnum} '{room.name}' - {len(room.description)} chars")
        
        report_lines.append("\n" + "=" * 80)
        return "\n".join(report_lines)

def main():
    """Main function to parse world files."""
    # Path to world files
    world_dir = "/home/zach/.openclaw/workspace/rparet-darkpawns/lib"
    output_dir = "/home/zach/.openclaw/workspace/darkpawns_repo/world_data"
    
    print("Starting Dark Pawns World Parser...")
    print(f"World directory: {world_dir}")
    
    # Create parser
    parser = WorldParser(world_dir)
    
    # Parse all .wld files
    print("\nParsing .wld files...")
    parser.parse_all_wld_files()
    
    # Create output directory
    os.makedirs(output_dir, exist_ok=True)
    
    # Export data
    print("\nExporting data...")
    
    # JSON export
    json_path = os.path.join(output_dir, "rooms.json")
    parser.export_json(json_path)
    print(f"  JSON data exported to: {json_path}")
    
    # CSV export
    csv_dir = os.path.join(output_dir, "csv")
    parser.export_csv(csv_dir)
    print(f"  CSV data exported to: {csv_dir}")
    
    # Graph data
    graph_data = parser.build_room_graph()
    graph_path = os.path.join(output_dir, "room_graph.json")
    with open(graph_path, 'w', encoding='utf-8') as f:
        json.dump(graph_data, f, indent=2, ensure_ascii=False)
    print(f"  Room graph exported to: {graph_path}")
    
    # Generate report
    report = parser.generate_report()
    report_path = os.path.join(output_dir, "parsing_report.txt")
    with open(report_path, 'w', encoding='utf-8') as f:
        f.write(report)
    print(f"  Parsing report exported to: {report_path}")
    
    # Print summary
    print("\n" + "=" * 80)
    print("PARSING COMPLETE")
    print("=" * 80)
    print(report.split("\n")[2:15])  # Print first part of report
    print("\nFull report available at:", report_path)

if __name__ == "__main__":
    main()