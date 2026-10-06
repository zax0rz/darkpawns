"""Run from repository root with retained output directory; optional control name."""
import pathlib
import subprocess
import sys

root = pathlib.Path(sys.argv[1])
root.mkdir(parents=True, exist_ok=True)
controls = []
for name, prefix, replacement in [
    ('force-single', 'game.Act(s.manager.world, true, s.player, target.player, nil, nil, fmt.Sprintf("$n has forced you to', 'target.Send(fmt.Sprintf("%s has forced you to'),
    ('force-loop', 'game.Act(caster.manager.world, true, caster.player, target.player, nil, nil, fmt.Sprintf("$n has forced you to', 'target.Send(fmt.Sprintf("%s has forced you to'),
]:
    p = pathlib.Path('pkg/session/wiz_communication.go')
    original = p.read_text()
    line = next(line for line in original.splitlines() if prefix in line)
    actor = 's' if name == 'force-single' else 'caster'
    command = 'forceCmd' if name == 'force-single' else 'command'
    old = '\t\ttarget.Send(fmt.Sprintf("%s has forced you to \'%s\'.\\r\\n", '+actor+'.player.Name, '+command+'))'
    controls.append((name, str(p), line, old, 'TestForceNotificationVisibilityAndSleep'))
controls.append(('force-shutdown', 'pkg/session/wiz_system.go',
    'game.Act(m.world, true, caster.player, target.player, nil, nil, "$n has forced you to \'all save\'.", "", game.ToVict)',
    'target.Send(fmt.Sprintf("%s has forced you to \'all save\'.\\r\\n", caster.player.Name))', 'TestForceShutdownNotificationAct'))
for name, payload in [('on','WHOD turned on'),('off','WHOD turned off'),('add','name added'),('remove','name removed')]:
    controls.append(('whod-'+name, 'pkg/game/whod.go', 'MudLog(payload, MudlogBrief, LVL_GOD, true)',
        'if !strings.HasPrefix(payload, "'+payload+'") { MudLog(payload, MudlogBrief, LVL_GOD, true) }', 'TestWhodMudlogProducers/'+name))
controls += [
 ('whod-type','pkg/game/whod.go','MudlogBrief, LVL_GOD','MudlogNormal, LVL_GOD','TestWhodMudlogProducers'),
 ('whod-threshold','pkg/game/whod.go','MudlogBrief, LVL_GOD','MudlogBrief, LVL_GOD-1','TestWhodMudlogProducers'),
 ('whod-ack-order','pkg/game/whod.go','output(message)\n\t\tMudLog(payload, MudlogBrief, LVL_GOD, true)',
  'MudLog(payload, MudlogBrief, LVL_GOD, true)\n\t\toutput(message)', 'TestWhodMudlogProducers'),
]
controls.append(('whod-remainder','pkg/game/whod.go',
    'tokens := strings.Fields(argument)\n\targument = ""\n\tif len(tokens) > 0 {\n\t\targument = strings.ToLower(tokens[0])\n\t}',
    'argument = strings.TrimSpace(strings.ToLower(argument))', 'TestWhodMudlogIgnoresRemainder'))
for name, filename, before, after, test in controls:
    if len(sys.argv)>2 and sys.argv[2]!=name:
        continue
    path=pathlib.Path(filename)
    original=path.read_text()
    if before not in original:
        sys.exit('mutation missing: '+name)
    pattern='^'+test.replace('/', '$/^')+'$'
    def run(label):
        result=subprocess.run(['go','test','./pkg/session','-run',pattern,'-count=1'],capture_output=True,text=True)
        (root/(name+'-'+label+'.txt')).write_text(result.stdout+result.stderr)
        return result
    if run('green').returncode:
        sys.exit('initial failure: '+name)
    try:
        path.write_text(original.replace(before,after,1))
        result=run('revert')
        if not result.returncode or '--- FAIL: '+test.split('/')[0] not in result.stdout or '[build failed]' in result.stdout+result.stderr:
            sys.exit('invalid red: '+name)
    finally:
        path.write_text(original)
    if run('restore').returncode:
        sys.exit('restore failure: '+name)
    print(name+': 1 -> 0 -> 1',flush=True)
