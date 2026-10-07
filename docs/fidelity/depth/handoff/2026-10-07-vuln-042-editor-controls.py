#!/usr/bin/env python3
"""R5h routing control using a Go overlay; leaves census source untouched."""
import json
import os
import pathlib
import subprocess
import sys
import tempfile

root = pathlib.Path(sys.argv[1]).expanduser()
root.mkdir(parents=True, exist_ok=True)
source = pathlib.Path('pkg/session/session_login.go').resolve()
original = source.read_text()
needle = "\t// C's process_input exposes the raw line to an active snooper before any"
assert original.count(needle) == 1
mutation = '\tif s.player != nil && s.tryExecuteNow(cmd.Command, cmd.Args, cmd.RawArgs) { return nil }\n\n'
command = ['go', 'test', '-p', '2', './pkg/session', '-run', '^TestInputQueueCapLaggedEditorPaste$', '-count=1']
env = dict(os.environ, GOMAXPROCS='2')

def run(name, args):
    result = subprocess.run(args, env=env, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    (root / (name + '.txt')).write_text(result.stdout)
    return result

assert run('green', command).returncode == 0
with tempfile.TemporaryDirectory() as directory:
    replacement = pathlib.Path(directory) / 'session_login.go'
    replacement.write_text(original.replace(needle, mutation + needle))
    overlay = pathlib.Path(directory) / 'overlay.json'
    overlay.write_text(json.dumps({'Replace': {str(source): str(replacement)}}))
    result = run('revert', command[:2] + ['-overlay=' + str(overlay)] + command[2:])
    assert result.returncode != 0 and '[build failed]' not in result.stdout, result.stdout
    for kind in ['editor', 'board']:
        assert '--- FAIL: TestInputQueueCapLaggedEditorPaste/' + kind in result.stdout, result.stdout
assert source.read_text() == original
assert run('restore', command).returncode == 0
print('PASS editor/board routing: green/revert/restore 0/1/0; both paste assertions fail')
