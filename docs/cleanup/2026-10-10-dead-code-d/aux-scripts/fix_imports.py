#!/usr/bin/env python3
"""Remove imports that became unused after symbol deletion (static, no build)."""
import re, subprocess, sys
from pathlib import Path

files = subprocess.run(['git', 'diff', '--name-only'], capture_output=True, text=True).stdout.split()
for f in files:
    p = Path(f)
    if not p.exists() or p.suffix != '.go':
        continue
    text = p.read_text()
    m = re.search(r'import \(\n(.*?)\n\)', text, re.S)
    if not m:
        continue
    block = m.group(1)
    body = text[:m.start()] + text[m.end():]
    body_nc = re.sub(r'//.*', '', body)
    keep, drop = [], []
    for line in block.splitlines():
        im = re.match(r'\s*(?:(\w+)\s+)?"([^"]+)"', line)
        if not im:
            keep.append(line)
            continue
        alias, path = im.group(1), im.group(2)
        name = alias or path.split('/')[-1]
        if re.search(r'\b' + re.escape(name) + r'\.', body_nc):
            keep.append(line)
        else:
            drop.append(f'{name} ({path})')
    if drop:
        newblock = '\n'.join(keep)
        text = text[:m.start()] + 'import (\n' + newblock + '\n)' + text[m.end():]
        p.write_text(text)
        print(f'{f}: dropped {", ".join(drop)}')
