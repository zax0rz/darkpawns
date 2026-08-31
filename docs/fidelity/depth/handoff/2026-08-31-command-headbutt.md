# Depth handoff — 2026-08-31 — `headbutt`

## Frontier and queue position

- Started from clean `main` at `d7c093f6c` after the merged `help` / `?`
  handoff, pulled `main`, confirmed `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus `2026-08-31-command-help.md`.
- The starting frontier was 2,051 total, with 1,992 proven/delegated, 16
  blocked, and 43 excluded. The headbutt manifest adds 26 proven/delegated
  cases and one explicit exclusion, producing 2,078 total, 2,018
  proven/delegated, 16 blocked, and 44 excluded; actionable completion is
  2,018/2,034 (99.2%).
- `headbutt` is registered at `src/interpreter.c:487`. The `?` row at
  `src/interpreter.c:488` was already claimed by the help family because it
  reaches the same C handler. An exact manifest sweep found `heh` at
  `src/interpreter.c:489` is the next unmanifested command family; the next
  session must return to clean `main`, pull, rerun the frontier check, reread
  this handoff, and begin `heh`.

## C call path and branch inventory

The C registration and handler were traced before changing Go:

- The table row is `POS_FIGHTING`, level 1, `do_headbutt`, `subcmd=0`.
  The ordinary interpreter position/level rejection is shared with other
  fighting skills.
- `new_cmds.c:368-460` first checks the peaceful-room gate, then the learned
  skill, then mounted state. It parses only the first target token with
  `one_argument`; a visible target is selected, otherwise `FIGHTING(ch)` is
  used, and an empty fallback emits `Headbutt who?`.
- Self-targeting emits the wall message. A mortal attacking a non-NPC
  immortal receives both direct C lines and is moved to sitting. The ordinary
  roll has the sleeping/level auto-success arms, the `MOB_NOBASH` zero-percent
  quirk, and the fatal-recoil HP gate.
- The miss arm calls `damage(..., 0, SKILL_HEADBUTT)`; the hit arm applies the
  helm-dependent recoil, flat level damage, target sitting, and the shared
  skill-message/audience path. Both arms apply a three-pulse wait; player
  success improves the skill twice. The registered player row uses `subcmd=0`;
  the separate native fighter `subcmd=1` path is delegated to the existing
  `mob.fighter-headbutt` special-procedure proof.
- The descriptorless early return is explicitly excluded because a registered
  player command is descriptor-issued; no valid command-surface vehicle can
  reach that call without inventing another call site.

## Confirmed divergences and fix

The new depth vehicles were run against pre-fix `main` before the feature
branch. They found two player-visible divergences:

1. C's `one_argument` accepted `headbutt trainee trailing` as target
   `trainee`; Go joined the full token list and returned `Headbutt who?`.
2. C's self-target branch emitted `You bang your head into the nearest wall...`;
   Go invented `You contemplate headbutting yourself... maybe later.`.

Only those confirmed divergences were fixed. The command wrapper now passes the
first parsed target token to the game layer, and the game layer owns the C
self-target message. No source or C-oracle file was edited.

## Coverage proof

- Added `headbutt-depth.txt` for table-prefix resolution, no-argument,
  missing-target, one-argument, and the ordinary opener; separate vehicles
  cover peaceful, mounted, self-target, and immortal-target gates.
- The ordinary headbutt opener and target-resolution vehicle matched C at
  seeds `1, 2, 3, 5, 8`; the existing no-skill and combat opener vehicles
  remain green. The matrix covers direct messages, audiences, damage, wait,
  sitting, recoil, and both hit/miss RNG draw orders.
- Added focused registration, self-target, fighting-fallback, HP, damage,
  recoil, NOBASH, target-position, wait, improvement, skill-message, and draw
  order tests. Added 26 proving/delegating manifest rows plus the explicit
  descriptorless exclusion in `docs/fidelity/depth/headbutt.tsv`.

This follows R1/R2/R3/R4/R5e and R5c: C bytes and registration order remain
authoritative, raw target parsing and command aliases are preserved, RNG/state
proof is explicit, no behavior was invented, the actual call path was traced,
and shared combat/subcommand behavior was delegated to its owning manifests.

## Changes, gates, and integration

- PR #929 (`glm/depth-headbutt`, commit `6632bbb98`) passed hosted `test`,
  `lint`, and `security` checks after the single permitted workflow retry; the
  release-only build/deploy jobs were skipped as expected. It was merged only
  after all reported checks were green; merged `main` is `223b6f387`.
- Local gates passed: `make fidelity-depth`, `go build ./...`, `go vet ./...`,
  `go test ./...`, `golangci-lint run ./...`, `gofumpt -l .`, and
  `git diff --check`.

The next session must begin from clean `main`, pull, run `make fidelity-depth`,
reread this handoff, and continue the interpreter-table sweep with `heh` at
`src/interpreter.c:489`.
