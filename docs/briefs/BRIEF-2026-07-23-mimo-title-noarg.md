# BRIEF (mimo) — `title` with no args must clear the title, C-style

**Owner:** mimo-v2.5-pro (first MiMo brief — welcome). **Gate:** Claude runs the
differential oracle red→green.
**Git:** NONE. You are working in an isolated git worktree; do NOT run any git
commands (no branch/add/commit/push). Edit files, run tests, report. The operator
commits.
**Closes:** DP-1196. Sized: S (one branch removal + tests).
**Cite:** `src/act.other.c:595-619` (do_title); rules **R1**, **R4**
(`docs/fidelity/RULEBOOK.md`).

## The C truth

`do_title` (act.other.c:595) has **no empty-argument guard**. After
skip_spaces/delete_doubledollar/delete_ansi_controls, an empty argument falls
through every else-if (not NPC, not NOTITLE, no parens, not too long) into:

```c
set_title(ch, argument);            /* argument == "" */
sprintf(buf, "Okay, you're now %s %s.\r\n", GET_NAME(ch), GET_TITLE(ch));
```

So `title` alone CLEARS the title and prints `Okay, you're now Name .` — the
trailing space before the period (from `%s %s.` with an empty title) is
player-facing surface. Oracle-captured live: `Okay, you're now Infoactor .`

## The Go bug

`pkg/session/informative_cmds.go:34` cmdTitle: the `len(args) == 0` early
return sending `Set your title to what?` is an R4 invention. Everything after
it is already faithful (the switch mirrors C's else-if chain and the success
message already uses the `%s %s.` shape).

## Fix

1. Remove the `len(args) == 0` early return. Empty input must flow through the
   normal path: preprocessing on an empty string, the guard chain, then
   `SetTitle(player, "")` + the confirmation message.
2. Verify `game.SetTitle` accepts "" and stores it (read it; if it has its own
   empty-input rejection, that's part of this bug). The stored empty title must
   render `Okay, you're now Name .` exactly.
3. Check the with-args path still works unchanged.

## Tests

Extend the existing title tests (find them: grep `cmdTitle\|SetTitle` in
pkg/session and pkg/game tests):
- `title` (no args) → sends exactly `Okay, you're now <Name> .\r\n` and the
  player's title is now empty
- `title Foo the Tester` then `title` → title set, then cleared, both messages
  exact
- NOTITLE flag still rejects with C's message even with empty args (C checks
  the flag before the empty case matters — preserve order)

## Verification

`go build ./... && go vet ./pkg/session/... && go test ./pkg/session/... ./pkg/game/...`
All green. Do NOT run git. Report what you changed and the test output.
End your final message with the list of files modified.
