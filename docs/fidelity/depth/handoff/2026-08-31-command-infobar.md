# Depth-fidelity handoff: `infobar`

Date: 2026-08-31

## Queue position and frontier

This session continued the source-order `src/interpreter.c` command-family
sweep after the special-procedure inventory and the one-time
`objmagic.sleep-entry-gates` cast-sleep vehicle. The pre-slice frontier was
2,283 total cases: 2,223 proven/delegated, 16 blocked, and 44 excluded
(2,223 of 2,239 actionable cases, 99.3%). The 11 new `do_infobar` manifest
rows bring the frontier to 2,294 total: 2,234 proven/delegated, 16 blocked,
and 44 excluded (2,234 of 2,250 actionable cases, 99.3%).

The special-procedure inventory remains exhausted. The explicitly blocked
`objmagic.sleep-entry-gates` row remains blocked after its already-recorded
cast-sleep attempt; its reached vehicle remains separately proven as
`objmagic.sleep-entry-gates.cast`. No blocked or excluded row was repicked.

The feature slice was PR #972 from `glm/depth-infobar-20260831`, merged with
all hosted required checks green at main merge commit `21bd7a561` (feature
commit `77e20fa3c`). No non-green PR was merged.

The next source-order unclaimed command is `insult` at
`src/interpreter.c:518`.

## C call path and branch inventory

The registration is:

```text
src/interpreter.c:517: { "infobar", POS_DEAD, do_infobar, 0, 0 }
```

The handler is `src/act.display.c:112-155`. Its actual path is:

- `any_one_arg` consumes only the first token;
- no argument reports OFF, ON, or the unknown-state two-line reset;
- `off` either calls `InfoBarOff` or reports already off;
- `on` defaults a zero screen size to 25, either calls `InfoBarOn` or
  reports already on; and
- every other first token emits the exact usage line.

`InfoBarOn` is `src/act.display.c:158-216`; it clears the screen, sets the
scroll margin, writes the separator and labels, hides level/needed-exp fields
at `LVL_IMMORT`, stores the last values, writes values in the order hit/move/
mana/exp/(needed,level for mortals)/gold, and returns the cursor to `(0,0)`.
`InfoBarOff` is `src/act.display.c:218-224` and restores the full-screen
margin before the home-and-clear sequence. The helper positions and color
thresholds are `src/act.display.c:288-710`; `src/vt100.h:33-47` is the
authoritative VT100 byte table, including DEC save/restore `ESC 7` / `ESC 8`.

The hidden redraw path is `InfoBarUpdate` at `src/act.display.c:226-285`,
called by the infobar branch of `make_prompt` at `src/comm.c:1158-1193` after
last-value comparisons. C detects changed move, mana, hit, gold, and exp
values, then renders set bits in the order mana, move, hit, exp, gold. The
EXP branch also redraws level and needed experience only for mortals. Other
call sites are the `lines` redraw at `src/act.display.c:65-110`, quit cleanup
at `src/act.other.c:139-141`, and immortal-return redraw at
`src/interpreter.c:2220-2221`; those caller surfaces remain in their own
source-order queue positions or existing shared cleanup ownership.

## RED/GREEN result

The first `infobar-depth` probe was run against main-equivalent Go code before
the fix. It exposed confirmed player-facing divergences in the initial ON
frame: Go used the wrong home-clear sequence, a 12-cell separator instead of
the oracle's five-cell result from C's overlapping `sprintf`, move columns
48/60 instead of 53/63, hit/mana/move value order instead of hit/move/mana,
the wrong mortal threshold, and `find_exp(level+1)` instead of the C current
level. The ordinary event transport also inserted a CRLF between the raw
frame and the following status line. C's `ESC 7` / `ESC 8` save/restore bytes
and the complete `InfoBarUpdate` last-value path were then verified from the
actual C call path and implemented; no source or C-oracle file was edited.

The final vehicle has one first-player immortal actor and one ordinary mortal
peer. The actor proves OFF/ON status, default screen size, exact immortal
frame, already-on/off, first-token usage, and off-frame bytes. The peer proves
the mortal-only level and needed-experience frame branch. The focused update
test proves all five changed fields, C bit values and render order, exact
save/restore bytes, and last-value suppression on an unchanged prompt.

## Durable proof

- `cmd/dp-oracle-diff/scenarios/infobar-depth.txt` is annotated for all eight
  live transcript cases and runs the immortal/mortal audience split.
- `docs/fidelity/depth/infobar.tsv` contains 11/11 proven rows: one entry
  gate, seven direct/status/frame cases, one mortal audience case, and two
  focused state cases.
- `pkg/session/infobar_depth_test.go` contains
  `TestInfobarRegistrationUsesCEntryGate`,
  `TestInfobarUnknownStateResetsToOff`, and
  `TestInfobarUpdateUsesCLayoutAndBitOrder`.
- `infobar-depth --show-oracle --seed 1` is green with the intended C blocks
  visible; seeds 2, 3, 5, and 8 are also green.
- The focused session tests and `TestFindExp` pass.

## Verification

The complete local gates passed on the feature branch:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...`
- `gofumpt -l .` clean

The required hosted checks for PR #972 were also green: `test`, `lint`, and
`security`. `build-and-push` and `deploy` were skipped by repository workflow
policy. Because checks did not attach initially, the required single
`gh workflow run "Dark Pawns CI/CD" --ref glm/depth-infobar-20260831` retry
was used before merge.

## Next session

Start from clean `main`, pull, run `make fidelity-depth`, reread
`docs/fidelity/DEPTH_TESTING.md` and this newest handoff, then take only the
unclaimed `insult` family at `src/interpreter.c:518`. Continue in interpreter
table order with one `glm/depth-insult` PR, and leave another dated handoff.
