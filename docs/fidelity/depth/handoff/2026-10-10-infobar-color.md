# Infobar color prerequisite for DP-1386

Stop tier: player-facing ANSI bytes. Approved by Zach after the ON-tail
scenario exposed unconditional Go colors at color off. C src/act.display.c:
311-708 gates HP, mana, movement, experience, gold and every corresponding
reset at C_NRM. src/screen.h:45-56 makes this level >=2, equivalent to
PRF_COLOR_2. Cursor movement, clear/save/restore and uncolored labels retain
their unconditional escapes. No other display formatting changes.

One shared infobarState.color gate applies to all five value renderers,
used by both cmdInfoBarOn and cmdInfoBarUpdate. State is constructed afresh
from the recipient's preference flags on each render; no stored color setting
can go stale after do_color. Reader audit found only these two production
consumers of the value renderers. Telnet and WebSocket raw events consume
these rendered bytes; the terminal funnel preserves raw ANSI. Flag/vitals
getters release their own locks before render/send; no new locks or RNG.

TestInfobarColorLevels exercises real ON and repaint dispatch at levels 0-3,
checks all five fields and green/yellow/red vital values. Reset and color
bytes are absent at off/sparse, present at normal/complete. Existing repaint
order test now explicitly requests normal color instead of relying on the bug.
2026-10-10-infobar-color-controls.py compiles removed and inverted gates;
the named assertion fails and restored code passes (R5h).

Original targeted evidence: dp-1386-infobar-on-review NOT_CLEAN at 2dc25b70b.
Diagnostic evidence: dp-1386-infobar-on-diagnosis NOT_CLEAN at the same tip;
retained attempts show the sole difference at infobar on with color off.
Every ON-tail matrix prompt matched. A new combined census is required
for this production correction. Evidence is under
/home/zach/Archives/darkpawns/oracle-runs/2026-10-10/.
