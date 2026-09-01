# Depth-fidelity handoff — `load` — 2026-09-01

## Queue position

This session started from clean `main` at the post-`lock` handoff frontier,
ran `git pull --ff-only`, confirmed `make fidelity-depth`, and reread
`docs/fidelity/DEPTH_TESTING.md`, the standing brief, and the newest
`2026-09-01-command-lock.md` handoff. The special-procedure inventory remains
exhausted and the single blocked `objmagic.sleep-entry-gates` row remains
deferred; neither was repicked.

The interpreter sweep selected `load` at `src/interpreter.c:544`, registered as
`{ "load", POS_DEAD, do_load, LVL_IMMORT, 0 }`. The slice shipped in PR #1010
and was squash-merged to main as `c703b229f` after all hosted checks passed.

## C path and proof

The C implementation is `src/act.wizard.c:1224-1380`. The audit followed
`argument_interpreter`/`one_argument`, source-order mob/object prototype name
lookup, the first-byte `isdigit` gate and `atoi` prefix behavior, numeric
VNUM resolution, object limit/cost gates, `read_mobile`/`read_object`, quiet
placement, and the single `random_load_msg` `number(0,2)` draw. The Go changes
are limited to `pkg/session/wiz_object.go`, the quiet-spawn wrapper in
`pkg/game/world.go`, focused tests, and the two oracle vehicles; no `src/` or
`darkpawns-c-oracle/` file was edited.

The unchanged main vehicle was RED on C's name/abbreviation/parser branches
and on object success narration/state. The patched branch is GREEN for
`load-depth` and `load-depth-gates` at seeds `1,2,3,5,8`, with `--show-oracle`
run on both vehicles. The object vehicle proves actor, room peer, quiet spawn,
inventory placement, exact narration, and one shared RNG draw. The level-34
vehicle proves VNUMs 81/82/8095 and the cost cap over 6000.

The successful mob branch is explicitly blocked in
`docs/fidelity/depth/load.tsv`: three honest numeric vehicles (2100, 2109, 1)
and a named `puff` attempt produced empty C actor/room blocks even though the
source branch contains `char_to_room`, `random_load_msg`, and both creation
acts. No player-facing Go bytes were invented. This is an R1/R4/R5e proof gap,
not an exclusion.

## Gates and frontier

`make fidelity-depth` reports `2492 total, 2430 proven/delegated, 18 blocked,
44 excluded` and exits 0. Local `go build ./...`, `go vet ./...`,
`go test ./...`, `golangci-lint run ./...` (`0 issues`), and `gofumpt -l .`
all pass. Hosted PR #1010 lint, security, and test checks all passed; build and
deploy were skipped as expected for a PR.

The next unmanifested interpreter-table family is `love` at
`src/interpreter.c:545` (`do_action`, `POS_RESTING`, no minimum level), after
the shared `lock` audit and the now-claimed `load` slice. Continue in source
table order. Preserve the existing blocked special-procedure and sleep-entry
rows, and apply R1/R2/R3/R4/R5e plus R5b/R5c for shared behavior ownership.
