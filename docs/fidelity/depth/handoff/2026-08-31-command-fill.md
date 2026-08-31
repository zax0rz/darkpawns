# Depth handoff — 2026-08-31 — `fill`

## Frontier and queue position

- Started from clean `main` at `c462f06c3` after the merged fart handoff,
  ran `git pull --ff-only`, confirmed `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus the newest prior handoff,
  `2026-08-31-command-fart.md`.
- The frontier before this slice was 1,709 total, with 1,652
  proven/delegated, 16 blocked, and 41 excluded. The dedicated fill manifest
  adds 13 cases, all proven/delegated: 9 direct oracle cases, 2 direct unit
  cases, and 2 shared delegations. The post-slice frontier is 1,722 total,
  1,665 proven/delegated, 16 blocked, and 41 excluded; actionable completion
  is 1,665/1,681 (99.0%).
- The source-order command gap was `fill`, registered at
  `src/interpreter.c:443`. The next command-table gap is `finger` at line
  444; the next session must rescan from clean `main` before taking it.

## C call path and branch inventory

`src/interpreter.c:443` registers `fill` with `POS_STANDING`, no minimum level,
and `do_pour` in `src/act.item.c:1159-1335` with `SCMD_FILL`. The handler
parses two arguments with `two_arguments`, whose `one_argument` calls skip C
fill words and lowercase tokens. The fill-specific path is:

- empty target/source: `What do you want to fill?  And what are you filling it
  from?`;
- missing inventory target: `You can't find it!`;
- inventory target not a drink container: `You can't fill $p!`;
- missing source: `What do you want to fill $p from?`;
- missing room source: `There doesn't seem to be a <article> <name> here.`;
- room source not a fountain: `You can't fill something from $p.`;
- shared destination liquid conflict and capacity failures;
- successful actor/room messages `You gently fill $p from $P.` and
  `$n gently fills $p from $P.`, followed by liquid, amount, poison, alias,
  and weight mutation.

The source lookup is room-only for fill, while the target lookup is inventory;
the scenario uses separate spawned room objects to keep those boundaries
observable. Existing `pour.tsv` coverage does not claim this command-table
family, so these cases are owned by `fill.tsv`.

## Coverage proof

The live GREEN vehicle is `fill-depth`, covering usage, target/source lookup
failures, non-fountain source, non-drink target, existing-liquid conflict,
successful actor and room audiences, and full-target capacity. It drains a
one-unit tin cup before refilling so the success path and alias lookup are
reachable; `--show-oracle` confirmed every intended C block. The vehicle was
GREEN for seeds `1,2,3,5,8`.

`TestFillRegistrationUsesCEntryGate` covers the POS_STANDING registration,
`TestFillRestingActorHitsCPositionGate` covers the exact common rejection, and
`TestDoPour_FillFromFountain` covers the silent state mutation. The shared
communication gate delegates to the existing social matrix.

No `src/` or `darkpawns-c-oracle/` file was edited. The work follows R1/R2/R4,
R5e, and R5c: C bytes and lookup order remain authoritative, the actual
`do_pour` subcommand path was checked before adding cases, and shared behavior
is delegated rather than re-invented.

## Gates and merge

Local gates passed:

- `make fidelity-depth` — 1,722 total / 1,665 proven-or-delegated /
  16 blocked / 41 excluded;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...` — 0 issues;
- `gofumpt -l .` clean;
- `git diff --check` clean.

PR #861 (`test: prove fill command depth fidelity`) was merged only after
hosted `lint`, `security`, and `test` checks were all green. The workflow's
`build-and-push` and `deploy` jobs were skipped by policy. The next session
must return to clean `main`, pull, rerun the frontier check, and begin
`finger`.
