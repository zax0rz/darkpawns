# Depth handoff — 2026-08-31 — `grumble`

## Frontier and queue position

- Started from clean `main` at `843dac58b` after the merged `growl` handoff,
  pulled `main`, confirmed `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus `2026-08-31-command-growl.md`.
- The starting frontier was 2,014 total, with 1,956 proven/delegated, 16
  blocked, and 42 excluded. The `grumble` manifest adds 8 proven/delegated
  cases. The post-slice frontier is 2,022 total, with 1,964 proven/delegated,
  16 blocked, and 42 excluded; actionable completion is 1,964/1,980 (99.2%).
- `grumble` is covered at `src/interpreter.c:482`. The next unclaimed
  command-table family is `gsay` at `src/interpreter.c:483`; the next session
  must return to clean `main`, pull, rerun the frontier check, reread this
  handoff, and begin `gsay`.

## C call path and branch inventory

`src/interpreter.c:482` registers `grumble` as `POS_RESTING`, minimum level 0,
routed to generic `do_action`. The C social handler in
`src/act.social.c:102-130` and authored record in
`lib/misc/socials:370-373` were traced before claiming coverage:

- `find_action` is guaranteed by the registered command; its unknown-action
  error is not reachable from this command row. `PLR_NOSHOUT` is checked first
  and uses the shared emote refusal.
- The record is `grumble 0 0` with no hide flag, no minimum victim position,
  and a `#` target field. Because `char_found` is absent, C clears the argument
  path before target lookup. Typed, missing, and self-looking arguments all
  remain on the no-argument path.
- No argument emits the authored `You grumble about life.` actor line and
  `$n grumbles to $mself.` room line. There are no target, self, not-found,
  victim, or sleeping-target branches reachable from this record.

The registered C row and the Go generic `DoAction` path were compared against
`pkg/session/commands.go`, `pkg/game/act_social.go`, the shared social parser,
and canonical Act delivery. No Go behavior change was confirmed or needed.

## Coverage proof

The C-first `grumble-depth --seed 1 --show-oracle` vehicle showed the exact
actor/observer pair for no argument, a visible target plus trailing words, a
missing target, and a self-named target. The vehicle reported no normalized
divergence at seeds `1,2,3,5,8`. It never edits `src/` or
`darkpawns-c-oracle/`.

The manifest records 8 cases: entry gate, shared position gate, no argument,
ignored visible argument, ignored missing target, ignored self target, shared
noshout, and shared visibility. The focused `pkg/session/grumble_test.go` pins
the C entry gate, hide/position metadata, and all three authored social fields.

This follows R1/R2/R3/R4/R5e: C social bytes and command registration remain
authoritative, deterministic parity is proven across five seeds, no
unreachable target branch is invented, and the actual generic handler path was
verified. Shared social position, recipient visibility, and noshout behavior
follow R5b/R5c through named existing manifests.

## Changes, gates, and integration

- Added `cmd/dp-oracle-diff/scenarios/grumble-depth.txt` with the self-only
  argument vehicle and room observer.
- Added `docs/fidelity/depth/grumble.tsv` with 8 explicit rows.
- Added `pkg/session/grumble_test.go` for registration and authored-record
  proof.
- No implementation change was made: the existing Go path matched the C
  oracle on the complete vehicle.
- Local gates passed: `make fidelity-depth`, `go build ./...`, `go vet ./...`,
  `go test ./...`, `golangci-lint run ./...`, `gofumpt -l .`, and
  `git diff --check`.
- PR #923 (`glm/depth-grumble`) passed hosted `test`, `lint`, and `security`
  checks and was merged. Checks did not initially report, so the single
  permitted `gh workflow run "Dark Pawns CI/CD" --ref glm/depth-grumble` retry
  was used; the resulting checks were all green before merge.

The next session must begin from clean `main`, pull, run `make fidelity-depth`,
reread this handoff, and continue the interpreter-table sweep with `gsay` at
`src/interpreter.c:483`.
