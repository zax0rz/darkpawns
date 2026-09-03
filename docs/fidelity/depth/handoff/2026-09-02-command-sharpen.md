# Depth handoff — sharpen

Date: 2026-09-02  
Branch: `glm/depth-sharpen`  
Feature PR: #1237 (merged green)  
Feature commit: `b3ad2d139`  
Main merge: `ceefbc899`

## Queue position

The special-procedure inventory remains exhausted. The single blocked row,
`objmagic.sleep-entry-gates`, was already attempted once through the
cast-sleep outlaw/reagent vehicle and remains blocked for the unreachable
entry gates. After checking out `main`, pulling with `--ff-only`, running the
frontier, reading `DEPTH_TESTING.md`, and reviewing the newest dated handoff
(`2026-09-02-command-retsu.md`), the next genuinely unclaimed interpreter
family was `sharpen` at `src/interpreter.c:692`. Earlier source-order entries
were either already owned by a command manifest or were shared `do_action`,
`do_not_here`, or kuji-kiri behavior and were delegated under R5b/R5c.

Pre-slice frontier: 3,655 total, 3,554 proven/delegated, 48 blocked, 53
excluded.

The sharpen slice adds 18 manifest cases. Post-slice frontier: 3,673 total,
3,572 proven/delegated, 48 blocked, 53 excluded; actionable completion is
3,572/3,620 = 98.7%.

The next session must refresh `main`, rerun the frontier, reread the guide and
newest handoff, then continue after `sharpen` in interpreter-table order. The
fresh source-order audit identifies the next unclaimed family as the shared
`do_shutdown` pair, `shutdow`/`shutdown`, at `src/interpreter.c:698-699`;
confirm that claim and its actual C path before working it. Do not repick any
existing manifest or shared-class claim.

## C call path and observable contract

The command table registers:

```c
{ "sharpen"  , POS_RESTING , do_sharpen  , 0, 0 }
```

at `src/interpreter.c:692`. `do_sharpen` in `src/new_cmds.c:2743-2791`
first applies `half_chop`, then calls `get_obj_in_list_vis` against carrying
only. Missing input or a missing carried object emits `Sharpen what?`.
The object must be `ITEM_WEAPON` with `value[3] == TYPE_SLASH-TYPE_HIT`;
otherwise C emits `This weapon can not be sharpened.`. Any non-empty object
affect slot or `ITEM_MAGIC` emits `This weapon can not be sharpened any
further.`. Only then does C reject `FIGHTING(ch)` with the busy message.

The live branch draws exactly `number(1,101)` and succeeds only when the draw
is strictly below `GET_SKILL`. Success writes affect slot zero to
`APPLY_DAMROLL` with `1 + (level == 30) + (level > 25)`, sends the room act
before the actor line, and emits `You sharpen it to perfection!`. Failure
writes damroll `-1`, sends the room act before the actor line, emits
`You damage it trying to sharpen it!`, and calls `improve_skill` after the
actor message. A valid object with skill zero still reaches the draw and
failure mutation; C has no skill-zero early return in this handler.

No `src/` or `darkpawns-c-oracle/` file was edited.

## Proof artifacts

Scenarios:

- `cmd/dp-oracle-diff/scenarios/sharpen-depth.txt` — argument and carried
  lookup, half-chop behavior, non-weapon and magic gates, success actor/room
  audiences.
- `cmd/dp-oracle-diff/scenarios/sharpen-failure-depth.txt` — failure actor and
  room order with mutation/improvement timing.
- `cmd/dp-oracle-diff/scenarios/sharpen-fighting-depth.txt` — post-lookup
  fighting gate with missing and valid carried-object probes.

Manifest: `docs/fidelity/depth/sharpen.tsv` (18 rows).

Focused tests:

- `pkg/game/sharpen_depth_test.go` — carried-only/object predicates,
  affect-scaling and failure mutation, fighting order, and one-draw parity.
- `pkg/session/sharpen_depth_test.go` — C registration gate.
- `pkg/command/skill_commands_test.go` — live fighting command setup.

With `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle`, all three
scenarios produced `result: no normalized divergence` for seeds 1, 2, 3, 5,
and 8. Seed 1 was inspected with `--show-oracle` for the intended C blocks.
The RED on main was the Go skill-zero early return and invented object/error
messages, with no C-compatible object mutation or room audience. The Go
implementation now matches the confirmed C gates, bytes, state, ordering,
and draw bounds.

## Gates and review

All local gates passed on the final feature commit:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...`
- `gofumpt -l .` (clean)
- `git diff --check`

PR #1237's hosted lint, security, and test checks were green; build/deploy
were skipped by the workflow. It was self-merged only after the applicable
checks were green.

This slice follows R1 (player-facing bytes), R2 (registered command surface),
R3 (draw, state, and ordering parity), R4 (no invented behavior), R5 and R5e
(verify the actual C call path and let C win). Shared behavior was audited and
delegated under R5b/R5c.
