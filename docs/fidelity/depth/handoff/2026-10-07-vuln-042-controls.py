#!/usr/bin/env python3
"""R5h: shared queue-cap green/revert/restore; run from repository root."""
import pathlib
import subprocess
import sys

root = pathlib.Path(sys.argv[1]).expanduser()
root.mkdir(parents=True, exist_ok=True)
source = pathlib.Path('pkg/session/manager.go')
original = source.read_text()
command = ['go', 'test', './pkg/session', '-run', '^TestInputQueueCap', '-count=1']

def run(name):
    result = subprocess.run(command, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    (root / (name + '.txt')).write_text(result.stdout)
    return result

assert run('green').returncode == 0
try:
    needle = 'if count > maxQueuedInput-len(s.inputQueue) {'
    assert original.count(needle) == 1
    source.write_text(original.replace(needle, 'if false {'))
    result = run('revert')
    assert result.returncode != 0 and '[build failed]' not in result.stdout
    for test in ['TestInputQueueCapWaitFlood', 'TestInputQueueCapAliases']:
        assert '--- FAIL: ' + test in result.stdout, result.stdout
finally:
    source.write_text(original)
assert run('restore').returncode == 0
print('PASS shared admission cap: green/revert/restore 0/1/0; both named assertions fail')
