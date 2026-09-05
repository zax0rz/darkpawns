# Depth-fidelity handoff: `kick`

Date: 2026-08-31

## Queue position and result

This session started from clean main, pulled `origin/main`, ran
`make fidelity-depth`, reread `docs/fidelity/DEPTH_TESTING.md`, and read the
newest prior handoff, `2026-08-31-command-kai.md`. The special-procedure
inventory remains exhausted and the one blocked `objmagic.sleep-entry-gates`
row remains deferred; neither was repicked.

The fresh source/manifest sweep consumed the next actually unmanifested
interpreter-table command after `kai`: `kick`, registered at
`src/interpreter.c:528` as `do_kick` at `POS_FIGHTING` with a level-1 command
gate. `kill` and `kabuki` were already claimed by earlier manifests and were
not repicked.

The pre-slice frontier was 2,356 total cases, with 2,296 proven/delegated,
16 blocked, and 44 excluded. The 17 Kick rows bring the frontier to 2,373
total, with 2,313 proven/delegated, 16 blocked, and 44 excluded: 2,313/2,329
actionable cases, or 99.3%. The `do_kick` family is now 17/17.

Feature PR #985 (`fix: prove kick depth fidelity`) passed hosted `test`,
`lint`, and `security` checks; build-and-push and deploy were skipped by the
PR workflow. It merged to main as `3f913352c` (feature commit `2902410bc`).
No merge was performed while a required check was pending, and no workflow
retry was needed.

The next truly unmanifested command in source registration order is `kiss` at
`src/interpreter.c:529`.

## C call path and reachable branches

The authoritative path is `src/interpreter.c:528` →
`src/act.offensive.c:587-634` (`ACMD(do_kick)`) →
`src/fight.c:1023-1092` (`skill_message` through `damage`) and the shared
`src/comm.c:910-947` command-position gate. The handler first checks
`GET_SKILL(ch, SKILL_KICK)`, parses one target token with `one_argument`, and
resolves a visible room target. With no target token it uses the direct
`FIGHTING(ch)` pointer; otherwise it emits `Kick who?`. It then rejects self
targets and mounted callers before consuming the kick roll.

The roll is `((7 - (GET_AC(vict) / 10)) << 1) + number(1, 101)` against
`GET_SKILL(ch, SKILL_KICK)`. A miss calls `damage(ch, vict, 0, SKILL_KICK)`;
a hit calls `damage(ch, vict, GET_LEVEL(ch) >> 1, SKILL_KICK)` and then
`improve_skill`. Both arms pass through the set-134 `skill_message` corpus,
enroll combat through `damage`, and apply `WAIT_STATE(ch,
PULSE_VIOLENCE + 2)`. The shared damage/message path owns the detailed actor,
victim, room, death, and retaliation matrix; Kick-specific evidence pins its
set selection, state contract, and draw ordering.

## RED diagnosis and confirmed fixes

The original main-equivalent opener vehicle was green at seed 1 for the
single-target miss arm. Adding the C branch probes made the first RED finding
sharp: `kick trainee trailing words` emitted C's set-134 miss immediately,
while Go treated the whole phrase as a target and returned `Kick who?`. The
fix applies Go's `game.OneArgument` before target lookup, including C's fill-word
skip and trailing-token ignore.

The second RED vehicle proved the empty-argument branch: after the padded
combat opener, C used its direct multi-word `FIGHTING(ch)` target and emitted
the kick message, while Go reparsed the display name through the keyword
resolver and returned `Kick who?`. The fix adds an exact in-room
`FindFightingTargetInRoom` lookup for the stored fighting name and uses it only
as the C-equivalent fallback after ordinary resolution fails.

No `src/` or `darkpawns-c-oracle/` file was edited. These changes fix only
confirmed divergences and preserve R1/R2/R3/R4; the target-resolution change
was checked as a behavior-class correction under R5c and the actual C pointer
path was verified under R5e.

## Durable proof

The manifest `docs/fidelity/depth/kick.tsv` contains 17 rows covering the C
entry and position gates, unknown-skill refusal, no-argument and missing-target
branches, direct fighting fallback, one-argument/trailing-token parsing,
self-target and mounted guards, miss and hit contracts, set-134 message source,
combat follow-up, damage/enrollment, draw order, and the unconditional wait.

The durable vehicle is:

- `cmd/dp-oracle-diff/scenarios/combat-kick-opener.txt`.

Its four depth annotations cover the opener miss, padded combat follow-up,
trailing-argument parser, and no-argument fighting-pointer fallback. It ran
with `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle` at seeds 1, 2, 3,
5, and 8 with no normalized divergence; seed 1 was also run with
`--show-oracle` after the fixes and proved every intended block executed.
Focused tests pin the registration gate, target branches, set-134 routing,
hit/miss contracts, exact message source, combat enrollment, C draw order, and
the three-pulse wait.

The proof establishes R1 player-facing byte parity, R2 command-surface parity,
R3 deterministic draw/state parity, R4 absence of invented behavior, and R5e
verification of the actual C dispatch/callee path. Shared command-position,
skill-message, and combat behavior is delegated or bounded under R5b/R5c.

## Verification and continuation

The feature branch passed all local gates:

- `make fidelity-depth` (`do_kick: 17/17`);
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...`;
- `test -z "$(gofumpt -l .)"` and `git diff --check`.

After merging, main is clean at `3f913352c`. The next session must start from
clean main, pull, reconfirm the frontier, read the depth-testing guide and
this newest handoff, then map and prove `kiss` in table order. Continue to
leave one dated handoff per session.
