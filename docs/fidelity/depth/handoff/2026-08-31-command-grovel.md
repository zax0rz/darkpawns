# Depth handoff — 2026-08-31 — `grovel`

## Frontier and queue position

- Started from clean `main` at `f2668104b` after the merged `grope` slice,
  pulled `main`, confirmed `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus `2026-08-31-command-grope.md`.
- The starting frontier was 1,995 total, with 1,937 proven/delegated, 16
  blocked, and 42 excluded. The `grovel` manifest adds 11
  proven/delegated cases. The post-slice frontier is 2,006 total, with 1,948
  proven/delegated, 16 blocked, and 42 excluded; actionable completion is
  1,948/1,964 (99.2%).
- `grovel` is covered at `src/interpreter.c:480`. The next unclaimed
  command-table family is `growl` at `src/interpreter.c:481`; the next session
  must return to clean `main`, pull, rerun the frontier check, reread this
  handoff, and begin `growl`.

## C call path and branch inventory

`src/interpreter.c:480` registers `grovel` as `POS_RESTING`, minimum level 0,
routed to generic `do_action`. The actual C handler in
`src/act.social.c:102-148` and authored record in `lib/misc/socials:355-363`
were traced before claiming coverage:

- `find_action` is guaranteed by the registered command; its unknown-action
  error is not reachable from this command row. `PLR_NOSHOUT` is checked first
  and uses the shared emote refusal.
- The record header `grovel 1 0` means hide flag 1 and minimum victim position
  zero. Its `char_found` field is present, so C calls `one_argument` and uses
  the first target token while discarding trailing words.
- No argument emits `You grovel in the dirt.` to the actor and
  `$n grovels in the dirt.` to the room. A missing target emits `Who?`.
- A visible target at any position reaches the actor, TO_NOTVICT room, and
  victim records: `You grovel before $M.`, `$n grovels in the dirt before
  $N.`, and `$n grovels in the dirt before you.`. A self target reaches
  `That seems a little silly to me..` and the authored `#` room suppression.
- The zero minimum victim position means a sleeping target still receives the
  normal target/victim branch. Visibility, recipient filtering, hide-bit Act
  behavior, and the shared noshout gate remain owned by existing social
  manifests rather than being duplicated here.

## Coverage evidence

The C-first `grovel-depth --seed 1 --show-oracle` vehicle exercised the full
target matrix with an authoritative spawned cleaner, awake observer, and
sleeping player. It showed exact matching blocks for no argument, first-token
target success, actor/observer/victim audiences, self, missing target, and
sleeping target. The vehicle was GREEN at seeds `1,2,3,5,8`; no Go divergence
was found, so this is a pure coverage slice. It never edits `src/` or
`darkpawns-c-oracle/`.

The manifest records 11 cases: entry gate, shared position gate, no argument,
target success, target audience, one-argument parsing, self, missing target,
sleeping target, shared noshout, and shared visibility. The focused
`pkg/session/grovel_test.go` pins the C entry gate, hide/position metadata, and
all eight authored social fields.

This follows R1/R2/R3/R4/R5e: C social bytes and the command registration remain
authoritative, five-seed deterministic parity is recorded, no behavior is
invented, and the actual generic handler path was verified. Shared social
lookup, recipient, hide, and noshout behavior follows R5b/R5c through named
existing manifests.

## Changes, gates, and integration

- Added `cmd/dp-oracle-diff/scenarios/grovel-depth.txt` with the full audience
  and target-state vehicle.
- Added `docs/fidelity/depth/grovel.tsv` with 11 explicit rows.
- Added `pkg/session/grovel_test.go` for registration and authored-record
  proof.
- No Go implementation change was needed.
- Local gates passed on the complete slice: `make fidelity-depth`,
  `go build ./...`, `go vet ./...`, `go test ./...`,
  `golangci-lint run ./...`, `gofumpt -l .`, and `git diff --check`.
- PR #919 (`glm/depth-grovel`) passed hosted `test`, `lint`, and `security`
  checks and was merged. No workflow retry was required for that PR.

The next session must begin from clean `main`, pull, run `make fidelity-depth`,
reread this handoff, and continue the interpreter-table sweep with `growl` at
`src/interpreter.c:481`.
