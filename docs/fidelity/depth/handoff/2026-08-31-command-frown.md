# Depth handoff — 2026-08-31 — `frown`

## Frontier and queue position

- Started from clean `main` at `73c4e4fe5` after the merged `french` handoff,
  ran `git pull --ff-only`, confirmed `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus the newest prior handoff,
  `2026-08-31-command-french.md`.
- The frontier before this slice was 1,806 total, with 1,749
  proven/delegated, 16 blocked, and 41 excluded. The dedicated `frown`
  manifest adds seven proven/delegated cases: four direct cases and three
  shared delegations. The post-slice frontier is 1,813 total, 1,756
  proven/delegated, 16 blocked, and 41 excluded; actionable completion is
  1,756/1,772 (99.1%).
- The source-order command gap was `frown`, registered at
  `src/interpreter.c:454`. The next command-table gap is `fume` at line 455;
  the next session must rescan from clean `main` before taking it.

## C call path and branch inventory

`src/interpreter.c:454` registers `frown` with `POS_RESTING`, no minimum
level, and `do_action`. Its record is `lib/misc/socials:270-273`: zero hide
flag and zero victim position, `What's bothering you?` for the actor,
`$n frowns.` for the room, and `#` for the char-found slot. The actual C
handler path is `src/act.social.c:102-127`: because `char_found` is absent,
it does not parse or resolve a target and always emits the no-argument pair;
the shared PLR_NOSHOUT and position gates run before it.

The two-client vehicle uses an awake room observer and probes bare, missing,
self-looking, and leading-fill-word/trailing-token arguments. Every probe
therefore proves that the record remains self-only and that typed arguments
cannot enter the target branches. Shared position, noshout, and visibility
behavior is delegated to its existing owners.

## Coverage proof

The clean-main vehicle was GREEN; no Go behavior change was inferred.
`frown-depth --show-oracle` reported no normalized divergence for seeds
`1,2,3,5,8`. `TestFrownRegistrationUsesCEntryGate` pins the command gate and
the no-char-found social record. No `src/` or `darkpawns-c-oracle/` file was
edited.

The work follows R1/R2/R4, R5e, and R5c: C bytes and the actual `do_action`
call path remain authoritative, the command surface is represented in source
order, and shared behavior is delegated rather than duplicated.

## Gates and merge

Local gates passed:

- `make fidelity-depth` — 1,813 total / 1,756 proven-or-delegated /
  16 blocked / 41 excluded;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...` — 0 issues;
- `gofumpt -l .` clean;
- `git diff --check` clean.

Implementation PR #877 was merged only after hosted `lint`, `security`, and
`test` checks were all green. The workflow's `build-and-push` and `deploy`
jobs were skipped by policy. The next session must return to clean `main`,
pull, rerun the frontier check, reread this handoff, and begin `fume`.
