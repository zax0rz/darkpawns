"""R5h replay: python3 revert_proofs.py CASE; run from repository root."""
import json, subprocess, sys
from pathlib import Path
MUTATIONS = {
 "rename-guest-id": ("pkg/session/wiz_set.go", "release, allowed := s.manager.reserveOfflineRename(target.record.Name, value)", "for _, p := range s.manager.world.GetAllPlayers() { if p.GetID() == target.player.GetID() { s.Send(\"Sorry, you can't do that.\\r\\n\"); return nil } }; release, allowed := s.manager.reserveOfflineRename(target.record.Name, value)", "^TestEntryOfflineRenameGuard/offline-with-guests$"),
 "orphan-selector": ("pkg/session/reconnect.go", "m.playerLifecycleMu.Lock()\n\tdefer m.playerLifecycleMu.Unlock()", "// Deliberately omit lifecycle serialization.", "^TestEntryDuplicateTeardownSerialization$"),
 "orphan-admission": ("pkg/session/manager.go", "func (m *Manager) enterWorld(name string, s *Session) error {\n\tm.playerLifecycleMu.Lock()\n\tdefer m.playerLifecycleMu.Unlock()", "func (m *Manager) enterWorld(name string, s *Session) error {", "^TestEntryDuplicateTeardownSerialization/admission$"),
 "orphan-retirement": ("pkg/session/manager.go", "func (m *Manager) UnregisterSession(s *Session) {\n\tm.playerLifecycleMu.Lock()\n\tdefer m.playerLifecycleMu.Unlock()", "func (m *Manager) UnregisterSession(s *Session) {", "^TestEntryDuplicateTeardownSerialization/transport$"),

 "rename-guard": ("pkg/session/wiz_set.go", 'if target.file && field.name == "name" {', 'if false && target.file && field.name == "name" {', "^TestEntry(OfflineRenameGuard|DuplicateRenameTopologyExclusions)$"),
 "candidate-skip": ("pkg/session/reconnect.go", "m.world.DiscardLoadedPlayerObjects(s.player)", "_ = s.player", "^TestEntryDuplicateCandidateObjects$"),
 "candidate-owner": ("pkg/session/reconnect.go", "m.world.DiscardLoadedPlayerObjects(s.player)", "m.world.ExtractRentedObjects(s.player)", "^TestEntryDuplicateCandidateObjects$"),
 "password-prefix": ("pkg/session/char_creation.go", "if choice != s.charPassword {", "if choice[:min(8,len(choice))] != s.charPassword[:min(8,len(s.charPassword))] {", "^TestEntryFullPasswordComparison$"),
 "editor-buffer": ("pkg/session/menu.go", "s.textEdit.buffer, s.textEdit.original = current, current", "s.textEdit.buffer, s.textEdit.original = \"\", current", "^TestEntryDescriptionEditor$"),
 "editor-abort": ("pkg/session/menu.go", "apply(original)", "apply(\"\")", "^TestEntryDescriptionEditor$"),
 "editor-bound": ("pkg/session/menu.go", "maxDescriptionLength  = 240", "maxDescriptionLength  = 4096", "^TestEntryDescriptionEditor$"),
 "editor-prompt": ("pkg/session/menu.go", 's.sendPromptText("] ")', 's.sendPromptText("")', "^TestEntryDescriptionEditor$"),
 "editor-dollar": ("pkg/session/menu.go", 'strings.ReplaceAll(input.Choice, "$", "$$")', 'input.Choice', "^TestEntryDescriptionEditor$"),
 "editor-empty-pointer": ("pkg/session/menu.go", 'current != "" || (s.player != nil && !s.creationSaved)', 'current != ""', "^TestEntryDescriptionEmptyPointer$"),
}
name=sys.argv[1]
filename,before,after,test=MUTATIONS[name]
p=Path(filename);source=p.read_text();assert before in source,(name,before)
root=Path('/home/zach/Archives/darkpawns/oracle-runs/2026-10-02/dp-1371-entry-finish-proofs')/name
root.mkdir(parents=True,exist_ok=True);exits=[]
try:
 for phase in ['fixed','reverted','restored']:
  p.write_text(source.replace(before,after,1) if phase=='reverted' else source)
  with (root/(phase+'.log')).open('w') as out:
   result=subprocess.run(['go','test','./pkg/session','-run',test,'-count=1'],stdout=out,stderr=subprocess.STDOUT)
  exits.append(result.returncode)
finally:p.write_text(source)
(root/'triple.json').write_text(json.dumps({'source':filename,'before':before,'after':after,'test':test,'exits':exits},indent=2)+'\n')
print(name,exits,flush=True);assert exits==[0,1,0]
assert '--- FAIL:' in (root/'reverted.log').read_text()
