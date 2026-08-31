# Depth handoff — 2026-08-31 — `hiccup`

## Frontier and queue position

- Started from clean `main` at `900dbed28` after the merged `hcontrol`
  handoff, pulled `main`, confirmed `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus
  `2026-08-31-command-hcontrol.md`.
- The starting frontier was 2,117 total, with 2,057 proven/delegated, 16
  blocked, and 44 excluded. The hiccup manifest adds 8 proven/delegated cases,
  producing 2,125 total, 2,065 proven/delegated, 16 blocked, and 44 excluded;
  actionable completion is 2,065/2,081 (99.2%).
- `hiccup` is registered at `src/interpreter.c:492` and reaches the shared
  `do_action` social path. An exact interpreter-table sweep leaves `hide` at
  `src/interpreter.c:493` as the next unmanifested command family; the next
  session must return to clean `main`, pull, rerun the frontier check, reread
  this handoff, and begin `hide`.

## C call path and branch inventory

The C registration, social catalog, and handler were traced before changing
the manifest or test vehicle:

- The row is `POS_RESTING`, level 0, and dispatches to `do_action`.
- `do_action` first checks the registered action and `PLR_NOSHOUT`; because
  the `hiccup` social record has `#`/NULL `char_found`, any typed argument is
  discarded and the handler emits only the no-argument actor and room
  messages.
- The C social record at `lib/misc/socials:375-379` is self-only:
  `*HIC*`, `$n hiccups.`, `#`. There is therefore no target-success,
  self-target, or target-not-found branch for this command. Position and
  noshout behavior are shared with the existing social vehicles.
- The Go social metadata and shared `DoAction` implementation already follow
  this C path. No source or C-oracle file was edited, and no Go behavior
  divergence was found.

## Coverage proof

- Added `hiccup-depth.txt` with actor/observer setup and no-argument,
  visible-target-plus-trailing-words, missing-target, and self-named probes.
  The vehicle matched the C oracle at seeds `1, 2, 3, 5, 8`; one run used
  `--show-oracle` to verify the intended C blocks executed.
- Added focused `TestHiccupRegistrationUsesCEntryGate` and 8 manifest rows in
  `docs/fidelity/depth/hiccup.tsv`. The position gate, noshout refusal, and
  room visibility are delegated to already-proven shared social vehicles.

This follows R1/R2/R3/R4/R5e and R5c: the authored player-facing bytes and
registered command gate remain authoritative, deterministic multiseed oracle
parity was checked, no unreachable target branches were invented, and the
actual interpreter-to-social call path was verified.

## Changes, gates, and integration

- PR #935 (`glm/depth-hiccup`, commit `d7275c40a`) passed hosted `test`,
  `lint`, and `security` checks; the release-only build/deploy jobs were
  skipped as expected. It was merged only after every reported check was
  green; merged `main` is `432435bca`.
- Local gates passed: `make fidelity-depth`, `go build ./...`,
  `go vet ./...`, `go test ./...`, `golangci-lint run ./...`,
  `gofumpt -l .`, and `git diff --check`.

The next session must begin from clean `main`, pull, run `make fidelity-depth`,
reread this handoff, and continue the interpreter-table sweep with `hide` at
`src/interpreter.c:493`.
