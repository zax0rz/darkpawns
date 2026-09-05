# Depth handoff — 2026-08-31 — `farewell`

## Frontier and queue position

- Started from clean `main` at `63e1ec7c4` after the merged `faint` handoff,
  ran `git pull --ff-only`, confirmed `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus the newest prior handoff,
  `2026-08-30-command-faint.md`.
- The frontier before this slice was 1,692 total, with 1,635
  proven/delegated, 16 blocked, and 41 excluded. The farewell manifest adds 10
  cases, all proven/delegated: 6 direct oracle cases, 1 direct unit case, and
  3 shared delegations. The post-slice frontier is 1,702 total, 1,645
  proven/delegated, 16 blocked, and 41 excluded; actionable completion is
  1,645/1,661 (99.0%).
- The source-order command gap was `farewell`, registered at
  `src/interpreter.c:441`. The next command-table gap is `fart` at line 442;
  the next session must rescan from clean `main` before taking it.

## C call path and branch inventory

`src/interpreter.c:441` registers `farewell` with `POS_RESTING`, no minimum
level, and `do_action` in `src/act.social.c:102-151`. Its social record is
`lib/misc/socials:1476-1484`:

- no target: `You wave farewell.` to the actor and `$n waves farewell.` to the
  room;
- visible other target: actor, non-victim room, and victim messages with the
  `$M`/`$N` substitutions;
- missing target: `Farewell to whom?`;
- self target: `Eh?` to the actor and `#` (no room message);
- the record's minimum victim position is zero, so no victim-position branch is
  reachable for this social.

The C handler consumes only the first argument token with `one_argument`, then
uses `get_char_room_vis`; the trailing words do not affect target selection.
The shared `PLR_NOSHOUT`, command-position, visibility, and `Act` audience
classes are delegated to existing depth matrices. The Go social table and
`DoAction` follow the same record and branch structure.

## Coverage proof

The live GREEN vehicle is `farewell-depth`, covering no argument, a visible
target and all three audiences, ignored trailing words after the first target
token, self-target, and missing-target behavior. `--show-oracle` confirmed the
intended C blocks, and the vehicle was GREEN for seeds `1,2,3,5,8`.

The entry gate is covered by `TestFarewellRegistrationUsesCEntryGate`. The
shared POS_RESTING rejection delegates to `fade.position-gate`; shared
PLR_NOSHOUT and Act visibility delegate to `dance-noshout` and `socials-depth`.

No `src/` or `darkpawns-c-oracle/` file was edited. The work follows R1/R2/R4,
R5e, and R5c: C bytes and reachability remain authoritative, the actual
`do_action` path was checked before claiming the victim-position branch
unreachable, and shared behavior is delegated rather than re-invented.

## Gates and merge

Local gates passed:

- `make fidelity-depth` — 1,702 total / 1,645 proven-or-delegated /
  16 blocked / 41 excluded;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...` — 0 issues;
- `gofumpt -l .` clean;
- `git diff --check` clean.

PR #857 (`test: prove farewell command depth fidelity`) was merged only after
hosted `lint`, `security`, and `test` checks were all green. The workflow's
`build-and-push` and `deploy` jobs were skipped by policy. The next session
must return to clean `main`, pull, rerun the frontier check, and begin `fart`.
