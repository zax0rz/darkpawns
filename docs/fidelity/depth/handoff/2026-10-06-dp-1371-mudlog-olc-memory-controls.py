"""R5h: run from train tip, with a retained output directory."""
import pathlib
import subprocess
import sys

root = pathlib.Path(sys.argv[1]); root.mkdir(parents=True, exist_ok=True)
controls = []
for command in ['redit', 'zedit', 'oedit', 'medit', 'sedit']:
    path = 'pkg/session/' + command + '.go'
    text = pathlib.Path(path).read_text()
    line = next(line for line in text.splitlines() if 'game.MudLog(fmt.Sprintf("OLC: %s edits' in line)
    test = 'TestOLCMemoryMudlog' + command.title()
    controls.append((command, path, line, '// removed producer', test))
    controls.append((command + '-type', path, line, line.replace('MudlogComplete', 'MudlogNormal'), test))
    controls.append((command + '-threshold', path, line, line.replace('LVL_IMMORT,', 'LVL_IMMORT-1,'), test))
    controls.append((command + '-writing-order', path, line, 's.player.SetPlrFlag(game.PlrWriting, false)\n' + line, test))
for command in ['redit', 'zedit', 'oedit', 'medit', 'sedit']:
    path = 'pkg/session/' + command + '.go'
    text = pathlib.Path(path).read_text()
    line = next(line for line in text.splitlines() if 'game.MudLog(fmt.Sprintf("OLC: %s edits' in line)
    test = 'TestOLCMemoryMudlog' + command.title()
    if command in ['oedit', 'medit']:
        start = text.index('\t\t\ts.save' + command.title() + 'InternallyLocked()')
    else:
        start = text.index('\t\tsaveMu :=', text.index('func (s *Session) finish' + command.title() + 'Locked'))
    end = text.index(line, start) + len(line)
    block = text[start:end]
    controls.append((command + '-commit-order', path, block, line + '\n' + block.replace(line, ''), test))
    if command == 'redit':
        changed = 's.reditSend("Room saved to memory.\\r\\n")\n' + line
        controls.append((command + '-ack-order', path, line, changed, test))
    elif command == 'medit':
        ack = '\t\t\ts.meditSendLocked("Saving mobile to memory.\\r\\n")'
        start = text.index(ack); end = text.index(line, start) + len(line)
        block = text[start:end]
        controls.append((command + '-ack-order', path, block, block.replace(ack, '') + '\n' + ack, test))
    else:
        flush = 's.flush' + command.title() + 'OutputLocked()'
        start = text.rindex(flush, 0, text.index(line))
        controls.append((command + '-ack-order', path, text[start:text.index(line)+len(line)], line + '\n' + flush, test))
for name, filename, before, after, test in controls:
    if len(sys.argv) > 2 and sys.argv[2] != name:
        continue
    path = pathlib.Path(filename); original = path.read_text()
    assert before in original, name
    def run(label):
        r = subprocess.run(['go', 'test', './pkg/session', '-run', '^' + test + '$', '-count=1'], capture_output=True, text=True)
        (root / (name + '-' + label + '.txt')).write_text(r.stdout + r.stderr)
        return r
    if run('green').returncode:
        sys.exit('initial failure: ' + name)
    try:
        path.write_text(original.replace(before, after, 1))
        r = run('revert')
        if not r.returncode or '--- FAIL: ' + test not in r.stdout or '[build failed]' in r.stdout + r.stderr:
            sys.exit('invalid red: ' + name)
    finally:
        path.write_text(original)
    if run('restore').returncode:
        sys.exit('restore failed: ' + name)
    print(name + ': 1 -> 0 -> 1', flush=True)
