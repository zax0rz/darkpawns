"""Render a mud.py event log as <pre class="mud-log"> HTML.

Server bytes go through a minimal SGR state machine (the codes Dark Pawns
sends: 0, 1, 31-37). The player's typed command is written after the prompt,
where a client with local echo shows it. Usage:
  ansi2html.py log.json START_INPUT END_MARKER > out.html
START_INPUT: index (0-based) of the input event to start from.
END_MARKER: stop after the output chunk containing this text.
"""
import html, json, re, sys

events = json.load(open(sys.argv[1]))
start = int(sys.argv[2]); end_marker = sys.argv[3]
inputs = [i for i, e in enumerate(events) if 'in' in e]
# Begin on the prompt the first command was typed at.
prev = [e for e in events[:inputs[start]] if 'out' in e][-1]['out']
cut = prev.rfind('\r\n')
events = [{'out': prev[cut + 2:] if cut >= 0 else prev}] + events[inputs[start]:]

out = []; fg = None; bold = False; open_span = False
def set_style():
    global open_span
    if open_span: out.append('</span>'); open_span = False
    cls = []
    if fg: cls.append(f'c{fg}')
    if bold: cls.append('b')
    if cls: out.append(f'<span class="{" ".join(cls)}">'); open_span = True
def text(s):
    out.append(html.escape(s, quote=False))

for e in events:
    if 'in' in e:
        if open_span: out.append('</span>'); open_span = False
        out.append(f'<span class="in">{html.escape(e["in"], quote=False)}</span>\n')
        set_style()
        continue
    chunk = e['out'].replace('\r\n', '\n').replace('\r', '')
    for part in re.split(r'(\x1b\[[0-9;]*m)', chunk):
        m = re.fullmatch(r'\x1b\[([0-9;]*)m', part)
        if m:
            for code in (m.group(1) or '0').split(';'):
                n = int(code or 0)
                if n == 0: fg, bold = None, False
                elif n == 1: bold = True
                elif 30 <= n <= 37: fg = n
            set_style()
        elif part:
            part = re.sub(r'\x1b\[[0-9;]*[A-Za-z]', '', part)
            text(part)
    if end_marker in e['out']:
        break
if open_span: out.append('</span>')
body = ''.join(out).strip('\n')
print(f'<pre class="mud-log">{body}</pre>')
