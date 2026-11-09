#!/usr/bin/env python3
"""Reproduce the raw warmup token inventory, including editor input and sends."""
import collections
import pathlib
import re

root = pathlib.Path(__file__).resolve().parents[4]
commands = {}
for line in (root / 'src/interpreter.c').read_text().splitlines():
    match = re.search(r'\{\s*"([^"]+)"\s*,\s*\w+\s*,\s*(do_\w+)', line)
    if match:
        commands[match[1]] = match[2]
counts = collections.Counter()
scenarios = collections.defaultdict(set)
for path in sorted((root / 'cmd/dp-oracle-diff/scenarios').glob('*.txt')):
    section = ''
    for raw in path.read_text().splitlines():
        text = raw.strip()
        if text.startswith('['):
            section = text
        elif section.startswith('[warmup') and text and not text.startswith('#'):
            if text.startswith('send:'):
                text = text.split(None, 1)[1]
            token = text.split()[0]
            counts[token] += 1
            scenarios[token].add(path.stem)
print('token\tsteps\tscenarios\tC handler or input kind')
for token, count in sorted(counts.items()):
    kind = commands.get(token, 'editor/text input (not a registered verb)')
    if token.startswith(('~', '<', '/')):
        kind = 'harness directive' if not token.startswith('/') else 'text editor control'
    print(f'{token}\t{count}\t{len(scenarios[token])}\t{kind}')
