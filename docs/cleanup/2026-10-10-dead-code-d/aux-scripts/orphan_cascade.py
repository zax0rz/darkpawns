#!/usr/bin/env python3
"""Delete declarations golangci-lint's `unused` flags, iterating to clean."""
import re, subprocess, sys
from pathlib import Path

for round in range(8):
    out = subprocess.run(['golangci-lint', 'run', './...'], capture_output=True, text=True)
    hits = re.findall(r'^([^ :\n]+\.go):(\d+):(\d+): (?:func |type |var |const |field )?\(?\S*\)? ?([\w.]+) is unused', out.stdout, re.M)
    if not hits:
        print(f'round {round}: lint clean')
        break
    byfile = {}
    for path, ln, _col, name in hits:
        byfile.setdefault(path, []).append((int(ln), name))
    progressed = False
    for path, items in byfile.items():
        p = Path(path)
        lines = p.read_text().splitlines()
        # Locate each decl: walk up from the flagged line to `func|type|var|const`
        spans = []
        for ln, name in items:
            start = ln
            while start > 1 and not re.match(r'^(func|type|var|const)\b', lines[start - 1]):
                start -= 1
            end = start
            if not lines[end - 1].rstrip().endswith('}'):
                while end < len(lines) and lines[end] != '}':
                    end += 1
            # include contiguous doc comment above
            s = start
            while s - 2 >= 0 and lines[s - 2].lstrip().startswith('//'):
                s -= 1
            spans.append((s, end + 1, name))
        for s, e, name in sorted(spans, reverse=True):
            del lines[s - 1:e]
        p.write_text('\n'.join(lines) + '\n')
        print(f'{path}: removed {len(spans)}: {", ".join(n for _, n in sorted(items))}')
        progressed = True
    if not progressed:
        print('stuck'); sys.exit(1)
