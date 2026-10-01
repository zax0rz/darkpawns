#!/usr/bin/env python3
"""R5h assertion-only mutation triples in a disposable proof checkout."""
from pathlib import Path
import subprocess
import sys
candidate,proof,logs=map(lambda p:Path(p).resolve(),sys.argv[1:4])
logs.mkdir(parents=True,exist_ok=True)
files=subprocess.check_output(['git','diff','origin/main','--name-only'],cwd=candidate,text=True).splitlines()+subprocess.check_output(['git','ls-files','--others','--exclude-standard'],cwd=candidate,text=True).splitlines()
for f in files:
 if f.endswith('.go'):
  target=proof/f;target.parent.mkdir(parents=True,exist_ok=True);target.write_bytes((candidate/f).read_bytes())
mutations=[
 ('deleted-gate','pkg/session/session_login.go','&& game.CharacterDataDeleted(rec.CharacterData)','&& false && game.CharacterDataDeleted(rec.CharacterData)','./pkg/session','TestEntryDeletedRecordStartsFresh|TestEntryDeletedWebSocketBoundary'),
 ('deleted-fold','pkg/session/session_login.go','s.startNewCharFlow(strings.ToLower(login.PlayerName))','s.startNewCharFlow(login.PlayerName)','./pkg/session','TestEntryDeletedRecordStartsFresh'),
 ('sole-bootstrap','pkg/session/char_creation.go','count == 1 && s.creationReplacement != nil','false','./pkg/session','TestEntryDeletedSoleRecordBootstrap'),
 ('replacement-save','pkg/session/char_creation.go','if s.creationReplacement != nil {','if false && s.creationReplacement != nil {','./pkg/session','TestEntryDeletedReplacementLifecycle'),
 ('N-reset','pkg/session/char_creation.go','case "N":\n\t\t\ts.creationReplacement = nil','case "N":','./pkg/session','TestEntryDeletedAbandonKeepsRecord'),
 ('save-failure','pkg/session/char_creation.go','if saveErr != nil {','if false && saveErr != nil {','./pkg/session','TestEntryDeletedReplacementFailureCloses'),
 ('menu-marker','pkg/session/menu.go','s.player.GetLevel() < game.LVL_GRGOD || deleted','false','./pkg/session','TestEntryDeletedMenuReuse'),
 ('menu-objects','pkg/session/menu.go','record.Inventory, record.Equipment = []byte("[]"), []byte("{}")','// omit crash-object removal','./pkg/session','TestEntryDeletedMenuReuse'),
 ('menu-level','pkg/session/menu.go','s.player.GetLevel() < game.LVL_GRGOD','s.player.GetLevel() <= game.LVL_GRGOD','./pkg/session','TestEntryDeletedMenuLevelGate'),
 ('snapshot-guard','pkg/db/player.go',"COALESCE(character_data,'{}') IS ?",'(? IS NOT NULL)','./pkg/db','TestDeletedReplacementAtomic'),
 ('rollback','pkg/db/player.go','if err := tx.QueryRow(query, args...).Scan(&newID); err != nil {\n\t\treturn err','if err := tx.QueryRow(query, args...).Scan(&newID); err != nil {\n\t\t_ = tx.Commit()\n\t\treturn err','./pkg/db','TestDeletedReplacementAtomic'),
]
rows=['mutation\tbefore\tmutant\tafter\tfailure\n']
for name,f,old,new,pkg,symbol in mutations:
 path=proof/f;original=path.read_text();assert old in original,name
 codes=[]
 try:
  for stage in ['before','mutant','after']:
   path.write_text(original.replace(old,new,1) if stage=='mutant' else original)
   r=subprocess.run(['go','test',pkg,'-run','^('+symbol+')$','-count=1'],cwd=proof,text=True,capture_output=True)
   out=r.stdout+r.stderr;(logs/(name+'-'+stage+'.log')).write_text(out);codes.append(r.returncode)
   if stage=='mutant':assert '--- FAIL: Test' in out and '[build failed]' not in out,out
 finally:path.write_text(original)
 assert codes==[0,1,0],(name,codes)
 rows.append(f'{name}\t0\t1\t0\tassertion\n');print(name, '0/1/0 assertion',flush=True)
(candidate/'docs/fidelity/depth/evidence/2026-10-01-entry-deleted/revert-triples.tsv').write_text(''.join(rows))
