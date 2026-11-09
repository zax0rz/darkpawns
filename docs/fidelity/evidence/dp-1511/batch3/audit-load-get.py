#!/usr/bin/env python3
"""Inventory adjacent direct warmup load-obj/get pairs; candidates, not verdicts."""
import argparse
import csv
import io
import pathlib
import subprocess
import sys
import tarfile

ROOT = pathlib.Path(__file__).resolve().parents[5]
REVISION = 'ed3c89f988983ffbd6807da44ad3e6e5e4df60b6'
parser = argparse.ArgumentParser()
parser.add_argument('--working-tree', action='store_true')
args = parser.parse_args()
if args.working_tree:
    files = {str(p.relative_to(ROOT)): p.read_text() for p in (ROOT / 'cmd/dp-oracle-diff/scenarios').glob('*.txt')}
else:
    archive = subprocess.run(['git', 'archive', REVISION, 'cmd/dp-oracle-diff/scenarios'],
                             cwd=ROOT, check=True, capture_output=True).stdout
    with tarfile.open(fileobj=io.BytesIO(archive)) as tar:
        files = {m.name: tar.extractfile(m).read().decode() for m in tar.getmembers() if m.isfile()}
writer = csv.writer(sys.stdout, delimiter='\t', lineterminator='\n')
writer.writerow(['scenario', 'load_line', 'load_input', 'get_line', 'get_input'])
for name, text in sorted(files.items()):
    if not name.endswith('.txt'):
        continue
    previous = None
    section = ''
    for number, raw in enumerate(text.splitlines(), 1):
        line = raw.strip()
        if line.startswith('['):
            section = line
            previous = None
        if not section.startswith('[warmup') or not line or line.startswith('#'):
            continue
        words = line.split()
        if words[0] == 'get' and previous:
            before = previous[1].split()
            if len(before) >= 3 and before[0] == 'load' and before[1] in ['obj', 'object']:
                writer.writerow([pathlib.Path(name).stem, previous[0], previous[1], number, line])
        previous = (number, line)
