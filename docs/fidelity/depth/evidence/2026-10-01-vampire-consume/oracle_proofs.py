#!/usr/bin/env python3
"""Retain five-seed proofs and live 0/1/0 triples using only census start/wait."""
import os
import subprocess
from pathlib import Path
root = Path('/home/zach/Archives/darkpawns/oracle-runs/2026-10-01')
evidence = Path('docs/fidelity/depth/evidence/2026-10-01-vampire-consume')
scenarios = 'vampire-consume-day,vampire-consume-sunset,vampire-consume-dark'
env = os.environ.copy()
env['ORACLE_REGRESSION_LOG_ROOT'] = str(root/'dp-1371-p4-vampire-final-attempts')
def census(name, seed=1, expected=0):
 env['ORACLE_REGRESSION_SEED'] = str(seed)
 r=subprocess.run(['scripts/census.sh','start','--name',name,'--scenarios',scenarios],env=env,text=True,capture_output=True)
 assert r.returncode==0,r.stdout+r.stderr
 print(r.stdout.strip(),flush=True)
 while True:
  r=subprocess.run(['scripts/census.sh','wait','--run',str(root/name),'--max-seconds','40'],env=env,text=True,capture_output=True)
  print(r.stdout.strip(),flush=True)
  if r.returncode!=3:break
 assert r.returncode==expected,r.stdout+r.stderr
 return root/name
for seed in [2,3,5,8]:
 census(f'dp-1371-p4-vampire-final-seed{seed}',seed)
source=Path('pkg/game/item_consumable.go')
original=source.read_text()
clause='(GetSunlight() == SunSet || GetSunlight() == SunDark)'
assert original.count(clause)==2
rows=['scenario\tbefore\tmutant\tafter\tmutation']
try:
 for kind,value,bands in [('night','false',['sunset','dark']),('day','true',['day'])]:
  source.write_text(original.replace(clause,value))
  run=census(f'dp-1371-p4-vampire-{kind}-mutant',expected=1)
  results={line.split('\t')[1]:line.split('\t')[0] for line in (run/'results.tsv').read_text().splitlines()}
  for band in bands:assert results[f'vampire-consume-{band}']=='FAIL',results
  source.write_text(original)
  census(f'dp-1371-p4-vampire-{kind}-restored')
  for band in bands:rows.append(f'vampire-consume-{band}\t0\t1\t0\t{kind}-gate')
finally:source.write_text(original)
(evidence/'oracle-revert-triples.tsv').write_text('\n'.join(rows)+'\n')
