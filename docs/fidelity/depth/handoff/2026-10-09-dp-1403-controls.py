#!/usr/bin/env python3
"""R5h: compile each removed terminal color route, fail by assertion, restore."""
import json
from pathlib import Path
import subprocess
import tempfile

def run(command, failures=()):
    result = subprocess.run(command, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    print(result.stdout, end='')
    if failures:
        assert result.returncode == 1 and '[build failed]' not in result.stdout
        for name in failures:
            assert '--- FAIL: '+name in result.stdout, name
    else:
        assert result.returncode == 0

shared = Path('pkg/session/terminal_colors.go').resolve()
text = shared.read_text()
a = text.index('\tif color {')
b = text.index('\treturn frame, ok',a)
removed = text[:a] + '\t_ = color\n' + text[b:]
# strings remains used by the shared expansion function.
command = ['go','test','./pkg/session','./pkg/telnet','-run','^(TestTerminalColor|TestWriteLoopRecipientColors)', '-count=1']
names = ['TestTerminalColorCodes','TestTerminalColorPerRecipient','TestTerminalColorEntryOnce','TestTerminalColorBrowserAndPrompts','TestWriteLoopRecipientColors']
telnet = Path('pkg/telnet/listener.go').resolve()
old_telnet = telnet.read_text().replace('s.RenderTerminalFrame(msg)', 'session.RenderTerminalFrame(msg)')
entry = Path('pkg/session/char_creation.go').resolve()
entry_text = entry.read_text()
entry_start = entry_text.index('func (s *Session) sendCharCreatePromptWithSecret')
entry_point = entry_text.index('\tdata := CharCreateData{', entry_start)
old_entry = entry_text[:entry_point] + '\tif s.charColor { prompt = expandTerminalColors(prompt) }\n' + entry_text[entry_point:]
for source, mutation, failures in [(shared,removed,names),(telnet,old_telnet,[names[-1]]),(entry,old_entry,[names[2]])]:
    run(command)
    with tempfile.TemporaryDirectory() as directory:
        root=Path(directory)
        file=root/source.name
        file.write_text(mutation)
        overlay=root/'overlay.json'
        overlay.write_text(json.dumps({'Replace':{str(source):str(file)}}))
        run(command[:2]+['-overlay='+str(overlay)]+command[2:],failures)
    run(command)
