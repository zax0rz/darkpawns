# Depth-fidelity handoff — `split`

Date: 2026-09-03

Branch: `glm/depth-split` (feature merged); handoff branch:
`handoff/2026-09-03-command-split`

Feature PR: #1273 (merged green)

Feature commit: `446524345`

Main merge: `7cfe36a9a`

## Queue position and result

This round checked out `main`, pulled with `git pull --ff-only`, ran
`make fidelity-depth`, reread `docs/fidelity/DEPTH_TESTING.md` and the newest
dated handoff, and audited the interpreter table in source order. The
special-procedure inventory remains exhausted. The one blocked row,
`objmagic.sleep-entry-gates`, remains blocked after its one allowed
cast-sleep outlaw/reagent vehicle and was not repicked.

The next genuinely unclaimed interpreter-table family after `socials` was
`split` at `src/interpreter.c:726`. The following source-order row is the
unmanifested social `spackle` at `src/interpreter.c:727`; confirm that claim
from a fresh `main` checkout before starting it. Do not repick `split`.

Pre-slice frontier: 3,818 total, 3,715 proven/delegated, 48 blocked, and 55
excluded. The split manifest adds eight proven/delegated cases. Post-slice
frontier: 3,826 total, 3,723 proven/delegated, 48 blocked, and 55 excluded;
actionable completion is 3,723/3,771 = 98.7%.

## C call path and observable contract

The registered C row is:

```c
/* src/interpreter.c:726 */
{ "split"    , POS_SITTING , do_split   , 1, 0 },
```

`do_split` in `src/act.other.c:823-890` first rejects NPC callers, consumes
only the first token with `one_argument`, and validates the token with C's
digit-only `is_number`. Empty or zero input reaches
`Sorry, you can't do that.`; malformed input reaches
`How many coins do you wish to split with your group?`; an amount above the
actor's gold reaches `You don't seem to have that much gold to split.`.

For a valid funded amount, C counts same-room, same-group player members,
including a grouped solo actor, divides with integer truncation, subtracts
the non-actor shares from the actor, and sends the exact actor summary plus
one recipient message per follower. A player without `AFF_GROUP`, or with no
group members, receives `With whom do you wish to share your gold?`.

The focused vehicle starts three same-room grouped players with 100 gold and
proves the empty, malformed, zero, over-gold, first-token/trailing-word, and
actor/follower audience branches. The no-group refusal is separately proved
through the registered command in `TestSplitNoGroupUsesCMessage`; shared
`group`/`follow` mechanics remain delegated to their own vehicles under
R5b/R5c.

## Confirmed divergence and implementation boundary

The clean-main RED vehicle exposed three confirmed Go divergences:

- malformed input emitted the invented `That doesn't look like a number.`
  instead of C's amount prompt, and `fmt.Sscanf` accepted numeric prefixes
  that C's `is_number` rejects;
- over-gold and successful grouped distribution produced no player-visible
  Go output because the handler re-entered player locks through getters and
  setters; and
- Go rejected a grouped solo member with `num <= 1`, while C permits the
  `num == 1` integer-share case.

The Go fix uses the C first-token parser contract, preserves C's branch order,
counts grouped members without holding a lock across lock-taking helpers, and
mutates gold under one direct lock-safe section. It adds no invented output
and does not edit `src/` or `darkpawns-c-oracle/`.

Durable evidence:

- `cmd/dp-oracle-diff/scenarios/split-depth.txt`;
- `docs/fidelity/depth/split.tsv`; and
- `pkg/session/split_depth_test.go`.

The final `split-depth` vehicle reported `result: no normalized divergence`
at seeds 1, 2, 3, 5, and 8. Seed 1 was run with `--show-oracle` and showed
the intended C blocks for every annotated case.

## Gates and review

The final local gates passed after the manifest and focused tests were added:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...`
- `gofumpt -l .` clean
- `git diff --check`

PR #1273's hosted lint, security, and full test checks completed green;
conditional build-and-push and deploy were skipped. CI fired normally, so no
workflow retry was used. The PR was self-merged only after all applicable
checks were green, per the 2026-08-27 amendment.

This slice follows R1 (player-facing bytes), R2 (registered command surface),
R3 (seed matrix, exact arithmetic, and audience ordering), R4 (no invented
behavior), R5/R5e (verify the actual C path and let C win), and R5b/R5c
(shared group/follow ownership).

## Continuation

The next session must checkout `main`, pull with `--ff-only`, rerun
`make fidelity-depth`, reread the guide and newest handoff, and audit/claim
`spackle` at `src/interpreter.c:727` before touching any implementation. Do
not repick `split` or any earlier claimed family.
