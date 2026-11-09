#!/usr/bin/env python3
"""Reproduce the raw warmup token inventory, including editor input and sends."""
import collections
import io
import subprocess
import tarfile
import pathlib
import re

root = pathlib.Path(__file__).resolve().parents[4]
# Freeze the inventory at the origin/main revision the brief was begun from.
revision = '441cadb40381faedc5719598cfc23570180ebc6a'
archive = subprocess.run(['git','archive',revision,'src/interpreter.c','cmd/dp-oracle-diff/scenarios'],cwd=root,check=True,capture_output=True).stdout
with tarfile.open(fileobj=io.BytesIO(archive)) as tar:
    files={member.name:tar.extractfile(member).read().decode() for member in tar.getmembers() if member.isfile()}
commands = {}
for line in files['src/interpreter.c'].splitlines():
    match = re.search(r'\{\s*"([^"]+)"\s*,\s*\w+\s*,\s*(do_\w+)', line)
    if match:
        commands[match[1]] = match[2]
forced = collections.Counter()
counts = collections.Counter()
scenarios = collections.defaultdict(set)
for name, body in sorted(files.items()):
    path = pathlib.Path(name)
    if not name.startswith('cmd/dp-oracle-diff/scenarios/') or path.suffix != '.txt':
        continue
    section = ''
    for raw in body.splitlines():
        text = raw.strip()
        if text.startswith('['):
            section = text
        elif section.startswith('[warmup') and text and not text.startswith('#'):
            if text.startswith('send:'):
                text = text.split(None, 1)[1]
            token = text.split()[0]
            if token == "force":
                forced[text.split()[2]] += 1
            counts[token] += 1
            scenarios[token].add(path.stem)
print('token\tsteps\tscenarios\tC handler or input kind\tforced steps')
for token in sorted(counts.keys() | forced.keys()):
    count = counts[token]
    kind = commands.get(token, 'editor/text input (not a registered verb)')
    if token.startswith(('~', '<', '/')):
        kind = 'harness directive' if not token.startswith('/') else 'text editor control'
    print(f'{token}\t{count}\t{len(scenarios[token])}\t{kind}\t{forced[token]}')
