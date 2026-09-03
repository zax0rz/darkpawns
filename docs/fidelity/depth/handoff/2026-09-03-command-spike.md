# Depth-fidelity handoff — `spike`

Date: 2026-09-03

Branch: `glm/depth-spike` (feature merged); handoff branch:
`handoff/2026-09-03-command-spike`

Feature PR: #1280 (merged green)

Feature commit: `9aa6add4e`

Main merge: `07e6d8085`

## Queue position and result

This round checked out `main`, pulled with `git pull --ff-only`, ran
`make fidelity-depth`, reread `docs/fidelity/DEPTH_TESTING.md` and the newest
dated handoff, and audited the interpreter table in source order. The special-
procedure inventory remains exhausted. The one blocked row,
`objmagic.sleep-entry-gates`, remains blocked after its one allowed cast-sleep
outlaw/reagent vehicle and was not repicked.

The next unclaimed interpreter-table family after `spew` was `spike` at
`src/interpreter.c:730`. The registered `stake` alias at line 734 uses the
same `do_spike` handler and is recorded only as a shared-handler delegation in
the spike manifest; its source-order alias does not require a second product
implementation. The next fresh queue boundary is `spit` at
`src/interpreter.c:731`; do not repick `spike` or earlier claimed families.

Pre-spike frontier: 3,859 total, 3,756 proven/delegated, 48 blocked, and 55
excluded. The spike manifest adds 18 proven/delegated cases and one excluded
caller case. Post-spike frontier: 3,878 total, 3,774 proven/delegated, 48
blocked, and 56 excluded; actionable completion is 3,774/3,822 = 98.7%.

## C call path and observable contract

The registered C rows are:

```c
/* src/interpreter.c:730 and :734 */
{ "spike"    , POS_STANDING, do_spike, 0, SCMD_SPIKE },
{ "stake"    , POS_STANDING, do_spike, 0, SCMD_STAKE },
```

The handler is `src/new_cmds.c:1098-1188`. It selects the weapon name from
`subcmd`, rejects fighting callers, prompts on no argument, resolves a visible
room target, checks the wielded object's `OBJN()` keywords, rejects peaceful
rooms and self-targets, requires the target's matching innate affect, blocks
the attacker's own nightbreed flag, protects immortal targets from mortal
attackers, and then chooses success or failure from the level comparison,
`number(0, LEVEL_IMMORT)`, or a sleeping victim.

The successful branch emits actor, non-victim room, and victim acts before
clearing player nightbreed flags, incrementing PK/death counters, calling
`raw_kill`, and waiting two violence pulses. The failure branch emits all
three miss audiences and waits two pulses. `breed_killer` in
`src/spec_procs2.c:1679-1722` directly invokes this handler for autonomous NPC
attacks; that caller belongs to the exhausted special-procedure inventory and
is excluded from the registered player command surface under R2/R5b/R5c/R5e.

## Confirmed divergences and evidence

The clean-main RED vehicle confirmed four player-visible/state divergences:

- Go invented a generic skill-knowledge gate although C has none;
- Go's target lookup used the wrong missing-target literal;
- Go searched the weapon short description and rejected NPC targets, while C
  searches `OBJN()` keywords and accepts affected NPCs; and
- Go used an exclusive upper bound for the C inclusive
  `number(0, LEVEL_IMMORT)` arm and finalized raw-kill state before the
  authored audience acts.

The fix is limited to those confirmed paths. The registered vehicle uses an
innate werewolf mob, a silver spike object, actor/observer peers, and the
required `~dpclock pulse 20` padding after the kill. The final
`spike-depth` runs at seeds 1, 2, 3, 5, and 8 all reported
`result: no normalized divergence`; seed 1 with `--show-oracle` exposed the
exact success acts and death-cry ordering.

Durable evidence:

- `cmd/dp-oracle-diff/scenarios/spike-depth.txt`;
- `docs/fidelity/depth/spike.tsv`;
- `pkg/game/spike_depth_test.go`; and
- `pkg/session/spike_depth_test.go`.

No file under `src/` or `darkpawns-c-oracle/` was edited.

## Gates and review

The final local gates passed:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...`
- `gofumpt -l .` clean
- `git diff --check`

PR #1280's hosted lint, security, and full test checks completed green;
conditional build-and-push and deploy were skipped. CI fired normally, so no
workflow retry was used. The PR was self-merged only after all applicable
checks were green, per the 2026-08-27 amendment.

This slice follows R1 (player-facing bytes), R2 (registered command surface),
R3 (seed, ordering, and state parity), R4 (no invented behavior), R5/R5e
(verify the actual C call path and let C win), and R5b/R5c (shared ownership
and autonomous-caller boundaries).

## Continuation

The next session must checkout `main`, pull with `--ff-only`, rerun
`make fidelity-depth`, reread the guide and newest handoff, and audit/claim
`spit` at `src/interpreter.c:731` before touching any implementation. Do not
repick `spike` or any earlier claimed family.
