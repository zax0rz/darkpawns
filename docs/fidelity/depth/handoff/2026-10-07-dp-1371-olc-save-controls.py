"""Run in an isolated worktree; compiling mutations must fail named assertions (R5h)."""
import argparse,pathlib,subprocess
parser=argparse.ArgumentParser();parser.add_argument('--output',required=True);parser.add_argument('--case');args=parser.parse_args()
root=pathlib.Path(args.output);root.mkdir(parents=True,exist_ok=True)
(root/'HEAD.txt').write_bytes(subprocess.check_output(['git','rev-parse','HEAD']))
listing=pathlib.Path('pkg/olc/save_list.go');command=pathlib.Path('pkg/session/olc_control.go');registry=pathlib.Path('pkg/session/commands.go')
final=next(line for line in command.read_text().splitlines(True) if 'game.MudLog(fmt.Sprintf("OLC:' in line)
ack=next(line for line in command.read_text().splitlines(True) if 'saved for zone %d.' in line)
error_return='\t\t\tslog.Error("olc disk save failed", "player", s.playerName, "kind", entry.Kind, "zone", entry.Zone, "error", err)\n\t\t\treturn\n'
cases=[
('ordered-view',listing,'func (s *SaveList) Ordered() []DirtyEntry {\n\ts.mu.Lock()\n\tdefer s.mu.Unlock()\n\treturn slices.Clone(s.order)\n}','func (s *SaveList) Ordered() []DirtyEntry { return s.List() }','./pkg/olc','TestSaveListCInsertionOrder'),
('repeat-mark',listing,'\tif _, exists := s.dirty[entry]; exists {\n\t\treturn\n\t}\n','','./pkg/olc','TestSaveListCInsertionOrder'),
('remove-order',listing,'\tif index := slices.Index(s.order, entry); index >= 0 {\n\t\ts.order = slices.Delete(s.order, index, index+1)\n\t}\n','','./pkg/olc','TestSaveListCInsertionOrder'),
('registration',registry,'\tregisterCommand("olc", wrapArgs(cmdOlc), "List or save unsaved OLC components.")\n','','./pkg/session','TestOLCDispatchAndSaveInfo'),
('save-info-order',command,'func (s *Session) olcSaveInfo() {\n\tentries := olcSaveList.Ordered()','func (s *Session) olcSaveInfo() {\n\tentries := olcSaveList.List()','./pkg/session','TestOLCDispatchAndSaveInfo'),
('save-prefix',command,'strings.HasPrefix(first, "save")','first == "save"','./pkg/session','TestOLCSaveAllSuccess'),
('final-producer',command,final,'','./pkg/session','TestOLCSaveAllSuccess'),
('final-type',command,final,final.replace('game.MudlogComplete','game.MudlogNormal'),'./pkg/session','TestOLCSaveAllSuccess'),
('final-level',command,final,final.replace('game.LVL_IMMORT','game.LVL_GOD'),'./pkg/session','TestOLCSaveAllSuccess'),
('final-file',command,final,final.replace(', true)',', false)'),'./pkg/session','TestOLCSaveAllSuccess'),
('acting-name',command,final,final.replace('s.player.GetName()','s.playerName'),'./pkg/session','TestOLCSaveAllOnlyDirtyKinds'),
('actual-writer',command,'\treturn writer(world, &zone)','\t_ = writer\n\t_ = zone\n\tclearOLCDirty(entry.Kind, entry.Zone)\n\treturn nil','./pkg/session','TestOLCSaveAllSuccess'),
('bounded-retry',command,error_return,error_return.replace('\t\t\treturn\n','\t\t\tcontinue\n'),'./pkg/session','TestOLCSaveAllBoundedFailure'),
('failed-marker',command,'\t\t\tlogOLCSaveAllFailure(s.manager.world, entry, err)','\t\t\tclearOLCDirty(entry.Kind, entry.Zone)\n\t\t\tlogOLCSaveAllFailure(s.manager.world, entry, err)','./pkg/session','TestOLCSaveAllBoundedFailure'),
('error-diagnostic',command,'\t\t\tlogOLCSaveAllFailure(s.manager.world, entry, err)\n','','./pkg/session','TestOLCSaveAllBoundedFailure'),
('failure-success-log',command,error_return,error_return.replace('\t\t\treturn\n',final+'\t\t\treturn\n'),'./pkg/session','TestOLCSaveAllBoundedFailure'),
]
# Move the acknowledgement after the complete writer/error branch. It remains
# before the final success producer, so only the error-time ordering proof catches it.
body=command.read_text();start=body.index(ack);end=body.index('\n\t}\n\t// src/olc.c:343',start)
fragment=body[start:end]
cases.append(('ack-before-writer',command,fragment,fragment.replace(ack,'')+'\n'+ack.rstrip('\n'),'./pkg/session','TestOLCSaveAllBoundedFailure'))
if args.case:assert args.case in [c[0] for c in cases],args.case
for name,path,old,new,package,test in cases:
 if args.case and args.case!=name:continue
 original=path.read_text();assert original.count(old)==1,(name,original.count(old))
 try:
  for stage,source in [('green',original),('revert',original.replace(old,new)),('restore',original)]:
   path.write_text(source)
   result=subprocess.run(['go','test',package,'-run','^'+test+'$','-count=1'],stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True,timeout=120)
   (root/(name+'-'+stage+'.txt')).write_text(result.stdout+'\nEXIT='+str(result.returncode)+'\n')
   if stage=='revert':assert result.returncode!=0 and '--- FAIL: '+test in result.stdout and '[build failed]' not in result.stdout and 'panic: test timed out' not in result.stdout,result.stdout
   else:assert result.returncode==0,result.stdout
  print(name+': 1 -> 0 -> 1',flush=True)
 finally:path.write_text(original)
