#!/usr/bin/env python3
"""Add Script: lines to mob files for confirmed Lua script deployments."""
import os
import re
from pathlib import Path

REPO = Path("/Users/zach/.openclaw/workspace-daeron/darkpawns_repo")
MOB_DIR = REPO / "lib" / "world" / "mob"

# Confirmed mappings: script -> [(vnum, mob_file)]
# One script per mob — no conflicts
MAPPINGS = {
    # Zone-specific scripts (from archive)
    'anhkheg.lua': [(9146, '91.mob')],
    'aurumvorax.lua': [(9147, '91.mob')],
    'bearcub.lua': [(9111, '91.mob')],
    'werewolf.lua': [(5510, '55.mob')],
    'dracula.lua': [(7903, '79.mob')],
    'head_shrinker.lua': [(7920, '79.mob')],
    'medusa.lua': [(14101, '141.mob')],
    'griffin.lua': [(2746, '27.mob')],
    'puff.lua': [(1, '0.mob')],
    'strike.lua': [(15118, '151.mob')],
    'pet_store.lua': [(21245, '212.mob')],
    'dragon_breath.lua': [(4209, '42.mob')],
    'cabinguard.lua': [(19114, '191.mob'), (19118, '191.mob')],
    'crystal_forger.lua': [(11701, '117.mob')],
    'golem_miner.lua': [(11700, '117.mob'), (11702, '117.mob')],
    'enchanter.lua': [(1314, '13.mob')],
    # Generic scripts (from archive, matched to specific mobs by name)
    'baker_flour.lua': [(8015, '80.mob')],  # baker mob in zone 80
    'beggar.lua': [(7018, '70.mob'), (8071, '80.mob')],
    'carpenter.lua': [(8067, '80.mob')],
    'citizen.lua': [(2749, '27.mob'), (2750, '27.mob'), (4802, '48.mob'), (8062, '80.mob')],
    'cityguard.lua': [(21200, '212.mob'), (21201, '212.mob'), (2747, '27.mob')],
    'clerk.lua': [(18210, '182.mob'), (18228, '182.mob'), (2767, '27.mob')],
    'eq_thief.lua': [(14223, '142.mob')],
    'troll.lua': [(12126, '121.mob'), (12130, '121.mob')],
    'tyr.lua': [(8093, '80.mob')],
    'warg.lua': [(8063, '80.mob')],
    'zealot.lua': [(8069, '80.mob')],
    'aversin.lua': [(8059, '80.mob')],
    'take_jail.lua': [(8088, '80.mob')],
    'donation.lua': [(19641, '196.mob')],
    'bhang.lua': [(14202, '142.mob')],
    # Scripts already deployed (from zone dirs)
    'blacksmith.lua': [(21210, '212.mob')],
    '144/hisc.lua': [(14412, '144.mob')],
    '122/healer.lua': [(12220, '122.mob')],
    '212/blacksmith.lua': [(21210, '212.mob')],
    # Existing generic template assignments (already have Script: lines)
    # These mobs already have scripts — skip them
}

def add_script_to_mob(filepath, vnum, script_name):
    """Add a Script: line to a mob file before the E marker for the given vnum."""
    lines = filepath.read_text(errors='replace').split('\n')
    new_lines = []
    found = False
    i = 0
    while i < len(lines):
        line = lines[i]
        stripped = line.strip()
        
        # Check if this is the vnum line
        if stripped == f'#{vnum}':
            # Find the E marker after this mob's stats
            new_lines.append(line)
            i += 1
            # Copy all lines until we hit the E marker
            while i < len(lines) and lines[i].strip() != 'E':
                new_lines.append(lines[i])
                i += 1
            # Add Script: before E
            if i < len(lines) and lines[i].strip() == 'E':
                new_lines.append(f'Script: {script_name}')
                new_lines.append(lines[i])  # E marker
                found = True
                i += 1
                # Copy remaining mobs
                while i < len(lines):
                    new_lines.append(lines[i])
                    i += 1
                break
            else:
                # E marker not found after vnum — shouldn't happen
                print(f"  WARNING: E marker not found after #{vnum} in {filepath.name}")
        else:
            new_lines.append(line)
            i += 1
    
    if found:
        filepath.write_text('\n'.join(new_lines))
    return found

def main():
    total_added = 0
    skipped = 0
    
    for script, targets in sorted(MAPPINGS.items()):
        for vnum, mob_file in targets:
            filepath = MOB_DIR / mob_file
            if not filepath.exists():
                print(f"  ERROR: {mob_file} not found")
                continue
            
            # Check if mob already has a script
            content = filepath.read_text(errors='replace')
            # Find the mob and check for existing Script: line
            lines = content.split('\n')
            has_script = False
            for i, line in enumerate(lines):
                if line.strip() == f'#{vnum}':
                    # Check lines until next # or end
                    j = i + 1
                    while j < len(lines) and not lines[j].strip().startswith('#'):
                        if lines[j].strip().startswith('Script:'):
                            has_script = True
                            break
                        j += 1
                    break
            
            if has_script:
                skipped += 1
                continue
            
            # Add the Script: line
            if add_script_to_mob(filepath, vnum, script):
                print(f"  ADDED: #{vnum} → {script}")
                total_added += 1
            else:
                print(f"  FAILED: #{vnum} in {mob_file}")
    
    print(f"\nTotal Script: lines added: {total_added}")
    print(f"Skipped (already have script): {skipped}")

if __name__ == '__main__':
    main()
