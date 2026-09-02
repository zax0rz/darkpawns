# Depth-fidelity handoff — `search`

Date: 2026-09-02

## Queue position and result

This round began from synced `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, rereading `docs/fidelity/DEPTH_TESTING.md`, and reading
the newest prior handoff, `2026-09-02-command-show.md`. The special-procedure
inventory remains exhausted. The one-time blocked
`objmagic.sleep-entry-gates` row was already attempted through the cast-sleep
outlaw/reagent vehicle and was not repicked. The interpreter sweep consumed the
next genuinely unmanifested family after `credits`, `search`, at
`src/interpreter.c:411`.

The pre-slice frontier was 3,607 total cases, with 3,506 proven/delegated, 48
blocked, and 53 excluded. The search manifest contributes thirteen fully
proven/delegated cases. The resulting frontier is:

- 3,620 total cases
- 3,519 proven/delegated
- 48 blocked
- 53 excluded

Actionable completion is 3,519/3,567 = 98.7%.

## C call path and behavior surface

The registered row is:

```c
/* src/interpreter.c:411 */
{ "search"   , POS_STANDING, do_detect   , 0, 0 },
```

The handler is `src/new_cmds2.c:500-541`. It first rejects a non-elf with no
search skill, then rejects a blind player, and only then emits the room-check
line and draws `number(1, 101)`. A failed roll emits the authored failure line
and applies `WAIT_STATE(ch, PULSE_VIOLENCE + 1)`. A successful roll scans the
six exits in the C `dirs[]` order, uses case-sensitive `strstr(..., "secret")`,
emits the direction-specific wall/ceiling/floor text, and sets
`ROOM_SECRET_MARK`; it does not add a failure suffix when no secret exit is
found. The command argument is ignored by the handler.

The Go command was already registered through `command.CmdDetect`, but its
skillset display name needed to map `search` to the behavior key `detect`. The
Go implementation was corrected to use the C race gate, blind gate, pre-roll
room-check output, inclusive 1..101 roll range, exact failure wait, ordered
exit scan, case-sensitive keyword match, authored direction text, and room
secret mark. No `src/` or C-oracle file was edited.

## Evidence and verification

The C-first vehicles are:

- `cmd/dp-oracle-diff/scenarios/search-depth.txt` — no-secret success and
  trailing arguments ignored;
- `cmd/dp-oracle-diff/scenarios/search-gates-depth.txt` — ordinary no-skill
  entry gate;
- `cmd/dp-oracle-diff/scenarios/search-secret-depth.txt` — all six secret
  exits and output order;
- `cmd/dp-oracle-diff/scenarios/search-failure-depth.txt` — failed roll and
  wait-state drain with `~dpclock pulse 20` padding; and
- `cmd/dp-oracle-diff/scenarios/search-case-depth.txt` — uppercase `SECRET`
  case sensitivity.

The focused proofs are `pkg/game/detect_depth_test.go` and
`pkg/session/search_depth_test.go`. The five vehicles were run with
`DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle`; seed 1 was shown and
seeds 2, 3, 5, and 8 were also checked. Every run reported `result: no
normalized divergence`.

Durable evidence is:

- `cmd/dp-oracle-diff/scenarios/search-*.txt`;
- `docs/fidelity/depth/search.tsv`; and
- `pkg/game/detect_depth_test.go` plus `pkg/session/search_depth_test.go`.

The manifest delegates the shared standing-position rejection to the existing
movement-position proof under R5b/R5c.

## Integration and gates

The required local gates passed on `glm/depth-search`:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...  # 0 issues
gofumpt -l .             # clean
git diff --check
```

Feature commit: `f599a3c90` (`fix: match search depth behavior (R1 R2 R3 R4 R5)`).

Feature PR: #1229 (`glm/depth-search`). Hosted lint, security, and test
checks completed green; conditional build-and-push and deploy jobs were
skipped. CI fired normally, so no workflow retry was used. The PR was
self-merged only after all applicable checks were green. The resulting `main`
merge commit is `dd0ce70a3`.

This round follows R1/R2 (player-facing bytes and command surface), R3
(multi-seed and RNG-boundary evidence), R4 (no invented output), R5/R5e (the
actual C call path), and R5b/R5c (shared entry-gate and position ownership).

## Continuation

The next session must return to clean `main`, pull, rerun
`make fidelity-depth`, reread the guide and this handoff, and perform a fresh
live interpreter-table-versus-manifest sweep. Continue with the next command
family after `search` that is genuinely unmanifested in `src/interpreter.c`
table order. Do not repick a family already claimed by a handoff, and do not
follow a stale continuation that points at a family already manifested.

