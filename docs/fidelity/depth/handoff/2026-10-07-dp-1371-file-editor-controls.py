"""Run in a clean isolated worktree; each compiling revert must fail its named assertion."""
import argparse, pathlib, subprocess
parser=argparse.ArgumentParser();parser.add_argument('--output',required=True);parser.add_argument('--case');args=parser.parse_args()
root=pathlib.Path(args.output);root.mkdir(parents=True,exist_ok=True)
(root/'HEAD.txt').write_bytes(subprocess.check_output(['git','rev-parse','HEAD']))
tedit=pathlib.Path('pkg/session/tedit.go');helper=pathlib.Path('pkg/session/file_editor_mudlog.go');lua=pathlib.Path('pkg/session/luaedit.go')
cases=[]
for name,fragment,test in [
 ('save-producer','game.MudLog(fmt.Sprintf("OLC: %s saves', 'TestFileEditorSaveMudlog'),
 ('delete-producer','game.MudLog(fmt.Sprintf("OLC: %s deletes','TestFileEditorDeleteMudlog'),
 ('delete-error-producer','game.MudLog(fmt.Sprintf("SYSERR: Can\'t delete','TestFileEditorDeleteErrorMudlog'),
 ('open-error-producer','game.MudLog(fmt.Sprintf("SYSERR: Can\'t write','TestFileEditorOpenParentMudlog')]:
 line=next(line for line in tedit.read_text().splitlines(True) if fragment in line)
 cases.append((name,tedit,line,'',test))
 if name in ['save-producer','delete-producer']:
  cases.append((name+'-threshold',tedit,line,line.replace('game.LVL_GOD','game.LVL_IMPL'),test))
 else:
  cases.append((name+'-threshold',tedit,line,line.replace('game.LVL_IMPL','game.LVL_IMPL-1'),test))
cases.extend([
 ('tedit-storage',tedit,'"text/" + field.filename','"text/" + field.name','TestFileEditorStorageIdentity'),
 ('lua-storage',lua,'storageDir + "/" + filename','"scripts/" + filename','TestFileEditorStorageIdentity'),
 ('readability',helper,'unix.Access(path, unix.R_OK) != nil','unix.Access(path, unix.R_OK) != nil && path == ""','TestFileEditorDeleteMudlog'),
 ('acting-body',helper,'return s.switchedMob.GetName()','return s.playerName','TestFileEditorActingBody'),
 ('open-stage',helper,'failure.Op != "open"','false','TestFileEditorOpenStageClassifier'),
 ('open-root-path',helper,'failure.Path != parent','false','TestFileEditorOpenStageClassifier'),
])
if args.case:assert args.case in [c[0] for c in cases],args.case
for name,path,old,new,test in cases:
 if args.case and args.case!=name:continue
 original=path.read_text();assert original.count(old)==1,(name,original.count(old))
 try:
  for stage,source in [('green',original),('revert',original.replace(old,new)),('restore',original)]:
   path.write_text(source)
   result=subprocess.run(['go','test','./pkg/session','-run','^'+test+'$','-count=1'],stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True)
   (root/(name+'-'+stage+'.txt')).write_text(result.stdout+'\nEXIT='+str(result.returncode)+'\n')
   if stage=='revert':assert result.returncode!=0 and '--- FAIL: '+test in result.stdout and '[build failed]' not in result.stdout,result.stdout
   else:assert result.returncode==0,result.stdout
  print(name+': 1 -> 0 -> 1',flush=True)
 finally:path.write_text(original)
