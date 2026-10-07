#!/usr/bin/env python3
"""R5h controls using Go overlays; leaves the checkout unchanged."""
import argparse, json, os, pathlib, subprocess, tempfile
p=argparse.ArgumentParser();p.add_argument('--output', required=True);p.add_argument('--case');a=p.parse_args()
root=pathlib.Path(a.output).expanduser();root.mkdir(parents=True,exist_ok=True)
(root/'HEAD.txt').write_bytes(subprocess.check_output(['git','rev-parse','HEAD']))
controls=[('zone-open','pkg/session/zedit.go','logOLCOpenParentFailure(s.manager.world, "zon", fmt.Sprintf("SYSERR: OLC: zedit_save_to_disk:  Can\'t write zone %d.", zone.Number), err)','', 'TestZeditOpenParentMudlog')]
for name,file,before,after,test in controls:
 if a.case and a.case!=name:continue
 source=pathlib.Path(file).read_text();assert source.count(before)==1,(name,source.count(before))
 folder=root/name;folder.mkdir(exist_ok=True)
 with tempfile.TemporaryDirectory() as tmp:
  replacement=pathlib.Path(tmp)/pathlib.Path(file).name
  replacement.write_text(source.replace(before,after))
  overlay=pathlib.Path(tmp)/'overlay.json';overlay.write_text(json.dumps({'Replace':{str(pathlib.Path(file).resolve()):str(replacement)}}))
  for stage in ['green','revert','restore']:
   cmd=['go','test','-p','2']+(['-overlay='+str(overlay)] if stage=='revert' else [])+['./pkg/session','-run','^'+test+'$','-count=1']
   r=subprocess.run(cmd,env=dict(os.environ,GOMAXPROCS='2'),text=True,stdout=subprocess.PIPE,stderr=subprocess.STDOUT)
   (folder/(stage+'.txt')).write_text(r.stdout+'\nEXIT='+str(r.returncode)+'\n')
   if stage=='revert':assert r.returncode!=0 and '[build failed]' not in r.stdout and '--- FAIL: '+test in r.stdout,r.stdout
   else:assert r.returncode==0,r.stdout
 print(name,'PASS 0/1/0',flush=True)
