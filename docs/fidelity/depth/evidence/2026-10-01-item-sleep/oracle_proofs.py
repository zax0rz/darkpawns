#!/usr/bin/env python3
"""Run only retained census start/wait; then prove all four live entries can fail."""
import os
import subprocess
from pathlib import Path
root=Path('/home/zach/Archives/darkpawns/oracle-runs/2026-10-01')
selection=','.join('item-sleep-'+s for s in ['outlaw-sand','outlaw-bare','refusal-sand','refusal-bare'])
env=os.environ.copy();env['ORACLE_REGRESSION_LOG_ROOT']=str(root/'dp-1371-p4-item-sleep-attempts')
def census(name,seed=1,expected=0):
 env['ORACLE_REGRESSION_SEED']=str(seed)
 r=subprocess.run(['scripts/census.sh','start','--name',name,'--scenarios',selection],env=env,text=True,capture_output=True)
 assert r.returncode==0,r.stdout+r.stderr
 print(r.stdout.strip(),flush=True)
 while True:
  r=subprocess.run(['scripts/census.sh','wait','--run',str(root/name),'--max-seconds','40'],env=env,text=True,capture_output=True)
  print(r.stdout.strip(),flush=True)
  if r.returncode!=3:break
 assert r.returncode==expected,r.stdout+r.stderr
 return root/name
for seed in [1,3,5,8]:census(f'dp-1371-p4-item-sleep-seed{seed}',seed)
p=Path('pkg/spells/call_magic.go');original=p.read_text()
assert original.count('if si == nil {')==1
try:
 p.write_text(original.replace('if si == nil {','if si == nil || spellNum == SpellSleep {',1))
 d=census('dp-1371-p4-item-sleep-oracle-mutant',expected=1)
 results=[s.split('\t') for s in (d/'results.tsv').read_text().splitlines()]
 assert len(results)==4 and all(s[0]=='FAIL' for s in results),results
finally:p.write_text(original)
census('dp-1371-p4-item-sleep-oracle-restored')
e=Path('docs/fidelity/depth/evidence/2026-10-01-item-sleep')
(e/'oracle-revert-triples.tsv').write_text('scenario\tbefore\tmutant\tafter\tfailure\n'+''.join(f'{s}\t0\t1\t0\ttranscript\n' for s in selection.split(',')))
