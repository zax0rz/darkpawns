# Depth handoff — 2026-08-31 — `freeze`

## Frontier and queue position

- Started from clean `main` at `ad6e239e5` after the merged `fondle` slice,
  ran `git pull --ff-only`, confirmed `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus the newest prior handoff,
  `2026-08-31-command-flirt.md`.
- The frontier before this slice was 1,787 total, with 1,730
  proven/delegated, 16 blocked, and 41 excluded. The dedicated `freeze`
  manifest adds nine proven cases. The post-slice frontier is 1,796 total,
  1,739 proven/delegated, 16 blocked, and 41 excluded; actionable completion
  is 1,739/1,755 (99.1%).
- The source-order command gap was `freeze`, registered at
  `src/interpreter.c:452`. The next command-table gap is `french` at line
  453; the next session must rescan from clean `main` before taking it.

## C call path and branch inventory

`src/interpreter.c:452` registers `freeze` with `POS_DEAD`,
`LVL_FREEZE` (the C value is `LVL_GRGOD`, 38), and `do_wizutil` with
`SCMD_FREEZE`. The actual handler path is `src/act.wizard.c:2077-2155`:
the command-level authority guard, C `one_argument`, no-target prompt,
`get_char_vis` miss, visible-NPC rejection, higher-immortal rejection,
self-protection, already-frozen guard, victim/actor/room messages, and the
`PLR_FROZEN` plus `GET_FREEZE_LEV` mutation.

The three-client vehicle places the implementor, a victim, and a room
observer together and injects the vnum-16303 trainee in room 8162. It covers
the empty target, missing target, `self` alias, visible NPC, first freeze,
already-frozen state, and a leading fill word with trailing arguments. The
shared `do_wizutil` branches outside `SCMD_FREEZE` remain owned by their
separate command-table families.

## Coverage proof

The clean-main RED vehicle found three confirmed divergences: Go did not
resolve `self` through the world character resolver; a visible NPC was
reported as a missing player instead of reaching C's explicit mob rejection;
and standalone `cmdFreeze` passed only the first token instead of C's
fill-word-skipping `one_argument`. The Go fix now resolves C character targets
before the existing session fallback, emits the shared NPC rejection, and
parses the joined command arguments with `game.OneArgument`.

`freeze-depth --show-oracle` reported no normalized divergence for seeds
`1,2,3,5,8`. `TestFreezeRegistrationUsesCEntryGate` pins the C registration;
`TestCmdFreezeResolvesCCharacterTargets` pins self/fill-word parsing, victim
state and audience bytes, and the visible-mob branch. No `src/` or
`darkpawns-c-oracle/` file was edited.

The work follows R1/R2/R4, R5e, and R5c: C player-facing bytes and the actual
`do_wizutil` call path remain authoritative, the command surface is preserved,
and the resolver change is limited to confirmed RED behavior while existing
session-only fixtures retain their fallback.

## Gates and merge

Local gates passed:

- `make fidelity-depth` — 1,796 total / 1,739 proven-or-delegated /
  16 blocked / 41 excluded;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...` — 0 issues;
- `gofumpt -l .` clean;
- `git diff --check` clean.

Implementation PR #873 was merged only after hosted `lint`, `security`, and
`test` checks were all green. The workflow's `build-and-push` and `deploy`
jobs were skipped by policy. The next session must return to clean `main`,
pull, rerun the frontier check, reread this handoff, and begin `french`.
