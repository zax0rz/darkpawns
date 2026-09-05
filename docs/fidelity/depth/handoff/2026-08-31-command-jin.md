# Depth-fidelity handoff: `jin`

Date: 2026-08-31

## Queue position and result

This session started from clean main, pulled `origin/main`, ran
`make fidelity-depth`, reread `docs/fidelity/DEPTH_TESTING.md`, and read the
newest prior handoff, `2026-08-31-command-jeer.md`. The special-procedure
inventory remains exhausted and the one blocked `objmagic.sleep-entry-gates`
row remains deferred; neither was repicked.

The fresh source/manifest sweep consumed the next actually unmanifested
interpreter-table command after `jeer`: `jin`, registered at
`src/interpreter.c:523` as `do_kuji_kiri` at `POS_STANDING` with no
command-level minimum. `junk`, `kabuki`, and `kill` were already claimed by
earlier manifests and were not repicked.

The pre-slice frontier was 2,330 total cases, with 2,270 proven/delegated,
16 blocked, and 44 excluded. The 12 Jin rows bring the frontier to 2,342
total, with 2,282 proven/delegated, 16 blocked, and 44 excluded: 2,282/2,298
actionable cases, or 99.3%.

Feature PR #981 (`fix: prove jin kuji-kiri fidelity`) passed hosted `test`,
`lint`, and `security` checks; build-and-push and deploy were skipped by the
PR workflow. It merged to main as `22704bcbb` (feature commit `526395833`).
No merge was performed while a required check was pending, and no workflow
retry was needed.

The next truly unmanifested command in source registration order is `kai` at
`src/interpreter.c:527`.

## C call path and reachable branches

The authoritative path is `src/interpreter.c:523` →
`src/new_cmds.c:1500-1541` (`check_kk_success`) →
`src/new_cmds.c:1552-1739` (`ACMD(do_kuji_kiri)`) →
`src/handler.c:473-499` (`affect_join`) and `src/comm.c:2392-2555`
(`act`). The registered C row admits a standing caller at level 0; the
procedure itself then evaluates these gates in order:

- non-Ninja mortals receive `You know nothing of kuji-kiri!`;
- fighters receive `You are too busy fighting to practice kuji-kiri!`;
- callers with the aggregate `AFF_KUJI_KIRI` flag receive
  `You can not practice kuji-kiri again right now!`;
- mounted callers receive `Dismount first!`;
- `check_kk_success()` consumes `number(1, 101)` before the Jin mastery
  branch, even when the skill is unlearned;
- an unmastered Jin receives `You have not mastered this art yet!`;
- a learned successful Jin receives `Interlacing your fingers, you focus on
  recooperation.` and the room receives the meditation act;
- a learned failed Jin receives `You try the art of kuji-kiri, but can't
  concentrate!` and still emits the meditation room act.

The C function initializes both default records, then sends both through
`affect_join`. For Jin both records are `type 162` at `APPLY_SPELL`: the
successful later record leaves one `AFF_KUJI_KIRI` effect, while the failed
later `AFF_NOTHING` record replaces the first and leaves the aggregate
lockout flag clear, allowing a retry. This is why the Go implementation now
uses `JoinAffect` and preserves the inert failure record. The room string is
resolved through the C-style actor pronouns before the shared literal-result
sender, preserving the peer bytes.

## RED diagnosis and confirmed fixes

The main-equivalent baseline vehicle was RED at seed 1: C's first Jin emitted
the success and room lines, then its second use emitted the lockout; Go
reported the mastery refusal both times. The C-first trace isolated the skill
slot mismatch (`spells[162]` is displayed as `kuji-kiri jin`, while the command
reads the `SkillKkJin` key), the pre-mastery RNG ordering, and the unresolved
`$n/$s` room act.

A second RED vehicle at seed 1 isolated the state branch: after a forced
skill-1 failure, C allowed a retry while Go rejected it as locked out. Reading
the actual `affect_join` call path showed that C's later default
`AFF_NOTHING/APPLY_SPELL` record replaces the failed primary record. The Go
fix therefore also corrected the shared kuji record application from
independent `AddAffect` calls to C-equivalent `JoinAffect` calls, without
inventing a failure lockout.

No `src/` or `darkpawns-c-oracle/` file was edited. The actual C call path,
including the initially non-obvious `affect_join` replacement behavior, was
verified under R5e; the shared kuji change is retained under R5b/R5c.

## Durable proof

The manifest `docs/fidelity/depth/jin.tsv` contains 12 rows covering the C
entry and position gates, class/fighting/mounted/lockout/mastery branches,
catalog skill-slot mapping, success actor/room output, joined success state,
failure actor/room output, and joined inert failure state.

The durable vehicles are:

- `cmd/dp-oracle-diff/scenarios/jin-depth.txt`;
- `cmd/dp-oracle-diff/scenarios/jin-failure-depth.txt`.

`jin-depth` ran at seeds 1, 2, 3, 5, and 8 with no normalized divergence.
`jin-failure-depth` ran at seed 1 with no normalized divergence; it reaches
the failure arm, then changes the skill and proves the retry success and room
audience. Focused tests pin the C gate, catalog key, pre-mastery draw, joined
success record, all Jin entry guards, and joined inert failure record.

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

After merging, main is clean at `22704bcbb`. The next session must start from
clean main, pull, reconfirm the frontier, read the depth-testing guide and
this newest handoff, then map and prove `kai` in table order. Continue to
leave one dated handoff per session.
