# Mobile command-call lane — 2026-10-25

This handoff records the direct mobile-command audit from `origin/main` at
`4fc5c8ab2` on `codex/mob-command-lane`. The work is in one draft PR for
Claude's batch review; it has not been merged or deployed.

## Inventory result

| Measure | Before | Draft result |
|---|---:|---:|
| Consolidated direct-call rows | 0 | 173 |
| Fixed | 0 | 127 |
| Delegated | 0 | 6 |
| Unreachable | 0 | 28 |
| Blocked | 0 | 12 |

The map records each C file and line, caller, command, exact C-built argument,
Go route, status, proof and notes in
`docs/fidelity/depth/surface/mob-command-calls.tsv`. The 28 unreachable sites
were checked against their C guards or `src/spec_assign.c`; the mayor,
mickey, mallory, conductor, fearface, tipster and evillead specials have no
mob assignment. `kender_steal` has no caller in the audited C sources. The two
mobile-activity wake/stand sites are behind the asleep guard.

The 12 blocked rows are the 11 door-command calls held for GLM's follow-up
after #2146 and the `breed_killer` spike/stake call: the shared Go command
currently has no mobile `SCMD_SPIKE`/`SCMD_STAKE` body. No door helper or
`src/` file was changed.

## Port and C call paths

The shared entry is `World.NPCCommandDirect`: it preserves the raw argument,
dispatches the named body without the interpreter prelude or gates, and then
uses the mobile-native fallback when the session registry has no actor body.
The session route reaches `dispatchCharacterCommandDirect` with the same
actor and argument. The call sites now use this path for communication,
movement/follow, equipment, combat skills, Lua actions, and room/mobile
specials; sites already reaching the same command body are marked delegated.

The seed-8 fighter investigation found a missing C refusal at
`src/new_cmds.c:406-415`: `do_headbutt` checks an immortal player before its
percent roll, sits the mobile and sends the victim-only line. Go now models
that gate, and `TestSpecFighterHeadbuttRefusesImmortalDirectTarget` proves the
direct command, exact line, posture and no-draw return. The live seed-8
`spec-proc-fighter` transcript still differs: C omits that line while Go emits
it, even though draw parity matches. The actor/target state behind this
scenario difference remains unresolved; the map records the red run rather
than claiming a five-seed byte proof for that case.

The complete C→Go call and proof inventory is the TSV map. It covers at least
`src/spec_procs.c`, `src/spec_procs2.c`, `src/spec_procs3.c`, `src/mobact.c`,
`src/graph.c`, `src/fight.c` and `src/scripts.c`, with test names as revert
controls for the fixed rows.

## Retained oracle evidence

The five final batches each cover the same 34 targeted scenarios with
`--draw-parity`:

| Seed | Census | Verdict | Draw divergence |
|---:|---|---|---:|
| 1 | `luna-mob-command-lane-seed1-final` | CLEAN | 0 |
| 2 | `luna-mob-command-lane-seed2-final` | CLEAN | 0 |
| 3 | `luna-mob-command-lane-seed3-final` | CLEAN_AFTER_RECHECK | 0 |
| 5 | `luna-mob-command-lane-seed5-final` | CLEAN | 0 |
| 8 | `luna-mob-command-lane-seed8-final` | NOT_CLEAN: `spec-proc-fighter` bytes | 0 |

The seed-8 failure was reproduced twice in
`luna-mob-command-lane-seed8-fighter-diagnostic`; retained attempt logs are
under `luna-mob-command-lane-seed8-fighter-diag-logs/`. Its diff is only the
extra Go immortal-refusal line. The work therefore meets the requested
zero-draw-divergence condition but does not claim a clean five-seed byte
batch.

The initial 35-scenario seed-1 run `luna-mob-command-lane-seed1` included
`spec-proc-key-seller`, whose setup has an existing draw mismatch (C
`Number(0,18)`, Go `Number(1,6)`). Archived runs before this work show the same
divergence. That case is excluded from the five-seed batch and recorded in the
key-seller rows' notes; its direct tell argument is covered by
`TestTellFromMobUsesDirectTellArgument`.

All census artifacts are retained under
`/home/zach/Archives/darkpawns/oracle-runs/2026-10-25/`.

## Checks

Passed individually after the code change: `make fmt`, `go build ./...`,
`go vet ./...`, `go test ./...`, `go test ./pkg/game/...`,
`golangci-lint run ./...` (0 issues), `make deadcode-ratchet` (52 baselined,
0 new), `make fidelity-units` (1,949 claims), `make string-census` (ratchet
passes; generated reports are already stale), `gosec -severity=high -quiet
./...`, and the proof-token check (no missing scenarios).

`make fidelity-depth` fails on the new requested inventory map because its
generic surface-map validator requires the unrelated header
`c_site,function,classification,case_ids,notes`, while the goal requires
columns for C file/line, caller, command, argument, Go site, status, proof and
notes. The validator reports `43 map(s), 1 error(s)`. Tooling is out of scope
for this goal, so the map was kept in the specified format and the gate result
is recorded here and in the PR.

## Delivery

Elapsed time at handoff: approximately 1h 16m. PR count: 1 draft. The draft
remains open for review; no merge or deploy was performed.

## Draft PR

See the attached draft PR for the grouped C call paths, per-group revert
controls and **Decisions for Zach**.
