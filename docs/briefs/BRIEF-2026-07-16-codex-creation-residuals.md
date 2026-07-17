# BRIEF (codex) — PR #382 follow-up: two whitespace residuals to reach GREEN (DP-1173)

**Owner:** codex (follow-up commit on the same branch `codex/dp1173-character-creation-1to1`). **Gate:** Claude runs `--scenario character-creation` red→green.

PR #382 is **~95% green** — the motd, bracket menus, unified nanny flow, dedup, race/class/hometown/roll/menu/room-entry all byte-match C. Two whitespace residuals remain in a single diff hunk. Both are player-facing under the 1:1 north star, so they block green. **Do not touch `scenarios/character-creation.txt` — Claude owns it** (and has already aligned the port inputs to C's now-unified flow for the gate).

The remaining normalized diff:
```
@@ (greeting → name area)
-            (C has 2 blank lines here that the port lacks)   <-- Residual 1
 By what name do you wish to be known? Please remember to choose an appropriate fantasy-oriented name.
 Did I get that right, Ccreator (Y/N)? New character.
 Give me a password for Ccreator:
-Please retype password:                                       <-- C: color prompt on its OWN line
-Do you want ANSI color (Y/N)? What is your sex (M/F)?
+Please retype password: Do you want ANSI color (Y/N)? What is your sex (M/F)?   <-- port: glued
```

## Residual 1 — greeting blank lines
The port's greeting emits **2 fewer blank lines** than C just before the `By what name…` prompt. Match C's greeting whitespace exactly. C source: `~/.openclaw/workspace/darkpawns-c-oracle/src/config.c` `GREETINGS` — its tail is:
```
...credits three lines...\r\n
\r\n
   As of 10-17-2008 there has been a pwipe.  Enjoy your new adventures!\r\n
\r\n\r\n
```
(i.e. a blank line, the pwipe line, then **two** trailing blank lines before the name prompt). The oracle normalizer drops the volatile `As of …` line, so for the **gate** only the blank-line count matters — but match C's byte structure for real fidelity. Port side: the greeting constant in `pkg/telnet/listener.go` (~:291, `greetingsLogo`). Add the missing trailing blank lines so the port's greeting whitespace equals C's.

## Residual 2 — color prompt must start on its own line (clean, not bug-for-bug)
C shows:
```
Please retype password:
Do you want ANSI color (Y/N)?
```
The port glues them onto one line. **Root cause in C:** `echo_on()` (`src/comm.c:964`) sends a *malformed* telnet string `{IAC, WONT, TELOPT_ECHO, TELOPT_NAOFFD(12), TELOPT_NAOCRD(10), 0}` — after the valid 3-byte `IAC WONT ECHO`, the trailing bytes `12`(`\f`) and **`10`(`\n`)** leak to the client, and that stray `\n` is the line break.

**Decision (Zach): do the telnet correctly, but reproduce the player-visible line break.**
- **Do NOT** replicate C's malformed echo bytes — the port's clean `IAC WONT ECHO` is correct and stays. The telnet negotiation is invisible transport; keep it clean.
- **DO** emit a normal `\r\n` immediately before `Do you want ANSI color (Y/N)? ` so it starts on a fresh line, matching C's observable layout. (Verified: the normalizer collapses both C's `\f\n` and a plain `\r\n` to one newline, so this lands byte-identical.)

The color prompt is sent from the unified password-confirm path (`pkg/session/char_creation.go` / the reconciled flow). Prepend the `\r\n` there. This is the north-star seam in miniature: **clean transport, 1:1 player-facing output.**

## Out of scope
- Everything already green — don't perturb it. The scenario file (Claude's). The malformed C telnet bytes (we do NOT carry them forward).

## Tests
- Extend the golden telnet transcript test so the color prompt appears on its own line after `Please retype password:`, and the greeting blank-line count matches C.

## Gate
Claude re-runs `--scenario character-creation` on the branch; target is **no normalized divergence**. On green, Claude commits the scenario with `[creation:port]` aligned to C and marks DP-1173 done.

## PR hygiene
- Commits end with: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`
- (Same PR/body already open — no new PR needed.)
