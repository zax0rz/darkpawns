# Depth handoff — 2026-09-02 — command `shadow`

The `shadow` depth slice is complete. The C path was audited from the
registered `src/interpreter.c:687` row through `src/act.movement.c:883-951`
(`do_follow`) and `src/utils.c:397-498` (`stop_follower` and
`add_follower`). The command is registered at POS_STANDING with no level gate
and uses the quiet `do_follow` subcommand. Its call path establishes the shared
target, charm, self, circle, and leader-switch relation gates, then branches on
the C `number(0,101)` skill roll. Success applies one level-duration
SKILL_SHADOW affect with AFF_DODGE, emits only the actor's quiet follow line,
and uses quiet follower state. Failure falls through to ordinary follower
audiences. Shadow-aware self-stop removes the affect and emits only the actor
stop line. Shared relation behavior remains delegated to the existing follow
matrix under R5b/R5c.

The pre-fix RED probes were concrete: the Go shadow stop used ordinary follow
stop bytes and notified the leader and room, while a failed quiet roll omitted
the C leader and room audience lines. The Go fix is limited to those confirmed
divergences and the C success branch: it preserves the skill draw before the
immortal shortcut, replaces prior shadow state in C order, and keeps the
AFF_DODGE/ SKILL_SHADOW transition explicit. No `src/` or
`darkpawns-c-oracle/` file was edited.

Evidence is recorded in `docs/fidelity/depth/shadow.tsv` and the vehicles
`shadow-depth` and `shadow-failure-depth`. `shadow-depth` is green on seeds 1,
2, 3, 5, and 8, with seed 1 run using `--show-oracle`; it proves the quiet
actor audience, shadow affect path, and shadow-only self-stop. The failure
vehicle is green on seeds 1, 2, 3, 5, and 8, with seed 1 run using
`--show-oracle`; it proves the ordinary actor, awake-leader, and room audience
bytes after a failed quiet roll. Focused tests cover the C registration gate,
success affect shape, failure no-affect state, exact one-draw ordering,
shadow-aware stop cleanup, and replacement of prior shadow state.

The repository gates passed before publication:
`make fidelity-depth`, `go build ./...`, `go vet ./...`, `go test ./...`,
`golangci-lint run ./...`, clean `gofumpt -l .`, clean `git diff --check`, and
clean oracle-tree diff. Feature PR #1161 (`glm/depth-shadow`) passed hosted
security, test, and lint checks and was self-merged to `main` as
`1bd5db55a`.

The frontier after this merge is:

- 3238 total cases
- 3156 proven/delegated
- 30 blocked
- 52 excluded

The one-time blocked-row vehicle remains satisfied and bounded:
`objmagic.sleep-entry-gates.cast` is green through the cast-sleep outlaw and
reagent arms in `sleep-spell-depth`; the separate object-magic entry row stays
blocked in `docs/fidelity/depth/object-magic.tsv` and must not be re-picked.

The next unclaimed source-order interpreter entry is `shame` at
`src/interpreter.c:688`, registered to `do_action` with POS_RESTING, level 0,
and no fighting restriction. Begin it only after a fresh `main` checkout/pull,
`make fidelity-depth`, depth-guide read, newest-handoff read, and source/table
audit. Fidelity basis remains R1/R2/R3/R4/R5e, with shared ownership bounded by
R5b/R5c.
