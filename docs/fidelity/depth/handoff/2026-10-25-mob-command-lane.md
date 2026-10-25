# Mobile command-call lane — 2026-10-25

This handoff records the direct mobile-command audit from `origin/main` at
`4fc5c8ab2` on `codex/mob-command-lane`. The work is in one draft PR for
Claude's batch review; it has not been merged or deployed.

## Inventory result

| Measure | Before | Draft result |
|---|---:|---:|
| Consolidated direct-call rows | 0 | 173 |
| Fixed | 0 | 127 |
| Proven | 0 | 1 |
| Delegated | 0 | 10 |
| Unreachable | 0 | 28 |
| Blocked | 0 | 7 |

The map records each C file and line, caller, command, exact C-built argument,
Go route, status, proof and notes in
`docs/fidelity/depth/surface/mob-command-calls.tsv`. The 28 unreachable sites
were checked against their C guards or `src/spec_assign.c`; the mayor,
mickey, mallory, conductor, fearface, tipster and evillead specials have no
mob assignment. `kender_steal` has no caller in the audited C sources. The two
mobile-activity wake/stand sites are behind the asleep guard.

The door review cleared five of the eleven `do_gen_door` rows after #2146 and
#2147 merged: the hunter call is proven through `mobGenDoor`, and the four mayor
calls are delegated through the same handler with C's literal `"gate"`. Six
butler calls remain blocked because Go selects a container and calls
`butlerDoorToggle` directly instead of passing C's `case`, `cabinet`, or `chest`
argument through `mobGenDoor`. That R5e mismatch is reported below and was not
changed here. The seventh blocked row is `breed_killer`: the shared Go command
currently has no mobile `SCMD_SPIKE`/`SCMD_STAKE` body. No door helper or `src/`
file was changed.

## Port and C call paths

The shared entry is `World.NPCCommandDirect`: it preserves the raw argument,
dispatches the named body without the interpreter prelude or gates, and then
uses the mobile-native fallback when the session registry has no actor body.
The session route reaches `dispatchCharacterCommandDirect` with the same
actor and argument. The call sites now use this path for communication,
movement/follow, equipment, combat skills, Lua actions, and room/mobile
specials; sites already reaching the same command body are marked delegated.

The initial seed-8 mismatch was a Go port error. In the actual
`do_headbutt` call path, `src/new_cmds.c:413-418`, both `send_to_char` calls
target `ch`, the attacking mobile. A mobile has no descriptor, so neither
refusal line reaches the immortal player; the only observable effect is that
the mobile sits before returning without a percent draw. Go now retains that
posture and return while emitting no victim line. The regression test asserts
silence, posture and no draw. Its revert control was exercised: restoring the
victim `SendMessage` made the test fail on the extra bytes, and removing it
made the test pass.

The direct dispatch check is exact-name based. `dispatchCharacterCommandDirect`
passes the normalized full name to `cmdRegistry.Lookup`, whose implementation
does a case-insensitive map lookup (`pkg/command/registry.go:113-119`); it
does not resolve prefixes. That matches C's direct `ACMD` call.

The complete C→Go call and proof inventory is the TSV map. It covers at least
`src/spec_procs.c`, `src/spec_procs2.c`, `src/spec_procs3.c`, `src/mobact.c`,
`src/graph.c`, `src/fight.c` and `src/scripts.c`, with test names as revert
controls for the fixed rows.

## Retained oracle evidence

The five 34-scenario runs below were captured before the headbutt correction.
They remain in the archive as the original draft's evidence; the seed-8 red
was fixed and rechecked in the focused runs that follow.

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
`luna-mob-command-lane-seed8-fighter-diagnostic`; those retained attempt logs
describe the pre-fix draft. The source check corrected the diagnosis: C sends
both refusal strings to the descriptor-less mobile, rather than to the
immortal player.

After commit `53a751987`, the focused `spec-proc-fighter` scenario ran on each
required seed with `--draw-parity`:

| Seed | Census | Verdict | Draw divergence |
|---:|---|---|---:|
| 1 | `luna-mob-command-fighter-fix-53a751-seed1` | CLEAN | 0 |
| 2 | `luna-mob-command-fighter-fix-53a751-seed2` | CLEAN | 0 |
| 3 | `luna-mob-command-fighter-fix-53a751-seed3` | CLEAN | 0 |
| 5 | `luna-mob-command-fighter-fix-53a751-seed5` | CLEAN | 0 |
| 8 | `luna-mob-command-fighter-fix-53a751-seed8` | CLEAN | 0 |

The initial 35-scenario seed-1 run `luna-mob-command-lane-seed1` included
`spec-proc-key-seller`, whose setup has an existing draw mismatch (C
`Number(0,18)`, Go `Number(1,6)`). Archived runs before this work show the same
divergence. That case is excluded from the five-seed batch and recorded in the
key-seller rows' notes; its direct tell argument is covered by the
caller-specific `TestSpecKeySellerTellArgumentsMatchC`.

All census artifacts are retained under
`/home/zach/Archives/darkpawns/oracle-runs/2026-10-25/`.

## Checks

Passed individually at the review follow-up commit: `make fmt`, `go build ./...`,
`go vet ./...`, `go test ./...`, `go test ./pkg/game/...`,
`golangci-lint run ./...` (0 issues; it printed a warning for an absent sibling
worktree path), `make deadcode-ratchet` (52 baselined, 0 new),
`make fidelity-depth` (42 maps, 0 errors), `make fidelity-units` (1,949 claims),
`make string-census` (ratchet passes; generated reports are already stale),
`gosec -severity=high -quiet ./...`, and the proof-token check (no missing
scenarios).

Zach approved a narrow exception in `scripts/surface_map.py`: the custom
`mob-command-calls.tsv` is skipped with `spell-matrix.tsv` during per-file
output-map validation and map counting. The comment identifies it as a C-call
inventory, not a per-file output map, and states this script does not validate
its schema. No validator was added.

## Review follow-up — 2026-10-25

The R5h proof was tightened at commit `915007fe9`. The 56 mapped
special-procedure tell calls now have caller-specific table tests that record
the exact command and target-prefixed argument at the direct-command boundary.
The 13 `hunt_victim` command calls have
`TestHuntVictim_CommandArgumentsMatchC`; puff's say calls are asserted by the
complete roll-to-command table and the explicit dead-mobile test. The map names
each C call line and describes shared literals where separate branches use
them. A temporary mutation of the recruiter training argument made
`TestSpecRecruiterTellArgumentsMatchC/training` fail on the exact argument
mismatch; the C-matching value was restored and the focused and full Go suites
pass.

The tell-call audit also found three caller paths that did not reach the
command body as C does. `remorterTell` now routes through `tellFromMob`;
`conSellerTell` uses `NPCCommandDirect("tell", target + text)`; and
`specEviltrade` now handles C's `give` path instead of the Go-only `trade`
behavior. Their exact arguments and caller outcomes are covered by
`TestSpecRemorterTellArgumentsMatchC`, `TestSpecConSellerTellArgumentsMatchC`,
`TestSpecEviltradeTellArgumentsMatchC`, and `TestEviltradeSetsFlag`. There is no
oracle scenario for eviltrade, so that path has unit proof only.

The exact dispatch test confirms `tell` resolves its registered body while
`tel` is unsupported, matching direct C `ACMD` lookup with no prefix expansion.

The five targeted census runs cover the six remorter/con-seller scenarios on
each required seed and use `--draw-parity`:

| Seed | Census | Verdict | Draw divergence |
|---:|---|---|---:|
| 1 | `luna-mob-command-review-seed1` | CLEAN | 0 |
| 2 | `luna-mob-command-review-seed2` | CLEAN | 0 |
| 3 | `luna-mob-command-review-seed3` | CLEAN | 0 |
| 5 | `luna-mob-command-review-seed5` | CLEAN | 0 |
| 8 | `luna-mob-command-review-seed8` | CLEAN | 0 |

The merged-door R5e recheck found hunter `src/graph.c:338` builds the same
`<keyword list> <travel direction>` string that Go passes to `mobGenDoor`;
the multi-word-keyword test exercises that parse. Mayor lines 906, 907, 911,
and 912 all pass the same literal `"gate"` to `mobGenDoor`. Butler lines 172,
178, 183, 190, 191, and 192 do not: `specButler` calls `butlerDoorToggle`
directly with a selected object. These six rows are marked blocked for this
separate mismatch and were left untouched as requested.

## Delivery

Elapsed time at initial handoff: 1h 25m; seed-8 correction follow-up: about
15m; this review follow-up: about 18m (roughly 1h 58m total). PR count: 1 draft
(#2148). The draft remains open for Claude's re-review; no merge or deploy was
performed.

## Draft PR

See [draft PR #2148](https://github.com/zax0rz/darkpawns/pull/2148) for the
grouped C call paths, per-group revert controls and **Decisions for Zach**.
