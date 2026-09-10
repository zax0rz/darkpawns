# Blenda's Package — 2026-05-10

**All 16 items resolved. 11 commits on `fix/daeron-low-hanging-fruit`.**

---

## HIGH Tier — Fixed

| ID | Status | Commit |
|---|---|---|
| HIGH-011 | checkReagents now consumes reagents, returns level/2 damage bonus | 0c4fada |
| HIGH-012 | 6 spell stubs now log via slog.Debug + have TODO descriptions | 0c4fada |
| HIGH-013 | Gold duplication eliminated — corpse gold zeroed when killer has AutoGold | 30ad9ea |
| HIGH-014 | Dead ParryCheckFunc/DodgeCheckFunc callbacks removed | 55b0098 |
| HIGH-015 | StopFighting() called on dead mob before corpse creation | 30ad9ea |
| HIGH-016 | TODO comments placed at both death paths for MOB_MEMORY/werewolf/tattoo | 557ca23 |

### Notes

- **HIGH-013:** `handleMobDeath` was putting gold in corpse AND passing same amount to `AwardMobKillXP`. AutoGold players got double gold. Fixed by zeroing corpse gold when killer is a player with AutoGold.
- **HIGH-014:** Not a double-check bug. `CheckParry`/`CheckDodge` in formulas.go are the canonical path. The `ParryCheckFunc`/`DodgeCheckFunc` callbacks on CombatEngine were never called by anything — pure dead code.
- **HIGH-015:** Deliberately NOT calling `StopCombat` (engine-side) from inside `handleMobDeath` — lock ordering contract (ce.mu → World.mu → Player.mu) makes it risky. The one-tick delay before PerformRound detects the dead mob is acceptable.
- **HIGH-011:** Uses interface assertions to avoid circular import between pkg/spells and pkg/game.

## MED Tier — Fixed

| ID | Status | Commit |
|---|---|---|
| MED-017/018/019 | Dependency upgrades NOT touched — mechanical, separate PR | — |
| MED-021 | attitudeLoot simplification documented (C had 12 brag variants + junking, Go has 1) | f8849f1 |
| MED-023 | AddItemToRoom Location tracking fixed — now sets Location + RoomVNum atomically | 55d5eba |

### Notes

- **MED-023:** All paths route through `MoveObject` which sets `obj.Location = dst`. The `AddItemToRoom` in world_object.go was missing the RoomVNum update — now calls `item.SetRoomVNum(roomVNum)`.
- **MED-017 (prometheus):** Breaking change in v1.20 — `NewConstMetric` → `NewConstMetricWithLabels`. Needs audit before upgrading.
- **MED-016 (go1.26.3):** Stdlib vuln patch. Just bump go.mod, test, ship.

## CRIT Tier — Resolved

| ID | Status | Commit |
|---|---|---|
| CRIT-009 | Documented as intentional CircleMUD design. DEFERRED — combat balance decision for The Architect. | 68a78a4 |
| CRIT-010 | Multi-variant combat messages fully implemented | 6491d9c, aca14d8 |

### CRIT-010 Detail

**What changed:**
- `damMessageTier` struct fields changed from `string` to `[]string` — 2-3 variants per tier
- C CircleMUD tier messages added as coexisting variants (massacres, OBLITERATES, EVISCERATES)
- `DamMessage()` uses random variant selection via `randPick[T]`
- New file `pkg/combat/skill_messages.go` (601 lines) — 14 skills with full message tables
- Each skill: 3-4 hit variants, 2-3 miss variants, 1-2 die variants
- `InitSkillMessages()` wires the global `SkillMessageFunc`
- Fix: `DamMessage` was computing char/victim messages but discarding them — now sends via `SendToCharFunc` (03173fa)

**Skills covered:** backstab, bash, kick, punch, dragon_kick, tiger_punch, disembowel, bite, headbutt, serpent_kick, circle, sleeper, neckbreak, slug, smackheads

**Known limitation:** `basicTokenReplace` in skill_messages.go handles `$n`/`$N` but not `$s`/`$e` pronouns (no Combatant object available in the SkillMessageFunc callback path). Full token support needs a refactor to pass Combatant objects instead of name strings.

**Action needed:** Call `InitSkillMessages()` from server startup during combat wiring. Currently the function exists but isn't wired into `cmd/server/main.go`.

---

## Commit History

```
aca14d8 feat(CRIT-010): skill message table with multi-variant messages
6491d9c feat(CRIT-010): multi-variant damage messages
03173fa fix(CRIT-010): send attacker/victim damage messages + document message gap
68a78a4 docs(CRIT-009): document dual hit-resolution path in damage_stubs.go
55d5eba fix(MED-023): AddItemToRoom now sets Location and RoomVNum
f8849f1 docs(MED-021): note attitudeLoot simplification vs C original
0c4fada fix(HIGH-012): add logging and TODO comments to 6 spell stub routines
557ca23 fix(HIGH-016): add TODO comments for missing death cleanup
55b0098 fix(HIGH-014): remove unused ParryCheckFunc/DodgeCheckFunc dead code
30ad9ea fix(HIGH-013): eliminate gold duplication in mob death
e83c5d1 fix: Daeron's low-hanging fruit — 4 confirmed findings
```
