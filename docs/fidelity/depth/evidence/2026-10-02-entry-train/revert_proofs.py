#!/usr/bin/env python3
"""R5h replay. Mutations must fail an assertion, then restore exact source bytes."""
import argparse,json,subprocess
from pathlib import Path
p=argparse.ArgumentParser();p.add_argument('case');p.add_argument('output',type=Path);a=p.parse_args();a.output.mkdir(parents=True,exist_ok=True)
mutations={
 'bootstrap-remort':('pkg/game/world_player.go','case ClassThief:', 'case ClassThief, ClassAssassin:', './pkg/game', '^TestEntryBootstrapRemortKits$'),
 'bootstrap-practices':('pkg/game/character.go','p.SetPractices(0) // src/db.c','p.SetPractices(2) // src/db.c','./pkg/session','^TestEntryBootstrapMatrix$'),
 'bootstrap-pack':('pkg/game/world_player.go','w.giveStartingPackItem(p, pack, picks)','_ = picks // omit starting picks','./pkg/session','^TestEntryBootstrapMatrix$'),
 'bootstrap-kit':('pkg/session/char_creation.go','s.manager.world.GiveStartingItems(s.player)','// omit starting kit','./pkg/session','^TestEntryBootstrapMatrix$'),

 'stats-choice':('pkg/session/char_creation.go', 'choice = firstCreationByte(strings.TrimLeft(input.Choice, " \\t\\n\\r\\v\\f"))', 'choice = strings.TrimSpace(input.Choice)', './pkg/session', '^TestEntryStatsRerollMatrix$'),
 'stats-draw':('pkg/session/char_creation.go', 's.charStats = game.RollRealAbils(s.charClass, s.charRace)', '_ = game.RollRealAbils(s.charClass, s.charRace); s.charStats = game.RollRealAbils(s.charClass, s.charRace)', './pkg/session', '^TestEntryStatsRerollMatrix$'),
 'stats-accept':('pkg/session/char_creation.go', 's.charSex, s.charStats)', 's.charSex, game.CharStats{})', './pkg/session', '^TestEntryStatsRerollMatrix$'),

 'race-choice':('pkg/session/char_creation.go','switch firstCreationByte(upperChoice) {','switch upperChoice {','./pkg/session','^TestEntryCreationChoiceMatrix$'),
 'creation-choices':('pkg/session/char_creation.go','if s.charStage != "race" && len(choice) > 0 {','if false && s.charStage != "race" && len(choice) > 0 {','./pkg/session','^TestEntryCreationChoiceMatrix$'),
 'race-help':('pkg/session/char_creation.go','help = "That is not a race..\\r\\n"','help = "\\r\\nThat is not a race..\\r\\n"','./pkg/session','^TestEntryRaceHelpMatrix$'),
 'password-security':('pkg/session/session_login.go','if s.manager.accountLockouts != nil {','if false && s.manager.accountLockouts != nil {','./pkg/session','^TestHandleLogin_(LockedAccountReturnsErrorAndCloses|NewlyLockedClosesImmediately)$'),
 'password-durable':('pkg/session/session_login.go','if s.manager.accountLockouts == nil {','if false && s.manager.accountLockouts == nil {','./pkg/session','^TestEntryPasswordAccounting$'),
 'password-leading':('pkg/session/session_login.go','login.Password = strings.TrimLeft(login.Password, " \\t\\n\\r\\v\\f")','login.Password = login.Password','./pkg/session','^TestEntryPasswordAccounting$'),
 'password-retries':('pkg/session/menu.go','failedPasswords[0] > 0','failedPasswords[0] < 0','./pkg/session','^TestEntryPasswordAccounting$'),
 'password-empty':('pkg/session/session_login.go','s.sendRawEvent("\\r\\n") // src/interpreter.c:1871: echo_on precedes empty close.','s.sendRawEvent("wrong") // reverted echo','./pkg/session','^TestEntryPasswordEmptyEchoClose$'),
 'new-password':('pkg/session/char_creation.go','choice = strings.TrimLeft(choice, " \\t\\n\\r\\v\\f")','choice = choice','./pkg/session','^TestEntryNewPasswordMatrix$'),
 'password-byte-view':('pkg/session/session_login.go','rec.FailedLoginAttempts & 0xff','rec.FailedLoginAttempts','./pkg/session','^TestEntryPasswordCounterByteView$'),
}
file,old,new,pkg,test=mutations[a.case];path=Path(file);source=path.read_bytes();text=source.decode();assert old in text
codes=[]
try:
 for phase in ['fixed','reverted','restored']:
  path.write_bytes(text.replace(old,new).encode() if phase=='reverted' else source)
  r=subprocess.run(['go','test',pkg,'-run',test,'-count=1'],capture_output=True,text=True)
  (a.output/(phase+'.log')).write_text(r.stdout+r.stderr);codes.append(r.returncode)
  if phase=='reverted':
   assert r.returncode!=0 and '--- FAIL:' in r.stdout and '[build failed]' not in r.stdout and 'timeout waiting' not in r.stdout, r.stdout+r.stderr
  else: assert r.returncode==0,r.stdout+r.stderr
finally:path.write_bytes(source)
(a.output/'triple.json').write_text(json.dumps({'case':a.case,'exits':codes,'file':file,'mutation':old+' -> '+new},indent=2)+'\n')
print(a.case,codes)
