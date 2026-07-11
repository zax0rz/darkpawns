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

def get_trigger_bitmask(script_name):
    paths = [
        os.path.join("lib/world/scripts/mob", script_name),
        os.path.join("test_scripts/mob/archive", os.path.basename(script_name)),
        os.path.join("test_scripts/mob", script_name)
    ]
    
    path = None
    for p in paths:
        if os.path.exists(p):
            path = p
            break
            
    if not path:
        if "fighter" in script_name or "cleric" in script_name or "magic_user" in script_name or "sorcery" in script_name:
            return 256
        if "shopkeeper" in script_name or "banker" in script_name:
            return 24
        return 0
        
    with open(path, 'r', encoding='utf-8', errors='ignore') as f:
        content = f.read()
        
    # Simple regex to find defined functions: function name(
    funcs = re.findall(r'function\s+(\w+)\s*\(', content)
    
    # Also check if it's delegating via dofile/call
    if "dofile" in content:
        if "fighter.lua" in content:
            funcs.append("fight")
        if "magic_user.lua" in content or "sorcery.lua" in content:
            funcs.append("fight")
        if "cityguard.lua" in content:
            funcs.append("fight")
            funcs.append("onpulse_pc")
        if "take_jail.lua" in content:
            funcs.append("oncmd")
            
    bitmask = 0
    mapping = {
        "bribe": 2,
        "greet": 4,
        "ongive": 8,
        "sound": 16,
        "death": 32,
        "onpulse_all": 64,
        "onpulse_pc": 128,
        "fight": 256,
        "oncmd": 512
    }
    for func in funcs:
        if func in mapping:
            bitmask |= mapping[func]
            
    if bitmask == 0:
        if "fighter" in script_name or "cleric" in script_name or "magic_user" in script_name or "sorcery" in script_name:
            return 256
        if "shopkeeper" in script_name or "banker" in script_name:
            return 24
            
    return bitmask

def add_script_to_mob(mob_file, vnum, script_name):
    """Add or update a Script: line to a mob definition in a .mob file.

    Each mob block runs from ``#<vnum>`` up to (but not including) the next
    ``\\n#<number>`` or end of file. The Script: line is inserted just before
    the final standalone ``E`` terminator *within that bounded block*, so it
    can never land in a neighbouring mob's block. The file is written via a
    temp file + atomic replace.
    """
    with open(mob_file, 'r') as f:
        content = f.read()

    # Bound the target mob's block: from its #vnum line to the next #<number>
    # line (or EOF). This prevents the Script line from crossing into an
    # adjacent mob block when the target block is missing its own terminator.
    block_start_pattern = rf'(?m)^#{re.escape(str(vnum))}\s*$'
    block_start = re.search(block_start_pattern, content)
    if not block_start:
        return False, "mob not found"

    rest = content[block_start.end():]
    # Next mob starts at a line beginning with #<digits>.
    next_mob = re.search(r'(?m)^#\-?\d+\s*$', rest)
    block_body = rest[:next_mob.start()] if next_mob else rest
    block_body_end_rel = (next_mob.start() if next_mob else len(rest))

    # Find the LAST standalone 'E' line within the bounded block. A standalone
    # E is an 'E' on its own line (not '... 1000 E').
    e_matches = list(re.finditer(r'(?m)^E\s*$', block_body))
    if not e_matches:
        return False, "mob block has no E terminator"
    last_e = e_matches[-1]

    # mob_block is everything in the block before the final E line (excluding
    # the newline that separates them); end_marker is that separating newline
    # plus the E line, preserved verbatim.
    e_line_start = last_e.start()
    # Include the preceding newline in the end marker so the Script line we
    # insert lands on its own line before E.
    if e_line_start > 0 and block_body[e_line_start - 1] == '\n':
        sep_newline = '\n'
        mob_block = block_body[:e_line_start - 1]
    else:
        sep_newline = ''
        mob_block = block_body[:e_line_start]
    end_marker = sep_newline + block_body[e_line_start:]

    bitmask = get_trigger_bitmask(script_name)
    script_line = f'Script: {script_name}'
    if bitmask > 0:
        script_line += f' {bitmask}'

    # Check if a Script: line already exists in the block
    script_match = re.search(r'Script:\s+(\S+)(?:\s+(\d+))?', mob_block)

    if script_match:
        existing_script = script_match.group(1)
        existing_bitmask = int(script_match.group(2)) if script_match.group(2) else 0

        # If it matches exactly (same script and same bitmask), skip
        if existing_script == script_name and existing_bitmask == bitmask:
            return False, "already correctly wired"

        # Otherwise, replace the existing script line
        new_mob_block = mob_block.replace(script_match.group(0).strip(), script_line)
        new_block = new_mob_block + end_marker
        msg = f"updated bitmask {existing_bitmask} → {bitmask}"
    else:
        # Add Script: line before the final E
        new_block = mob_block + f'\n{script_line}' + end_marker
        msg = f"wired (bitmask: {bitmask})"

    # Reassemble: prefix + block_start line + new_block + remainder after block.
    prefix = content[:block_start.end()]
    suffix = rest[block_body_end_rel:]
    new_content = prefix + new_block + suffix

    # Atomic write via temp file + os.replace.
    tmp_path = mob_file + '.tmp'
    with open(tmp_path, 'w') as f:
        f.write(new_content)
    os.replace(tmp_path, mob_file)

    return True, msg

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
            bitmask = get_trigger_bitmask(script_name)
            print(f"  WOULD WIRE: {vnum} → {script_name} (bitmask: {bitmask}) in {os.path.basename(mob_file)}")
            wired += 1
        else:
            ok, msg = add_script_to_mob(mob_file, vnum, script_name)
            if ok:
                print(f"  ✓ {vnum} → {script_name} ({msg}) in {os.path.basename(mob_file)}")
                wired += 1
            else:
                print(f"  SKIP: {vnum} → {script_name}: {msg}")
                skipped += 1
    
    print(f"\n{'Would wire' if dry_run else 'Wired'}: {wired}, Skipped: {skipped}, Failed: {failed}")

if __name__ == '__main__':
    main()
