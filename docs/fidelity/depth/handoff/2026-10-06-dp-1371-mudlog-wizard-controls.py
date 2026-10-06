"""Run from the repository root; argument is a retained evidence directory.
Each control must compile and fail the specific producer subtest (R5h).
"""
import pathlib
import subprocess
import sys

root = pathlib.Path(sys.argv[1])
root.mkdir(parents=True, exist_ok=True)
controls = [
    ('load-mob', 'wiz_object.go', 'game.MudLog(fmt.Sprintf("(GC) %s loaded %s at %s."'),
    ('load-object', 'wiz_object.go', 'game.MudLog(fmt.Sprintf("(GC) %s loaded %s at %s"'),
    ('purge-player', 'wiz_object.go', 'game.MudLog(fmt.Sprintf("(GC) %s has purged'),
    ('force-single', 'wiz_communication.go', 'game.MudLog(fmt.Sprintf("(GC) %s forced %s'),
    ('force-room', 'wiz_communication.go', 'game.MudLog(fmt.Sprintf("(GC) %s forced room'),
    ('force-all', 'wiz_communication.go', 'game.MudLog(fmt.Sprintf("(GC) %s forced all'),
    ('reset-zone', 'wiz_zone.go', 'game.MudLog(fmt.Sprintf("(GC) %s reset zone'),
    ('reset-world', 'wiz_zone.go', 'game.MudLog(fmt.Sprintf("(GC) %s reset entire'),
    ('newbie', 'wiz_system.go', 'game.MudLog(fmt.Sprintf("(GC) %s newbied'),
]
for name, filename, prefix in controls:
    if len(sys.argv) > 2 and name != sys.argv[2]:
        continue
    p = pathlib.Path('pkg/session') / filename
    original = p.read_text()
    start = original.index(prefix)
    # All producer statements end at the true file-flag argument.
    end = original.index('true)', start) + len('true)')
    cmd = ['go', 'test', './pkg/session', '-run', '^TestWizardMudlogProducers$/^'+name+'$', '-count=1']
    marker = '--- FAIL: TestWizardMudlogProducers/' + name
    def run(label):
        r = subprocess.run(cmd, capture_output=True, text=True)
        (root / (name+'-'+label+'.txt')).write_text(r.stdout+r.stderr)
        return r
    r = run('green')
    if r.returncode:
        sys.exit('initial green failed: '+name)
    try:
        p.write_text(original[:start]+'// producer removed for R5h'+original[end:])
        r = run('revert')
        if not r.returncode or marker not in r.stdout or '[build failed]' in r.stdout+r.stderr:
            sys.exit('invalid red: '+name)
    finally:
        p.write_text(original)
    if run('restore').returncode:
        sys.exit('restore failed: '+name)
    print(name+': 1 -> 0 -> 1', flush=True)

# Boundaries discovered by the producer reader audit, beyond call presence.
extras = [
 ('force-single-wrong-type', 'pkg/session/wiz_communication.go',
  'target.player.GetName(), forceCmd), game.MudlogNormal,',
  'target.player.GetName(), forceCmd), game.MudlogBrief,',
  '^TestWizardMudlogProducers$/^force-single$'),
 ('load-mob-wrong-threshold', 'pkg/session/wiz_object.go',
  'mob.GetName(), s.manager.world.GetRoomInWorld(roomVNum).Name),\n\t\t\tgame.MudlogBrief, s.player.GetLevel()+1, true)',
  'mob.GetName(), s.manager.world.GetRoomInWorld(roomVNum).Name),\n\t\t\tgame.MudlogBrief, s.player.GetLevel(), true)',
  '^TestWizardMudlogProducers$/^load-mob$'),
 ('force-raw-routing', 'pkg/session/commands.go',
  '\tif cmd == "force" && rawArgs != "" {\n\t\treturn cmdForceText(s, rawArgs)\n\t}\n', '',
  '^TestWizardMudlogForceRawRemainder$'),
 ('force-nested-routing', 'pkg/session/wiz_communication.go',
  '\t_, rawArgs := wiznetHalfChop(command)\n\tif err := executeCommandRaw(target, cmd, args, false, rawArgs); err != nil {',
  '\tif err := executeCommand(target, cmd, args, false); err != nil {',
  '^TestWizardMudlogNestedForceRemainder$'),
 ('purge-room-act', 'pkg/session/wiz_object.go',
  '\t\t\t\tgame.Act(s.manager.world, false, s.player, victim, nil, nil, "$n disintegrates $N.", "", game.ToNotVict)\n', '',
  '^TestWizardMudlogThresholdAndOrder$/^purge_act_then_log_before_teardown$'),
]
for name, filename, before, after, test in extras:
    if len(sys.argv) > 2 and name != sys.argv[2]:
        continue
    p = pathlib.Path(filename)
    original = p.read_text()
    if before not in original:
        sys.exit('missing mutation: '+name)
    cmd = ['go', 'test', './pkg/session', '-run', test, '-count=1']
    def run_extra(label):
        r = subprocess.run(cmd, capture_output=True, text=True)
        (root / (name+'-'+label+'.txt')).write_text(r.stdout+r.stderr)
        return r
    if run_extra('green').returncode:
        sys.exit('initial green failed: '+name)
    try:
        p.write_text(original.replace(before, after, 1))
        r = run_extra('revert')
        if not r.returncode or '--- FAIL: '+test.split('$/')[0].strip('^$') not in r.stdout or '[build failed]' in r.stdout+r.stderr:
            sys.exit('invalid red: '+name)
    finally:
        p.write_text(original)
    if run_extra('restore').returncode:
        sys.exit('restore failed: '+name)
    print(name+': 1 -> 0 -> 1', flush=True)
