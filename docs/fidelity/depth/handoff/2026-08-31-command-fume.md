# Depth handoff — 2026-08-31 — `fume`

## Frontier and queue position

- Started from clean `main` at `50cbeed8e` after the merged `frown` handoff,
  ran `git pull --ff-only`, confirmed `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus the newest prior handoff,
  `2026-08-31-command-frown.md`.
- The frontier before this slice was 1,813 total, with 1,756
  proven/delegated, 16 blocked, and 41 excluded. The dedicated `fume`
  manifest adds ten proven/delegated cases: seven direct cases and three
  shared delegations. The post-slice frontier is 1,823 total, 1,766
  proven/delegated, 16 blocked, and 41 excluded; actionable completion is
  1,766/1,782 (99.1%).
- The source-order command gap was `fume`, registered at
  `src/interpreter.c:455`. The next command-table gap is `future` at line
  456; the next session must rescan from clean `main` before taking it.

## C call path and branch inventory

`src/interpreter.c:455` registers `fume` with `POS_RESTING`, no minimum level,
and `do_action`. Its record is `lib/misc/socials:275-282`: hide flag one,
victim minimum `POS_RESTING` (record field five), actor/room no-target text,
the fuming target trio, a record-specific not-found response, and self-target
actor/room text. The actual shared path is `src/act.social.c:102-151`,
including PLR_NOSHOUT, C `one_argument`, visible room resolution, self,
victim-position, and audience branches.

The standing vehicle uses an awake peer and observer to expose no-target,
target-success actor/room/victim output, not-found, self, and leading
fill-word/trailing-token parsing. The companion vehicle uses warmup `force`
to put the target asleep, proving the record's POS_RESTING victim gate before
audience emission. Shared position, noshout, and visibility behavior is
delegated to existing owners. An initial fixture-name setup was rejected by
C before gameplay; replacing the synthetic names with valid neutral names
was a harness correction, not a proof attempt or behavior change.

## Coverage proof

The corrected `fume-depth --show-oracle` and `fume-sleeping-depth
--show-oracle` vehicles reported no normalized divergence for seeds
`1,2,3,5,8`. `TestFumeRegistrationUsesCEntryGate` pins the command gate and
the social record metadata. No Go behavior change was inferred, and no
`src/` or `darkpawns-c-oracle/` file was edited.

The work follows R1/R2/R4, R5e, and R5c: C bytes and the actual shared
`do_action` call path remain authoritative, the record's position constant was
verified against C's position enum, and shared behavior is delegated rather
than duplicated.

## Gates and merge

Local gates passed:

- `make fidelity-depth` — 1,823 total / 1,766 proven-or-delegated /
  16 blocked / 41 excluded;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...` — 0 issues;
- `gofumpt -l .` clean;
- `git diff --check` clean.

The first hosted check query for implementation PR #879 reported no checks;
the one permitted `gh workflow run "Dark Pawns CI/CD" --ref glm/depth-fume`
retry caused the pull-request run to attach. PR #879 was merged only after
its attached hosted `lint`, `security`, and `test` checks were all green; the
workflow's `build-and-push` and `deploy` jobs were skipped by policy. The next
session must return to clean `main`, pull, rerun the frontier check, reread
this handoff, and begin `future`.
