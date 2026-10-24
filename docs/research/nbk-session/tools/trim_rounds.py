"""Drop whole prompt-led rounds from a rendered log, keeping colours.
trim_rounds.py DIR name prompt-prefix ... : reads DIR/name.html (from
ansi2html.py) and writes DIR/name.trim.html and DIR/name.trim.txt, dropping
every round whose prompt starts with one of the prefixes (e.g. "165H ")."""
import re, sys, html
S = sys.argv[1].rstrip('/') + '/'
name, drops = sys.argv[2], sys.argv[3:]
h = open(S + name + '.html').read()
body = h[len('<pre class="mud-log">'):h.rindex('</pre>')]
# Split into lines, closing and reopening any span that crosses a newline.
lines, cur, open_cls = [], '', None
for tok in re.split(r'(<span class="[^"]*">|</span>)', body):
    m = re.fullmatch(r'<span class="([^"]*)">', tok)
    if m: open_cls = m.group(1); cur += tok; continue
    if tok == '</span>': open_cls = None; cur += tok; continue
    parts = tok.split('\n')
    for i, p in enumerate(parts):
        cur += p
        if i < len(parts) - 1:
            if open_cls: cur += '</span>'
            lines.append(cur); cur = f'<span class="{open_cls}">' if open_cls else ''
lines.append(cur)
plain = lambda l: html.unescape(re.sub(r'<[^>]+>', '', l))
keep, dropping = [], False
for l in lines:
    t = plain(l)
    if re.match(r'^-?\d+H \d+M \d+V >', t):
        dropping = any(t.startswith(d) for d in drops)
    if not dropping: keep.append(l)
out = '\n'.join(keep)
out = re.sub(r'<span class="[^"]*"></span>', '', out)
# A truly empty line would end the <figure> HTML block in Markdown; keep it non-blank.
out = '\n'.join(l if l.strip() else '<span></span>' for l in out.split('\n'))
open(S + name + '.trim.html', 'w').write('<pre class="mud-log">' + out + '</pre>\n')
open(S + name + '.trim.txt', 'w').write('\n'.join(plain(l) for l in keep).rstrip() + '\n')
print(name, len(lines), '->', len(keep))
