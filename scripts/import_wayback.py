#!/usr/bin/env python3
"""Import wayback archive content into Hugo content files for Dark Pawns website."""
import os
import re
import html

CONTENT_DIR = os.path.expanduser("~/darkpawns/website/content")
WAYBACK_DIR = os.path.expanduser("~/darkpawns/docs/wayback")
HELP_DIR = os.path.join(WAYBACK_DIR, "help-files")

DATE = "2026-04-28"

def clean_html(text):
    """Convert HTML entities to plain text."""
    text = html.unescape(text)
    # Remove MUD color codes like &c, &n, &b etc
    text = re.sub(r'&[a-zA-Z0-9]', '', text)
    return text

def write_page(path, title, section, description, body, aliases=None):
    """Write a Hugo content page with frontmatter."""
    os.makedirs(os.path.dirname(path), exist_ok=True)
    frontmatter_lines = [
        '---',
        f'title: "{title}"',
        f'date: {DATE}',
        'draft: false',
        f'section: "{section}"',
    ]
    if description:
        frontmatter_lines.append(f'description: "{description}"')
    if aliases:
        frontmatter_lines.append(f'aliases: {aliases}')
    frontmatter_lines.append('---')
    
    content = '\n'.join(frontmatter_lines) + '\n\n' + body
    with open(path, 'w') as f:
        f.write(content)
    return path

def yaml_safe_title(title):
    """Make a title safe for YAML double-quoted strings."""
    # Replace double quotes with single quotes inside the title
    return title.replace('"', "'")

def write_help_page(path, title, slug, section, body, aliases=None):
    """Write a help entry page."""
    os.makedirs(os.path.dirname(path), exist_ok=True)
    safe_title = yaml_safe_title(title)
    frontmatter_lines = [
        '---',
        f'title: "{safe_title}"',
        f'date: {DATE}',
        'draft: false',
        f'section: "{section}"',
    ]
    if aliases:
        frontmatter_lines.append(f'aliases: {aliases}')
    frontmatter_lines.append('---')
    
    content = '\n'.join(frontmatter_lines) + '\n\n' + body
    with open(path, 'w') as f:
        f.write(content)
    return path

def process_html_content_file(filename, dest_path, title, section, description):
    """Process a single HTML content md file."""
    src = os.path.join(WAYBACK_DIR, filename)
    with open(src, 'r') as f:
        text = f.read()
    text = clean_html(text)
    path = os.path.join(CONTENT_DIR, dest_path)
    write_page(path, title, section, description, text)
    print(f"  {dest_path}")

def split_hlp_file(filename):
    """Split a .hlp file into entries by # delimiter. Returns list of (title, body)."""
    src = os.path.join(HELP_DIR, filename)
    with open(src, 'r') as f:
        content = f.read()
    
    # Split on lines that are just '#'
    entries = []
    current_lines = []
    
    for line in content.split('\n'):
        stripped = line.strip()
        if stripped == '#':
            if current_lines:
                entries.append('\n'.join(current_lines))
                current_lines = []
        elif stripped == '$':
            # End of file marker
            if current_lines:
                entries.append('\n'.join(current_lines))
                current_lines = []
            break
        else:
            current_lines.append(line)
    
    if current_lines:
        entries.append('\n'.join(current_lines))
    
    results = []
    for entry in entries:
        entry = entry.strip()
        if not entry:
            continue
        lines = entry.split('\n')
        # First line is the command name(s)
        title_line = lines[0].strip()
        if not title_line:
            continue
        # Use full title line slugified for uniqueness
        full_slug = slugify(title_line)
        if not full_slug:
            # Fallback to first word
            first_word = title_line.split()[0].strip('"').lower()
            full_slug = first_word
        body = '\n'.join(lines[1:]).strip() if len(lines) > 1 else ''
        results.append((title_line, full_slug, body))
    
    return results

def slugify(name):
    """Convert a command name to a safe filename slug."""
    # Remove quotes, replace spaces with hyphens, lowercase
    name = name.strip('"').strip().lower()
    name = re.sub(r'[^a-z0-9]+', '-', name).strip('-')
    return name

def process_hlp_file(filename, dest_dir, section):
    """Process a .hlp file and create individual pages."""
    entries = split_hlp_file(filename)
    count = 0
    for title, slug, body in entries:
        if not slug or slug == '$':
            continue
        safe_slug = slugify(slug)
        if not safe_slug:
            safe_slug = slug
        
        path = os.path.join(CONTENT_DIR, dest_dir, f"{safe_slug}.md")
        aliases = [f"/help/{safe_slug}"]
        write_help_page(path, title, safe_slug, section, body, aliases)
        count += 1
    print(f"  {filename}: {count} entries -> {dest_dir}/")
    return count

def main():
    total = 0
    
    print("=== HTML Content Files ===")
    # Major content pages
    process_html_content_file(
        "background.html-content.md",
        "lore/world-creation.md",
        "World Creation — The Letter of Friar Drake",
        "lore",
        "Friar Drake's letter describing the creation myth of the Dark Pawns world"
    )
    total += 1
    
    process_html_content_file(
        "class.html-content.md",
        "world/classes/classes.md",
        "Classes",
        "world",
        "All base and remort classes available in Dark Pawns"
    )
    total += 1
    
    process_html_content_file(
        "faq.html-content.md",
        "help/faq.md",
        "Frequently Asked Questions",
        "help",
        "Common questions and answers for new Dark Pawns players"
    )
    total += 1
    
    process_html_content_file(
        "features.html-content.md",
        "about/features.md",
        "Features",
        "about",
        "Key features that make Dark Pawns unique"
    )
    total += 1
    
    process_html_content_file(
        "main.html-content.md",
        "news/historical-posts.md",
        "Historical News Posts",
        "news",
        "News posts from the original Dark Pawns website"
    )
    total += 1
    
    process_html_content_file(
        "wizlist.html-content.md",
        "credits/wizlist.md",
        "Wizard List",
        "credits",
        "The wizards and creators of Dark Pawns"
    )
    total += 1
    
    print("\n=== Help Files ===")
    # Help files
    total += process_hlp_file("commands.hlp", "help/commands", "help")
    total += process_hlp_file("info.hlp", "help/info", "help")
    total += process_hlp_file("spells.hlp", "help/spells", "help")
    total += process_hlp_file("socials.hlp", "help/socials", "help")
    total += process_hlp_file("wizhelp.hlp", "help/wizhelp", "help")
    
    print(f"\nTotal files created: {total}")

if __name__ == "__main__":
    main()
