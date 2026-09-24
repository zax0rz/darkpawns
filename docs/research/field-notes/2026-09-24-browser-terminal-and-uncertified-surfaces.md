# 2026-09-24: the browser terminal, and what the census cannot see

Field notes from making `/play` show telnet's bytes (DP-1320) and from the
things that work turned up. Claims here are backed by the ledger rows named in
each section.

## 1. One renderer: 104 web-only failures to one

On 2026-09-23 the census, run through the real browser client, failed 104 of
976 scenarios while telnet was green (RO-008). The cause was two renderers:
the telnet listener turned session frames into bytes and routed input lines,
and the browser client did neither. The fix moved the telnet listener's
renderer, name dialogue and line router into `pkg/session` and had both
transports drive it. Telnet keeps only IAC framing, and the browser receives
rendered bytes. The browser-client census went from 104 failures to one,
which section 3 explains. The telnet census stayed green.

The shape of the fix matters for the method. The first survey listed the
failures one by one: glued lines, no pager, no prompts, collapsed spacing, a
different greeting. Patching the browser client for each would have produced a
second renderer that agreed for now. Sharing the renderer makes the web path
correct by construction, and the browser census then checks that it stays so
(RO-010).

## 2. Prompts are uncertified by construction

A real-browser pass turned up two prompt bugs within minutes, both on telnet
too: the pager prompt gets a trailing line ending and a stray `> ` after it,
and no prompt appears after entering the game (DP-1326). Neither could fail
the census. `Normalize` drops prompt-only lines, and even the raw ANSI mode
leaves prompts out of scope. So every prompt bug in the port is green by
construction, and nothing in the census output says so.

The general point: a normalizer is part of the claim. Every line it drops is a
surface the oracle does not certify. That's the same lesson as the terminator
blind spot recorded earlier, a different surface this time (RO-011).

## 3. Scenarios whose peers never logged in

The one browser-census failure was `cutthroat-failure`. Its thief's password
was 13 characters. C refuses passwords over 10 (`MAX_PWD_LENGTH`), and so does
the port, so on both servers the thief sat at a password prompt for the whole
run, and every block involving it compared two identical refusals. Five more
scenarios had the same flaw in an observer or victim. The browser client's
typeahead handling changed how the two refusals lined up, and that is the
only reason it surfaced.

With real passwords, five of the six pass with characters that actually play.
The sixth, `groinrip-sleeping-depth`, fails for real: C refuses to hurt a
level-1 player through `damage()`'s protections, and the port's groinrip
skips them (DP-1327). Its manifest row claimed a damage path that C never
reaches for that victim. The harness now fails any setup that was refused at
the password prompt (RO-012).

This is the coincidence-green pattern in its purest form: agreement between
two servers that are both doing nothing. A differential oracle needs a
positive check that the scenario did what it meant to do, not only that both
sides agree.
