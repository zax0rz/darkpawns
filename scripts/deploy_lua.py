#!/usr/bin/env python3
"""Deploy Lua scripts from archive to live world directories.

Reads vnum references from script comments/code and creates the directory
structure: lib/world/scripts/mob/<vnum>/<script>.lua

Also adds Script: lines to mob files where missing.
"""
import os
import re
import shutil
from pathlib import Path

REPO = Path(__file__).parent.parent
ARCHIVE = REPO / "test_scripts" / "mob" / "archive"
MOB_DIR = REPO / "lib" / "world" / "mob"
DEPLOYED = REPO / "lib" / "world" / "scripts" / "mob"

# Scripts already deployed (skip these)
DEPLOYED_NAMES = {f.name for f in DEPLOYED.rglob("*.lua")}

# Scripts that are generic templates (deploy to all matching mobs, not zone-specific)
GENERIC_TEMPLATES = {
    'cleric.lua', 'fighter.lua', 'magic_user.lua', 'sorcery.lua',
    'beggar.lua', 'citizen.lua', 'cityguard.lua', 'shopkeeper.lua',
    'backstabber.lua', 'thief.lua', 'mercenary.lua',
    'banker.lua', 'prisoner.lua', 'jailguard.lua', 'guard_captain.lua',
    'identifier.lua', 'teacher.lua', 'healer.lua',
    'merchant_inn.lua', 'merchant_walk.lua', 'shop_give.lua',
    'teleporter.lua', 'town_teleport.lua', 'teleport_vict.lua',
    'stable.lua', 'minstrel.lua', 'petitioner.lua', 'recruiter.lua',
    'tattoo.lua', 'rescuer.lua', 'paladin.lua',
    'janitor.lua', 'carpenter.lua', 'miller.lua', 'clerk.lua',
    'conjured.lua', 'creation.lua', 'minion.lua',
    'never_die.lua', 'no_get.lua', 'remove_curse.lua',
    'weatherworker.lua', 'towncrier.lua', 'singingdrunk.lua',
    'mime.lua', 'hermit.lua',
}

def extract_vnums(path):
    """Extract mob vnum references from a Lua script."""
    text = path.read_text(errors='replace')
    vnums = set()
    
    # Comment patterns
    for m in re.finditer(r'(?:attached to|target|source|mob|vnum)\s+(\d{4,5})', text, re.I):
        vnums.add(int(m.group(1)))
    
    # Code patterns
    for m in re.finditer(r'(?:\.vnum\s*==\s*|\.vnum\s*=\s*)(\d{4,5})', text):
        vnums.add(int(m.group(1)))
    
    # dofile references
    dofiles = []
    for m in re.finditer(r'dofile\(["\']([^"\']+)["\']', text):
        dofiles.append(m.group(1))
    
    return vnums, dofiles, text

def parse_mobs():
    """Parse all mob files and return dict of vnum -> mob info."""
    mobs = {}
    for f in sorted(MOB_DIR.glob("*.mob")):
        text = f.read_text(errors='replace')
        current = None
        for line in text.split('\n'):
            line_stripped = line.strip()
            
            m = re.match(r'^#(\d+)$', line_stripped)
            if m:
                if current:
                    mobs[current['vnum']] = current
                current = {
                    'vnum': int(m.group(1)),
                    'name': '',
                    'short_desc': '',
                    'level': 0,
                    'has_script': False,
                    'script_name': '',
                    'file': f.name,
                }
                continue
            
            if current is None:
                continue
            
            if current['name'] == '' and '~' not in line_stripped:
                current['name'] = line_stripped
                continue
            
            if current['short_desc'] == '' and '~' not in line_stripped and current['name']:
                current['short_desc'] = line_stripped
                continue
            
            if current['level'] == 0 and re.match(r'^\d+', line_stripped):
                try:
                    current['level'] = int(line_stripped.split()[0])
                except:
                    pass
            
            if line_stripped.startswith('Script:'):
                current['has_script'] = True
                parts = line_stripped.split()
                if len(parts) >= 2:
                    current['script_name'] = parts[1]
        
        if current:
            mobs[current['vnum']] = current
    
    return mobs

def main():
    mobs = parse_mobs()
    
    # Categorize scripts
    specific = []  # scripts with clear vnum references
    generic = []   # templates
    unknown = []   # can't determine target
    
    for f in sorted(ARCHIVE.glob("*.lua")):
        if f.name in DEPLOYED_NAMES:
            continue
        
        vnums, dofiles, text = extract_vnums(f)
        
        if f.name in GENERIC_TEMPLATES:
            generic.append((f, vnums, dofiles))
        elif vnums:
            specific.append((f, vnums, dofiles))
        else:
            unknown.append((f, vnums, dofiles))
    
    print(f"Specific scripts (have vnum refs): {len(specific)}")
    print(f"Generic templates: {len(generic)}")
    print(f"Unknown (need manual review): {len(unknown)}")
    print(f"Already deployed (skipped): {len(DEPLOYED_NAMES)}")
    
    # Deploy specific scripts
    deployed_count = 0
    for f, vnums, dofiles in specific:
        for vnum in vnums:
            if vnum not in mobs:
                print(f"  WARNING: {f.name} references mob #{vnum} which doesn't exist")
                continue
            
            mob = mobs[vnum]
            if mob['has_script']:
                # Already has a script, check if it's the same
                if mob['script_name'] == f.name:
                    continue  # Already correct
                else:
                    print(f"  CONFLICT: #{vnum} {mob['name']} already has {mob['script_name']}, would overwrite with {f.name}")
                    continue
            
            # Create directory and copy
            dest_dir = DEPLOYED / str(vnum)
            dest_dir.mkdir(parents=True, exist_ok=True)
            dest_file = dest_dir / f.name
            
            if not dest_file.exists():
                shutil.copy2(f, dest_file)
                print(f"  DEPLOY: {f.name} → {vnum}/ (mob: {mob['name'][:40]})")
                deployed_count += 1
    
    print(f"\nDeployed {deployed_count} specific scripts")
    
    # Report unknowns
    if unknown:
        print(f"\nUNKNOWN SCRIPTS (need manual review):")
        for f, vnums, dofiles in unknown:
            triggers = []
            text = f.read_text(errors='replace')
            for fname in ['fight', 'death', 'oncmd', 'onpulse_pc', 'onpulse_all', 
                          'ongive', 'sound', 'onmove', 'onact', 'onhear', 'code']:
                if re.search(rf'function\s+{fname}\s*\(', text):
                    triggers.append(fname)
            print(f"  {f.name}: triggers={','.join(triggers) if triggers else 'none'}, dofile={dofiles}")

if __name__ == '__main__':
    main()
