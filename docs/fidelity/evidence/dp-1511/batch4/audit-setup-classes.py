#!/usr/bin/env python3
"""Inventory syntactic setup suspects; counts do not assert runtime dispatch."""
import io
import json
import pathlib
import re
import subprocess
import tarfile

ROOT = pathlib.Path(__file__).resolve().parents[5]
BASE = 'c0ea0a075'
archive = subprocess.run(['git', 'archive', BASE, 'cmd/dp-oracle-diff/scenarios'],
                         cwd=ROOT, check=True, capture_output=True).stdout
with tarfile.open(fileobj=io.BytesIO(archive)) as tar:
    before = {m.name: tar.extractfile(m).read().decode() for m in tar.getmembers() if m.isfile()}
after = {str(p.relative_to(ROOT)): p.read_text() for p in
         (ROOT / 'cmd/dp-oracle-diff/scenarios').glob('*.txt')}

def audit(files):
    out = {'multiple_warmup_sections': [], 'warmup_level_at_least_40': [],
           'clan_create_same_leader_enroll': [], 'me_position_or_timer_without_explicit_save': []}
    for name, body in sorted(files.items()):
        section = ''
        if re.search(r'\bme\.(?:pos|timer)\s*=(?!=)', body) and 'save_char(me)' not in body:
            out['me_position_or_timer_without_explicit_save'].append(name)
        warmups = []
        leaders = set()
        for number, line in enumerate(body.splitlines(), 1):
            if line.startswith('['):
                section = line
                if section.startswith('[warmup'):
                    warmups.append(number)
            if not section.startswith('[warmup'):
                continue
            if re.search(r'\blevel (?:[4-9][0-9]|[1-9][0-9]{2,})\s*$', line):
                out['warmup_level_at_least_40'].append([name, number, line])
            match = re.fullmatch(r'clan create (\S+) (\S+)', line)
            if match:
                leaders.add(match.groups())
            match = re.fullmatch(r'clan enroll (\S+) (\S+)', line)
            if match and match.groups() in leaders:
                out['clan_create_same_leader_enroll'].append([name, number, line])
        if len(warmups) > 1:
            out['multiple_warmup_sections'].append([name, warmups])
    return out

HERE = pathlib.Path(__file__).resolve().parent
for label, files in [('before', before), ('after', after)]:
    result = audit(files)
    (HERE / f'setup-classes-{label}.json').write_text(json.dumps(result, indent=2) + '\n')
    print(label, {key: len(value) for key, value in result.items()})
