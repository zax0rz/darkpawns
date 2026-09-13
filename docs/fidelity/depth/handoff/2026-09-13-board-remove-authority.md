# Board removal authority boundary — 2026-09-13

## Disposition

The board-removal authority debt identified in the merged audit handoff
[`2026-09-12-modernization-phase6-3-audit.md`](2026-09-12-modernization-phase6-3-audit.md#c-concrete-separate-fidelity-debt-board-removal-authority-bypass)
is corrected on branch `glm/fix-board-remove-authority`. This is one bounded
behavior correction, not completion of board fidelity or consolidation of the
authority-level family. The correction is unmerged and awaiting human review.

The implementation checkpoint tested by the live runs is
`99b9818506b8b361e084bb2405f06d574ac7ff70` (`fix: match board removal
authority boundary`), based on the verified tip of merged audit PR #1450.
The final documentation-only checkpoint follows that implementation commit;
the code and scenario tested below are unchanged.

## C → Go call path

The complete exercised path is:

- C assigns `gen_board` to the real board objects at
  `src/spec_assign.c:535-555`; `src/boards.c:128-140` selects the first
  assigned board object in the actor's room; and `src/boards.c:179-210`
  dispatches `remove` to `Board_remove_msg`.
- C's normal command path reaches object specials through
  `src/interpreter.c:883-948,1407-1480`. `Board_remove_msg` parses the
  message number, checks empty/range/heading, applies the ownership and board
  remove-level gate, applies the author-level comparison, validates the slot,
  rejects active writing, frees storage, compacts the message index, emits
  actor/room output, and saves at `src/boards.c:371-440`; persistence is
  `src/boards.c:444-481`.
- Go scans the live room items in `pkg/game/boards.go:18-35`, registers the
  real object assignments in `pkg/game/spec_assign.go:328-355`, and dispatches
  `remove` from `pkg/game/boards.go:45-94`. The session's live room-special
  interception is `pkg/session/commands.go:694-733`. The board implementation
  performs number validation, the read gate, empty/range/heading checks,
  ownership/remove-level authorization, author-level authorization, slot and
  active-writing checks, storage/index mutation, actor/observer output, and
  persistence in `pkg/boards/boards.go:505-592`.

The C board table gives board 19652 (and the independent 19601/19627 boards)
read, write, and remove level 0 at `src/boards.c:92-105`, so the boundary
vehicle reaches the author-level check without an earlier board gate. The Go
read gate remains covered by the existing focused test; the correction does
not alter any preceding gate.

## Proven behavior

C defines `LVL_IMPL=40` at `src/structs.h:610`. C's
`Board_remove_msg` checks `GET_LEVEL(ch) < LVL_IMPL-1` at
`src/boards.c:403-405`, so the higher-author-level rejection is active below
39 and bypassed at 39. The prior Go condition used literal 59 at the audit
site. The fix uses the dependency-neutral compile-time value 39 in
`pkg/boards/boards.go:37-40,550-553`; no package cycle, runtime configuration,
save-format change, or authority-family consolidation was introduced.

The disposable C-first scenario creates a completed level-40 post through
the real assigned board objects in separate rooms, then exercises separate
messages with level-38, level-39, and level-40 actors. The pre-fix run at the
same scenario input preserved the intended divergence:

- level 38: C and Go both emit `You can't remove a message holier than
  yourself.` and retain the message;
- level 39: C emits `Message removed.` to the actor and
  `Boardmid just removed message 1.` to the room, while pre-fix Go emitted the
  higher-author rejection;
- level 40: equal-author-level removal succeeds.

The post-fix `--show-oracle` run is oracle-green and shows those same C blocks
for both implementations. Separate rooms/messages prevent an earlier removal
from invalidating a later case. The vehicle and its depth rows are in
[`boards-remove-authority-depth.txt`](../../../../cmd/dp-oracle-diff/scenarios/boards-remove-authority-depth.txt)
and [`boards.tsv`](../boards.tsv).

Focused state tests in `pkg/boards/boards_test.go:157-290` independently assert
the literal C-derived 38/39/40 expectations, rejection preservation of the
message/storage/saved file, successful removal persistence, adjacent-message
preservation and compaction, and exact actor/observer output. Existing
`ReadLvl`, active-writing, ownership, room-echo, and board-removal tests also
remain in the focused test selection.

## Sibling audit (R5c)

The bounded search covered `pkg/boards`, `pkg/game`, `pkg/session`,
`pkg/combat`, `pkg/spells`, and `pkg/scripting`. The only confirmed same-policy
instance of the mistaken 59 threshold was the corrected
`pkg/boards/boards.go` author-level check. Other 59 matches are unrelated
spell ID/catalog/table data or combat test data. Existing `LVL_IMPL-1` uses in
other C/Go command and privilege paths are separate policies and were left
unchanged. No same-policy sibling defect was confirmed, so the PR scope was
not broadened.

## Validation and evidence

Evidence is preserved outside the disposable harness at
`/home/zach/board-remove-authority-evidence-2026-09-12/`.

- Pre-fix proof: `go run ./cmd/dp-oracle-diff -scenario
  boards-remove-authority-depth -seed 1 -show-oracle` exited 3 with only the
  level-39 actor/mid-observer divergence. SHA-256:
  `4eb1660abdaab793b666ac9d75406465a3d62609cfa2a48385f1070be1e3f9d5` for
  `pre-fix-seed-1-show-oracle.log`.
- Post-fix proof: the same command exited 0 with `result: no normalized
  divergence`; SHA-256:
  `30a3c0cac6bdf5b50d6c190e2f28ce631f02a4f61b3d9f2b3c252a26506c4f90` for
  `post-fix-seed-1-show-oracle.log`.
- Focused seeds 1, 2, 3, 5, and 8: 5/5 exited 0 with no normalized
  divergence; the recorded tally has SHA-256
  `f1ca1d44012f20d28e2cb12ad136dc69c1e913a9423996ef681342f49ac79c03`.
- `make fidelity-depth`: 4,816 total cases; 4,697 proven/delegated; 68
  blocked; 51 excluded.
- `make expected-divergences-check`: passed; 26 existing ledger rows across
  10 scenarios and pins OK. No pins, exclusions, normalization, or runner
  classification changed.
- Full `make oracle-regression` with seed 1, timeout 240s, jobs 4:
  `scenarios=941 passed=931 expected=9 unpinnable=1 stale=0 failed=0
  infra=0 timed_out=0`; exit 2 is solely the specifically human-cleared
  `accuse-noarg-depth` baseline. Independent pre-summary accounting found
  941 classified results, 941 unique scenarios, no missing or duplicate runs.
  SHA-256:
  `80905d74c86d29725cd46737318f74ed3073eaff40403dabcdcb8f1bad094df5` for
  `full-oracle-regression.log`.
- Four infrastructure-shaped attempts were retried by the existing bounded
  worker policy (`mortal-prefs-batch`, `social-fillword`,
  `spec-proc-newbie-zone-entrance-low`, and `threaten-depth`); each resolved
  to PASS. No final infrastructure classification remained.
- Required local gates passed at the implementation checkpoint:
  `make fmt`, `make check-fmt`, `go build ./...`, `go vet ./...`,
  `go test ./...`, and `golangci-lint run ./...`.

## Fidelity rationale and boundaries

This follows R1 by matching C's player-visible acceptance and output, R2 by
proving the live command/special path, R3 by using independent fresh board
state and multiple seeds, R4 by introducing only the C-proven boundary value,
and R5 by tracing the actual C/Go call paths and auditing the sibling class.
The work does not claim broader board-fidelity completion, authority-level
consolidation, or repair of unrelated board lifecycle/storage behavior. Any
additional board defects exposed outside this threshold remain separate debt.
