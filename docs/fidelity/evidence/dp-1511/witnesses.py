#!/usr/bin/env python3
"""Find actual C probe lines for refusal tests; never synthesize witnesses."""
import json
import pathlib
import re
root = pathlib.Path(__file__).resolve().parents[4]
source = (root/'internal/oraclediff/setuprefusal_inventory.go').read_text()
entries=[]
for m in re.finditer(r'(?:refusalPattern\(|\{)("(?:[^"\\]|\\.)*"), ("(?:[^"\\]|\\.)*")',source):
    template,cite=map(json.loads,m.groups())
    expr='';pos=0
    for slot in re.finditer(r'%[-+ #0.0-9hlz]*[sdiuoxXfFeEgGc]|\$[nNmMsSeEoOpPaATF]',template):
        expr+=re.escape(template[pos:slot.start()])+('[+-]?[0-9]{1,20}' if slot[0][-1] in 'diu' and slot[0][0]=='%' else '[^\r\n]{1,1000}')
        pos=slot.end()
    expr+=re.escape(template[pos:])
    entries.append((template,cite,re.compile('^'+expr+'$')))
witness_path=root/'docs/fidelity/evidence/dp-1511/refusal-witnesses.json'
found={row['template']:row for row in json.loads(witness_path.read_text())} if witness_path.exists() else {}
current={entry[0] for entry in entries}
found={key:row for key,row in found.items() if key in current}
remaining=[entry for entry in entries if entry[0] not in found]
archive=pathlib.Path.home()/'Archives/darkpawns/oracle-runs'
files=[]
for date in ['2026-11-08','2026-11-07','2026-11-06','2026-11-05']:
    files.extend(sorted((archive/date).glob('*/census-dump/*.txt')))
seen_lines=set()
for path in files:
    text=path.read_text(errors='replace')
    if not text.startswith('normalized C oracle blocks:'):continue
    for number,line in enumerate(text.splitlines(),1):
        if line.startswith(('normalized Go','Go port','normalized port')):break
        if line.startswith(('--- [','normalized C')):continue
        line=re.sub(r'\x1b\[[0-9;]*m','',line).strip()
        if line in seen_lines:continue
        seen_lines.add(line)
        for template,cite,pattern in remaining:
            if pattern.fullmatch(line):
                found[template]={'template':template,'cite':cite,'line':line,'dump':str(path),'dump_line':number}
    remaining=[entry for entry in remaining if entry[0] not in found]
    if not remaining:break
out=root/'docs/fidelity/evidence/dp-1511/refusal-witnesses.json'
out.write_text(json.dumps(list(found.values()),indent=2)+'\n')
missing=[t for t,c,p in entries if t not in found]
(root/'docs/fidelity/evidence/dp-1511/missing-witnesses.json').write_text(json.dumps(missing,indent=2)+'\n')
print(len(found),'witnessed;',len(missing),'missing; searched',len(files),'retained dumps')
