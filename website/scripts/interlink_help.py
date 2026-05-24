#!/usr/bin/env python3
import os
import re
import yaml
from pathlib import Path

HELP_DIR = Path("/Users/zach/.openclaw/workspace/darkpawns_repo/website/content/help")

# Common English uppercase stop words to NEVER linkify in body text
STOP_WORDS = {
    "A", "AN", "THE", "AND", "OR", "BUT", "IF", "OF", "BY", "FOR", "IN", "ON", "AT", "TO", 
    "IS", "IT", "HE", "SHE", "WE", "US", "ME", "MY", "YOU", "NO", "YES", "NOT", "SO", "DO", 
    "GO", "UP", "ON", "BE", "AM", "ARE", "WAS", "WERE", "BEING", "BEEN", "HAS", "HAVE", "HAD", 
    "DOING", "DID", "DONE", "THIS", "THAT", "THESE", "THOSE", "WHO", "WHOM", "WHICH", "WHAT",
    "HOW", "WHY", "WHERE", "WHEN", "WITH", "FROM", "ABOUT", "ABOVE", "BELOW", "OUT", "INTO",
    "ONCE", "HERE", "THERE", "ALL", "ANY", "BOTH", "EACH", "FEW", "MORE", "MOST", "OTHER",
    "SOME", "SUCH", "THAN", "TOO", "VERY", "CAN", "WILL", "JUST", "SHOULD", "COULD", "WOULD",
    "MAY", "MIGHT", "MUST", "GET", "PUT", "SEE", "ALSO"
}

def clean_word(word):
    """Normalize a keyword to lowercase and strip all punctuation from ends."""
    word = word.strip()
    if word in ["!", "^"]:
        return word
    # Strip leading/trailing non-alphanumeric characters except ! or ^
    cleaned = re.sub(r"^[^a-zA-Z0-9!^]+|[^a-zA-Z0-9!^]+$", "", word)
    return cleaned.lower()

def extract_frontmatter_and_content(file_path):
    content = file_path.read_text(errors='replace')
    parts = content.split('---')
    if len(parts) >= 3:
        fm_text = parts[1]
        body = '---'.join(parts[2:])
        try:
            fm = yaml.safe_load(fm_text)
        except Exception as e:
            fm = {}
        return fm, body, parts[0], fm_text
    return {}, content, "", ""

def get_keywords(file_path, fm):
    keywords = set()
    
    # 1. From filename (without .md)
    name = file_path.stem
    keywords.add(clean_word(name))
    for part in re.split(r'[-_]', name):
        p = clean_word(part)
        if p and p not in ["", "!"]:
            keywords.add(p)
            
    # 2. From title in frontmatter
    title = fm.get("title", "")
    if title:
        keywords.add(clean_word(title))
        for part in re.split(r'[\s\-_/,]+', title):
            p = clean_word(part)
            if p:
                keywords.add(p)
                
    # 3. From aliases
    aliases = fm.get("aliases", [])
    for alias in aliases:
        alias_name = alias.replace("/help/", "").rstrip("/")
        keywords.add(clean_word(alias_name))
        for part in re.split(r'[\s\-_/,]+', alias_name):
            p = clean_word(part)
            if p:
                keywords.add(p)
                
    # Filter out empty or single letters (unless they are ! or ^)
    filtered = set()
    for kw in keywords:
        if kw in ["!", "^"] or (len(kw) >= 2 and kw.isalnum()):
            filtered.add(kw)
    return filtered

def main():
    keyword_to_url = {}
    url_to_keywords = {}
    
    # 1. First Pass: Scan all help files to map keywords to permalinks
    files_to_process = []
    
    for root, dirs, files in os.walk(HELP_DIR):
        for file in files:
            if not file.endswith(".md"):
                continue
            file_path = Path(root) / file
            fm, body, prefix, fm_text = extract_frontmatter_and_content(file_path)
            
            # Compute permalink
            rel_path = file_path.relative_to(HELP_DIR.parent)
            permalink = "/" + str(rel_path.with_suffix("")).lower() + "/"
            
            keywords = get_keywords(file_path, fm)
            url_to_keywords[permalink] = keywords
            
            files_to_process.append((file_path, fm, body, prefix, fm_text))
            
            for kw in keywords:
                if kw in keyword_to_url:
                    existing_url = keyword_to_url[kw]
                    # Prioritize commands or spells over info or socials/wizhelp for exact hits
                    if "commands" in permalink or "spells" in permalink:
                        keyword_to_url[kw] = permalink
                else:
                    keyword_to_url[kw] = permalink

    print(f"Index built: {len(keyword_to_url)} keywords mapped to {len(url_to_keywords)} URLs.")

    # 2. Second Pass: Linkify body text of each file in-place
    # Sort keywords by length descending to match multi-word phrases first
    sorted_keywords = sorted(keyword_to_url.keys(), key=lambda x: len(x), reverse=True)
    
    modified_count = 0
    
    for file_path, fm, body, prefix, fm_text in files_to_process:
        original_body = body
        temp_body = body
        
        # We replace each keyword in the body
        for kw in sorted_keywords:
            if kw.upper() in STOP_WORDS:
                continue
            
            kw_upper = kw.upper()
            escaped = re.escape(kw_upper)
            
            # Formulate regex pattern based on word boundary or whitespace boundary
            if kw_upper.isalnum():
                kw_pattern = r'\b' + escaped + r'\b'
            else:
                kw_pattern = r'(?<!\S)' + escaped + r'(?!\S)'
                
            # Full regex to match existing markdown links, backticks, or our target keyword
            # This prevents replacing text already within a link [TEXT](url) or code block `code`
            pattern = re.compile(r'(\[[^\]]+\]\([^)]+\)|`[^`]+`|' + kw_pattern + ')')
            
            url = keyword_to_url[kw]
            
            def replace_callback(match):
                text = match.group(0)
                # If it's a markdown link or code block, return it untouched
                if text.startswith('[') or text.startswith('`'):
                    return text
                # Otherwise, it's our keyword — linkify it!
                return f"[{text}]({url})"
            
            temp_body = pattern.sub(replace_callback, temp_body)
            
        # Write back if changed
        if temp_body != original_body:
            # Reconstruct the file content
            new_content = f"{prefix}---{fm_text}---{temp_body}"
            file_path.write_text(new_content, encoding='utf-8')
            modified_count += 1
            
    print(f"Interlinking complete! Modified {modified_count} of {len(files_to_process)} help files.")

if __name__ == "__main__":
    main()
