# Depth handoff — 2026-08-31 — `flirt`

## Frontier and queue position

- Started the slice from clean `main` at `9a53bd40b` after the merged `flip`
  handoff, ran `git pull --ff-only`, confirmed `make fidelity-depth`, and
  reread `docs/fidelity/DEPTH_TESTING.md` plus the newest prior handoff,
  `2026-08-31-command-flip.md`.
- The frontier before this slice was 1,751 total, with 1,694
  proven/delegated, 16 blocked, and 41 excluded. The `flirt` manifest adds ten
  cases, all proven/delegated: seven direct oracle/unit cases and three shared
  delegations. The post-slice frontier is 1,761 total, 1,704
  proven/delegated, 16 blocked, and 41 excluded; actionable completion is
  1,704/1,720 (99.1%).
- The source-order command gap was `flirt`, registered at
  `src/interpreter.c:449`. The next unmanifested command-table family is
  `follow` at line 450.

## C call path and branch inventory

`src/interpreter.c:449` registers `flirt` with `POS_RESTING`, no minimum level,
and `do_action` in `src/act.social.c:102-151`. The social record is
`lib/misc/socials:240-248`: first field 1 enables hidden-invisible actor
filtering, second field 5 requires a resting victim, and the record supplies
no-argument actor/room lines, target actor/non-victim-room/victim lines, the
dearly-beloved not-found line, and self-target actor/room lines.

The handler resolves the social, applies the shared `PLR_NOSHOUT` gate, parses
the first target token with C `one_argument`, resolves a visible room target,
handles self and minimum victim position, and emits the direct `act` audiences.
The three-client vehicle uses a standing target and room observer to expose
the no-argument pair and complete target trio, plus self, not-found, and
leading-fill-word/trailing-token parsing. A focused game-layer test sets the
victim to sleeping and proves the exact minimum-position rejection. The
shared command position, noshout, and visibility classes are delegated to
existing manifests; the social table parity test owns the record metadata.

## Coverage proof

The clean-main baseline was already GREEN for the complete `flirt-depth`
vehicle, so this was a pure-coverage round with no Go behavior change. The
seed-1 `--show-oracle` run confirmed hidden-aware no-argument audience output,
all three target audiences, self-target output, not-found, and C parser
behavior. The vehicle reported no normalized divergence for seeds
`1,2,3,5,8`.

`TestFlirtRegistrationUsesCEntryGate` verifies the C POS_RESTING/level-zero
registration. `TestDoActionFlirtSleepingTargetHitsPositionGate` verifies the
social minimum victim-position branch and its silent target audience. The
manifest records the direct and delegated cases. No `src/` or
`darkpawns-c-oracle/` file was edited. The work follows R1/R2/R4, R5e, and
R5c: C bytes and the actual social call path remain authoritative, and shared
social behavior is delegated rather than duplicated.

## Gates and merge

Local gates passed:

- `make fidelity-depth` — 1,761 total / 1,704 proven-or-delegated /
  16 blocked / 41 excluded;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...` — 0 issues;
- `gofumpt -l .` clean;
- `git diff --check` clean.

PR #869 (`test: prove flirt command depth fidelity`) was merged only after
hosted `lint`, `security`, and `test` checks were all green. The workflow's
`build-and-push` and `deploy` jobs were skipped by policy. The next session
must return to clean `main`, pull, rerun the frontier check, reread the newest
handoff, and begin `follow`.
