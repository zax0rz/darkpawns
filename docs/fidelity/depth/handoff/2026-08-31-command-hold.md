# Depth handoff — 2026-08-31 — `hold`

## Frontier and queue position

- Started from clean `main` at `e4fd21b65` after the merged `hit` handoff,
  pulled `main`, confirmed `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus `2026-08-31-command-hit.md`.
- The starting frontier was 2,163 total, with 2,103 proven/delegated, 16
  blocked, and 44 excluded. The hold manifest adds 4 cases, producing 2,167
  total, 2,107 proven/delegated, 16 blocked, and 44 excluded; actionable
  completion remains 2,107/2,123 = 99.2%.
- The exact interpreter-table sweep advances past `hire` (already covered by
  the do_not_here family), `holler` (channels), and `holylight` (gen-tog).
  `home` at `src/interpreter.c:500`, routed to `do_home`, is the next
  unmanifested command family; the next session must return to clean `main`,
  pull, rerun the frontier check, reread this handoff, and begin `home`.

## C call path and branch inventory

The C source was traced before recording coverage:

- `hold` and `grab` both dispatch to `do_grab`; `hold` is registered at
  `src/interpreter.c:497` with `POS_RESTING` and minimum level 1, while `grab`
  is the level-0 row at `:472`.
- `do_grab` calls `one_argument`, answers `Hold what?` for empty post-fill
  input, resolves visible inventory objects, emits C's article-aware missing
  object line, rejects non-holdable objects except wand/staff/scroll/potion,
  and otherwise calls `perform_wear(..., WEAR_HOLD)`.
- The shared `perform_wear` path owns hold-slot occupancy, two-handed wield
  conflicts, invalid-class handling, inventory/equipment state, and actor/room
  wear messages. Those branches are already proven by `grab` and delegated
  under R5b/R5c; this round adds the distinct `hold` registration boundary.

## Coverage and result

- Added `hold-depth.txt` with the level-1 player's bare `hold` command. The
  C oracle and Go port are byte-equal (`Hold what?`), and `--show-oracle`
  confirmed the intended `do_grab` block.
- This is a pure-coverage round: no Go behavior changed. The existing
  `TestGrabAndHoldKeepDistinctCGates` pins the separate C command entries and
  both POS_RESTING gates; `hold.tsv` records the direct proof plus delegation
  of the shared object/equipment matrix.

This follows R1/R2/R3/R4/R5e: the command surface and authored bytes are
preserved, no RNG-bearing claim was made, no behavior was invented, and the
actual shared-handler path was verified. Delegations follow R5b/R5c.

## Changes, gates, and integration

- PR #943 (`glm/depth-hold`, commit `53ef80043`) passed hosted `test`, `lint`,
  and `security`; release-only build/deploy jobs were skipped as expected. It
  was merged only after every reported check was green; merged `main` is
  `53fca9011`.
- Local gates passed: `make fidelity-depth`, `go build ./...`, `go vet ./...`,
  `go test ./...`, `golangci-lint run ./...`, `gofumpt -l .`, and
  `git diff --check`.

