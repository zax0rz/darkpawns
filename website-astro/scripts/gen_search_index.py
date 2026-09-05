#!/usr/bin/env python3
"""
Dark Pawns Unified Search Index Generator (Astro version)
Parses:
  1. Astro Content Collections (src/content/):
     - help/ (commands, spells, socials, info, wizhelp)
     - docs/ (agents, getting-started, research, server)
     - world/
     - blog/
     - archive/
  2. Game Database (data/database.json):
     - Mobs (1,319 entities, linked to /mobs/[vnum]/ and /database)
     - Items (1,672 entities, linked to /items/[vnum]/ and /database)
  3. World Zones (map/world.json):
     - Zones (92 zones, linked to /zones/[id]/ and /map)
  4. Core Static Pages:
     - /play/, /map/, /database/, /about/, /credits/, /contact/, /privacy/

Outputs:
  website/static/data/search-index.json (served as /data/search-index.json by Astro)
"""

import os
import re
import json
import sys
from pathlib import Path

# Base paths
SCRIPTS_DIR = Path(__file__).resolve().parent
ASTRO_ROOT = SCRIPTS_DIR.parent
PROJECT_ROOT = ASTRO_ROOT.parent
CONTENT_DIR = ASTRO_ROOT / "src" / "content"
STATIC_DIR = PROJECT_ROOT / "website" / "static"
DATA_DIR = STATIC_DIR / "data"
MAP_DIR = STATIC_DIR / "map"
OUTPUT_FILE = DATA_DIR / "search-index.json"

def searchable_body(text: str) -> str:
    """The whole document as one lowercase line of words.

    Search previously matched titles, descriptions and slug-derived keywords
    only, so a phrase from inside a 2004 forum post, a help file or a blog post
    was unfindable. This is the text those queries need. It is never displayed;
    the description is still what a result shows.
    """
    text = re.sub(r'^---.*?^---', ' ', text, flags=re.S | re.M)
    text = re.sub(r'`{1,3}[^`]*`{1,3}', ' ', text)
    text = re.sub(r'!?\[([^\]]*)\]\([^)]*\)', r'\1', text)
    text = re.sub(r'<[^>]+>', ' ', text)
    text = re.sub(r'[#>*_|\\]', ' ', text)
    return re.sub(r'\s+', ' ', text).strip().lower()


def clean_text(text: str, max_len: int = 200) -> str:
    """Strip markdown, html, newlines, and truncate."""
    if not text:
        return ""
    text = re.sub(r'```[\s\S]*?```', '', text)
    text = re.sub(r'<[^>]+>', '', text)
    text = re.sub(r'\[([^\]]+)\]\([^)]+\)', r'\1', text)
    text = re.sub(r'[#*`_~]', '', text)
    text = re.sub(r'\s+', ' ', text).strip()
    if len(text) > max_len:
        text = text[:max_len].rsplit(' ', 1)[0] + "…"
    return text

def parse_frontmatter(content: str):
    """Simple YAML frontmatter parser for Astro markdown files."""
    if not content.startswith('---'):
        return {}, content
    
    parts = content.split('---', 2)
    if len(parts) < 3:
        return {}, content
    
    fm_raw = parts[1]
    body = parts[2]
    
    meta = {}
    nested = []
    for raw_line in fm_raw.splitlines():
        if not raw_line.strip() or raw_line.lstrip().startswith('#'):
            continue
        indented = raw_line[:1].isspace() or raw_line.lstrip().startswith('- ')
        line = raw_line.strip()
        if ':' in line:
            key, val = line.split(':', 1)
            key = key.strip().lstrip('- ')
            val = val.strip().strip('"\'')
            if indented:
                # A list item under participants/topics. Collected for search
                # keywords, never allowed to shadow a top-level field.
                if key in ('name', 'title', 'author') and val:
                    nested.append(val)
                continue
            if val.lower() == 'true':
                val = True
            elif val.lower() == 'false':
                val = False
            meta[key] = val
    if nested:
        meta['_nested'] = nested

    return meta, body

def index_markdown_content():
    """Index help, docs, world, blog, and archive content."""
    entries = []
    
    # 1. Help Collection
    help_dir = CONTENT_DIR / "help"
    if help_dir.exists():
        category_map = {
            "commands": "Command",
            "spells": "Spell",
            "socials": "Social",
            "info": "Info & Lore",
            "wizhelp": "Wizard Help"
        }
        for md_file in help_dir.rglob("*.md"):
            if md_file.name.startswith("_"):
                continue
            rel_path = md_file.relative_to(help_dir)
            parts = rel_path.parts
            if len(parts) < 2:
                continue
            cat = parts[0]
            slug = md_file.stem
            
            try:
                content = md_file.read_text(encoding='utf-8', errors='ignore')
                meta, body = parse_frontmatter(content)
                if meta.get("draft") is True:
                    continue
                
                title = meta.get("title", slug.replace("-", " ").upper())
                desc = meta.get("description", clean_text(body, 160))
                subtype = category_map.get(cat, cat.title())
                
                # Help keyword aliases
                keywords = [slug.replace("-", " ")]
                if cat == "spells":
                    keywords.extend(["spell", "magic", "cast"])
                elif cat == "commands":
                    keywords.extend(["command", "syntax", "how to"])
                
                entries.append({
                    "t": title,
                    "c": "help",
                    "s": subtype,
                    "u": f"/help/{cat}/{slug}/",
                    "k": " ".join(set(keywords)),
                    "d": desc,
                    "b": searchable_body(body),
                    "v": 0
                })
            except Exception as e:
                print(f"Warning parsing help file {md_file}: {e}", file=sys.stderr)

    # 2. Docs Collection
    docs_dir = CONTENT_DIR / "docs"
    if docs_dir.exists():
        for md_file in docs_dir.rglob("*.md"):
            if md_file.name.startswith("_"):
                continue
            rel_path = md_file.relative_to(docs_dir)
            parts = rel_path.parts
            section = parts[0] if len(parts) > 1 else ""
            slug = md_file.stem
            url = f"/docs/{section}/{slug}/" if section else f"/docs/{slug}/"
            
            try:
                content = md_file.read_text(encoding='utf-8', errors='ignore')
                meta, body = parse_frontmatter(content)
                if meta.get("draft") is True:
                    continue
                title = meta.get("title", slug.replace("-", " ").title())
                desc = meta.get("description", clean_text(body, 160))
                
                entries.append({
                    "t": title,
                    "c": "docs",
                    "s": f"Docs · {section.title()}" if section else "Docs",
                    "u": url,
                    "k": f"documentation {section} {slug.replace('-', ' ')}",
                    "d": desc,
                    "b": searchable_body(body),
                    "v": 0
                })
            except Exception as e:
                print(f"Warning parsing docs file {md_file}: {e}", file=sys.stderr)

    # 3. World Collection
    world_dir = CONTENT_DIR / "world"
    if world_dir.exists():
        for md_file in world_dir.glob("*.md"):
            if md_file.name.startswith("_"):
                continue
            slug = md_file.stem
            try:
                content = md_file.read_text(encoding='utf-8', errors='ignore')
                meta, body = parse_frontmatter(content)
                if meta.get("draft") is True:
                    continue
                title = meta.get("title", slug.replace("-", " ").title())
                desc = meta.get("description", clean_text(body, 160))
                
                entries.append({
                    "t": title,
                    "c": "world",
                    "s": "World Lore",
                    "u": f"/world/{slug}/",
                    "k": f"world lore {slug.replace('-', ' ')}",
                    "d": desc,
                    "b": searchable_body(body),
                    "v": 0
                })
            except Exception as e:
                print(f"Warning parsing world file {md_file}: {e}", file=sys.stderr)

    # 4. Blog Collection
    blog_dir = CONTENT_DIR / "blog"
    if blog_dir.exists():
        for md_file in blog_dir.glob("*.md"):
            if md_file.name.startswith("_"):
                continue
            slug = md_file.stem
            try:
                content = md_file.read_text(encoding='utf-8', errors='ignore')
                meta, body = parse_frontmatter(content)
                if meta.get("draft") is True:
                    continue
                title = meta.get("title", slug.replace("-", " ").title())
                desc = meta.get("description", clean_text(body, 160))
                
                entries.append({
                    "t": title,
                    "c": "docs",
                    "s": "Blog · Dispatch",
                    "u": f"/blog/{slug}/",
                    "k": f"blog dispatch article {slug.replace('-', ' ')}",
                    "d": desc,
                    "b": searchable_body(body),
                    "v": 0
                })
            except Exception as e:
                print(f"Warning parsing blog file {md_file}: {e}", file=sys.stderr)

    # 5. Archive Collection
    archive_dir = CONTENT_DIR / "archive"
    if archive_dir.exists():
        for md_file in archive_dir.glob("*.md"):
            if md_file.name.startswith("_"):
                continue
            slug = md_file.stem
            try:
                content = md_file.read_text(encoding='utf-8', errors='ignore')
                meta, body = parse_frontmatter(content)
                if meta.get("draft") is True:
                    continue
                title = meta.get("title", slug.replace("-", " ").title())
                desc = meta.get("description", clean_text(body, 160))
                
                # Who spoke, and which board. These are the keys a returning
                # player actually types, and they exist nowhere in the prose.
                people = " ".join(meta.get("_nested", []))
                board = meta.get("board", "")
                kind = meta.get("kind", "").replace("-", " ")
                section_map = {
                    "dp-players.com": "dp-players.com, 2004",
                    "darkpawns.com": "darkpawns.com, 2002-2005",
                }
                entries.append({
                    "t": title,
                    "c": "archive",
                    "s": section_map.get(meta.get("sourceSite", ""), "Community Archive"),
                    "u": f"/archive/{slug}/",
                    "k": f"archive {kind} {board} {people}".strip(),
                    "d": desc,
                    "b": searchable_body(body),
                    "v": 0
                })
            except Exception as e:
                print(f"Warning parsing archive file {md_file}: {e}", file=sys.stderr)

    return entries

def index_database():
    """Index mobs and items from database.json."""
    db_file = DATA_DIR / "database.json"
    if not db_file.exists():
        print(f"Warning: {db_file} not found. Skipping mobs and items.", file=sys.stderr)
        return []
    
    entries = []
    try:
        with open(db_file, 'r', encoding='utf-8') as f:
            data = json.load(f)
            
        # Index Mobs (dict of vnum -> mob)
        mobs = data.get("mobs", {})
        mob_list = mobs.values() if isinstance(mobs, dict) else mobs
        for mob in mob_list:
            vnum = mob.get("v", 0)
            name = mob.get("s") or mob.get("k") or f"Mob #{vnum}"
            level = mob.get("lvl", 1)
            align_val = mob.get("alg", 0)
            align = "Good" if align_val > 350 else ("Evil" if align_val < -350 else "Neutral")
            keywords = mob.get("k", "")
            long_desc = mob.get("l") or mob.get("d") or f"Level {level} {align} NPC in Dark Pawns."
            
            entries.append({
                "t": name,
                "c": "mobs",
                "s": f"Lvl {level} {align} NPC",
                "u": f"/mobs/{vnum}/",
                "k": f"mob npc {keywords} #{vnum}",
                "d": clean_text(long_desc, 150),
                "v": vnum
            })
            
        # Index Items (dict of vnum -> item)
        items = data.get("items", {}) or data.get("objects", {})
        item_list = items.values() if isinstance(items, dict) else items
        for item in item_list:
            vnum = item.get("v", 0)
            name = item.get("s") or item.get("k") or f"Item #{vnum}"
            item_type = str(item.get("type", "ITEM")).upper()
            wear_list = item.get("wear", [])
            wear_str = f" · {' '.join(wear_list)}" if wear_list else ""
            keywords = item.get("k", "")
            action_desc = item.get("l") or f"VNUM {vnum} {item_type}{wear_str}."
            
            entries.append({
                "t": name,
                "c": "items",
                "s": f"{item_type}{wear_str}",
                "u": f"/items/{vnum}/",
                "k": f"item object equipment {keywords} #{vnum}",
                "d": clean_text(action_desc, 150),
                "v": vnum
            })
            
    except Exception as e:
        print(f"Error indexing database.json: {e}", file=sys.stderr)
        
    return entries

def index_world_map():
    """Index zones from world.json."""
    world_file = MAP_DIR / "world.json"
    if not world_file.exists():
        print(f"Warning: {world_file} not found. Skipping world zones.", file=sys.stderr)
        return []
    
    entries = []
    try:
        with open(world_file, 'r', encoding='utf-8') as f:
            data = json.load(f)
            
        zones = data.get("zones", [])
        for zone in zones:
            zone_id = zone.get("id") or zone.get("zone_number") or zone.get("vnum", 0)
            name = zone.get("name", f"Zone {zone_id}")
            builders = zone.get("builders") or zone.get("author") or ""
            bottom = zone.get("bottom", 0)
            top = zone.get("top", 0)
            room_count = len(zone.get("rooms", [])) if "rooms" in zone else 0
            
            desc = f"VNUMs {bottom}–{top}."
            if room_count > 0:
                desc += f" {room_count} rooms mapped."
            if builders:
                desc += f" Built by {builders}."
                
            entries.append({
                "t": name,
                "c": "world",
                "s": f"Map Zone #{zone_id}",
                "u": f"/zones/{zone_id}/",
                "k": f"zone map area world room #{zone_id} {builders}",
                "d": desc,
                "v": zone_id
            })
    except Exception as e:
        print(f"Error indexing world.json: {e}", file=sys.stderr)
        
    return entries

def index_static_pages():
    """Index primary application and core landing pages."""
    return [
        {
            "t": "Play in Browser",
            "c": "pages",
            "s": "CRT Terminal",
            "u": "/play/",
            "k": "play terminal xterm telnet client connect web client",
            "d": "Interactive CRT terminal console to connect and play Dark Pawns directly in your browser.",
            "v": 0
        },
        {
            "t": "Interactive World Map",
            "c": "world",
            "s": "Cartography",
            "u": "/map/",
            "k": "map cartography world atlas rooms navigation graph d3",
            "d": "Interactive vector world map visualizing over 90 zones and connected rooms.",
            "v": 0
        },
        {
            "t": "Mob & Item Codex",
            "c": "pages",
            "s": "Database",
            "u": "/database/",
            "k": "database codex inspector mobs items equipment search monsters",
            "d": "Complete searchable codex of all monsters, NPCs, weapons, armor, and equipment in the game.",
            "v": 0
        },
        {
            "t": "About Dark Pawns",
            "c": "pages",
            "s": "History",
            "u": "/about/",
            "k": "about history story timeline dpreturns lineage engine diku merc circlemud",
            "d": "The sourced history of Dark Pawns, from its 1994 founding to the 2026 Go restoration.",
            "v": 0
        },
        {
            "t": "Credits & Staff",
            "c": "pages",
            "s": "Credits",
            "u": "/credits/",
            "k": "credits staff wizards immortals developers authors builders",
            "d": "Historical staff rosters, immortals, builders, and restoration contributors.",
            "v": 0
        },
        {
            "t": "Contact & Feedback",
            "c": "pages",
            "s": "Contact",
            "u": "/contact/",
            "k": "contact feedback bug report message admin",
            "d": "Send private feedback, bug reports, or restoration inquiries directly to the team.",
            "v": 0
        },
        {
            "t": "Privacy Policy",
            "c": "pages",
            "s": "Policy",
            "u": "/privacy/",
            "k": "privacy policy data cookies analytics security",
            "d": "Dark Pawns privacy policy and data governance practices.",
            "v": 0
        }
    ]

def main():
    print("Generating Dark Pawns unified search index (Astro)...")
    DATA_DIR.mkdir(parents=True, exist_ok=True)
    
    entries = []
    
    # 1. Static Core Pages
    static_pages = index_static_pages()
    entries.extend(static_pages)
    
    # 2. Markdown Content (Help, Docs, World, Blog, Archive)
    md_entries = index_markdown_content()
    entries.extend(md_entries)
    print(f"  → Indexed {len(md_entries)} Markdown topics across Help, Docs, World, Blog, and Archive")
    
    # 3. Mobs & Items from Database
    db_entries = index_database()
    entries.extend(db_entries)
    print(f"  → Indexed {len(db_entries)} Mobs and Items from database")
    
    # 4. World Map Zones
    zone_entries = index_world_map()
    entries.extend(zone_entries)
    print(f"  → Indexed {len(zone_entries)} World Zones")
    
    # Write to output file (minified)
    with open(OUTPUT_FILE, 'w', encoding='utf-8') as f:
        json.dump(entries, f, ensure_ascii=False, separators=(',', ':'))
        
    size_kb = OUTPUT_FILE.stat().st_size / 1024
    print(f"✓ Saved {len(entries)} search entries to {OUTPUT_FILE} ({size_kb:.1f} KB)")

if __name__ == "__main__":
    main()
