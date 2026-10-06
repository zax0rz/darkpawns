"""Compiling R5h triples from train tip; output directory, optional control."""
import pathlib
import subprocess
import sys

root=pathlib.Path(sys.argv[1]);root.mkdir(parents=True,exist_ok=True)
controls=[]
path='pkg/session/zedit.go';text=pathlib.Path(path).read_text()
line=next(line for line in text.splitlines() if 'game.MudLog(fmt.Sprintf("OLC: %s creates new zone' in line)
test='TestNewZoneMudlogSuccess'
controls.extend([
 ('success',path,line,'// producer removed',test),
 ('success-type',path,line,line.replace('MudlogBrief','MudlogComplete'),test),
 ('success-minimum',path,line,line.replace('LVL_IMMORT,','LVL_IMMORT-1,'),test),
 ('success-invis',path,line,line.replace('LVL_IMMORT,','max(LVL_IMMORT, s.player.GetInvisLevel()),'),test),
])
ack='\ts.zeditSend("Zone created.\\r\\n")'
controls.append(('success-ack-order',path,line+'\n'+ack,ack+'\n'+line,test))
start=text.index('\tif _, ok := world.CreateZone(number)');end=text.index(line,start)+len(line)
block=text[start:end]
controls.append(('success-world-order',path,block,line+'\n'+block.replace(line,''),test))
start=text.index('\tfor _, ext :=',start);block=text[start:end]
controls.append(('success-index-order',path,block,line+'\n'+block.replace(line,''),test))
path='pkg/session/zedit_new_zone_mudlog.go';text=pathlib.Path(path).read_text()
for name,word in [('Zon','zone'),('Wld','world'),('Mob','mob'),('Obj','obj'),('Shp','shop')]:
 line=next(line for line in text.splitlines() if "Can't write new "+word+' file' in line)
 test='TestNewZoneMudlog'+name
 controls.extend([
 (name.lower(),path,line,'// producer removed',test),
 (name.lower()+'-type',path,line,line.replace('MudlogBrief','MudlogComplete'),test),
 (name.lower()+'-minimum',path,line,line.replace('game.LVL_IMPL,','game.LVL_IMPL-1,'),test),
 ])
controls.append(('open-only',path,'failure.Op != "open"','failure.Op == "open"','TestNewZoneMudlogOpenClassifier'))
path='pkg/olc/new_zone.go';text=pathlib.Path(path).read_text();a=text.index('\t\t{Extension: "zon"');b=text.index('\t\t{Extension: "mob"',a)
block=text[a:b];lines=block.splitlines(keepends=True)
controls.append(('file-order',path,block,''.join(reversed(lines)),'TestNewZoneMudlogZon'))
for name,filename,before,after,test in controls:
 if len(sys.argv)>2 and sys.argv[2]!=name:continue
 path=pathlib.Path(filename);original=path.read_text()
 if before not in original:sys.exit('missing mutation: '+name)
 def run(label):
  r=subprocess.run(['go','test','./pkg/session','-run','^'+test+'$','-count=1'],capture_output=True,text=True)
  (root/(name+'-'+label+'.txt')).write_text(r.stdout+r.stderr)
  return r
 if run('green').returncode:sys.exit('initial failure: '+name)
 try:
  path.write_text(original.replace(before,after,1))
  r=run('revert')
  if not r.returncode or '--- FAIL: '+test not in r.stdout or '[build failed]' in r.stdout+r.stderr:sys.exit('invalid red: '+name)
 finally:path.write_text(original)
 if run('restore').returncode:sys.exit('restore failed: '+name)
 print(name+': 1 -> 0 -> 1',flush=True)
