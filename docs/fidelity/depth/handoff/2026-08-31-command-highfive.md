# Depth handoff — 2026-08-31 — `highfive`

## Frontier and queue position

- Started from clean `main` at `7b865877c` after the merged `hide` handoff,
  pulled `main`, confirmed `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus `2026-08-31-command-hide.md`.
- The starting frontier was 2,146 total, with 2,086 proven/delegated, 16
  blocked, and 44 excluded. The highfive manifest adds 11 proven/delegated
  cases, producing 2,157 total, 2,097 proven/delegated, 16 blocked, and 44
  excluded; actionable completion is 2,097/2,113 (99.2%).
- `highfive` is registered at `src/interpreter.c:494` and reaches shared
  `do_action`. An exact interpreter-table sweep leaves `hit` at
  `src/interpreter.c:495` as the next unmanifested command family; the next
  session must return to clean `main`, pull, rerun the frontier check, reread
  this handoff, and begin `hit`.

## C call path and branch inventory

The C registration, social catalog, and shared handler were traced before
changing the manifest:

- The command row is `POS_RESTING` with minimum level 0 and dispatches to
  `do_action`. The social record at `lib/misc/socials:380-388` has
  `hide_invisible=1`, minimum victim position `POS_STANDING` (5), and all
  eight actor/room/victim/not-found/self message slots.
- `do_action` covers the registered-action and `PLR_NOSHOUT` early returns,
  no-argument actor/room pair, first-token target parsing, target lookup and
  not-found message, self-target actor/room pair, minimum victim position
  refusal, and successful actor/non-victim-room/victim audiences.
- The social hide flag suppresses the actor from observers who cannot see an
  invisible actor. Noshout and shared position/visibility behavior belong to
  existing social manifests and are delegated rather than duplicated.
- Go's existing `DoAction` and social metadata matched this C path. No player-
  facing divergence was confirmed, so no Go behavior change was made.

## Coverage proof

- Added `highfive-depth.txt` with an actor, room observer, and visible target
  for no-argument, first-token/trailing-word, successful audience, self, and
  missing-target branches.
- Added `highfive-sleeping-depth.txt` with a sleeping visible target for the
  social's proper-position gate. The main vehicle matched C for seeds `1, 2,
  3, 5, 8`; the sleeping vehicle used `--show-oracle` to verify the intended
  C block executed.
- Added focused `TestHighfiveRegistrationUsesCEntryGate`, pinning the
  command gate, social hide/victim metadata, and all authored messages. Added
  11 manifest rows in `docs/fidelity/depth/highfive.tsv`.

This follows R1/R2/R3/R4/R5e and R5c: all authored audience bytes and the
registered command surface are covered, deterministic multiseed parity was
checked, no social branches were invented, the actual interpreter-to-social
path was verified, and shared behavior was delegated to its owning matrix.

## Changes, gates, and integration

- PR #939 (`glm/depth-highfive`, commit `fbaab1003`) passed hosted `test`,
  `lint`, and `security` checks; release-only build/deploy jobs were skipped
  as expected. It was merged only after every reported check was green; merged
  `main` is `b4c3b6e4a`.
- Local gates passed: `make fidelity-depth`, `go build ./...`,
  `go vet ./...`, `go test ./...`, `golangci-lint run ./...`,
  `gofumpt -l .`, and `git diff --check`.

The next session must begin from clean `main`, pull, run `make fidelity-depth`,
reread this handoff, and continue the interpreter-table sweep with `hit` at
`src/interpreter.c:495`.
