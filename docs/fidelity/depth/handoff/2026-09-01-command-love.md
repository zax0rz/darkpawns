# Depth-fidelity handoff — `love` — 2026-09-01

## Queue position

This round started from clean `main` at the post-`load` handoff frontier, ran
`git pull --ff-only`, confirmed `make fidelity-depth`, and reread
`docs/fidelity/DEPTH_TESTING.md`, the standing brief, and the newest
`2026-09-01-command-load.md` handoff. The special-procedure inventory remains
exhausted and `objmagic.sleep-entry-gates` remains the one previously attempted
blocked row; neither was repicked.

The interpreter sweep selected `love` at `src/interpreter.c:545`, registered as
`{ "love", POS_RESTING, do_action, 0, 0 }`. The pure-coverage slice shipped in
PR #1012 and was squash-merged to main as `e900c81f4` after all hosted checks
passed.

## C path and proof

The C path is `src/act.social.c:102-151`, using the `love` record in
`lib/misc/socials:470-478`. The audit covered `find_action`, the shared
`PLR_NOSHOUT` refusal, `one_argument` target parsing, no-argument actor/room
messages, visible target lookup, self-target handling, minimum victim position,
and the TO_CHAR/TO_NOTVICT/TO_VICT audience split. The Go social table already
matched the eight C-authored messages and the shared `do_action` implementation
already matched the behavior, so no Go behavior change was required.

The fresh-main ordinary vehicle was GREEN with `--show-oracle`, proving the
no-argument, target-success, separate observer/target audiences, fill-word and
trailing-input parsing, self-target, and missing-target branches. The separate
sleeping-target vehicle was also GREEN and proved the zero victim-position
minimum plus the TO_SLEEP actor delivery. Both vehicles matched at seeds
`1,2,3,5,8`; the social has no RNG branch, but the required multiseed matrix
guards the live call path. Focused `TestLoveRegistrationUsesCEntryGate` pins
the command gate and all eight authored message strings. No `src/` or
`darkpawns-c-oracle/` file was edited.

## Gates and frontier

`make fidelity-depth` reports `2503 total, 2441 proven/delegated, 18 blocked,
44 excluded` and exits 0. Local `go build ./...`, `go vet ./...`,
`go test ./...`, `golangci-lint run ./...` (`0 issues`), and `gofumpt -l .`
all pass. Hosted PR #1012 lint, security, and test checks all passed; build and
deploy were skipped as expected for a PR.

The next unmanifested interpreter-table family is `luaedit` at
`src/interpreter.c:546` (`do_luaedit`, POS_DEAD, LVL_BUILDER with LVL_HIGOD
hide metadata). Continue in source table order. Preserve the existing blocked
special-procedure and sleep-entry rows, and apply R1/R2/R3/R4/R5e plus R5b/R5c
for shared behavior ownership.
