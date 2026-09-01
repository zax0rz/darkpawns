# Depth-fidelity handoff: `lambada`

Date: 2026-08-31

## Queue position and result

This session started from clean main, pulled `origin/main`, ran
`make fidelity-depth`, reread `docs/fidelity/DEPTH_TESTING.md`, and read the
newest prior handoff, `2026-08-31-command-laugh.md`. The special-procedure
inventory remains exhausted and the one blocked `objmagic.sleep-entry-gates`
row remains deferred; neither was repicked.

The fresh source/manifest sweep consumed the next actually unmanifested
interpreter-table command after `laugh`: `lambada`, registered at
`src/interpreter.c:534` as `do_action` at `POS_STANDING` with no command-level
restriction. The generic social family was already covered, but this
per-command slice adds the complete `lambada` record and direct branch proof.

The pre-slice frontier was 2,407 total cases, with 2,347 proven/delegated,
16 blocked, and 44 excluded. The 10 Lambada rows bring the frontier to 2,417
total, with 2,357 proven/delegated, 16 blocked, and 44 excluded: 2,357/2,373
actionable cases, or 99.3%. The `do_action` family is now 441/441.

Feature PR #993 (`test: prove lambada depth fidelity`) passed hosted `test`,
`lint`, and `security` checks; build-and-push and deploy were skipped by the
PR workflow. Its checks required the one permitted workflow retry because no
checks initially appeared. It merged to main as `1297de429` (feature commit
`736470b11`). No merge was performed while a required check was pending.

The next truly unmanifested command in source registration order is `last` at
`src/interpreter.c:535`.

## C call path and reachable branches

The authoritative path is `src/interpreter.c:534` →
`src/act.social.c:102-151` (`ACMD(do_action)`) →
`src/comm.c:2392-2555` (`perform_act` and audience delivery), with the shared
`src/comm.c:910-947` command-position gate before dispatch. The handler loads
the `lambada` record from `lib/misc/socials:1486-1494`, checks the actor's
`PLR_NOSHOUT` bit before argument parsing, and uses C `one_argument` because
the record has a `char_found` message.

The record's eight fields provide a forbidden-dance no-argument actor/room
pair, actor/room/victim target trio, `Really now?` not-found refusal, and a
self-target actor/room pair. Its minimum level, hide flag, and minimum victim
position are all zero. Reachable branches therefore include no argument,
first-token target parsing with trailing words ignored, visible cleaner-mob
target, visible player target, missing target, and self target. The vehicle
exposes the actor, room, victim, and self audiences and checks both `$N` and
`$M` substitutions.

## RED diagnosis and confirmed fixes

The main-equivalent vehicle was green at seed 1 with `--show-oracle`. It
reached the exact forbidden-dance no-argument actor/room pair, cleaner-mob
target with `$N/$M` substitutions, player victim audience, first-token parser,
self-target pair, and not-found refusal with no normalized divergence. The
same vehicle was green at seeds 2, 3, 5, and 8.

No Go behavior change was justified. The focused test confirms the C entry
gate, Go social registration, zero metadata, and all eight authored messages.
No `src/` or `darkpawns-c-oracle/` file was edited. This preserves R1/R2/R3/R4,
follows the actual C path under R5e, and delegates shared position, Noshout,
and visibility behavior under R5b/R5c.

## Durable proof

The manifest `docs/fidelity/depth/lambada.tsv` contains 10 rows covering the C
entry and shared position gates, exact no-argument bytes, mob room audience,
player victim audience, one-argument parsing, self target, not-found refusal,
shared Noshout refusal, and shared visibility behavior.

The durable vehicle is:

- `cmd/dp-oracle-diff/scenarios/lambada-depth.txt`.

Its six depth annotations cover the direct observable branches. It ran with
`DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle` at seeds 1, 2, 3, 5,
and 8 with no normalized divergence; seed 1 was also run with `--show-oracle`
and proved the intended C block executed.

The proof establishes R1 player-facing byte parity, R2 command-surface parity,
R3 deterministic parity for this non-random social, and R4 absence of
invented behavior. R5e verifies the actual C dispatch/callee path; shared
command gates, Noshout, and Act visibility remain bounded by R5b/R5c.

## Verification and continuation

The feature branch passed all local gates:

- `make fidelity-depth` (`do_action: 441/441`);
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...`;
- `gofumpt -l .` clean and `git diff --check`.

After merging, main is clean at `1297de429`. The next session must start from
clean main, pull, reconfirm the frontier, read the depth-testing guide and
this newest handoff, then map and prove `last` in table order. Continue to
leave one dated handoff per session.
