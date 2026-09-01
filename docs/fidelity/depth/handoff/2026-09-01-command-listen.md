# Depth-fidelity handoff — `listen` — 2026-09-01

## Queue position

This session started from clean `main` at the post-`levels` handoff frontier,
ran `git pull --ff-only`, confirmed `make fidelity-depth`, and reread
`docs/fidelity/DEPTH_TESTING.md` plus
`docs/fidelity/depth/handoff/2026-09-01-command-levels.md`.

The source-order sweep checked `list` at `src/interpreter.c:539` and found it
already claimed by `docs/fidelity/depth/do-not-here.tsv` as part of the
`list/buy/sell` shop-fallback row, so `list` was not repicked. The next
actually unmanifested command was `listen` at `src/interpreter.c:540`; the
next command after this slice is `lick` at `src/interpreter.c:541`.

The special-procedure inventory remains exhausted and the one blocked
`objmagic.sleep-entry-gates` row remains deferred; neither was repicked.

## C call path and branch inventory

The C registration is `{ "listen", POS_RESTING, do_action, 0, 0 }` in
`src/interpreter.c:540`. `comm.c:910-947` applies the POS_RESTING and zero
level gates before invoking `ACMD(do_action)` in `src/act.social.c:102-151`.
The record at `lib/misc/socials:1386-1388` is self-only: it contains
`You listen.`, `$n listens.`, and `#`, with no `char_found` message. C's
`do_action` therefore uses the no-argument actor/room pair for no input, a
visible target, a missing target, and a self-named target. No target lookup,
not-found line, victim-position gate, or target trio is reached for this
record. Shared noshout, visibility, and command-position behavior are owned
by existing manifests.

## Proof and implementation

Added `cmd/dp-oracle-diff/scenarios/listen-depth.txt` with an actor and room
observer. The vehicle covers no argument, typed visible target, missing
target, and self target; seed 1 used `--show-oracle` and reached the exact
actor and observer blocks for every case. Seeds 2, 3, 5, and 8 were also
oracle-green. No Go behavior change was confirmed. The focused test
`pkg/session/listen_depth_test.go` pins the C entry gate, zero social
metadata, and all three authored messages. No `src/` or
`darkpawns-c-oracle/` file was edited.

## Durable proof and verification

`docs/fidelity/depth/listen.tsv` contains eight rows: entry gate, delegated
position gate, no argument, typed-target ignored, missing-target ignored,
self-target ignored, delegated noshout, and delegated visibility. The
post-slice frontier is `2456 total, 2395 proven/delegated, 17 blocked, 44
excluded`; actionable completion is `2395/2412 = 99.3%`, and `do_action` is
`460/460`.

The feature branch passed all local gates:

- `make fidelity-depth`;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...` (`0 issues`);
- `gofumpt -l .` clean and `git diff --check` clean.

Feature PR #1003 (`glm/depth-listen-20260901`) merged with hosted `test`,
`security`, and `lint` green in run `33469327483` at main commit
`764e9e496`; optional build/deploy jobs were skipped by workflow policy. No
merge was performed while a required check was pending.

This note is the required dated handoff for the session. Continue from clean
`main`, recheck the frontier and newest handoff, and take `lick` next. The
slice follows R1/R2/R3/R4/R5e; shared social behavior is bounded under
R5b/R5c.
