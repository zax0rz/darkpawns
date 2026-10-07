"""Compiling, assertion-failing producer and classifier controls; isolated worktree only."""
import argparse,pathlib,subprocess
parser=argparse.ArgumentParser();parser.add_argument('--output',required=True);parser.add_argument('--case');args=parser.parse_args()
root=pathlib.Path(args.output);root.mkdir(parents=True,exist_ok=True)
(root/'HEAD.txt').write_bytes(subprocess.check_output(['git','rev-parse','HEAD']))
cases=[]
for name,extension,noun in [('redit','wld','room'),('oedit','obj','objects'),('medit','mob','mob'),('sedit','shp','shop')]:
 old='\t\t\tlogOLCOpenParentFailure(s.manager.world, "'+extension+'", "SYSERR: OLC: Cannot open '+noun+' file!", err)\n'
 cases.append((name,pathlib.Path('pkg/session/'+name+'.go'),old,'','Test'+name.capitalize()+'OpenParentMudlog'))
classifier=pathlib.Path('pkg/session/olc_open_mudlog.go')
cases.extend([
 ('operation-guard',classifier,'(failure.Op != "open" && failure.Op != "mkdir")','false','TestOLCOpenParentObstructionClassifier'),
 ('cause-guard',classifier,'!errors.Is(err, os.ErrNotExist) && !errors.Is(err, syscall.ENOTDIR)','false','TestOLCOpenParentObstructionClassifier'),
 ('parent-guard',classifier,'return !info.IsDir()','return info.IsDir() || !info.IsDir()','TestOLCOpenParentObstructionClassifier'),
])
for name,path,old,new,test in cases:
 if args.case and args.case != name:continue
 original=path.read_text();assert original.count(old)==1,(name,original.count(old))
 try:
  for stage,source in [('green',original),('revert',original.replace(old,new)),('restore',original)]:
   path.write_text(source)
   result=subprocess.run(['go','test','./pkg/session','-run','^'+test+'$','-count=1'],stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True)
   (root/(name+'-'+stage+'.txt')).write_text(result.stdout+'\nEXIT='+str(result.returncode)+'\n')
   if stage=='revert':
    assert result.returncode!=0 and '--- FAIL: '+test in result.stdout and '[build failed]' not in result.stdout,result.stdout
   else:assert result.returncode==0,result.stdout
  print(name+': 1 -> 0 -> 1',flush=True)
 finally:path.write_text(original)
