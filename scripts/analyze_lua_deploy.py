#!/usr/bin/env python3
"""Analyze Lua scripts and mob files to build deployment mapping."""
import os
import re
from pathlib import Path

REPO = Path(__file__).parent.parent
ARCHIVE = REPO / "test_scripts" / "mob" / "archive"
MOB_DIR = REPO / "lib" / "world" / "mob"
DEPLOYED = REPO / "lib" / "world" / "scripts" / "mob"

def extract_vnums_from_script(path):
    """Extract mob vnum references from a Lua script."""
    text = path.read_text(errors='replace')
    vnums = set()
    
    # Comment patterns: "Attached to mob 12915", "mob 9147", "vnum 14401"
    for m in re.finditer(r'(?:attached to|target|source|mob|vnum)\s+(\d{4,5})', text, re.I):
        vnums.add(int(m.group(1)))
    
    # Code patterns: obj.vnum == 12915, room.char[i].vnum == 9111
    for m in re.finditer(r'(?:\.vnum\s*==\s*|\.vnum\s*=\s*)(\d{4,5})', text):
        vnums.add(int(m.group(1)))
    
    # dofile references
    dofiles = []
    for m in re.finditer(r'dofile\(["\']([^"\']+)["\']', text):
        dofiles.append(m.group(1))
    
    # Determine trigger types
    triggers = []
    for fname in ['fight', 'death', 'oncmd', 'onpulse_pc', 'onpulse_all', 
                   'ongive', 'sound', 'onmove', 'onact', 'onhear', 'code']:
        if re.search(rf'function\s+{fname}\s*\(', text):
            triggers.append(fname)
    
    return vnums, dofiles, triggers, text

def parse_mob_file(path):
    """Parse a .mob file and extract mob entries."""
    mobs = []
    text = path.read_text(errors='replace')
    
    current_mob = None
    for line in text.split('\n'):
        line = line.strip()
        
        # Mob vnum line: #14401
        m = re.match(r'^#(\d+)$', line)
        if m:
            if current_mob:
                mobs.append(current_mob)
            current_mob = {
                'vnum': int(m.group(1)),
                'name': '',
                'short_desc': '',
                'level': 0,
                'has_script': False,
                'script_name': '',
                'zone_file': path.name,
            }
            continue
        
        if current_mob is None:
            continue
            
        # Name line (first ~ line after vnum)
        if current_mob['name'] == '' and '~' not in line:
            current_mob['name'] = line
            continue
        
        # Short description
        if current_mob['short_desc'] == '' and '~' not in line and current_mob['name'] != '':
            current_mob['short_desc'] = line
            continue
        
        # Level line: numeric stats
        if current_mob['level'] == 0 and re.match(r'^\d+', line):
            parts = line.split()
            if len(parts) >= 1:
                try:
                    current_mob['level'] = int(parts[0])
                except:
                    pass
        
        # Script line
        if line.startswith('Script:'):
            current_mob['has_script'] = True
            parts = line.split()
            if len(parts) >= 2:
                current_mob['script_name'] = parts[1]
    
    if current_mob:
        mobs.append(current_mob)
    
    return mobs

def main():
    # 1. Analyze all scripts
    print("=" * 80)
    print("SCRIPT ANALYSIS")
    print("=" * 80)
    
    scripts = {}
    for f in sorted(ARCHIVE.glob("*.lua")):
        vnums, dofiles, triggers, text = extract_vnums_from_script(f)
        scripts[f.name] = {
            'vnums': vnums,
            'dofiles': dofiles,
            'triggers': triggers,
            'first_line': text.split('\n')[0][:100] if text else '',
        }
    
    # 2. Analyze all mobs
    print("\n" + "=" * 80)
    print("MOB INVENTORY")
    print("=" * 80)
    
    all_mobs = []
    for f in sorted(MOB_DIR.glob("*.mob")):
        mobs = parse_mob_file(f)
        all_mobs.extend(mobs)
    
    # 3. Build mapping
    print("\n" + "=" * 80)
    print("SCRIPT → MOB MAPPING")
    print("=" * 80)
    
    # Already deployed
    deployed_scripts = set()
    for d in DEPLOYED.rglob("*.lua"):
        deployed_scripts.add(d.name)
    
    for script_name, info in sorted(scripts.items()):
        base = script_name.replace('.lua', '')
        vnums = info['vnums']
        triggers = info['triggers']
        
        # Find matching mobs
        candidates = []
        for mob in all_mobs:
            if mob['vnum'] in vnums:
                candidates.append(mob)
            # Name matching
            mob_name_lower = mob['name'].lower().replace(' ', '_').replace('-', '_')
            if base in mob_name_lower or mob_name_lower.startswith(base):
                candidates.append(mob)
        
        # Deduplicate
        seen = set()
        unique_candidates = []
        for c in candidates:
            if c['vnum'] not in seen:
                seen.add(c['vnum'])
                unique_candidates.append(c)
        
        status = "DEPLOYED" if script_name in deployed_scripts else "ARCHIVE"
        has_script_line = any(m['has_script'] for m in unique_candidates)
        
        print(f"\n{script_name} [{status}]")
        print(f"  Triggers: {', '.join(triggers) if triggers else '(none)'}")
        print(f"  Vnums in code: {vnums if vnums else '(none found)'}")
        print(f"  dofile refs: {info['dofiles'] if info['dofiles'] else '(none)'}")
        if unique_candidates:
            for m in unique_candidates[:5]:
                script_flag = " [HAS SCRIPT]" if m['has_script'] else ""
                print(f"  → #{m['vnum']} {m['name']} (L{m['level']}){script_flag}")
        else:
            print(f"  → NO MATCHING MOBS FOUND")
    
    # 4. Mobs without scripts
    print("\n" + "=" * 80)
    print("MOBS WITH SCRIPT: LINE (already wired)")
    print("=" * 80)
    for mob in all_mobs:
        if mob['has_script']:
            print(f"  #{mob['vnum']} {mob['name']} → {mob['script_name']}")
    
    print(f"\nTotal mobs: {len(all_mobs)}")
    print(f"Mobs with scripts: {sum(1 for m in all_mobs if m['has_script'])}")
    print(f"Total scripts: {len(scripts)}")
    print(f"Deployed scripts: {len(deployed_scripts)}")

if __name__ == '__main__':
    main()
