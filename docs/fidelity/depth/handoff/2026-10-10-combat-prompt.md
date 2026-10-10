# DP-1386 combat prompt

Stop tier: terminal/session prompt bytes. C src/comm.c:970-1025,1120-1156
renders target and tank inside INFOBAR_OFF, after vitals. Target uses
one_argument(PERS(...)), lowercases/skips fill words then uppercases its first
byte; tank uses the full PERS name. A thin exported PersName bridge reuses
Act's visibility rule (AFF_HIDE is not invisibility under C PERS). Shared
integer percentage thresholds exactly follow get_status, including maximum
<=0 and negative integer truncation. C_CMP red/reset requires color level 3.

AFK and INACTIVE overwrite prior fields, with INACTIVE winning. This priority applies
inside INFOBAR_OFF; editor prompt remains earlier. The separate INFOBAR_ON
tail (src/comm.c:1194-1201) emits AFK > with C_CMP red/reset when AFK is
set, otherwise > . It discards the invisibility prefix and ignores INACTIVE. The implementation
preserves raw ANSI so recipient-aware ampersand expansion does not alter it.

R5h: combat_prompt_depth_test.go covers target/tank bytes, all condition
thresholds, absent fighting relationships, color levels 0–3, status priority
and infobar behavior. TestPromptInfobarOnTail covers ON × none/AFK/INACTIVE/both
× invisibility 0/1 × color off/complete. Its compiling AFK-arm removal
control fails and restoration passes. The compiling overlay controls separately remove
target, tank and coloring and require the named assertion failure/restoration.

Readers: SendPrompt, terminal async prompt/rendering and WebSocket prompt
messages share promptText. GMCP vitals remain separately reported. World
PERS readers still use the same canSeeForPers implementation. No RNG.
Locks: body/flag/vitals getters snapshot under their existing locks; no
manager, world or player lock spans prompt delivery. Fighting references
are read by identity through GetFightingBody, never display-name lookup.

Full fmt/build/vet/tests/game tests/lint/fidelity-depth/fidelity-units/string-census and focused race gates pass (1467/1467 claims). Compiling controls pass; combined census remains pending before PR. Claude
reviews the actual train diff against src and spot-checks its R5h controls.

Gate/control logs retained at ~/Archives/darkpawns/oracle-runs/2026-10-10/inactive-prompt-gates/.

Review fix targeted census: prompt-infobar-status-depth exercises the joint
ON status/invisibility/color matrix with prompts retained; prompt-playing-depth
guards the ordinary prompt and infobar-depth guards infobar transitions.
These cover the changed ON tail and its adjacent OFF behavior.
