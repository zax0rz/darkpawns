#!/usr/bin/env python3
"""Find rejected sex inputs in recognizable new-character setup frames."""
import argparse
import io
import json
import pathlib
import subprocess
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
rows = []
for name, text in sorted(files.items()):
    if not name.endswith('.txt'):
        continue
    section = ''
    lines = []
    for line in text.splitlines() + ['[end]']:
        if line.startswith('['):
            if (section.startswith('[setup:') and len(lines) >= 7 and lines[1].lower() == 'y'
                    and lines[2] == lines[3] and lines[4].lower() in ['y', 'n']
                    and lines[5][:1].lower() not in ['m', 'f']):
                # C's CON_QSEX switches on *arg, not the entire word.
                recovery = next((value for value in lines[6:] if value[:1].lower() in ['m', 'f']), None)
                rows.append({'scenario': pathlib.Path(name).stem, 'section': section,
                             'name': lines[0], 'sex_token': lines[5],
                             'later_accepted_sex_input': recovery})
            section = line
            lines = []
        elif section.startswith('[setup:') and line and not line.startswith('#'):
            lines.append(line)
print(json.dumps(rows, indent=2))
