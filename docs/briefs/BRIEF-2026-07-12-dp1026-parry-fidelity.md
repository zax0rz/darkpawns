# Brief: Restore C Parry/Dodge Fidelity — 2026-07-12

**Workspace:** `/Users/zach/.openclaw/workspace/darkpawns_repo`
**Repo:** `git@github-darkpawns:zax0rz/darkpawns.git` (branch: `main`)
**Build gate:** `go build ./... && go vet ./... && go test ./...` — ALL THREE MUST PASS.

---

## Fix 1: DP-1026 — Parry/Dodge Skill Wiring and Combat Model (High)

**File:** `pkg/game/combat_wire.go`, `pkg/combat/formulas.go`, `pkg/combat/engine.go`

**Problem:**
The game callback for `GetSkill(name, skillNum)` looked up `p.GetSkill(name)`, using the character name as the skill key. That made parry, retreat, and escape skill callbacks return zero. Separately, the Go melee engine modeled parry/dodge as per-hit negation using a 1-101 skill roll, while C checks parry/dodge once per round and reduces the attacker's attack count.

**Fix:**
Map C skill numbers to game skill names before reading `Player.GetSkill()`. Correct the C skill numbers for retreat, escape, and parry. Rework parry to use `number(0,10000) <= GET_SKILL(ch, SKILL_PARRY)` once per round. Rework dodge as NPC-only `AFF_DODGE` with `number(0,100) < GET_LEVEL(ch)`. Apply successful parry/dodge by reducing attack count using the defender's `dex_app[].defensive` rule.

**Cite:** C source — `src/spells.h:197,205,220` (`SKILL_RETREAT=149`, `SKILL_ESCAPE=157`, `SKILL_PARRY=172`). `src/fight.c:1949-1973` checks player parry and NPC `AFF_DODGE` once per round. `src/fight.c:1999-2004` reduces the parried attacker's attack count.

**Regression Test:** `pkg/game/combat_skill_names_test.go`, `pkg/combat/formulas_test.go`, `pkg/combat/engine_test.go`
- Pin the C skill-number mapping used by combat callbacks.
- Pin parry's 0-10000 range and mutual-fighting requirements.
- Pin dodge as NPC `AFF_DODGE`, not player skill.
- Pin successful parry reducing a two-attack round to one hit.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Execution Order

1. Correct combat skill constants and game-layer skill-name mapping.
2. Port parry and dodge formulas to the C model.
3. Apply parry/dodge as an attack-count reducer in the engine.
4. Update focused combat tests and deterministic roller sequences.

## After Fix

```bash
git add pkg/game/combat_wire.go pkg/game/combat_skill_names.go pkg/game/combat_skill_names_test.go pkg/combat/fight_core.go pkg/combat/formulas.go pkg/combat/formulas_test.go pkg/combat/engine.go pkg/combat/engine_test.go docs/briefs/BRIEF-2026-07-12-dp1026-parry-fidelity.md
git commit -m "fix: restore C parry fidelity (DP-1026)"
git push -u origin fix/dp1026-parry-skill-fidelity
gh pr create --title "fix: restore C parry fidelity (DP-1026)" --body "Fixes DP-1026. See docs/briefs/BRIEF-2026-07-12-dp1026-parry-fidelity.md for details."
```
