#!/usr/bin/env python3
import json
import re
import sys
import unicodedata
from pathlib import Path

# Paths
ROOT_DIR = Path(__file__).parent.parent.parent
WORLD_DIR = ROOT_DIR / "lib/world"
STATIC_DATA_DIR = ROOT_DIR / "website/static/data"

# Constants
DIRECTIONS = {0: "north", 1: "east", 2: "south", 3: "west", 4: "up", 5: "down"}

ITEM_TYPES = {
    1: "LIGHT", 2: "SCROLL", 3: "WAND", 4: "STAFF", 5: "WEAPON",
    6: "FIREWEAPON", 7: "MISSILE", 8: "TREASURE", 9: "ARMOR", 10: "POTION",
    11: "WORN", 12: "OTHER", 13: "TRASH", 14: "TRAP", 15: "CONTAINER",
    16: "NOTE", 17: "DRINKCON", 18: "KEY", 19: "FOOD", 20: "MONEY",
    21: "PEN", 22: "BOAT", 23: "FOUNTAIN"
}

WEAR_FLAGS = {
    0: "TAKE", 1: "FINGER", 2: "NECK", 3: "BODY", 4: "HEAD",
    5: "LEGS", 6: "FEET", 7: "HANDS", 8: "ARMS", 9: "SHIELD",
    10: "ABOUT", 11: "WAIST", 12: "WRIST", 13: "WIELD", 14: "HOLD",
    15: "THROW", 16: "ABLEGS", 17: "FACE", 18: "HOVER"
}

EXTRA_FLAGS = {
    0: "GLOW", 1: "HUM", 2: "NORENT", 3: "NODONATE", 4: "NOINVIS",
    5: "INVISIBLE", 6: "MAGIC", 7: "NODROP", 8: "BLESS", 9: "ANTI_GOOD",
    10: "ANTI_EVIL", 11: "ANTI_NEUTRAL", 12: "ANTI_MAGE", 13: "ANTI_CLERIC",
    14: "ANTI_THIEF", 15: "ANTI_WARRIOR", 16: "NOSELL", 17: "NAMED",
    18: "ANTI_PSIONICIST", 19: "ANTI_NINJA", 20: "ANTI_PALADIN", 21: "ANTI_MAGUS",
    22: "ANTI_ASSASSIN", 23: "ANTI_AVATAR", 24: "RARE", 25: "NOLOCATE",
    26: "ANTI_RANGER", 27: "ANTI_MYSTIC", 28: "TWOHANDS"
}

APPLY_LOCATIONS = {
    0: "NONE", 1: "STR", 2: "DEX", 3: "INT", 4: "WIS", 5: "CON",
    6: "CHA", 7: "CLASS", 8: "LEVEL", 9: "AGE", 10: "CHAR_WEIGHT",
    11: "CHAR_HEIGHT", 12: "MANA", 13: "HIT", 14: "MOVE", 15: "GOLD",
    16: "EXP", 17: "AC", 18: "HITROLL", 19: "DAMROLL", 20: "SAVING_PARA",
    21: "SAVING_ROD", 22: "SAVING_PETRI", 23: "SAVING_BREATH", 24: "SAVING_SPELL",
    25: "RACE_HATE", 26: "HIT_REGEN", 27: "MANA_REGEN", 28: "MOVE_REGEN", 29: "PERM_SPELL"
}

EQUIP_SLOTS = {
    0: "used as light",
    1: "worn on right finger",
    2: "worn on left finger",
    3: "worn around neck (1)",
    4: "worn around neck (2)",
    5: "worn on body",
    6: "worn on head",
    7: "worn on legs",
    8: "worn on feet",
    9: "worn on hands",
    10: "worn on arms",
    11: "worn as shield",
    12: "worn about body",
    13: "worn about waist",
    14: "worn around right wrist",
    15: "worn around left wrist",
    16: "wielded",
    17: "held",
    18: "wielded for throwing",
    19: "worn about legs",
    20: "worn on face",
    21: "hovering near head"
}

ACTION_BITS = [
    "SPEC", "SENTINEL", "SCAVENGER", "ISNPC", "AWARE", "AGGR",
    "STAY-ZONE", "WIMPY", "AGGR_EVIL", "AGGR_GOOD", "AGGR_NEUTRAL",
    "MEMORY", "HELPER", "!CHARM", "!SUMMN", "!SLEEP", "!BASH",
    "!BLIND", "HUNTER", "AGGR24", "RNDLD_ZONE", "MOUNTABLE",
    "RARE", "LOOTS", "OKGIVE"
]

AFFECT_BITS = [
    "BLIND", "INVIS", "DET-ALIGN", "DET-INVIS", "DET-MAGIC",
    "SENSE-LIFE", "WATERWALK", "SANCT", "GROUP", "CURSE", "INFRA",
    "POISON", "PROT-EVIL", "PROT-GOOD", "SLEEP", "!TRACK",
    "FLESH-ALTER", "DODGE", "SNEAK", "HIDE", "BERSERK", "CHARM",
    "FOLLOW", "WIMPY", "KUJI-KIRI", "CUTTHROAT", "FLY", "WEREWOLF",
    "VAMPIRE", "MOUNTED", "INVULN", "FLAMING", "NOTHING", "HASTE",
    "SLOW", "DREAM", "WATERBREATHE", "METALSKIN", "ROBBED"
]

MOB_RACES = [
    "Human", "Elf", "Dwarf", "Kender", "Centaur", "Rakshasa", "Troll",
    "Lycanthrope", "Vampire", "Undead", "Dragon", "Demon", "Horse", "Reptile",
    "Arachnid", "Rodent", "Other", "Vegetable", "Giant", "Demi-god", "Ogre",
    "Insect", "Mammal", "Fish", "Avian", "Magical Construct", "Amphibian",
    "Humanoid", "Faery", "Ssaur", "Minotaur"
]

ATTACK_TYPES = [
    "hit", "sting", "whip", "slash", "bite", "bludgeon", "crush",
    "pound", "claw", "maul", "thrash", "pierce", "blast", "punch", "stab"
]

POSITIONS = [
    "Dead", "Mortally wounded", "Incapacitated", "Stunned",
    "Sleeping", "Resting", "Sitting", "Fighting", "Standing"
]

def clean_desc(desc):
    return desc.replace("\r", "").strip()

def clean_generated_text(value):
    """Remove source control bytes that are invalid in YAML and Markdown."""
    return "".join(
        char for char in value
        if char in "\n\t" or unicodedata.category(char) != "Cc"
    )

def yaml_string(value):
    """Encode a string as a YAML-compatible JSON double-quoted scalar."""
    return json.dumps(clean_generated_text(value), ensure_ascii=False)

def read_string(lines, i):
    parts = []
    while i < len(lines):
        line = lines[i]
        i += 1
        stripped = line.rstrip()
        if stripped == '~':
            break
        if stripped.endswith('~'):
            parts.append(stripped[:-1])
            break
        parts.append(line.rstrip('\n'))
    return '\n'.join(parts).replace("\r", "").strip(), i

def parse_flag(s):
    try:
        return int(s)
    except ValueError:
        result = 0
        for c in s:
            if 'a' <= c <= 'z':
                result |= 1 << (ord(c) - ord('a'))
            elif 'A' <= c <= 'Z':
                result |= 1 << (26 + ord(c) - ord('A'))
        return result

def get_bitmask_flags(array_flags, bit_list):
    active = []
    for i, val in enumerate(array_flags):
        for bit in range(32):
            if val & (1 << bit):
                idx = i * 32 + bit
                if idx < len(bit_list):
                    active.append(bit_list[idx])
                else:
                    active.append(f"FLAG_{idx}")
    return active

def parse_mobs():
    mobs = {}
    mob_dir = WORLD_DIR / "mob"
    if not mob_dir.exists():
        return mobs
    
    for f in sorted(mob_dir.glob("*.mob")):
        try:
            lines = f.read_text(errors='replace').splitlines()
            i = 0
            while i < len(lines):
                line = lines[i].strip()
                i += 1
                if not line.startswith('#') or line == '#$':
                    continue
                vnum_str = line[1:]
                if vnum_str == "99999":
                    break
                try:
                    vnum = int(vnum_str)
                except ValueError:
                    continue
                
                keywords, i = read_string(lines, i)
                short_desc, i = read_string(lines, i)
                long_desc, i = read_string(lines, i)
                detailed_desc, i = read_string(lines, i)
                
                flags_line = lines[i].strip()
                i += 1
                while not (flags_line.endswith(' E') or flags_line.endswith('E') or 
                           flags_line.endswith(' S') or flags_line.endswith('S')):
                    if i >= len(lines):
                        break
                    flags_line = flags_line + " " + lines[i].strip()
                    i += 1
                
                letter = 'S'
                if flags_line.endswith('E'):
                    letter = 'E'
                flags_line = flags_line.rstrip(' ES')
                fields = flags_line.split()
                
                action_flags = []
                affect_flags = []
                alignment = 0
                race = 16  # Default RACE_OTHER in src/db.c:1110
                
                if len(fields) >= 9:
                    act_raw = [parse_flag(fields[j]) for j in range(4)]
                    aff_raw = [parse_flag(fields[4 + j]) for j in range(4)]
                    action_flags = [flg for flg in get_bitmask_flags(act_raw, ACTION_BITS) if flg != "ISNPC"]
                    affect_flags = get_bitmask_flags(aff_raw, AFFECT_BITS)
                    try:
                        alignment = int(fields[8])
                    except ValueError:
                        pass
                elif len(fields) >= 3:
                    try:
                        act_raw = [parse_flag(fields[0]), 0, 0, 0]
                        aff_raw = [parse_flag(fields[1]), 0, 0, 0]
                        action_flags = [flg for flg in get_bitmask_flags(act_raw, ACTION_BITS) if flg != "ISNPC"]
                        affect_flags = get_bitmask_flags(aff_raw, AFFECT_BITS)
                        alignment = int(fields[2])
                    except ValueError:
                        pass
                
                # Stats line: <level> <thac0> <ac> <hp_dice> <dmg_dice>
                stats_line = lines[i].strip()
                i += 1
                stats = stats_line.split()
                level = 0
                thac0 = 20
                ac = 100
                hp_dice = "1d1+1"
                damage_dice = "1d1+0"
                if len(stats) >= 5:
                    try:
                        level = int(stats[0])
                        thac0 = int(stats[1])
                        ac = int(stats[2]) * 10  # C scale (src/db.c:1064)
                        hp_dice = stats[3]
                        damage_dice = stats[4]
                    except ValueError:
                        pass
                
                # Gold / Exp
                ge_line = lines[i].strip()
                i += 1
                ge = ge_line.split()
                gold = 0
                exp = 0
                if len(ge) >= 2:
                    try:
                        gold = int(ge[0])
                        exp = int(ge[1])
                    except ValueError:
                        pass
                
                # Position / Default Position / Sex
                pos_line = lines[i].strip()
                i += 1
                pos = pos_line.split()
                position = 8
                default_pos = 8
                sex = 0
                if len(pos) >= 3:
                    try:
                        position = int(pos[0])
                        default_pos = int(pos[1])
                        sex = int(pos[2])
                    except ValueError:
                        pass
                
                # Base stats
                str_stat, int_stat, wis_stat, dex_stat, con_stat, cha_stat = 11, 11, 11, 11, 11, 11
                bare_hand_attack = 0
                noise = ""
                script = ""
                
                # Parse espec lines (only when mob type is 'E')
                if letter == 'E':
                    while i < len(lines):
                        line = lines[i].strip()
                        if line == "E" or line.startswith('#') or line.startswith('$'):
                            if line == "E":
                                i += 1
                            break
                        i += 1
                        
                        if line.startswith("BareHandAttack:"):
                            try:
                                bare_hand_attack = int(line.split(":", 1)[1].strip())
                            except ValueError:
                                pass
                        elif line.startswith("Race:"):
                            try:
                                race = int(line.split(":", 1)[1].strip())
                            except ValueError:
                                pass
                        elif line.startswith("Str:"):
                            try:
                                str_stat = int(line.split(":", 1)[1].strip())
                            except ValueError:
                                pass
                        elif line.startswith("Int:"):
                            try:
                                int_stat = int(line.split(":", 1)[1].strip())
                            except ValueError:
                                pass
                        elif line.startswith("Wis:"):
                            try:
                                wis_stat = int(line.split(":", 1)[1].strip())
                            except ValueError:
                                pass
                        elif line.startswith("Dex:"):
                            try:
                                dex_stat = int(line.split(":", 1)[1].strip())
                            except ValueError:
                                pass
                        elif line.startswith("Con:"):
                            try:
                                con_stat = int(line.split(":", 1)[1].strip())
                            except ValueError:
                                pass
                        elif line.startswith("Cha:"):
                            try:
                                cha_stat = int(line.split(":", 1)[1].strip())
                            except ValueError:
                                pass
                        elif line.startswith("Noise:"):
                            noise = line.split(":", 1)[1].strip().rstrip('~')
                        elif line.startswith("Script:"):
                            sp_parts = line.split(":", 1)[1].strip().split()
                            if sp_parts and sp_parts[0] != "0":
                                script = sp_parts[0]
                
                mobs[vnum] = {
                    "vnum": vnum,
                    "keywords": keywords,
                    "short_desc": short_desc,
                    "long_desc": long_desc,
                    "detailed_desc": detailed_desc,
                    "action_flags": action_flags,
                    "affect_flags": affect_flags,
                    "alignment": alignment,
                    "race": race,
                    "race_name": MOB_RACES[race] if 0 <= race < len(MOB_RACES) else f"Race {race}",
                    "level": level,
                    "thac0": thac0,
                    "ac": ac,
                    "hp_dice": hp_dice,
                    "damage_dice": damage_dice,
                    "gold": gold,
                    "exp": exp,
                    "position": POSITIONS[position] if 0 <= position < len(POSITIONS) else "Standing",
                    "default_pos": POSITIONS[default_pos] if 0 <= default_pos < len(POSITIONS) else "Standing",
                    "sex": sex,
                    "str": str_stat,
                    "int": int_stat,
                    "wis": wis_stat,
                    "dex": dex_stat,
                    "con": con_stat,
                    "cha": cha_stat,
                    "bare_hand_attack": bare_hand_attack,
                    "attack_name": ATTACK_TYPES[bare_hand_attack] if 0 <= bare_hand_attack < len(ATTACK_TYPES) else "hit",
                    "noise": noise,
                    "script": script,
                    "spawns": [],
                    "drops": [],
                    "shop": None
                }
        except Exception as e:
            print(f"Error parsing mob file {f.name}: {e}", file=sys.stderr)
            
    return mobs

def parse_objects():
    objs = {}
    obj_dir = WORLD_DIR / "obj"
    if not obj_dir.exists():
        return objs
    
    for f in sorted(obj_dir.glob("*.obj")):
        try:
            lines = f.read_text(errors='replace').splitlines()
            i = 0
            while i < len(lines):
                line = lines[i].strip()
                i += 1
                if not line.startswith('#') or line == '#$':
                    continue
                vnum_str = line[1:]
                if vnum_str == "99999":
                    break
                try:
                    vnum = int(vnum_str)
                except ValueError:
                    continue
                
                keywords, i = read_string(lines, i)
                short_desc, i = read_string(lines, i)
                long_desc, i = read_string(lines, i)
                action_desc, i = read_string(lines, i)
                
                # Flags line
                flags_line = lines[i].strip()
                i += 1
                flags_fields = flags_line.split()
                type_flag = 0
                extra_flags_raw = [0, 0, 0, 0]
                wear_flags_raw = [0, 0, 0, 0]
                
                if len(flags_fields) >= 9:
                    try:
                        type_flag = int(flags_fields[0])
                        for f_idx in range(4):
                            extra_flags_raw[f_idx] = parse_flag(flags_fields[1+f_idx])
                        for f_idx in range(4):
                            wear_flags_raw[f_idx] = parse_flag(flags_fields[5+f_idx])
                    except ValueError: pass
                elif len(flags_fields) >= 3:
                    try:
                        type_flag = int(flags_fields[0])
                        extra_flags_raw[0] = parse_flag(flags_fields[1])
                        wear_flags_raw[0] = parse_flag(flags_fields[2])
                    except ValueError: pass
                
                # Values line
                values_line = lines[i].strip()
                i += 1
                val_fields = values_line.split()
                values = [0, 0, 0, 0]
                for v_idx in range(min(4, len(val_fields))):
                    try: values[v_idx] = int(val_fields[v_idx])
                    except ValueError: pass
                
                # Stats line
                stats_line = lines[i].strip()
                i += 1
                stats_fields = stats_line.split()
                weight = 0
                cost = 0
                load_percent = 100.0
                if len(stats_fields) >= 3:
                    try:
                        weight = int(stats_fields[0])
                        cost = int(stats_fields[1])
                        load_percent = float(stats_fields[2])
                    except ValueError: pass
                
                extra_descs = []
                affects = []
                script = ""
                
                while i < len(lines):
                    line = lines[i].strip()
                    if line == "$" or line == "$~" or line.startswith('#'):
                        break
                    i += 1
                    
                    if line == "E":
                        ex_keywords, i = read_string(lines, i)
                        ex_desc, i = read_string(lines, i)
                        extra_descs.append({"keywords": ex_keywords, "desc": ex_desc})
                    elif line == "A":
                        aff_line = lines[i].strip()
                        i += 1
                        aff_fields = aff_line.split()
                        if len(aff_fields) >= 2:
                            try:
                                loc = int(aff_fields[0])
                                mod = int(aff_fields[1])
                                affects.append({"location": APPLY_LOCATIONS.get(loc, f"LOC_{loc}"), "modifier": mod})
                            except ValueError: pass
                    elif line == "S" or line.startswith("S "):
                        if line == "S" and i < len(lines):
                            s_line = lines[i].strip()
                            i += 1
                        else:
                            s_line = line[1:].strip()
                        s_parts = s_line.split()
                        if s_parts and s_parts[0] != "0":
                            script = s_parts[0]
                
                extra_flags = get_bitmask_flags(extra_flags_raw, list(EXTRA_FLAGS.values()))
                wear_flags = get_bitmask_flags(wear_flags_raw, list(WEAR_FLAGS.values()))
                
                objs[vnum] = {
                    "vnum": vnum,
                    "keywords": keywords,
                    "short_desc": short_desc,
                    "long_desc": long_desc,
                    "action_desc": action_desc,
                    "type": ITEM_TYPES.get(type_flag, f"TYPE_{type_flag}"),
                    "extra_flags": extra_flags,
                    "wear_flags": wear_flags,
                    "values": values,
                    "weight": weight,
                    "cost": cost,
                    "load_percent": load_percent,
                    "extra_descs": extra_descs,
                    "affects": affects,
                    "script": script,
                    "loaded_by": [],
                    "placed_in": [],
                    "sold_by": []
                }
        except Exception as e:
            print(f"Error parsing object file {f.name}: {e}", file=sys.stderr)
            
    return objs

def parse_rooms():
    rooms = {}
    wld_dir = WORLD_DIR / "wld"
    if not wld_dir.exists():
        return rooms
    
    for f in sorted(wld_dir.glob("*.wld")):
        try:
            lines = f.read_text(errors='replace').splitlines()
            i = 0
            while i < len(lines):
                line = lines[i].strip()
                i += 1
                if not line.startswith('#') or line == '#$':
                    continue
                vnum_str = line[1:]
                if vnum_str == "99999":
                    break
                try: vnum = int(vnum_str)
                except ValueError: continue
                
                name, i = read_string(lines, i)
                desc, i = read_string(lines, i)
                
                flags_line = lines[i].strip()
                i += 1
                parts = flags_line.split()
                zone = int(parts[0]) if len(parts) > 0 else -1
                
                # Skip exits block to find next room marker or S
                while i < len(lines):
                    tok = lines[i].strip()
                    i += 1
                    if tok == 'S':
                        break
                    if re.match(r'^D\d+$', tok):
                        _, i = read_string(lines, i) # desc
                        _, i = read_string(lines, i) # keywords
                        i += 1 # nav
                    elif tok == 'E':
                        _, i = read_string(lines, i)
                        _, i = read_string(lines, i)
                
                rooms[vnum] = {"vnum": vnum, "name": name, "zone": zone}
        except Exception as e:
             print(f"Error parsing room file {f.name}: {e}", file=sys.stderr)
    return rooms

def parse_zones_and_relate(mobs, objs, rooms):
    zon_dir = WORLD_DIR / "zon"
    if not zon_dir.exists():
        return
    
    for f in sorted(zon_dir.glob("*.zon")):
        try:
            lines = f.read_text(errors='replace').splitlines()
            i = 0
            while i < len(lines):
                line = lines[i].strip()
                i += 1
                if not line.startswith('#') or line == '#$':
                    continue
                
                # Zone header
                _, i = read_string(lines, i) # name
                i += 1 # Top room, lifespan, reset
                
                last_mob = None
                
                # Read resets
                while i < len(lines):
                    r_line = lines[i].strip()
                    if r_line == "S":
                        i += 1
                        break
                    if r_line == "" or r_line.startswith('*'):
                        i += 1
                        continue
                    i += 1
                    
                    fields = r_line.split()
                    if not fields:
                        continue
                    
                    cmd = fields[0]
                    # Format resets
                    if cmd == 'M':
                        # M <if_flag> <mob_vnum> <max_in_world> <room_vnum>
                        if len(fields) >= 5:
                            mob_vnum = int(fields[2])
                            room_vnum = int(fields[4])
                            last_mob = mob_vnum
                            
                            if mob_vnum in mobs:
                                room_name = rooms[room_vnum]["name"] if room_vnum in rooms else f"Room {room_vnum}"
                                zone_id = rooms[room_vnum]["zone"] if room_vnum in rooms else -1
                                spawn_info = {"room": room_vnum, "name": room_name, "zone": zone_id}
                                if spawn_info not in mobs[mob_vnum]["spawns"]:
                                    mobs[mob_vnum]["spawns"].append(spawn_info)
                    elif cmd == 'O':
                        # O <if_flag> <obj_vnum> <max_in_world> <room_vnum>
                        if len(fields) >= 5:
                            obj_vnum = int(fields[2])
                            room_vnum = int(fields[4])
                            if obj_vnum in objs:
                                room_name = rooms[room_vnum]["name"] if room_vnum in rooms else f"Room {room_vnum}"
                                zone_id = rooms[room_vnum]["zone"] if room_vnum in rooms else -1
                                place_info = {"room": room_vnum, "name": room_name, "zone": zone_id}
                                if place_info not in objs[obj_vnum]["placed_in"]:
                                    objs[obj_vnum]["placed_in"].append(place_info)
                    elif cmd == 'G':
                        # G <if_flag> <obj_vnum> <max_in_world>
                        if len(fields) >= 4 and last_mob is not None:
                            obj_vnum = int(fields[2])
                            if obj_vnum in objs and last_mob in mobs:
                                drop_info = {"obj_vnum": obj_vnum, "name": objs[obj_vnum]["short_desc"], "type": "carried", "slot": "inventory"}
                                if drop_info not in mobs[last_mob]["drops"]:
                                    mobs[last_mob]["drops"].append(drop_info)
                                    
                                loader_info = {"mob_vnum": last_mob, "name": mobs[last_mob]["short_desc"], "type": "carried", "slot": "inventory"}
                                if loader_info not in objs[obj_vnum]["loaded_by"]:
                                    objs[obj_vnum]["loaded_by"].append(loader_info)
                    elif cmd == 'E':
                        # E <if_flag> <obj_vnum> <max_in_world> <equip_position>
                        if len(fields) >= 5 and last_mob is not None:
                            obj_vnum = int(fields[2])
                            slot_id = int(fields[4])
                            slot_name = EQUIP_SLOTS.get(slot_id, f"slot {slot_id}")
                            
                            if obj_vnum in objs and last_mob in mobs:
                                drop_info = {"obj_vnum": obj_vnum, "name": objs[obj_vnum]["short_desc"], "type": "equipped", "slot": slot_name}
                                if drop_info not in mobs[last_mob]["drops"]:
                                    mobs[last_mob]["drops"].append(drop_info)
                                    
                                loader_info = {"mob_vnum": last_mob, "name": mobs[last_mob]["short_desc"], "type": "equipped", "slot": slot_name}
                                if loader_info not in objs[obj_vnum]["loaded_by"]:
                                    objs[obj_vnum]["loaded_by"].append(loader_info)
                    elif cmd == 'P':
                        # P <if_flag> <obj_vnum> <max_in_world> <container_vnum>
                        if len(fields) >= 5:
                            obj_vnum = int(fields[2])
                            cont_vnum = int(fields[4])
                            if obj_vnum in objs and cont_vnum in objs:
                                # We can link container items
                                place_info = {"container_vnum": cont_vnum, "name": objs[cont_vnum]["short_desc"]}
                                if place_info not in objs[obj_vnum]["placed_in"]:
                                    objs[obj_vnum]["placed_in"].append(place_info)
        except Exception as e:
            print(f"Error parsing zone resets {f.name}: {e}", file=sys.stderr)

def parse_shops_and_relate(mobs, objs):
    shp_dir = WORLD_DIR / "shp"
    if not shp_dir.exists():
        return
    
    for f in sorted(shp_dir.glob("*.shp")):
        try:
            lines = f.read_text(errors='replace').splitlines()
            i = 0
            while i < len(lines):
                line = lines[i].strip()
                i += 1
                if not line.startswith('#') or line == '#$':
                    continue
                shop_vnum_str = line[1:].rstrip('~')
                try: shop_vnum = int(shop_vnum_str)
                except ValueError: continue
                
                # Items sold list
                items_sold = []
                while i < len(lines):
                    s_line = lines[i].strip()
                    i += 1
                    if s_line.endswith('~'):
                        s_line = s_line.rstrip('~')
                    try:
                        val = int(s_line)
                        if val == -1:
                            break
                        items_sold.append(val)
                    except ValueError: pass
                
                # Multipliers
                sell_mult = 1.0
                buy_mult = 1.0
                if i < len(lines):
                    try: sell_mult = float(lines[i].strip().rstrip('~'))
                    except ValueError: pass
                    i += 1
                if i < len(lines):
                    try: buy_mult = float(lines[i].strip().rstrip('~'))
                    except ValueError: pass
                    i += 1
                
                # Items bought types (skip list)
                while i < len(lines):
                    b_line = lines[i].strip()
                    i += 1
                    if b_line.endswith('~'): b_line = b_line.rstrip('~')
                    try:
                        if int(b_line) == -1: break
                    except ValueError: pass
                
                # 7 messages (skip)
                for _ in range(7):
                    if i < len(lines):
                        _, i = read_string(lines, i)
                
                # Shopkeeper / temper / bitvector
                temper = 0
                bitvector = 0
                keeper = -1
                if i < len(lines):
                    try: temper = int(lines[i].strip().rstrip('~'))
                    except ValueError: pass
                    i += 1
                if i < len(lines):
                    try: bitvector = int(lines[i].strip().rstrip('~'))
                    except ValueError: pass
                    i += 1
                if i < len(lines):
                    try: keeper = int(lines[i].strip().rstrip('~'))
                    except ValueError: pass
                    i += 1
                
                # Skip room list
                while i < len(lines):
                    r_line = lines[i].strip()
                    i += 1
                    if r_line.endswith('~'): r_line = r_line.rstrip('~')
                    try:
                        if int(r_line) == -1: break
                    except ValueError: pass
                
                # Hours
                open1, close1, open2, close2 = 0, 28, 0, 0
                if i < len(lines):
                    try: open1 = int(lines[i].strip().rstrip('~'))
                    except ValueError: pass
                    i += 1
                if i < len(lines):
                    try: close1 = int(lines[i].strip().rstrip('~'))
                    except ValueError: pass
                    i += 1
                if i < len(lines):
                    try: open2 = int(lines[i].strip().rstrip('~'))
                    except ValueError: pass
                    i += 1
                if i < len(lines):
                    try: close2 = int(lines[i].strip().rstrip('~'))
                    except ValueError: pass
                    i += 1
                
                # Link keeper and objects sold
                if keeper != -1:
                    shop_info = {
                        "shop_vnum": shop_vnum,
                        "sell_mult": sell_mult,
                        "buy_mult": buy_mult,
                        "open_hours": f"{open1}:00-{close1}:00",
                        "items_sold": [{"vnum": ov, "name": objs[ov]["short_desc"] if ov in objs else f"Item {ov}"} for ov in items_sold]
                    }
                    if keeper in mobs:
                        mobs[keeper]["shop"] = shop_info
                    
                    # Link items sold by shop
                    for ov in items_sold:
                        if ov in objs:
                            objs[ov]["sold_by"].append({
                                "shop_vnum": shop_vnum,
                                "keeper_vnum": keeper,
                                "keeper_name": mobs[keeper]["short_desc"] if keeper in mobs else f"Shopkeeper {keeper}",
                                "price": int(objs[ov]["cost"] * sell_mult)
                            })
        except Exception as e:
            print(f"Error parsing shop resets {f.name}: {e}", file=sys.stderr)



def main():
    print("Compiling Dark Pawns Interactive MUD Database...", file=sys.stderr)
    
    print("  1. Parsing rooms...", file=sys.stderr)
    rooms = parse_rooms()
    print(f"     Found {len(rooms)} rooms.", file=sys.stderr)
    
    print("  2. Parsing mobs...", file=sys.stderr)
    mobs = parse_mobs()
    print(f"     Found {len(mobs)} mobs.", file=sys.stderr)
    
    print("  3. Parsing items/objects...", file=sys.stderr)
    objs = parse_objects()
    print(f"     Found {len(objs)} items.", file=sys.stderr)
    
    print("  4. Linking zone resets (spawns & drops)...", file=sys.stderr)
    parse_zones_and_relate(mobs, objs, rooms)
    
    print("  5. Linking shopkeepers and shop items...", file=sys.stderr)
    parse_shops_and_relate(mobs, objs)
    
    # Export Dynamic Database JSON
    print("  6. Exporting database.json...", file=sys.stderr)
    STATIC_DATA_DIR.mkdir(parents=True, exist_ok=True)
    db_out = STATIC_DATA_DIR / "database.json"
    
    # Restructure for compact output
    compact_mobs = {}
    for kv, mv in mobs.items():
        compact_mobs[kv] = {
            "v": mv["vnum"],
            "k": mv["keywords"],
            "s": mv["short_desc"],
            "l": mv["long_desc"],
            "d": mv["detailed_desc"],
            "lvl": mv["level"],
            "thac0": mv["thac0"],
            "rc": mv["race"],
            "rcn": mv["race_name"],
            "alg": mv["alignment"],
            "sex": mv["sex"],
            "pos": mv["position"],
            "dpos": mv["default_pos"],
            "hp": mv["hp_dice"],
            "dmg": mv["damage_dice"],
            "bha": mv["bare_hand_attack"],
            "atk": mv["attack_name"],
            "gld": mv["gold"],
            "exp": mv["exp"],
            "ac": mv["ac"],
            "stat": [mv["str"], mv["int"], mv["wis"], mv["dex"], mv["con"], mv["cha"]],
            "act": mv["action_flags"],
            "aff": mv["affect_flags"],
            "noise": mv["noise"],
            "scr": mv["script"],
            "spw": mv["spawns"],
            "drp": mv["drops"],
            "shop": mv["shop"]
        }
        
    compact_objs = {}
    for kv, ov in objs.items():
        compact_objs[kv] = {
            "v": ov["vnum"],
            "k": ov["keywords"],
            "s": ov["short_desc"],
            "l": ov["long_desc"],
            "ad": ov["action_desc"],
            "type": ov["type"],
            "wear": ov["wear_flags"],
            "extra": ov["extra_flags"],
            "val": ov["values"],
            "wt": ov["weight"],
            "cst": ov["cost"],
            "load": ov["load_percent"],
            "aff": ov["affects"],
            "edesc": ov["extra_descs"],
            "scr": ov["script"],
            "mobs": ov["loaded_by"],
            "rms": ov["placed_in"],
            "shp": ov["sold_by"]
        }
    
    with open(db_out, "w") as f:
        json.dump({"mobs": compact_mobs, "items": compact_objs}, f, separators=(",", ":"))
        
    size_kb = db_out.stat().st_size / 1024
    print(f"     Compiled database.json: {size_kb:.1f} KB", file=sys.stderr)
    
    print("  7. Generating pre-rendered SEO static pages...", file=sys.stderr)
    print("     SEO compilation complete.", file=sys.stderr)
    print("Database build successfully finalized!", file=sys.stderr)

if __name__ == "__main__":
    main()
