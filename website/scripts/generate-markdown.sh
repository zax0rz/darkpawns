#!/bin/bash
# Generate .md files from Hugo content for content negotiation
# Run after: hugo --minify

WEBSITE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PUBLIC_DIR="$WEBSITE_DIR/public"
CONTENT_DIR="$WEBSITE_DIR/content"

echo "Generating markdown files for content negotiation..."

# Generate .md for regular pages (not _index.md)
find "$CONTENT_DIR" -name '*.md' -not -name '_index.md' | while read -r src; do
    rel="${src#$CONTENT_DIR/}"
    rel="${rel%.md}"
    dest="$PUBLIC_DIR/$rel/index.md"
    
    mkdir -p "$(dirname "$dest")"
    
    # Strip YAML frontmatter (two --- blocks) then write
    awk '/^---$/{n++; next} n>=2{print}' "$src" > "$dest"
done

# Generate .md for section pages (_index.md files)
find "$CONTENT_DIR" -name '_index.md' | while read -r src; do
    rel="${src#$CONTENT_DIR/}"
    rel="${rel%_index.md}"
    rel="${rel%/}"
    
    if [ -n "$rel" ]; then
        dest="$PUBLIC_DIR/$rel/index.md"
        mkdir -p "$(dirname "$dest")"
        awk '/^---$/{n++; next} n>=2{print}' "$src" > "$dest"
    fi
done

echo "Done. Markdown files generated."
