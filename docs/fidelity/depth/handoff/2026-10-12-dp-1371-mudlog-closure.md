# DP-1371 D7 mudlog — closure (2026-10-12)

Consolidation PR: stale pointers replaced with the landed named proofs, the
208-site enumeration reconciled against the C sources, the `disabled-C` rationale
corrected, the aggregate updated only where a named proof exists, the missing
non-PK death arm proven, and the residual list fixed. Aggregate closure
(`mudlog.unported-sites`) is **recommended for closure at case level** with the
residuals below recorded.

## 1. Resolved — case inventory (`docs/fidelity/depth/mudlog.tsv`, 133 cases)

| Status | Before | After | Meaning |
|---|---|---|---|
| `unit-green` | 62 | **114** | a named landed unit proof exists (was mostly scenario pointers) |
| `excluded` | 13 | 13 | ruled out of valid play; no test warranted |
| `divergent-approved` | 3 | 3 | approved divergences with named tests |
| `oracle-green-multiseed` | 51 | **2** | scenario-pointer evidence only (residual list) |
| `blocked` | 1 | 1 | the `mudlog.unported-sites` tracker row itself |

Rows whose `proof` cell now names a landed test: **125 of 133** (was 76). The
replacement set is drawn from tests actually merged: `TestQuitMudlog*`,
`TestCloseSocket*`, `TestWizutilMudlog*`, `TestSkillsetMudlogTapMetadata`,
`TestHouseMudlog*`, `TestIdleForceRent*`, `TestPKDeathMudlog*`,
`TestReconnectMudlog*`, `TestLoginConnectMudlog`, `TestNewPlayerMudlog`,
`TestEntryRestrictionMudlogProducers`, `TestBadPasswordMudlog`,
`TestSelfDeleteMudlog`, `TestWizardMudlogProducers`, `TestWhodMudlogProducers`,
`TestOLCDiskMudlogProducers`, `TestOLCMemoryMudlog*`, `TestNewZoneMudlog*`,
`TestMudlogLocal*`, `TestHelpMudlogActingBody`, `TestMilestoneMudlog*`,
`TestStealZoneMudlog*`, `TestCrashLoadEntry*`, `TestReportMudlog*`,
`TestReditInsertionMudlogBoundary`, `TestGameDiagnostic*`, plus the PR-4 lua
proofs and the ban/ident producers.

## 2. Resolved — small missing proof added here

`pkg/game/pk_death_mudlog_test.go: TestPKDeathMudlogMobKillerArm` — the game-side
non-PK arm (`fight.c:1678` through `pkg/game/death.go:591-609`): a **mob** killer
logs the unprefixed `"X killed by Y at <room>"` line at BRF / `LVL_IMMORT` / file
TRUE. This closes the residual that batch 7 recorded.

## 3. Two non-PK death paths, distinguished

| Path | Producer | Proof |
|---|---|---|
| Game side, PC victim + NPC killer (`fight.c:1678` else arm) | `pkg/game/death.go:591-609` | `TestPKDeathMudlogMobKillerArm` **(added here)** |
| Combat fallback, `onDeath == nil` (`fight_core.go:589` else arm) | `pkg/combat/fight_core.go` | `TestPlayerSelfDeathMudlogThroughCombatCallback` |

The PK arms of both are also proven (`TestPKDeathMudlogProducer`,
`TestPKDeathMudlogThroughCombatCallback`).

## 4. Corrected — `disabled-C` rationale

`src/comm.c:1581` (`"New connection from [%s]"`, CMP / LVL_GOD / file FALSE) sits
behind **`#if 0` at `src/comm.c:1578-1582`** (`2026-10-06-dp-1371-mudlog-sites-check.py`
asserts those exact lines). The reference oracle cannot run it, so no Go
counterpart is required. Its **live sibling is a different site** —
`src/comm.c:1573` (`"Connection attempt denied from [%s]"`, CMP / LVL_GOD / file
**TRUE**) — implemented at `pkg/telnet/listener.go:244` and
`pkg/session/manager.go:1265` and proven by
`TestBannedConnectionRefusalBytesAndLog` / `TestWebSocketBannedRefusalLogs`.
The earlier wording credited this row to the sibling's Go code; the row's
`go_owner` and `proof` cells now say so explicitly.

## 5. Preserved limitation — fresh-login invisibility

`interpreter.c:1927`'s level is `MAX(LVL_IMMORT, GET_INVIS_LEV(...))`, but the
fresh login restores the player from storage, while `Player.InvisLevel` is
currently runtime-only (`pkg/game/player.go`) and is not restored by
`db.RecordToPlayer`. `TestLoginConnectMudlog` proves the
payload, the BRF type, the file flag and the `LVL_IMMORT` floor; the invisibility
leg remains outside this proof; do not claim the floor test covers it.

## 6. Reconciliation and required checks (as run)

| Check | Result |
|---|---|
| `2026-10-06-dp-1371-mudlog-sites-check.py` | PASS: 208 retained sites / 207 active, all calls reconciled; no duplicates, complete rows, disabled-C source checks and eight-site group limit pass. `game-diagnostics` has eight sites; the existing hunting producer is grouped separately as `hunting-diagnostics` (one site). |
| `go build ./...` / `go vet ./...` | 0 / 0 |
| `go test ./pkg/game/... ./pkg/session/... -count=1` | 0 |
| `go test ./... -count=1` | 0 |
| `golangci-lint run ./...` | 0 issues |
| `make fmt` / `make check-fmt` / `make deadcode-ratchet` | 0 / 0 / ok |

## 7. Residuals

1. `mudlog.unported-sites` — the tracker row itself, `blocked` by design; this PR
   recommends closing it at case level (see §8).
2. `olc-medit-checked-write`, `olc-sedit-checked-write` — `excluded-valid-play` by
   Zach's ruling (buffered stdio shows no diagnostic; needs a corrupt file). No
   test warranted.
3. `olc.save-info`, `game-spike-stake` — scenario-pointer evidence only; no named
   unit proof exists. Candidates for a small follow-up if unit-level proof is
   wanted.

## 8. Closure recommendation

Close `mudlog.unported-sites` **at case level for the D7 scope**: 114 of 133 cases
carry a named landed unit proof, 3 more are approved divergences with named tests,
13 are excluded by ruling, and the 2 remaining scenario-only rows plus the tracker
row are recorded above. The 208-site enumeration still reconciles with the C
sources. The original checker failure was resolved by separating the hunting
producer from the eight-site game-diagnostics group without changing or weakening
the checker.
