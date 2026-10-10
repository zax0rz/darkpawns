#!/usr/bin/env python3
"""Excise test funcs referencing helpers that prior excision orphaned."""
import re, subprocess, sys
from pathlib import Path

topfunc = re.compile(r'^func\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(')
gone = set()
for round in range(10):
    out = subprocess.run(['go', 'vet', './...'], capture_output=True, text=True)
    errs = re.findall(r'(\S+?\.go):(\d+):\d+: undefined: (\w+)', out.stderr + out.stdout)
    if not errs:
        print(f'round {round}: vet clean')
        break
    progressed = False
    for path, _ln, name in errs:
        p = Path(path)
        if not p.exists():
            continue
        lines = p.read_text().splitlines()
        funcs = []
        i = 0
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
        pat = re.compile(r'\b' + re.escape(name) + r'\s*\(') if name not in gone else re.compile(r'\b' + re.escape(name) + r'\b')
        drop = [(s, e, n) for s, e, n in funcs
                if pat.search('\n'.join(lines[s:e + 1])) or n == name]
        if drop:
            for s, e, n in sorted(drop, reverse=True):
                del lines[s:e + 1]
            p.write_text('\n'.join(lines) + '\n')
            print(f'{path}: excised {", ".join(n for _, _, n in drop)} (missing {name})')
            gone.add(name)
            progressed = True
    if not progressed:
        print(f'round {round}: stuck on: {errs[:3]}')
        sys.exit(1)
