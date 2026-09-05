# 2026-09-01 — `mindlink` depth slice

## Frontier and queue

- Started from refreshed `main` at the post-map frontier: 2,510 total cases,
  2,447 proven/delegated, 18 blocked, and 45 excluded; `make fidelity-depth`
  passed before source work.
- Read `docs/fidelity/DEPTH_TESTING.md` and the newest handoff,
  `2026-09-01-command-map.md`, before selecting the next source-order family.
- Special-procedure inventory remains exhausted. The one blocked
  `objmagic.sleep-entry-gates` row remains blocked after its single cast-sleep
  outlaw/reagent attempt.
- The next un-manifested interpreter-table family is `moan`,
  `src/interpreter.c:550`, registered to `do_action`.

## C path and findings

- `src/interpreter.c:549` registers `mindlink` at `POS_STANDING` with no
  minimum level; `src/new_cmds2.c:254-325` is the handler.
- C consumes only the first argument with `one_argument`, then checks empty
  argument, visible target, self, skill, non-NPC target, fighting, actor HP,
  target mana, and finally the percentage path (R1/R2/R5e).
- The normal loaded NPC target begins with `GET_MANA = GET_MAX_MANA = 10`
  (`src/db.c:1069-1072,1757`), so C's target-resource gate is the ordinary
  live path. `IS_PSIONIC` and `IS_MYSTIC` both include `!IS_NPC`
  (`src/utils.h:366,418`), so the apparent success branch is unreachable after
  the valid NPC target path; no substitute class behavior was invented (R4,
  R5e).
- RED vehicles found the missing mob-mana gate, missing actor `You fail.`
  line, missing target-inclusive room audience, and multi-token target parsing
  mismatch. The post-roll unit path also confirmed missing fixed 100-HP drain,
  `improve_skill`, stunned position, post-room ordering, and C's `number(1,101)`
  draw.

## Proof and implementation

- Added `mindlink-entry-depth`, `mindlink-low-mana-depth`, and
  `mindlink-player-depth`; all three are GREEN at seeds 1, 2, 3, 5, and 8.
  The `--show-oracle` runs confirmed the intended C blocks.
- Added `docs/fidelity/depth/mindlink.tsv` with 13 proven/delegated rows and
  one explicit excluded unreachable-success row. The manifest frontier is now
  2,524 total, 2,460 proven/delegated, 18 blocked, and 46 excluded; actionable
  completion is 2,460/2,478 = 99.3%.
- Fixed `CmdMindlink` first-token parsing, C mob mana initialization, the
  target mana gate, NPC failure outcome and draw path, player-target audiences,
  and post-room actor message/position sequencing. Added focused gate, state,
  audience, and RNG tests. No `src/` or oracle-tree files were edited.
- Local gates passed: `make fidelity-depth`, `go build ./...`, `go vet ./...`,
  `go test ./...`, `golangci-lint run ./...` (`0 issues`), `gofumpt -l .`, and
  `git diff --check`.
- PR #1018 (`glm/depth-mindlink`) had green hosted lint, security, and test
  checks; build/deploy were correctly skipped. It was self-merged as
  `ee80ec398` after all checks were green.

## Next action

Refresh `main`, run `make fidelity-depth`, reread this handoff and
`DEPTH_TESTING.md`, then take `moan` in interpreter table order on
`glm/depth-moan`, with one PR and one dated handoff for that slice. Cite R1-R5
by number in the resulting evidence.
