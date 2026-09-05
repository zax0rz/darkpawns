# Depth-fidelity handoff: `kyo`

Date: 2026-08-31

## Queue position and result

This session started from clean main, pulled `origin/main`, ran
`make fidelity-depth`, reread `docs/fidelity/DEPTH_TESTING.md`, and read the
newest prior handoff, `2026-08-31-command-kiss.md`. The special-procedure
inventory remains exhausted and the one blocked `objmagic.sleep-entry-gates`
row remains deferred; neither was repicked.

The fresh source/manifest sweep consumed the next actually unmanifested
interpreter-table command after `kiss`: `kyo`, registered at
`src/interpreter.c:530` as `do_kuji_kiri` at `POS_STANDING` with no command
minimum level. The shared kuji gate class was already proven by Jin and Kai;
this slice adds Kyo's success/failure audience and affect-state evidence.

The pre-slice frontier was 2,383 total cases, with 2,323 proven/delegated,
16 blocked, and 44 excluded. The 14 Kyo rows bring the frontier to 2,397
total, with 2,337 proven/delegated, 16 blocked, and 44 excluded: 2,337/2,353
actionable cases, or 99.3%. The `do_kuji_kiri` family is now 40/40.

Feature PR #989 (`test: prove kyo depth fidelity`) passed hosted `test`,
`lint`, and `security` checks; build-and-push and deploy were skipped by the
PR workflow. Its checks required the one permitted workflow retry because no
checks initially appeared. It merged to main as `bc16395f2` (feature commit
`201e0942c`). No merge was performed while a required check was pending.

The next truly unmanifested command in source registration order is the next
row after `kyo` in the interpreter table; a fresh sweep is required at the
start of the next session.

## C call path and reachable branches

The authoritative path is `src/interpreter.c:530` →
`src/new_cmds.c:1552-1739` (`ACMD(do_kuji_kiri)`) →
`src/new_cmds.c:1500-1541` (`check_kk_success`) and
`src/handler.c:473-499` (`affect_join`), with the shared
`src/comm.c:910-947` command-position gate before dispatch and
`src/comm.c:2392-2555` audience delivery after the command returns.

The handler first rejects non-Ninja mortals, active fighters, an aggregate
`AFF_KUJI_KIRI` lockout, and mounted actors. It then consumes the
`number(1, 101)` success draw before the per-seal mastery check. Kyo's case
requires a nonzero `SKILL_KK_KYO`, sets `af[0]` to `APPLY_HITROLL` with
modifier +1, leaves `af[1]` as the default `APPLY_SPELL` lockout record, and
emits the actor battle-rage line plus the default meditation room act on
success. On failure it zeroes modifiers, changes `af[1]` to `AFF_NOTHING`,
keeps Kyo's primary hitroll record `AFF_KUJI_KIRI`, emits the generic failure
line plus unchanged room act, and leaves the aggregate lockout active.

## RED diagnosis and confirmed fixes

The main-equivalent success vehicle was green at seed 1 with `--show-oracle`:
it reached the Kyo success message, meditation room audience, and second-use
lockout. The failure vehicle was also green at seed 1 with `--show-oracle`:
skill 1 reached the generic failure and room act, and the retry reached the
lockout. The success vehicle was then green at seeds 2, 3, 5, and 8.

No Go behavior change was justified. Focused tests confirm the Kyo command
and registry gate, exact success/failure messages and room act, successful
`APPLY_HITROLL +1` plus default lockout records, and failed primary-lockout
plus inert-secondary records. No `src/` or `darkpawns-c-oracle/` file was
edited. This preserves R1/R2/R3/R4, follows the actual C path under R5e, and
delegates shared kuji gates under R5b/R5c.

## Durable proof

The manifest `docs/fidelity/depth/kyo.tsv` contains 14 rows covering the C
entry and shared position/class/fighting/lockout/mounted/mastery gates, the
class-wide skill-slot boundary, success actor and room bytes, success
lockout, success affects, failure actor and room bytes, failure lockout, and
failure affects.

The durable vehicles are:

- `cmd/dp-oracle-diff/scenarios/kyo-depth.txt`;
- `cmd/dp-oracle-diff/scenarios/kyo-failure-depth.txt`.

Their five depth annotations cover the direct success, success lockout,
skill-slot, failure, and failure lockout branches. `kyo-depth` ran with
`DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle` at seeds 1, 2, 3, 5,
and 8 with no normalized divergence; seed 1 was also run with `--show-oracle`.
`kyo-failure-depth` ran at seed 1 with `--show-oracle` and no normalized
divergence.

The proof establishes R1 player-facing byte parity, R2 command-surface parity,
R3 success-draw and affect-state parity, and R4 absence of invented behavior.
R5e verifies the actual C dispatch and callee path; shared command and kuji
gates remain bounded by R5b/R5c.

## Verification and continuation

The feature branch passed all local gates:

- `make fidelity-depth` (`do_kuji_kiri: 40/40`);
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...`;
- `gofumpt -l .` clean and `git diff --check`.

After merging, main is clean at `bc16395f2`. The next session must start from
clean main, pull, reconfirm the frontier, read the depth-testing guide and
this newest handoff, then identify and prove the next unmanifested command in
interpreter-table order. Continue to leave one dated handoff per session.
