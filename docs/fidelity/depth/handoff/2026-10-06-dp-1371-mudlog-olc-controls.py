"""Run at train tip; output directory and optional control name. R5h."""
import pathlib
import subprocess
import sys

root=pathlib.Path(sys.argv[1]);root.mkdir(parents=True,exist_ok=True)
controls=[]
for command in ['redit','zedit','oedit','medit','sedit']:
 path='pkg/session/'+command+'.go'
 text=pathlib.Path(path).read_text()
 line=next((line for line in text.splitlines() if 'game.MudLog(fmt.Sprintf("OLC:' in line), '')
 if line: controls.append((command,path,line,'// producer removed for control','TestOLCDiskMudlogProducers/'+command))
path='pkg/session/wiz_system.go'
text=pathlib.Path(path).read_text()
line=next(line for line in text.splitlines() if 'game.MudLog(fmt.Sprintf("(GC) %s forced all to save"' in line)
controls.append(('shutdown',path,line,'// producer removed for control','TestShutdownForceMudlog'))
for name,before,after in [
 ('olc-type','game.MudlogComplete, LVL_IMMORT','game.MudlogNormal, LVL_IMMORT'),
 ('olc-threshold','game.MudlogComplete, LVL_IMMORT','game.MudlogComplete, LVL_IMMORT-1'),
]:controls.append((name,'pkg/session/redit.go',before,after,'TestOLCDiskMudlogProducers/redit'))
text=pathlib.Path('pkg/session/redit.go').read_text()
line=next((line for line in text.splitlines() if 'game.MudLog(fmt.Sprintf("OLC:' in line), '')
ack='\t\ts.reditSend("Saving all rooms in zone.\\r\\n")'
controls.append(('olc-ack-order','pkg/session/redit.go',ack+'\n'+line,line+'\n'+ack,'TestOLCDiskMudlogProducers/redit'))
start=text.index(ack);end=text.index('\n\t\treturn nil',start)
block=text[start:end]
controls.append(('olc-write-order','pkg/session/redit.go',block,block.replace('\n'+line,'')+'\n'+line,'TestOLCDiskMudlogProducers/redit'))
controls.append(('shutdown-threshold','pkg/session/wiz_system.go','max(caster.player.GetLevel()+1, caster.player.GetInvisLevel())','max(caster.player.GetLevel(), caster.player.GetInvisLevel())','TestShutdownForceMudlog'))
controls += [
 ('shutdown-log-command','pkg/session/wiz_system.go','forced all to save','forced all to all save','TestShutdownForceMudlog'),
 ('shutdown-notice-command','pkg/session/wiz_system.go',"has forced you to 'save'", "has forced you to 'all save'", 'TestShutdownForceMudlog'),
]
for name,filename,before,after,test in controls:
 if len(sys.argv)>2 and sys.argv[2]!=name:continue
 path=pathlib.Path(filename);original=path.read_text()
 if before not in original:sys.exit('mutation missing: '+name)
 pattern='^'+test.replace('/','$/^') # command prefix includes success/failure subtests
 def run(label):
  r=subprocess.run(['go','test','./pkg/session','-run',pattern,'-count=1'],capture_output=True,text=True)
  (root/(name+'-'+label+'.txt')).write_text(r.stdout+r.stderr)
  return r
 if run('green').returncode:sys.exit('initial failure: '+name)
 try:
  path.write_text(original.replace(before,after,1))
  r=run('revert')
  if not r.returncode or '--- FAIL: '+test.split('/')[0] not in r.stdout or '[build failed]' in r.stdout+r.stderr:sys.exit('invalid red: '+name)
 finally:path.write_text(original)
 if run('restore').returncode:sys.exit('restore failed: '+name)
 print(name+': 1 -> 0 -> 1',flush=True)
