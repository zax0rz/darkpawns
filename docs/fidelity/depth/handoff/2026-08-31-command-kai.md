# Depth-fidelity handoff: `kai`

Date: 2026-08-31

## Queue position and result

This session started from clean main, pulled `origin/main`, ran
`make fidelity-depth`, reread `docs/fidelity/DEPTH_TESTING.md`, and read the
newest prior handoff, `2026-08-31-command-jin.md`. The special-procedure
inventory remains exhausted and the one blocked `objmagic.sleep-entry-gates`
row remains deferred; neither was repicked.

The fresh source/manifest sweep consumed the next actually unmanifested
interpreter-table command after `jin`: `kai`, registered at
`src/interpreter.c:527` as `do_kuji_kiri` at `POS_STANDING` with no
command-level minimum. `junk`, `kabuki`, and `kill` were already claimed by
earlier manifests and were not repicked.

The pre-slice frontier was 2,342 total cases, with 2,282 proven/delegated,
16 blocked, and 44 excluded. The 14 Kai rows bring the frontier to 2,356
total, with 2,296 proven/delegated, 16 blocked, and 44 excluded: 2,296/2,312
actionable cases, or 99.3%. The `do_kuji_kiri` family is now 26/26.

Feature PR #983 (`fix: prove kai kuji-kiri fidelity`) passed hosted `test`,
`lint`, and `security` checks; build-and-push and deploy were skipped by the
PR workflow. It merged to main as `996f723d7` (feature commit `e6d47c861`).
No merge was performed while a required check was pending, and no workflow
retry was needed.

The next truly unmanifested command in source registration order is `kick` at
`src/interpreter.c:528`.

## C call path and reachable branches

The authoritative path is `src/interpreter.c:527` →
`src/new_cmds.c:1500-1541` (`check_kk_success`) →
`src/new_cmds.c:1552-1739` (`ACMD(do_kuji_kiri)`)
→ `src/handler.c:473-499` (`affect_join`) and
`src/comm.c:2392-2555` (`act`). The registered C row admits a standing caller
at level 0; Kai then evaluates the shared non-Ninja, fighting, aggregate
`AFF_KUJI_KIRI`, and mounted gates, consumes the success draw before the
mastery branch, and reaches the Kai-specific success/failure state.

The learned success arm sends the exact actor text
`Interlacing your fingers, your body becomes your fortress.` and the room
receives `$n interlaces $s fingers and meditates deeply.`. Kai's two records
are distinct C locations: `APPLY_DAMROLL` -1 and `APPLY_AC` -10, both with
`AFF_KUJI_KIRI`, duration five, and spell type 161. The failure arm sends
`You try the art of kuji-kiri, but can't concentrate!`, keeps the same room
act, and joins a zeroed damroll lockout record plus a zeroed AC
`AFF_NOTHING` record, so the aggregate lockout remains set.

The C catalog labels all nine seals as `kuji-kiri <seal>`, while the command
handler reads the `SkillKk*` gameplay keys. The catalog-boundary mapping now
covers all nine seals, preserving the same skill slot for `skillset`,
practice, and character bootstrap (R1/R5e).

## RED diagnosis and confirmed fixes

The main-equivalent baseline vehicle was RED at seed 1: C's learned Kai
emitted the fortress and room lines, then its second use emitted the lockout;
Go reported the mastery refusal on both uses. The C-first trace isolated the
catalog skill-slot mismatch and the shared `$n/$s` room act path.

A second failure vehicle at seed 1 confirmed the C failure topology. The Go
port differed until the shared kuji implementation used `JoinAffect` and
always applied both C records, preserving Kai's separate locations and
failure lockout. This shared correction is retained under R5b/R5c after
verifying the actual `affect_join` call path under R5e.

No `src/` or `darkpawns-c-oracle/` file was edited. The changes introduce no
player-facing behavior beyond the confirmed C paths, preserving R1, R3, and
R4.

## Durable proof

The manifest `docs/fidelity/depth/kai.tsv` contains 14 rows covering the C
entry and position gates, delegated shared kuji gates, the class-wide catalog
slot mapping, success actor/room output, success lockout and two-record state,
failure actor/room output, failure lockout, and failure two-record state.

The durable vehicles are:

- `cmd/dp-oracle-diff/scenarios/kai-depth.txt`;
- `cmd/dp-oracle-diff/scenarios/kai-failure-depth.txt`.

Both scenarios ran with `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle`.
`kai-depth` matched at seeds 1, 2, 3, 5, and 8. `kai-failure-depth` matched at
seeds 1, 2, 3, 5, and 8; seed 1 is the manifest vehicle that explicitly
reaches the failure arm before retrying. Focused tests pin the C gate, all
nine catalog keys, Kai's joined success records, and Kai's failed lockout
records.

The proof establishes R1 player-facing byte parity, R2 command-surface parity,
R3 deterministic draw/state parity, R4 absence of invented behavior, and R5e
verification of the actual C dispatch/callee path. Shared gate and affect
ownership are delegated or corrected under R5b/R5c.

## Verification and continuation

The feature branch passed all local gates:

- `make fidelity-depth`;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...`;
- `test -z "$(gofumpt -l .)"` and `git diff --check`.

After merging, main is clean at `996f723d7`. The next session must start from
clean main, pull, reconfirm the frontier, read the depth-testing guide and
this newest handoff, then map and prove `kick` in table order. Continue to
leave one dated handoff per session.
