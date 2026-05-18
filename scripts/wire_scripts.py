#!/usr/bin/env python3
"""Wire Lua scripts to mobs in .mob files.

Usage: python3 wire_scripts.py [--dry-run]

Reads wiring_plan.txt (format: vnum|script_name) and adds Script: lines
to the appropriate mob files.
"""
import sys
import re
import os

MOB_DIR = "lib/world/mob"
PLAN_FILE = "scripts/wiring_plan_v2.txt"

def add_script_to_mob(mob_file, vnum, script_name):
    """Add a Script: line to a mob definition in a .mob file."""
    with open(mob_file, 'r') as f:
        content = f.read()
    
    # Find the mob definition: starts with #VNUM, ends with E on its own line
    # We need to find the LAST E before the next # or end of file
    pattern = rf'(#{re.escape(str(vnum))}\n.*?)(\nE\n)'
    match = re.search(pattern, content, re.DOTALL)
    
    if not match:
        return False, "mob not found"
    
    mob_block = match.group(1)
    end_marker = match.group(2)
    
    # Check if Script: already exists
    if f'Script: {script_name}' in mob_block:
        return False, "already wired"
    
    # Add Script: line before the final E
    new_block = mob_block + f'\nScript: {script_name}' + end_marker
    content = content[:match.start()] + new_block + content[match.end():]
    
    with open(mob_file, 'w') as f:
        f.write(content)
    
    return True, "wired"

def main():
    dry_run = '--dry-run' in sys.argv
    
    if not os.path.exists(PLAN_FILE):
        print(f"Plan file not found: {PLAN_FILE}")
        sys.exit(1)
    
    with open(PLAN_FILE, 'r') as f:
        lines = [l.strip() for l in f if l.strip() and not l.startswith('#') and '|' in l]
    
    wired = 0
    skipped = 0
    failed = 0
    
    for line in lines:
        parts = line.split('|')
        if len(parts) != 2:
            print(f"  SKIP: invalid line: {line}")
            skipped += 1
            continue
        
        vnum_str, script_name = parts
        vnum = int(vnum_str)
        
        # Find which mob file contains this vnum
        mob_file = None
        for fname in os.listdir(MOB_DIR):
            if fname.endswith('.mob') and not fname.startswith('.'):
                fpath = os.path.join(MOB_DIR, fname)
                with open(fpath, 'r') as f:
                    for l in f:
                        if l.strip() == f'#{vnum}':
                            mob_file = fpath
                            break
                if mob_file:
                    break
        
        if not mob_file:
            print(f"  FAIL: vnum {vnum} not found in any mob file")
            failed += 1
            continue
        
        if dry_run:
            print(f"  WOULD WIRE: {vnum} → {script_name} in {os.path.basename(mob_file)}")
            wired += 1
        else:
            ok, msg = add_script_to_mob(mob_file, vnum, script_name)
            if ok:
                print(f"  ✓ {vnum} → {script_name} in {os.path.basename(mob_file)}")
                wired += 1
            else:
                print(f"  SKIP: {vnum} → {script_name}: {msg}")
                skipped += 1
    
    print(f"\n{'Would wire' if dry_run else 'Wired'}: {wired}, Skipped: {skipped}, Failed: {failed}")

if __name__ == '__main__':
    main()
