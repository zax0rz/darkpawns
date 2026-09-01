# Depth-fidelity handoff: `kiss`

Date: 2026-08-31

## Queue position and result

This session started from clean main, pulled `origin/main`, ran
`make fidelity-depth`, reread `docs/fidelity/DEPTH_TESTING.md`, and read the
newest prior handoff, `2026-08-31-command-kick.md`. The special-procedure
inventory remains exhausted and the one blocked `objmagic.sleep-entry-gates`
row remains deferred; neither was repicked.

The fresh source/manifest sweep consumed the next actually unmanifested
interpreter-table command after `kick`: `kiss`, registered at
`src/interpreter.c:529` as `do_action` at `POS_RESTING` with no command-level
restriction. The `kiss` social record was already present in Go, so this
slice required evidence and focused registration assertions but no behavior
change.

The pre-slice frontier was 2,373 total cases, with 2,313 proven/delegated,
16 blocked, and 44 excluded. The 10 Kiss rows bring the frontier to 2,383
total, with 2,323 proven/delegated, 16 blocked, and 44 excluded: 2,323/2,339
actionable cases, or 99.3%. The `do_action` family is now 421/421.

Feature PR #987 (`test: prove kiss depth fidelity`) passed hosted `test`,
`lint`, and `security` checks; build-and-push and deploy were skipped by the
PR workflow. It merged to main as `c1b78f52f` (feature commit `7975f5aa3`).
No merge was performed while a required check was pending, and no workflow
retry was needed.

The next truly unmanifested command in source registration order is `kyo` at
`src/interpreter.c:530`.

## C call path and reachable branches

The authoritative path is `src/interpreter.c:529` →
`src/act.social.c:102-151` (`ACMD(do_action)`) →
`src/comm.c:2392-2555` (`perform_act` and audience delivery), with the shared
`src/comm.c:910-947` command-position gate before dispatch. The handler loads
the `kiss` record from `lib/misc/socials:430-438`, checks the actor's
`PLR_NOSHOUT` bit before argument parsing, and calls `one_argument` only when
the record has a `char_found` message.

The record's eight fields are: no-argument actor message, empty room branch,
actor-on-target message, room-on-target message, victim message, not-found
message, self-target message, and an empty self-target room branch. Its
`MinLevel`, hide flag, and minimum victim position are all zero. Reachable
branches therefore include no argument, first-token target parsing with
trailing words ignored, visible mob target, visible player target, missing
target, and self target. A standing peer and cleaner mob expose actor, room,
victim, and self audiences without relying on an invented branch.

## RED diagnosis and confirmed fixes

The main-equivalent vehicle was green at seed 1. It exercised the no-argument,
cleaner-mob target, player target, first-token/trailing-word parser, self, and
not-found branches and found no normalized divergence. The same vehicle was
green at seeds 2, 3, 5, and 8.

No Go behavior change was justified. The focused test confirms the C entry
gate, Go registry/social presence, zero metadata, and all eight authored
messages. No `src/` or `darkpawns-c-oracle/` file was edited. This preserves
R1/R2/R3/R4, follows the actual C dispatch path under R5e, and keeps shared
Noshout/position/visibility behavior delegated under R5b/R5c.

## Durable proof

The manifest `docs/fidelity/depth/kiss.tsv` contains 10 rows covering the C
entry and shared position gates, exact no-argument bytes, mob room audience,
player victim audience, one-argument parsing, self target, not-found refusal,
shared Noshout refusal, and shared visibility behavior.

The durable vehicle is:

- `cmd/dp-oracle-diff/scenarios/kiss-depth.txt`.

Its six depth annotations cover the direct observable branches. It ran with
`DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle` at seeds 1, 2, 3, 5,
and 8 with no normalized divergence; seed 1 was also run with `--show-oracle`
and proved the intended C block executed.

The proof establishes R1 player-facing byte parity, R2 command-surface parity,
R3 deterministic parity for this non-random social, and R4 absence of
invented behavior. R5e verifies the actual C call path; shared command gates,
Noshout, and Act visibility remain bounded by R5b/R5c.

## Verification and continuation

The feature branch passed all local gates:

- `make fidelity-depth` (`do_action: 421/421`);
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...`;
- `gofumpt -l .` clean and `git diff --check`.

After merging, main is clean at `c1b78f52f`. The next session must start from
clean main, pull, reconfirm the frontier, read the depth-testing guide and
this newest handoff, then map and prove `kyo` in table order. Continue to
leave one dated handoff per session.
