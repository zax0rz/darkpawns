#!/usr/bin/env python3
"""Fail-capable coverage: C branch witnesses plus compiling Go-byte mutations."""
import argparse,os,pathlib,subprocess
CASES=[('inventory','appendImprovementMessage("Got it!", improveSkillMessage(ch, SkillSteal))','appendImprovementMessage("BROKEN inventory witness", improveSkillMessage(ch, SkillSteal))','steal bread the trainee trailing words'),('missing-item',"$E hasn't got that item.",'$E has BROKEN missing item.','steal bread trainee'),('coins-zero',"You couldn't get any gold...",'BROKEN zero gold witness','steal coins trainee')]
def witnesses(text):
 blocks={}
 for chunk in text.split('--- [')[1:]:
  heading,body=chunk.split(']\n',1)
  blocks.setdefault(heading,[]).append(body.strip())
 assert any('Got it!' in s for s in blocks.get('steal bread the trainee trailing words',[])), 'inventory branch not reached'
 assert len(blocks.get('steal bread trainee',[]))==2 and all("hasn't got that item." in s for s in blocks['steal bread trainee']), 'missing-item branch not reached'
 assert any("You couldn't get any gold..." in s for s in blocks.get('steal coins trainee',[])), 'zero-gold branch not reached'
parser=argparse.ArgumentParser();parser.add_argument('--output',required=True);parser.add_argument('--check-dump');args=parser.parse_args()
root=pathlib.Path(args.output);root.mkdir(parents=True,exist_ok=True)
if args.check_dump:
 witnesses(pathlib.Path(args.check_dump).read_text());print('C branch witnesses: PASS');raise SystemExit(0)
p=pathlib.Path('pkg/game/skill_stealth.go');original=p.read_text()
(root/'HEAD.txt').write_bytes(subprocess.check_output(['git','rev-parse','HEAD']))
for name,old,new,step in CASES:
 assert original.count(old)==1,(name,original.count(old))
 try:
  for stage,source in [('green',original),('revert',original.replace(old,new)),('restore',original)]:
   p.write_text(source)
   dump=root/(name+'-'+stage+'-C');dump.mkdir(exist_ok=True)
   result=subprocess.run(['go','run','./cmd/dp-oracle-diff','--scenario','steal-depth','--show-oracle','--dump-oracle',str(dump)],stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True,env={**os.environ,'DP_ORACLE_BIN':os.environ.get('DP_ORACLE_BIN','/home/zach/darkpawns-c-oracle/bin/circle')})
   (root/(name+'-'+stage+'.txt')).write_text(result.stdout+'\nEXIT='+str(result.returncode)+'\n')
   assert '[build failed]' not in result.stdout and 'undefined:' not in result.stdout,result.stdout
   if stage=='revert':
    assert result.returncode!=0 and 'BROKEN ' in result.stdout and step in result.stdout,result.stdout
   else:
    assert result.returncode==0,result.stdout
    witnesses((dump/'steal-depth.txt').read_text())
  print(name+': 1 -> 0 -> 1',flush=True)
 finally:p.write_text(original)
