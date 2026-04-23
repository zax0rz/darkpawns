#!/bin/bash
# View generated Dark Pawns maps

echo "================================================"
echo "DARK PAWNS - GENERATED MAPS VIEWER"
echo "================================================"
echo ""

echo "Available maps in $(pwd):"
echo "-------------------------"
ls -la *.txt *.dot *.html *.py *.md *.json 2>/dev/null | awk '{print $9}'
echo ""

echo "1. View Text Map (ASCII):"
echo "   cat text_map.txt | less"
echo ""

echo "2. View Graphviz DOT file:"
echo "   cat map_graph.dot | less"
echo ""

echo "3. Open Interactive Map in browser:"
echo "   Open 'interactive_map.html' in your web browser"
echo ""

echo "4. View Map Generation Report:"
echo "   cat MAP_GENERATION_REPORT.md | less"
echo ""

echo "5. Regenerate all maps:"
echo "   python3 generate_text_map.py"
echo "   python3 generate_interactive_map.py"
echo "   # python3 generate_visual_map.py  # Requires Graphviz"
echo ""

echo "6. Quick preview of text map:"
echo "--------------------------------"
head -30 text_map.txt
echo "..."
echo ""

echo "7. File sizes:"
echo "--------------"
du -h *.txt *.dot *.html *.json 2>/dev/null

echo ""
echo "================================================"
echo "For website integration:"
echo "- Use interactive_map.html as standalone page"
echo "- Convert map_graph.dot to SVG/PNG with Graphviz"
echo "- Include text_map.txt in documentation"
echo "================================================"