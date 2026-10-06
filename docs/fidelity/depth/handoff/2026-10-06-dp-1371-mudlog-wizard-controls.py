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
