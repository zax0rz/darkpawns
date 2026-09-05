# Depth handoff — 2026-08-31 — `french`

## Frontier and queue position

- Started from clean `main` at `8edfe32d5` after the merged `freeze` handoff,
  ran `git pull --ff-only`, confirmed `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus the newest prior handoff,
  `2026-08-31-command-freeze.md`.
- The frontier before this slice was 1,796 total, with 1,739
  proven/delegated, 16 blocked, and 41 excluded. The dedicated French
  manifests add ten proven/delegated cases: eight direct cases across two
  live vehicles and two shared delegations. The post-slice frontier is 1,806
  total, 1,749 proven/delegated, 16 blocked, and 41 excluded; actionable
  completion is 1,749/1,765 (99.1%).
- The source-order command gap was `french`, registered at
  `src/interpreter.c:453`. The next command-table gap is `frown` at line
  454; the next session must rescan from clean `main` before taking it.

## C call path and branch inventory

`src/interpreter.c:453` registers `french` with `POS_RESTING`, no minimum
level, and `do_action`. Its record is `lib/misc/socials:260-265`: hide flag
zero, minimum victim position zero, `French whom??` for no target, a
passionate actor/room/victim trio for a resolved target, a despair not-found
line, and self-target actor/room text. The actual shared call path is
`src/act.social.c:102-151`, including the PLR_NOSHOUT gate, C
`one_argument`, visible room resolution, self branch, and the plain TO_VICT
delivery behavior.

The standing vehicle uses an awake peer plus observer to expose all three
successful target audiences, no-argument, self, not-found, and leading
fill-word/trailing-token parsing. The companion vehicle uses warmup `force`
to put the peer asleep, proving that the record's zero victim-position gate
admits the sleeping target while the plain TO_VICT path suppresses the
sleeping recipient's private line. Shared position, noshout, and visibility
branches are delegated to their existing owners.

## Coverage proof

The initial clean-main vehicle was GREEN; no Go behavior change was inferred.
After adding the companion sleeping-target vehicle, both
`french-depth --show-oracle` and `french-sleeping-depth --show-oracle` were
GREEN for seeds `1,2,3,5,8`. `TestFrenchRegistrationUsesCEntryGate` pins the
C command gate and the generated social metadata. No `src/` or
`darkpawns-c-oracle/` file was edited.

The work follows R1/R2/R4, R5e, and R5c: C bytes and the actual shared
`do_action` path remain authoritative, the command surface is represented by
its source-order row, and shared behavior is delegated instead of duplicated.

## Gates and merge

Local gates passed:

- `make fidelity-depth` — 1,806 total / 1,749 proven-or-delegated /
  16 blocked / 41 excluded;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...` — 0 issues;
- `gofumpt -l .` clean;
- `git diff --check` clean.

Implementation PR #875 was merged only after hosted `lint`, `security`, and
`test` checks were all green. The workflow's `build-and-push` and `deploy`
jobs were skipped by policy. The next session must return to clean `main`,
pull, rerun the frontier check, reread this handoff, and begin `frown`.
