# Depth handoff — 2026-08-31 — `flip`

## Frontier and queue position

- Started the slice from clean `main` at `c92ccbd26` after the merged `flex`
  handoff, ran `git pull --ff-only`, confirmed `make fidelity-depth`, and
  reread `docs/fidelity/DEPTH_TESTING.md` plus the newest prior handoff,
  `2026-08-31-command-flex.md`.
- The frontier before this slice was 1,741 total, with 1,684
  proven/delegated, 16 blocked, and 41 excluded. The `flip` manifest adds ten
  cases, all proven/delegated: eight direct oracle cases, one shared
  delegation, and one entry-gate/position unit case. The post-slice frontier
  is 1,751 total, 1,694 proven/delegated, 16 blocked, and 41 excluded;
  actionable completion is 1,694/1,710 (99.1%).
- The source-order command gap was `flip`, registered at
  `src/interpreter.c:448`. The next unmanifested command-table family is
  `flirt` at line 449.

## C call path and branch inventory

`src/interpreter.c:448` registers `flip` with `POS_STANDING`, no minimum level,
and `do_action` in `src/act.social.c:102-151`. Its social record is
`lib/misc/socials:231-239` with `0 0`: no-argument actor/room lines, target
actor/non-victim-room/victim lines, the `Who?` not-found line, and `#` for
both self-target messages. The zero minimum victim position is significant:
the target-found path must accept a sleeping target.

The handler first resolves the social, applies the shared `PLR_NOSHOUT` gate,
parses the first target token with C `one_argument`, resolves a visible room
target, handles self, then emits the direct `act` audiences. The final
three-client vehicle uses a standing peer for complete target success and as
the non-victim observer while a sleeping peer is targeted in a separate probe.
It covers no argument, target success, intentional self silence, missing
target, sleeping-target success, and leading-fill-word/trailing-token parsing.
The shared command position, noshout, and visibility classes are delegated to
existing manifests.

## Coverage proof

The clean-main baseline was already GREEN for the complete `flip-depth`
vehicle, so this was a pure-coverage round with no Go behavior change. The
seed-1 `--show-oracle` run confirmed the intended C blocks, including the
silent self branch and successful sleeping-target branch. The vehicle reported
no normalized divergence for seeds `1,2,3,5,8`.

`TestFlipRegistrationUsesCEntryGate` verifies the C POS_STANDING/level-zero
registration, and `TestFlipRestingActorHitsCPositionGate` verifies the
command-specific common rejection before `do_action`. The manifest records
the direct and delegated cases. No `src/` or `darkpawns-c-oracle/` file was
edited. The work follows R1/R2/R4, R5e, and R5c: C bytes and the actual social
call path remain authoritative, and shared social behavior is delegated rather
than duplicated.

## Gates and merge

Local gates passed:

- `make fidelity-depth` — 1,751 total / 1,694 proven-or-delegated /
  16 blocked / 41 excluded;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...` — 0 issues;
- `gofumpt -l .` clean;
- `git diff --check` clean.

The first four-peer harness attempt was rejected by the Go telnet listener's
per-IP connection cap; the final three-client topology is the supported
vehicle and is the one used for all retained evidence. PR #867 (`test: prove
flip command depth fidelity`) was merged only after hosted `lint`, `security`,
and `test` checks were all green. The workflow's `build-and-push` and `deploy`
jobs were skipped by policy. The next session must return to clean `main`,
pull, rerun the frontier check, reread the newest handoff, and begin `flirt`.
