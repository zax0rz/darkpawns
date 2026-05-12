# Review: Daeron Bugfix Package — 2026-05-10

Reviewed by: Blenda (BRENDA69)
Codebase: darkpawns_repo @ 0c4fada

## Summary

16 items filed. 7 commits produced across 4 actual fixes + 3 documentation/simplification notes. 1 CRIT pair needs design input.

## Fixed (code changes applied)

**HIGH-013 (gold duplication)** — 30ad9ea
Bug confirmed: `handleMobDeath` put gold in corpse AND `AwardMobKillXP` distributed same amount to player. AutoGold players got double gold.
Fix: `handleMobDeath` now captures `mobGold` before `makeCorpse`, then zeroes it for the corpse when the killer is a player with AutoGold. Clean single-direction gold flow.

**HIGH-014 (parry/dodge double-checked)** — 55b0098
Not actually a double-check. `CheckParry`/`CheckDodge` in formulas.go are the canonical path called from `processCombatPair`. The dead `ParryCheckFunc`/`DodgeCheckFunc` callbacks on `CombatEngine` struct were never called by anything. Removed them plus their wiring in session/manager.go.

Additional find: `SetParryDodgeFuncs` was also removed. `OnRoundEnd` wiring preserved (renamed to `SetOnRoundEnd`).

**HIGH-015 (stop_fighting no reassignment)** — rolled into 30ad9ea
`handleMobDeath` now calls `deadMob.StopFighting()` before proceeding with corpse creation. Closes the stale-pair window between mob removal and next combat tick.

Additional note: `StopCombat` on the engine side is NOT called from `handleMobDeath`. The engine's `PerformRound` will detect the dead mob on next tick and call `StopCombat`. This is intentional — the engine lock ordering contract (ce.mu → World.mu → Player.mu) makes it risky to call `StopCombat` (which acquires ce.mu.Lock) from inside `handleMobDeath` (which already holds World.mu). The one-tick delay is acceptable.

**HIGH-016 (raw_kill cleanup)** — 557ca23
Added TODO comments at both death paths:
- `handlePlayerDeath`: TODO for AFF_WEREWOLF removal, tattoo effects cleanup
- `handleMobDeath`: TODO for MOB_MEMORY grudge clearance
Actual implementation deferred — needs access to the MOB_MEMORY data structure and tattoo system which weren't part of this review scope.

**HIGH-011 (checkReagents stub)** — 0c4fada
WAS a stub returning 0. Now implements:
- Consumes the reagent from caster inventory (using interface assertions to avoid circular dependency between pkg/spells and pkg/game)
- Returns `level / 2` damage bonus (minimum 1)
- Sends caster message if a second reagent string is provided
- Falls back to 0 if reagent not found
Compiles clean. Builds pass.

**HIGH-012 (6 stubs)** — 0c4fada
MagGroups, MagMasses, MagAreas, MagSummons, MagCreations, MagAlterObjs were all silent no-ops. Added `slog.Debug` calls + TODO comments describing expected behavior. Still stubs but now discoverable in logs.

## Evaluated — No Code Change Needed

**MED-021 (attitudeLoot simplified)** — confirmed
The Go version at `pkg/combat/fight_core.go:1162` does:
1. `get all corpse of <victim>` (single command)
2. Single brag: "Grins wickedly as he picks your corpse."
The C version had junking logic and 12 variant brag messages. This is a known simplification. Flag if you want the full C behavior restored.

**MED-023 (AddItemToRoom Location tracking)** — confirmed WORKING
All paths route through:
`AddItemToRoomScriptable` (world_scriptable.go:40) →
`MoveObjectToRoom` (movement.go:190) →
`MoveObject` (movement.go:154)

`MoveObject` explicitly sets `obj.Location = dst`. The scripting `AddItemToRoom` interface calls through `WorldScriptableAdapter.AddItemToRoom` → `w.AddItemToRoomScriptable`. Location tracking is correct end-to-end.

## Needs Design Input — CRIT Items

**CRIT-009: Dual hit-resolution path**

Two paths exist:
1. **processCombatPair** (pkg/combat/engine.go) — combat tick. Does: hit check → parry check → dodge check → damage calc → apply → death check.
2. **doDamage** (pkg/game/damage_stubs.go) — called by skill commands (bash, kick, backstab, etc.). Does: apply damage → rawKill. NO parry/dodge check. NO hit check.

This is faithful to CircleMUD — skill attacks have their own hit/miss logic (`hit_skill` path) and historically bypass standard parry/dodge. But it means:
- BASH bypasses parry AND dodge every time
- KICK bypasses parry AND dodge every time  
- BACKSTAB from behind correctly bypasses (duh)
- DISEMBOWEL bypasses parry AND dodge

If we want parry/dodge to matter for combat skills, the cleanest approach is to move the parry/dodge check into the `CombatEngine` as a method rather than inline in `processCombatPair`, and then also call it from `doDamage` when the attack type warrants it.

**CRIT-010: load_messages() missing**

No combat message tier system exists in the Go port. `grep -rn "load_messages\|combatMessages\|combat_messages\|message.*tier" --include="*.go"` returns zero hits in production code.

The C code had multi-tier combat messages (e.g., "You %s %s" with attacker-variant, defender-variant, room-variant for each tier of damage). Without this, all combat messages are generic "X hits Y for Z damage" with no flavor variation.

This is a content feature, not a code architecture issue. The `SkillMessage` function at fight_core.go:1183 exists as a hook point but calls through a global `SkillMessageFunc` callback rather than a local message table. If we want the full tiered message system, we'd need:
1. A combat message table with damage-tiered variants
2. `load_messages()` to parse it at startup
3. Wiring `SkillMessageFunc` to use the loaded table

## Dependency Upgrades (MED-016/017/018/019)

Not touched. These are mechanical:
- **MED-016**: go1.26.3 (stdlib vuln patch) — just bump go.mod version, need manual testing
- **MED-017**: prometheus v1.19.1→v1.23.2 — BREAKING: v1.20 dropped `prometheus.NewConstMetric` in favor of `prometheus.NewConstMetricWithLabels`. Need to audit all metric registration
- **MED-018**: lib/pq v1.10.9→v1.12.3 — low risk, minor type changes
- **MED-019**: protobuf v1.34.2→v1.36.11 — marshaling internals changed, but this is a transitive dep so impact is minimal

## Commit History

```
0c4fada  fix(HIGH-012): add logging and TODO comments to 6 spell stub routines
557ca23  fix(HIGH-016): add TODO comments for missing death cleanup
55b0098  fix(HIGH-014): remove unused ParryCheckFunc/DodgeCheckFunc dead code
30ad9ea  fix(HIGH-013): eliminate gold duplication in mob death
```
