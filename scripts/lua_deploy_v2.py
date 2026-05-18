#!/usr/bin/env python3
"""Comprehensive Lua deployment — match scripts to mobs and generate edits."""
import os
import re
from pathlib import Path

REPO = Path("/Users/zach/.openclaw/workspace-daeron/darkpawns_repo")
ARCHIVE = REPO / "test_scripts" / "mob" / "archive"
MOB_DIR = REPO / "lib" / "world" / "mob"
DEPLOYED = REPO / "lib" / "world" / "scripts" / "mob"

# Parse all mobs into a dict: vnum -> {name, desc, keywords, file, line_offset, has_script}
def parse_all_mobs():
    mobs = {}
    for f in sorted(MOB_DIR.glob("*.mob")):
        lines = f.read_text(errors='replace').split('\n')
        current = None
        line_num = 0
        for i, line in enumerate(lines):
            stripped = line.strip()
            m = re.match(r'^#(\d+)$', stripped)
            if m:
                if current:
                    mobs[current['vnum']] = current
                current = {
                    'vnum': int(m.group(1)),
                    'name': '',
                    'short': '',
                    'desc': '',
                    'keywords': '',
                    'file': f.name,
                    'line_num': i,
                    'has_script': False,
                    'script_name': '',
                    'level': 0,
                }
                line_num = i
                continue
            if current is None:
                continue
            if current['name'] == '' and '~' not in stripped:
                current['name'] = stripped
                continue
            if current['short'] == '' and '~' not in stripped:
                current['short'] = stripped
                continue
            # Collect desc lines until ~
            if current['desc'] == '' and '~' not in stripped:
                current['desc'] = stripped[:200]
                # Next line is keywords
                if i+1 < len(lines) and '~' not in lines[i+1].strip():
                    current['keywords'] = lines[i+1].strip().rstrip('~')
                continue
            # Level line
            if current['level'] == 0 and re.match(r'^\d+\s+\d+', stripped):
                try:
                    current['level'] = int(stripped.split()[0])
                except:
                    pass
            if stripped.startswith('Script:'):
                current['has_script'] = True
                parts = stripped.split()
                if len(parts) >= 2:
                    current['script_name'] = parts[1]
        if current:
            mobs[current['vnum']] = current
    return mobs

# Known mappings from script analysis
KNOWN_MAPPINGS = {
    # Script filename -> list of mob vnums
    'aurumvorax.lua': [9147],
    'baker_dough.lua': [8015],  # note: 8015 is "Eldrich the sorcerer" — script is about baking, may be wrong
    'baker_flour.lua': [8015, 15100],
    'bearcub.lua': [9111],
    'cabinguard.lua': [19114, 19118],
    'crystal_forger.lua': [11701],
    'dragon_breath.lua': [4209],
    'dragon_forger.lua': [7917, 7903],
    'enchanter.lua': [1314],
    'guardian.lua': [1314],
    'head_shrinker.lua': [7920],
    'golem_from_crate.lua': [11701, 11702],
    'golem_miner.lua': [11700, 11701, 11702],
    'golem_to_crate.lua': [11700, 11701, 11702],
    'pet_store.lua': [21245],
    'werewolf.lua': [5510],
    # Scripts matched by name search
    'anhkheg.lua': [9146],
    'dracula.lua': [7903],
    'medusa.lua': [14101],
    'griffin.lua': [2746],
    'bhang.lua': [14202],
    'beggar.lua': [7018, 8071],
    'carpenter.lua': [8067],
    'citizen.lua': [2749, 2750, 4802, 8062],
    'cityguard.lua': [21200, 21201, 2747],
    'clerk.lua': [18210, 18228, 2767],
    'donation.lua': [19641],
    'eq_thief.lua': [14223],
    'troll.lua': [14408, 12126, 12130],
    'tyr.lua': [8093],
    'warg.lua': [8063],
    'zealot.lua': [8069],
    'aversin.lua': [8059],
    'take_jail.lua': [8088],
    'puff.lua': [1],
    'snake.lua': [1315],
    'strike.lua': [15118],
    # Scripts with no clear mob — generic combat AI, deploy to matching mob types
    'backstabber.lua': [],  # generic: any stealthy mob
    'thief.lua': [],  # generic: any thief mob
    'mercenary.lua': [],  # generic: any mercenary mob
    'shopkeeper.lua': [],  # generic: any shopkeeper
    'banker.lua': [],  # generic: any banker
    'prisoner.lua': [],  # generic: any prisoner
    'jailguard.lua': [],  # generic: any jail guard
    'guard_captain.lua': [],  # generic: any guard captain
    'identifier.lua': [],  # generic: any identifier
    'teacher.lua': [],  # generic: any teacher/trainer
    'merchant_inn.lua': [],  # generic: any inn merchant
    'merchant_walk.lua': [],  # generic: any walking merchant
    'shop_give.lua': [],  # generic: any give-based shop
    'teleporter.lua': [],  # generic: any teleporter
    'town_teleport.lua': [],  # generic: any town teleport
    'teleport_vict.lua': [],  # generic: any teleport-vict
    'stable.lua': [],  # generic: any stable
    'minstrel.lua': [],  # generic: any minstrel
    'petitioner.lua': [],  # generic: any petitioner
    'recruiter.lua': [],  # generic: any recruiter
    'tattoo.lua': [],  # generic: any tattoo artist
    'rescuer.lua': [],  # generic: any rescuer
    'paladin.lua': [],  # generic: any paladin
    'janitor.lua': [],  # generic: any janitor
    'miller.lua': [],  # generic: any miller
    'hermit.lua': [],  # generic: any hermit
    'mime.lua': [],  # generic: any mime
    'towncrier.lua': [],  # generic: any town crier
    'singingdrunk.lua': [],  # generic: any singing drunk
    'weatherworker.lua': [],  # generic: any weather worker
    'remove_curse.lua': [],  # generic: any curse remover
    'no_get.lua': [],  # generic: prevents item pickup
    'never_die.lua': [],  # generic: immortal mob
    'creation.lua': [],  # generic: summoning
    'conjured.lua': [],  # generic: summoned creature
    'minion.lua': [],  # generic: any minion
    # No-mob scripts (skip these)
    'autodraw.lua': [],  # needs create_event — skip
    'breed_killer.lua': [],  # complex multi-script — skip for now
    'bane.lua': [],  # needs valoran.lua pair — skip for now
    'valoran.lua': [],  # needs bane.lua pair — skip for now
    # Scripts I can't match
    'bradle.lua': [],
    'brain_eater.lua': [],
    'caerroil.lua': [],
    'cuchi.lua': [],
    'drake.lua': [],
    'elven_prostitute.lua': [],
    'ettin.lua': [],
    'gazer.lua': [],
    'kelpie.lua': [],
    'memory_moss.lua': [],
    'mindflayer.lua': [],
    'mount.lua': [],
    'mymic.lua': [],
    'neckbreak.lua': [],
    'paralyse.lua': [],
    'porcupine.lua': [],
    'quanlo.lua': [],
    'seiji.lua': [],
    'thornslinger.lua': [],
    'keep_sorcerer.lua': [],
    'souleater.lua': [],
    'sungod.lua': [],
    'farmer_wheat.lua': [],
    'fire_ant.lua': [],
    'fire_ant_larva.lua': [],
    'forester.lua': [],
    'triflower.lua': [],
}

def main():
    mobs = parse_all_mobs()
    
    # Find mobs by keyword search for unmatched scripts
    def find_mobs_by_keywords(keywords, exclude_vnums=None):
        if exclude_vnums is None:
            exclude_vnums = set()
        results = []
        for vnum, mob in mobs.items():
            if vnum in exclude_vnums or mob['has_script']:
                continue
            searchable = f"{mob['name']} {mob['short']} {mob['desc']} {mob['keywords']}".lower()
            if all(kw.lower() in searchable for kw in keywords):
                results.append(mob)
        return results
    
    # Try to match unmatched scripts by searching mob descriptions
    unmatched_lookups = {
        'bradle': [],  # No match in mobs
        'brain_eater': [],  # No match  
        'caerroil': [],  # No match
        'cuchi': ['cuchi'],
        'drake': ['drake'],
        'elven_prostitute': ['elven', 'prostitute'],
        'ettin': ['ettin'],
        'gazer': ['gazer'],
        'kelpie': ['kelpie'],
        'memory_moss': ['memory', 'moss'],
        'mindflayer': ['mind', 'flayer'],
        'mount': ['mount'],
        'mymic': ['mymic'],
        'neckbreak': ['neck', 'break'],
        'paralyse': ['paralyse', 'paralyze'],
        'porcupine': ['porcupine'],
        'quanlo': ['quanlo'],
        'seiji': ['seiji'],
        'thornslinger': ['thorn', 'sling'],
        'keep_sorcerer': ['keep', 'sorcerer'],
        'souleater': ['soul', 'eater'],
        'sungod': ['sun', 'god'],
        'farmer_wheat': ['farmer', 'wheat'],
        'fire_ant': ['fire', 'ant'],
        'fire_ant_larva': ['fire', 'ant', 'larva'],
        'forester': ['forester'],
        'triflower': ['triflower', 'tri', 'flower'],
    }
    
    print("=== UNMATCHED SCRIPT LOOKUPS ===")
    for script, keywords in sorted(unmatched_lookups.items()):
        if not keywords:
            print(f"  {script}: NO KEYWORDS")
            continue
        matches = find_mobs_by_keywords(keywords)
        if matches:
            for m in matches[:3]:
                print(f"  {script} → #{m['vnum']} {m['name'][:50]}")
        else:
            print(f"  {script}: NO MOBS FOUND")
    
    # Generate Script: line additions for mobs
    print("\n=== SCRIPT LINE ADDITIONS NEEDED ===")
    additions = 0
    for script, vnums in sorted(KNOWN_MAPPINGS.items()):
        if not vnums:
            continue
        for vnum in vnums:
            if vnum in mobs and not mobs[vnum]['has_script']:
                mob = mobs[vnum]
                print(f"ADD: #{vnum} ({mob['file']}) → Script: {script}")
                additions += 1
    print(f"\nTotal additions needed: {additions}")
    
    # Summary
    print(f"\n=== SUMMARY ===")
    print(f"Total mobs: {len(mobs)}")
    print(f"Mobs with scripts: {sum(1 for m in mobs.values() if m['has_script'])}")
    print(f"Mobs without scripts: {sum(1 for m in mobs.values() if not m['has_script'])}")
    print(f"Scripts with clear mappings: {sum(1 for v in KNOWN_MAPPINGS.values() if v)}")
    print(f"Scripts with no mapping: {sum(1 for v in KNOWN_MAPPINGS.values() if not v)}")

if __name__ == '__main__':
    main()
