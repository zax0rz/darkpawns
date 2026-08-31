# Depth handoff — 2026-08-30 — `enter`

## Frontier and queue position

- Started from clean `main` at `9340a3f15` after the merged `embrace` slice,
  ran `git pull --ff-only`, confirmed `make fidelity-depth`, and read
  `docs/fidelity/DEPTH_TESTING.md` plus the newest handoff,
  `2026-08-30-command-embrace.md`.
- The frontier before this slice was 1,627 total, with 1,572
  proven/delegated, 14 blocked, and 41 excluded. This slice adds six
  proven cases; the post-slice frontier is 1,633 total, 1,578
  proven/delegated, 14 blocked, and 41 excluded, with actionable completion
  1,578/1,592 (99.1%).
- The source-order command gap was `enter`, registered at
  `src/interpreter.c:432`. The next un-manifested command-table family is
  the next gap after `enter`; the next session must rescan the table and
  manifests from clean `main` before taking it.

## C call path and branch inventory

`src/interpreter.c:432` registers `enter` with `POS_STANDING` and
`do_enter`. The handler is `src/act.movement.c:642-674`. It calls C
`one_argument`, resolves a matching exit keyword in direction order, and
otherwise emits `There is no %s here.`. With no argument it first rejects an
already-indoor actor, then selects the first open exit whose destination has
`ROOM_INDOORS`, or emits `You can't seem to find anything to enter.`. The
dispatcher position gate runs before the handler.

The Go RED on clean `main` was the confirmed parser divergence: Go's
`firstWord` treated `the` as the exit keyword for `enter the fountain extra`,
while C's `one_argument` skipped the fill word and entered the named fountain
exit. The unrelated `goto` transport mismatch discovered in an exploratory
vehicle was removed from the proving probes and was not changed.

## Coverage proof and result

The C-first vehicles cover named-door resolution with leading fill words,
already-indoor refusal, automatic entry through the first open indoor exit,
the no-qualifying-exit refusal, and the registered standing-position gate.
The automatic vehicle explicitly clears `ROOM_INDOORS` on origin room 8162;
the base world marks both disposable rooms indoors, so this fixture is needed
to reach the C automatic branch. The named and automatic vehicles include
origin/destination peers to prove movement audience bytes. Seed 1 was run with
`--show-oracle` for the named, automatic, and empty vehicles; all three
vehicles were GREEN for seeds 1, 2, 3, 5, and 8 after the fix.

The only production change is `pkg/game/movement_commands.go`: `DoEnter` now
uses the existing C-compatible `oneArgument` parser. No `src/` or C oracle
files were edited.

## Gates

On `glm/depth-enter`:

- `make fidelity-depth` — PASS
- `go build ./...` — PASS
- `go vet ./...` — PASS
- `go test ./...` — PASS
- `golangci-lint run ./...` — PASS, 0 issues
- `gofumpt -l .` — clean
- `git diff --check` — clean

The required hosted PR checks must also be green before self-merging; if CI
does not fire, retry it once with the prescribed workflow dispatch and leave
the PR open if it remains non-green.

This slice follows R1/R2/R4 and R5e: C bytes and command registration remain
authoritative, the actual movement call path was traced before the fix, no
behavior was invented, and the unrelated transport divergence was kept out of
scope.
