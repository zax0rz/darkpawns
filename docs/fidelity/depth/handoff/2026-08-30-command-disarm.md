# Depth handoff — disarm command

Date: 2026-08-30
Queue slice: `src/interpreter.c:414`, `disarm` / `do_disarm`
Starting main: `67e83cb86`

## Queue decision

The special-procedure inventory remains exhausted and the one permitted
`objmagic.sleep-entry-gates` cast-sleep attempt remains blocked. The preceding
`dig` command slice is claimed by open PR #835; its lint and test checks passed,
but the hosted security check failed because `vuln.go.dev` returned HTTP 403.
Per the standing rule, that PR remains open and this queue advances to the next
table item, `disarm`. No source or C-oracle file was edited.

## C path and proof

The command table registers `disarm` at `src/interpreter.c:414` with
`POS_FIGHTING`, dispatching to `do_disarm` in `src/new_cmds2.c:191-276`.
The handler parses one argument with `one_argument`, but uses `FIGHTING(ch)`
whenever present; it then checks self-targeting, the victim's wielded object,
the actor/victim combat relationship, and rolls
`number(1, 101 + GET_LEVEL(vict))` against the normal skill. Success emits the
three act() audiences with the actual `$p` short description, transfers the
weapon to the victim, optionally calls `hit(vict, ch)` when the victim was not
already fighting, improves the skill, and applies `WAIT_STATE(ch,
PULSE_VIOLENCE*2)`. Failure emits the three stumble audiences, sets the actor
to `POS_SITTING`, optionally retaliates, and applies the same wait.

The original `disarm-depth` vehicle was RED on main: Go resolved the typed
`nobody` argument and returned `They don't seem to be here.`, while C ignored
that argument and disarmed the current opponent. After the repair, the same
vehicle is GREEN. The additional `disarm-no-weapon-depth` and
`disarm-failure-depth` vehicles are also GREEN with seed 1. Focused tests cover
player equipment transfer, exact `$p` messages, gate order, inclusive target
level range, failure state, wait/improvement result flags, and the one-way
combat retaliation flag.

The Go path now resolves `FIGHTING(ch)` before typed arguments, uses the C
fill-word parser for the fallback target, checks the victim's actual wielded
slot before combat validation, transfers player and mob weapons through their
typed equipment APIs, uses `number(1, 101+level)`, and preserves the C
audience-before-retaliation ordering. This follows R1/R2/R3/R4 and R5/R5e:
exact player bytes, command-surface fidelity, deterministic draw/state parity,
no invented behavior, and verification of the actual C dispatch and call path.

## Evidence and gates

Added:

- `cmd/dp-oracle-diff/scenarios/disarm-depth.txt`
- `cmd/dp-oracle-diff/scenarios/disarm-no-weapon-depth.txt`
- `cmd/dp-oracle-diff/scenarios/disarm-failure-depth.txt`
- `docs/fidelity/depth/disarm.tsv`
- `pkg/game/disarm_depth_test.go`

The local gates passed after adding five proven rows:

```
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
test -z "$(gofumpt -l .)"
git diff --check
```

The frontier is 1543 total cases, 1489 proven/delegated, 14 blocked, and 40
excluded; actionable completion is 1489/1503 (99.1%).

## Next queue item

After this slice's PR is handled, return to clean `main`, pull, refresh the
frontier, reread the testing guide and newest handoff, and take the next
un-manifested command-table family after `disarm`.
