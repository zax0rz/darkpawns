# Depth handoff — 2026-08-31 — `grope`

## Frontier and queue position

- Started from clean `main` at `520f41c6c` after the merged `groinrip`
  handoff, pulled `main`, confirmed `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus `2026-08-31-command-groinrip.md`.
- The starting frontier was 1,984 total, with 1,926 proven/delegated, 16
  blocked, and 42 excluded. The `grope` manifest adds 11
  proven/delegated cases. The post-slice frontier is 1,995 total, with 1,937
  proven/delegated, 16 blocked, and 42 excluded; actionable completion is
  1,937/1,953 (99.2%).
- `grope` is the next source-order command after `groinrip` at
  `src/interpreter.c:479`. The next session must return to clean `main`, pull,
  rerun the frontier check, reread this handoff, and take `grovel` at
  `src/interpreter.c:480`.

## C call path and branch inventory

`src/interpreter.c:479` registers `grope` as `POS_RESTING`, minimum level 0,
routed to the generic `do_action` handler. The C handler in
`src/act.social.c:102-148` was traced against the authored record at
`lib/misc/socials:345-353`:

- `find_action` is already guaranteed by the registered command and the
  unknown-action error is outside this command's reachable call path.
- `PLR_NOSHOUT` is checked before record handling and uses the shared emote
  refusal. The grope record has hide flag 5 and no victim-position minimum.
- Because `char_found` is authored as `Well, what sort of noise do you expect
  here?`, C parses one argument with `one_argument`. The no-argument record is
  `Whom do you wish to grope??` for the actor and `#` for the room.
- A visible target gets the actor, hidden `TO_NOTVICT` room, and victim lines;
  a missing target gets `Try someone who's here.`. A self target gets the
  authored YUCK actor and room lines. A sleeping target is rejected by the
  record's minimum victim position 5 with C's generic proper-position line,
  without social audience output.
- The generic visibility, target lookup, recipient filtering, and hide-bit Act
  behavior are shared social machinery; they are delegated to the existing
  `socials-depth` and `dance-noshout` proofs.

## Coverage evidence

The C-first `grope-depth --seed 1 --show-oracle` vehicle exercised all intended
branches and showed the exact actor/observer/sleeper blocks. It was GREEN at
seeds `1,2,3,5,8`; no implementation divergence was found, so this is a pure
coverage slice. The vehicle uses an awake observer, a spawned authoritative
cleaner mob, and a sleeping player, and does not edit `src/` or
`darkpawns-c-oracle/`.

The manifest records 11 cases: registration, shared position gate, no
argument, target success, target audience, one-argument parsing, self, missing
target, sleeping-target position gate, shared noshout, and shared visibility.
`pkg/session/grope_test.go` pins the C entry gate and all eight authored social
record fields. It is present on `main` at `cdb21d365`; an explicit
reapply-after-revert was needed because the branch-repair history otherwise
made GitHub treat the already-reviewed test commit as the merge base. The
final `main` tree and frontier are green.

This follows R1/R2/R3/R4/R5e: C social bytes and the registered command surface
remain authoritative, deterministic behavior is checked across five seeds, no
unreachable behavior is invented, and the actual `do_action` path was traced
before claiming proof. Shared social lookup, recipient, hide, and noshout
behavior follows R5b/R5c through named existing manifests rather than being
duplicated.

## Changes, gates, and integration

- Added `cmd/dp-oracle-diff/scenarios/grope-depth.txt` with the full audience
  and target-state vehicle.
- Added `docs/fidelity/depth/grope.tsv` with 11 explicit rows.
- Added `pkg/session/grope_test.go` for registration and authored-record proof.
- No Go implementation change was needed.
- Local gates passed on the complete slice: `make fidelity-depth`,
  `go build ./...`, `go vet ./...`, `go test ./...`,
  `golangci-lint run ./...`, `gofumpt -l .`, and `git diff --check`.
- PR #917 (`glm/depth-grope`) passed hosted `test`, `lint`, and `security`
  checks and was merged. No workflow retry was required.

The next session must begin from clean `main`, pull, run `make fidelity-depth`,
reread this handoff, and continue the interpreter-table sweep with `grovel` at
`src/interpreter.c:480`.
