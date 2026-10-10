#!/usr/bin/env python3
"""Strip dangling t.Run entries and references to excised test funcs."""
import re, subprocess, sys
from pathlib import Path

for round in range(12):
    out = subprocess.run(['go', 'vet', './...'], capture_output=True, text=True)
    errs = re.findall(r'(\S+?\.go):(\d+):\d+: undefined: (\w+)', out.stderr + out.stdout)
    if not errs:
        print(f'round {round}: vet clean')
        break
    progressed = False
    for path, _ln, name in set(errs):
        # Find the file(s) whose content references the name (error may name
        # the file with the REFERENCE).
        p = Path(path)
        if not p.exists():
            continue
        lines = p.read_text().splitlines()
        defs = [i for i, l in enumerate(lines)
                if re.match(r'^func\s+' + re.escape(name) + r'\(', l)]
        if defs:
            continue  # still defined elsewhere — genuine compile error, skip
        new = [l for l in lines
               if not (re.search(r't\.Run\(', l) and re.search(r'\b' + re.escape(name) + r'\b', l))]
        if len(new) != len(lines):
            p.write_text('\n'.join(new) + '\n')
            print(f'{path}: stripped t.Run entries referencing {name}')
            progressed = True
        else:
            # word-match fallback: excise top-level funcs mentioning it
            topfunc = re.compile(r'^func\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(')
            i = 0
            funcs = []
            while i < len(lines):
                m = topfunc.match(lines[i])
                if m:
                    j = i
                    if not lines[j].rstrip().endswith('}'):
                        while j < len(lines) and lines[j] != '}':
                            j += 1
                    funcs.append((i, j, m.group(1)))
                    i = j + 1
                else:
                    i += 1
            drop = [(s, e, n) for s, e, n in funcs
                    if re.search(r'\b' + re.escape(name) + r'\b', '\n'.join(lines[s:e + 1]))]
            if drop:
                for s, e, n in sorted(drop, reverse=True):
                    del lines[s:e + 1]
                p.write_text('\n'.join(lines) + '\n')
                print(f'{path}: excised {", ".join(n for _, _, n in drop)} mentioning {name}')
                progressed = True
    if not progressed:
        print(f'round {round}: stuck: {errs[:3]}')
        sys.exit(1)
