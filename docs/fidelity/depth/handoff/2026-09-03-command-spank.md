# Depth-fidelity handoff — `spank`

Date: 2026-09-03

Branch: `glm/depth-spank` (feature merged); handoff branch:
`handoff/2026-09-03-command-spank`

Feature PR: #1277 (merged green)

Feature commit: `9f0be5fbe`

Main merge: `c334fd93a`

## Queue position and result

This round checked out `main`, pulled with `git pull --ff-only`, ran
`make fidelity-depth`, reread `docs/fidelity/DEPTH_TESTING.md` and the newest
dated handoff, and audited the interpreter table in source order. The
special-procedure inventory remains exhausted. The one blocked row,
`objmagic.sleep-entry-gates`, remains blocked after its one allowed
cast-sleep outlaw/reagent vehicle and was not repicked.

The next genuinely unclaimed interpreter-table family after `spackle` was the
`do_action` social `spank` at `src/interpreter.c:728`. The following
source-order row is the unmanifested social `spew` at
`src/interpreter.c:729`; confirm that claim from a fresh `main` checkout
before starting it. Do not repick `spank` or earlier claimed families.

Pre-slice frontier: 3,837 total, 3,734 proven/delegated, 48 blocked, and 55
excluded. The spank manifest adds 11 proven/delegated cases. Post-slice
frontier: 3,848 total, 3,745 proven/delegated, 48 blocked, and 55 excluded;
actionable completion is 3,745/3,793 = 98.7%.

## C call path and observable contract

The registered C row is:

```c
/* src/interpreter.c:728 */
{ "spank"    , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-151` first resolves the social record,
checks `PLR_NOSHOUT`, consumes only the first target token with `one_argument`
when `char_found` exists, and then selects no-argument, missing-target,
self-target, proper-position refusal, or target-success branches. The `spank`
record at `lib/misc/socials:804-812` is `spank 0 8`: its C hide field is 0,
its minimum victim position is POS_STANDING, and it has eight authored message
slots.

The reachable player-visible branches are:

- no argument: actor `You spank WHO?  Eh?  How?  Naaah, you'd never.` and the
  room thin-air line;
- visible standing target: actor, non-victim observer, and victim receive the
  three authored target-success lines;
- leading fill word and trailing words: C resolves only the first target token;
- self target: the actor receives `Hmm, not likely.` and the authored room
  self-target slot is the sentinel `#`;
- missing target: the authored direct-send refusal is emitted with no room act;
  and
- sleeping target: the standing victim-position minimum emits only
  `$N is not in a proper position for that.` to the actor.

The command-level POS_RESTING gate, PLR_NOSHOUT refusal, visible lookup, and
hide/audience mechanics are shared behavior delegated to
`fade.position-gate`, `dance-noshout`, and `socials-depth` under R5b/R5c.

## Evidence and implementation boundary

The clean-main proof was GREEN: the awake and sleeping `spank` vehicles matched
the C oracle at seed 1 with `--show-oracle`, and no new Go behavior change was
needed. The first combined four-peer vehicle reached a harness connection
limit on its fourth client, so the proof was split into the established
three-client awake vehicle and a separate three-client sleeping-target
vehicle. The sleeping warmup uses `~dpclock pulse 20` after `force ... sleep`
as required for wait-setting commands.

Durable evidence:

- `cmd/dp-oracle-diff/scenarios/spank-depth.txt`;
- `cmd/dp-oracle-diff/scenarios/spank-sleeping-depth.txt`;
- `docs/fidelity/depth/spank.tsv`; and
- `pkg/session/spank_depth_test.go`.

Both final vehicles reported `result: no normalized divergence` at seeds 1, 2,
3, 5, and 8. No file under `src/` or `darkpawns-c-oracle/` was edited.

## Gates and review

The final local gates passed after the manifest and focused record test were
added:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...`
- `gofumpt -l .` clean
- `git diff --check`

PR #1277's hosted lint, security, and full test checks completed green;
conditional build-and-push and deploy were skipped. CI fired normally, so no
workflow retry was used. The PR was self-merged only after all applicable
checks were green, per the 2026-08-27 amendment.

This slice follows R1 (player-facing bytes), R2 (registered command surface),
R3 (seed and audience matrix), R4 (no invented behavior), R5/R5e (verify the
actual C path and let C win), and R5b/R5c (shared social gate, lookup, and
audience ownership).

## Continuation

The next session must checkout `main`, pull with `--ff-only`, rerun
`make fidelity-depth`, reread the guide and newest handoff, and audit/claim
`spew` at `src/interpreter.c:729` before touching any implementation. Do not
repick `spank` or any earlier claimed family.
