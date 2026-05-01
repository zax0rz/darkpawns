#!/usr/bin/env python3
"""Split a Go source file by line ranges into new files, keeping the original.

Usage:
    python3 do_split.py <source.go> <newfile>:<start>-<end>[,<start2>-<end2>] ...

Example:
    python3 do_split.py pkg/game/skills.go skill_combat.go:298-665 skill_stealth.go:666-849
"""

import subprocess, sys, re, os

def guess_imports(content):
    needed = []
    # Common stdlib + project packages
    candidates = [
        ('"fmt"', 'fmt.'), ('"log/slog"', 'slog.'), ('"strings"', 'strings.'),
        ('"strconv"', 'strconv.'), ('"os"', 'os.'), ('"io"', 'io.'),
        ('"path/filepath"', 'filepath.'), ('"time"', 'time.'), ('"sync"', 'sync.'),
        ('"math/rand"', 'rand.'), ('"math"', 'math.'), ('"encoding/json"', 'json.'),
        ('"context"', 'context.'), ('"sort"', 'sort.'), ('"errors"', 'errors.'),
        ('"net/http"', 'http.'), ('"github.com/gorilla/websocket"', 'websocket.'),
        ('"github.com/zax0rz/darkpawns/pkg/combat"', 'combat.'),
        ('"github.com/zax0rz/darkpawns/pkg/game"', 'game.'),
        ('"github.com/zax0rz/darkpawns/pkg/parser"', 'parser.'),
        ('"github.com/zax0rz/darkpawns/pkg/scripting"', 'scripting.'),
        ('"github.com/zax0rz/darkpawns/pkg/common"', 'common.'),
        ('"github.com/zax0rz/darkpawns/pkg/db"', 'db.'),
        ('"github.com/zax0rz/darkpawns/pkg/events"', 'events.'),
        ('"github.com/zax0rz/darkpawns/pkg/command"', 'command.'),
        ('"github.com/zax0rz/darkpawns/pkg/game/systems"', 'systems.'),
    ]
    for imp, prefix in candidates:
        if prefix in content:
            needed.append(imp)
    return needed

def build_import_block(needed):
    if not needed:
        return ""
    stdlib = [n for n in needed if '"github.com/' not in n]
    external = [n for n in needed if '"github.com/' in n]
    parts = []
    if len(stdlib) == 1:
        parts.append(f"import {stdlib[0]}")
    elif stdlib:
        parts.append("import (\n\t" + "\n\t".join(stdlib) + "\n)")
    if len(external) == 1:
        parts.append(f"import {external[0]}")
    elif external:
        parts.append("import (\n\t" + "\n\t".join(external) + "\n)")
    return "\n".join(parts)

def fix_imports():
    while True:
        result = subprocess.run(["go", "build", "./..."], capture_output=True, text=True, cwd="/home/zach/darkpawns")
        if result.returncode == 0:
            break
        stderr = result.stderr
        unused_pattern = re.compile(r'^(?P<file>.+\.go):\d+:\d*:?\s*"(?P<imp>[^"]+)" imported and not used')
        fixes = []
        for line in stderr.splitlines():
            m = unused_pattern.match(line.strip())
            if m:
                fixes.append((m.group("file"), m.group("imp")))
        if not fixes:
            print("Build issues remain:")
            print(stderr[:2000])
            return False
        for filepath, imp in fixes:
            with open(filepath) as f:
                content = f.read()
            content = re.sub(rf'^\s*{re.escape(imp)}\s*\n', '', content, flags=re.MULTILINE)
            # Also handle single import
            content = re.sub(rf'^import \(\s*\n\s*{re.escape(imp)}\s*\n\s*\)', '', content, flags=re.MULTILINE)
            with open(filepath, "w") as f:
                f.write(content)
    return True

def main():
    if len(sys.argv) < 3:
        print(__doc__)
        sys.exit(1)

    source = sys.argv[1]
    with open(source) as f:
        all_lines = f.readlines()

    pkg_line = all_lines[0].strip()
    for line in all_lines:
        if line.strip().startswith("package "):
            pkg_line = line.strip()
            break
    file_ranges = {}

    for arg in sys.argv[2:]:
        name_part, ranges_part = arg.split(":", 1)
        franges = []
        for r in ranges_part.split(","):
            s, e = r.split("-")
            franges.append((int(s), int(e)))
        file_ranges[name_part] = franges

    new_files = []
    extracted_lines = set()

    for name, franges in file_ranges.items():
        franges.sort()
        all_content_lines = []
        for start, end in franges:
            all_content_lines.extend(all_lines[start-1:end])
            for i in range(start, end+1):
                extracted_lines.add(i)

        content = "".join(all_content_lines)
        needed = guess_imports(content)
        imports = build_import_block(needed)

        out_path = os.path.join(os.path.dirname(source), name)
        with open(out_path, "w") as f:
            f.write(pkg_line + "\n\n")
            if imports:
                f.write(imports + "\n\n")
            f.write(content)
        new_files.append((name, len(all_content_lines)))

    # Remove extracted lines from source, keeping non-extracted
    new_source_lines = []
    for i, line in enumerate(all_lines, start=1):
        if i not in extracted_lines:
            new_source_lines.append(line)

    with open(source, "w") as f:
        f.writelines(new_source_lines)

    for name, count in new_files:
        print(f"Created {os.path.join(os.path.dirname(source), name)} ({count} lines)")
    print(f"Updated {source} ({len(new_source_lines)} lines kept)")

    print("\nBuilding and fixing imports...")
    if not fix_imports():
        sys.exit(1)
    print("Build succeeded!")

if __name__ == "__main__":
    main()
