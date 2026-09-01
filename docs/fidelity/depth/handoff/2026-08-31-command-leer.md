# Depth-fidelity handoff — `leer` — 2026-08-31

## Queue position

This session started from clean `main` at the post-`leave` handoff frontier,
ran `git pull --ff-only`, confirmed `make fidelity-depth`, and reread
`docs/fidelity/DEPTH_TESTING.md` plus
`docs/fidelity/depth/handoff/2026-08-31-command-leave.md`. The next
unproven source-table command was `leer` at `src/interpreter.c:537`; the
next command after this slice is `levels` at `src/interpreter.c:538`.

The special-procedure inventory remains exhausted and the one blocked
`objmagic.sleep-entry-gates` row remains deferred; neither was repicked.

## C call path and branch inventory

The C registration is `{ "leer", POS_RESTING, do_action, 0, 0 }` in
`src/interpreter.c:537`. `comm.c:910-947` applies the command position and
level gates before invoking `ACMD(do_action)` in `src/act.social.c:102-151`.
The social record at `lib/misc/socials:450-458` has `hide=0` and
`min_victim_position=0`, with these reachable authored branches:

1. no argument: actor `Keep those nasty thoughts to yourself!` and room
   `$n leers at no one in particular.`;
2. visible target: actor, non-victim room, and victim messages with the
   `$M`/`$N` substitutions;
3. self target: actor `Eh?  Why would you want to do that?` and no room
   message because the authored `others_auto` record is `#`;
4. missing target: `Smirk all you like.`;
5. a sleeping visible target still enters the target branch because the
   authored victim-position minimum is zero.

`do_action` calls `one_argument`, so a leading fill word selects the first
actual target token and trailing words are ignored. Shared PLR_NOSHOUT,
visibility, and POS_RESTING dispatcher behavior are owned by existing
manifests rather than duplicated here.

## Proof and implementation

Added `cmd/dp-oracle-diff/scenarios/leer-depth.txt` with an actor and peer,
covering no argument, visible target and target audience, leading fill-word
plus ignored trailing input, self target, and missing target. Added
`cmd/dp-oracle-diff/scenarios/leer-sleeping-depth.txt` with a sleeping peer
to prove the zero victim-position minimum. Both seed-1 runs used
`--show-oracle` and reached the intended C blocks; the actor, peer, and
sleeping-peer transcripts were byte-identical after normalization. The two
vehicles also ran green at seeds 2, 3, 5, and 8.

No Go behavior change was confirmed. `pkg/session/leer_depth_test.go` pins
the C entry gate, social metadata, and all eight authored messages. The
shared Go `DoAction` implementation remains bounded by the existing social
matrix. No `src/` or `darkpawns-c-oracle/` file was edited.

## Durable proof and verification

`docs/fidelity/depth/leer.tsv` contains eleven rows: entry gate, delegated
position gate, no argument, target success, target audience, first-token
parsing, self target, not-found, sleeping target, delegated noshout, and
delegated visibility. The post-slice frontier is `2442 total, 2381
proven/delegated, 17 blocked, 44 excluded`; actionable completion is
`2381/2398 = 99.3%`, and `do_action` is `452/452`.

The feature branch passed all local gates:

- `make fidelity-depth`;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...` (`0 issues`);
- `gofumpt -l .` clean and `git diff --check` clean.

Feature PR #999 (`glm/depth-leer-20260831`) merged with hosted `test`,
`security`, and `lint` green in run `33467676274` at main commit
`5cdb46798`; optional build/deploy jobs were skipped by workflow policy. No
merge was performed while a required check was pending.

This note is the required dated handoff for the session. Continue from clean
`main`, recheck the frontier and newest handoff, and take `levels` next. The
slice follows R1/R2/R3/R4/R5e; shared social behavior is bounded under
R5b/R5c.
