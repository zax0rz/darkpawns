# Depth handoff — 2026-08-30 — `escape`

## Frontier and queue position

- Started from clean `main` at `c98c88812` after the merged `enter` slice,
  ran `git pull --ff-only`, confirmed `make fidelity-depth`, and read
  `docs/fidelity/DEPTH_TESTING.md` plus the newest handoff,
  `2026-08-30-command-enter.md`.
- The frontier before this slice was 1,633 total, with 1,578
  proven/delegated, 14 blocked, and 41 excluded. This slice adds 15 proven
  cases; the post-slice frontier is 1,648 total, 1,593 proven/delegated, 14
  blocked, and 41 excluded, with actionable completion 1,593/1,607 (99.1%).
- The source-order command gap was `escape`, registered at
  `src/interpreter.c:434`. The next un-manifested command-table family is the
  next gap after `escape`; the next session must rescan the table and
  manifests from clean `main` before taking it. The later C alias `retreat`
  remains queued and was not added to this slice.

## C call path and branch inventory

`src/interpreter.c:434` registers `escape` with `POS_FIGHTING` and the shared
`do_retreat` handler in `src/act.offensive.c:1001-1075`. The handler selects
the Ninja `Escape`/`escape` spelling and `SKILL_ESCAPE`, or the ordinary
`Retreat`/`retreat` spelling and `SKILL_RETREAT`; rejects non-fighting
positions, missing skills, and no-FIGHTING callers; draws the percent roll;
uses `improve_skill` and a three-round wait on failure; then samples six
independent directions, rejects missing and death-trap exits, emits the
origin attempt Act, calls `do_simple_move`, and emits either the actor success
line or the room movement-failure line. Exhausting all six samples emits the
actor and room terminal cornered lines. `src/fight.c:1608` can call the same
handler from the automatic wimpy path.

The initial clean-main RED was the absent `escape` command/handler path. After
that implementation, the isolated Ninja vehicle exposed a second confirmed
divergence: C `skillset 'escape'` resolves catalog entry 157, whose display
name is `escape of the mongoose`, while Go combat reads the gameplay key
`escape`. `SkillStorageName(157)` now performs that single catalog-boundary
translation for skillset, practice, remort/bootstrap, and the shared skill
storage path. The manager's automatic wimpy callback now reaches
`cmdRetreat`, not `cmdFlee`.

## Coverage proof and result

The slice adds 15 manifest rows in `docs/fidelity/depth/escape.tsv`:

- position and no-skill entry gates, including ignored trailing arguments;
- ordinary and Ninja class-specific no-fight branches;
- forced percent failure;
- successful actor, origin-audience, destination-audience, and six-draw
  direction parity;
- terminal cornering after six unusable directions;
- occupied-tunnel movement failure;
- focused proof that retreat preserves the combat pair, applies the C
  three-round failure wait, and is the automatic wimpy hook target.

The proving vehicles are `escape-depth`, `escape-no-skill`,
`escape-ninja-no-fight`, `escape-no-fight`, `escape-failure`,
`escape-success`, `escape-cornered`, and `escape-movement-failure`. All live
vehicles were run with `--show-oracle` at least once. The command, class gate,
failure, success, terminal, and movement-failure vehicles are GREEN for seeds
1, 2, 3, 5, and 8; the success vehicle's peer arrival direction exposes the
selected six-draw stream. The focused tests are
`TestCmdRetreat_PreservesCombatStateAfterSuccess`,
`TestCmdRetreat_FailureSetsThreeRoundWait`, and
`TestSetFleeHooks_RetreatUsesRetreatHandler`.

No `src/` or `darkpawns-c-oracle/` file was edited. The implementation uses
the existing canonical movement path and leaves shared movement behavior to
the movement manifest. This follows R1/R2/R3/R4 and R5e: C bytes and command
registration are authoritative, draw order and audiences are exercised, no
behavior was invented, and the actual dispatch/call path was traced before
the two fixes; the catalog mismatch was audited at the shared skill-storage
boundary under R5c.

## Gates

On `glm/depth-escape`:

- `make fidelity-depth` — PASS (1,648 total / 1,593 proven-or-delegated /
  14 blocked / 41 excluded)
- `go build ./...` — PASS
- `go vet ./...` — PASS
- `go test ./...` — PASS
- `golangci-lint run ./...` — PASS, 0 issues
- `gofumpt -l .` — clean
- `git diff --check` — clean

The required hosted PR checks must be green before self-merging. If CI does
not fire, retry once with the prescribed `gh workflow run "Dark Pawns CI/CD"
--ref glm/depth-escape`; if it remains non-green, leave the PR open and move
on.
