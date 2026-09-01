# Depth-fidelity handoff — `moan`

Date: 2026-09-01  
Queue: un-manifested interpreter command families, source-table order  
Rules: R1, R2, R3, R5b, R5c, R5e

## Frontier

Started from fresh `main` after the `mindlink` handoff.  `make fidelity-depth`
reported 2,524 cases: 2,460 proven/delegated, 18 blocked, and 46 excluded
(99.3% actionable).  After adding the `moan` slice and merging it, the
frontier is 2,532 total: 2,468 proven/delegated, 18 blocked, and 46 excluded
(99.3% actionable).

The next un-manifested command family is `mold`, the registered C row at
`src/interpreter.c:551` (`POS_RESTING`, `LVL_IMMORT`, `do_mold`).  No existing
depth manifest row claims `mold`.

## C path and proof

The queue item was `{ "moan", POS_RESTING, do_action, 0, 0 }` at
`src/interpreter.c:550`.  Its social record is `moan 0 0` with only:

- `You start to moan.`
- `$n starts moaning.`
- `#`

in `lib/misc/socials:490-493`.  `src/act.social.c:102-121` therefore takes the
no-`char_found` branch: it does not resolve an argument and emits the actor
line plus the room line.  The preceding `PLR_NOSHOUT` gate is covered by the
shared dance vehicle.  The `POS_RESTING` entry gate is pinned by
`TestMoanRegistrationUsesCEntryGate`; the shared position gate and social
visibility are delegated to their existing vehicles.

Added:

- `cmd/dp-oracle-diff/scenarios/moan-depth.txt` — no argument, visible typed
  argument with trailing words, missing target, and self-looking argument.
- `cmd/dp-oracle-diff/scenarios/moan-noshout.txt` — muted actor refusal.
- `pkg/session/moan_test.go` — C command/social registration and metadata.
- `docs/fidelity/depth/moan.tsv` — eight manifest rows.

Both scenarios were GREEN on clean main before the slice commit; no Go
divergence was found and no player-facing implementation change was needed.
`moan-depth` and `moan-noshout` both passed at seeds 1, 2, 3, 5, and 8; the
seed-1 run exposed the normalized C blocks and confirmed the intended
no-argument path for every probe.

## Gates and merge

Local gates passed on `glm/depth-moan`:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...` (0 issues)
- `gofumpt -l .` clean
- `git diff --check`

Feature commit `b0684c2a9` was submitted as PR #1020 (`glm/depth-moan`).
Hosted `test`, `lint`, and `security` checks were green; `build-and-push` and
`deploy` were skipped by the workflow for this PR.  PR #1020 was self-merged
to `main` as `769821bec6753b8c10ff3d764b6c1666c67b51f4`.

No C source or oracle-tree files were edited.
