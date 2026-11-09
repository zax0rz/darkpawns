#!/usr/bin/env python3
"""List warmup set tokens absent from C's field-prefix table at batch-2 base."""
import argparse
import io
import json
import pathlib
import re
import subprocess
import tarfile

ROOT = pathlib.Path(__file__).resolve().parents[5]
REVISION = '4b617576753dc078fadba1efcda290bc6ace3238'
archive = subprocess.run(
    ['git', 'archive', REVISION, 'src/act.wizard.c', 'cmd/dp-oracle-diff/scenarios'],
    cwd=ROOT, check=True, capture_output=True,
).stdout
with tarfile.open(fileobj=io.BytesIO(archive)) as tar:
    files = {m.name: tar.extractfile(m).read().decode() for m in tar.getmembers() if m.isfile()}
parser = argparse.ArgumentParser()
parser.add_argument('--working-tree', action='store_true', help='audit current scenario inputs instead of the pinned base')
args = parser.parse_args()
if args.working_tree:
    files = {str(p.relative_to(ROOT)): p.read_text() for p in (ROOT / 'cmd/dp-oracle-diff/scenarios').glob('*.txt')} | {'src/act.wizard.c': files['src/act.wizard.c']}
source = files['src/act.wizard.c'].split('ACMD(do_set)', 1)[1]
source = source.split('fields[] = {', 1)[1].split('};', 1)[0]
fields = re.findall(r'\{\s*"([a-z]+)"', source)
assert len(fields) == 59 and 'hit' in fields and 'align' in fields
bad = []
for name, body in sorted(files.items()):
    if not name.startswith('cmd/dp-oracle-diff/scenarios/') or not name.endswith('.txt'):
        continue
    section = ''
    for number, line in enumerate(body.splitlines(), 1):
        if line.startswith('['):
            section = line
        if not section.startswith('[warmup'):
            continue
        words = line.split()
        if words and words[0].startswith('send:'):
            words = words[1:]
        if words and words[0] == 'force':
            words = words[2:]
        if not words or words[0] != 'set':
            continue
        args = words[1:]
        if args and args[0] in ['player', 'mob', 'file']:
            args = args[1:]
        if len(args) > 1 and not any(field.startswith(args[1]) for field in fields):
            bad.append({'scenario': pathlib.Path(name).stem, 'line': number,
                        'field': args[1], 'input': line})
print(json.dumps(bad, indent=2))
