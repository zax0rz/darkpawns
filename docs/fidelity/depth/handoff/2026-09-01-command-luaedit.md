# Depth-fidelity handoff — `luaedit` — 2026-09-01

## Queue position

This round started from clean `main` at the post-`love` handoff frontier, ran
`git pull --ff-only`, confirmed `make fidelity-depth`, and reread
`docs/fidelity/DEPTH_TESTING.md`, the standing brief, and the newest
`2026-09-01-command-love.md` handoff. The special-procedure inventory remains
exhausted and `objmagic.sleep-entry-gates` remains the single previously
attempted blocked row; neither was repicked.

The interpreter sweep selected `luaedit` at `src/interpreter.c:546`, registered
as `{ "luaedit", POS_DEAD, do_luaedit, LVL_BUILDER, LVL_HIGOD }`. The scope
slice shipped in PR #1014 and was squash-merged to main as `a2e84aa37` after
all hosted checks passed.

## C path and scope decision

The C handler in `src/luaedit.c:28-58` parses two arguments, selects the script
root or `mob`/`obj`/`room` subdirectory, lists matching Lua files when no file
is supplied, and otherwise calls `view_file` below `subcmd` or `edit_file` at
or above `subcmd`, sending `Invalid lua edit option.\r\n` on helper failure.

The repository's explicit port policy says the C OLC/builder toolchain,
including `luaedit`, is intentionally not ported: world editing is owned by the
mounted web admin and the Go server has no in-game OLC command surface. The
queue item is therefore recorded as `luaedit.superseded-by-web-admin` with
status `excluded` in `docs/fidelity/depth/luaedit.tsv`; no substitute player
bytes or Go handler were invented. This is a scope boundary under R4/R5e, not
a claim that the legacy C command was unreachable in its original server.

## Gates and frontier

`make fidelity-depth` reports `2504 total, 2441 proven/delegated, 18 blocked,
45 excluded` and exits 0. Local `go build ./...`, `go vet ./...`,
`go test ./...`, `golangci-lint run ./...` (`0 issues`), and `gofumpt -l .`
all pass. Hosted PR #1014 security, lint, and test checks all passed; build and
deploy were skipped as expected for a PR.

The next unmanifested interpreter-table family is `map` at
`src/interpreter.c:548` (`do_map`, POS_SLEEPING, no minimum level). Continue in
source table order. Preserve the existing blocked special-procedure and
sleep-entry rows, and apply R1/R2/R3/R4/R5e plus R5b/R5c for shared behavior
ownership.
