# Commit Diff Sentinel — 2026-05-18

Window: 2026-05-17 08:30 UTC → 2026-05-18 08:30 UTC
2 commits reviewed, 0 findings.

---

## Commit: f1ed5eb7c3b9f5800e7febefdc6369a740663051
**Author:** BRENDA69
**Date:** Sun May 17 09:24:12 2026 -0400
**Message:** fix: resolve 3 CRITICAL spell fidelity bugs (DP-172, DP-173, DP-174)

### Files changed (2):
- `pkg/spells/affect_spells.go` — +64 -22
- `pkg/spells/damage_spells.go` — +0 -3

### Changes:
- DP-172: PROTECT_EVIL/GOOD now calls `combat.RawKill()` on alignment violation (C: magic.c:1142-1180)
- DP-173: HELLFIRE level≤4 handling — now deals `GET_MAX_HIT*12` lethal damage instead of skipping (C: spells.c:729-756)
- DP-174: FLAMESTRIKE moved from `RoutineDamage` to `RoutineAffects`, applies AFF_FLAMING DOT with outdoor-only restriction (C: magic.c:1109-1129)
- Removed flamestrike damage formula from `MagDamage` in `damage_spells.go`

---

## Commit: 9806cb48999963a94109185e216c76d17d75cac6
**Author:** BRENDA69
**Date:** Sun May 17 09:44:32 2026 -0400
**Message:** fix: resolve 4 HIGH spell fidelity bugs (DP-175, DP-176, DP-177, DP-178)

### Files changed (2):
- `pkg/spells/affect_spells.go` — +41 -4
- `pkg/spells/call_magic.go` — +5 -11

### Changes:
- DP-175: CallMagic savetype mapping — wand/staff/scroll/potion → `SaveRodStaff`, default → `SaveBreath` (C: spell_parser.c:454-467)
- DP-176: Meteor swarm damage range — `rand.Intn(level*3+11)-10` now produces `[-10, level*3]` matching C `number(-10, level*3)`
- DP-177: Charm spell applies AFF_CHARM with C duration formula `24*18/GET_INT`, removes MOB_AGGRESSIVE and MOB_SPEC flags (C: spells.c:448-464)
- DP-178: Metalskin reagent flat +1 AC bonus — `checkReagents` supports `"flat:1"` override (C: magic.c:1300)

---

## Build & Test Results
```
go build ./...        — PASS
go vet ./...          — PASS
go test ./pkg/spells/ — PASS (0.193s)
go test ./...         — PASS (all cached)
```

## Analysis

**No regressions detected.** All changes are targeted fidelity fixes matching C source behavior. Key observations:

1. **No debug prints** — clean commits
2. **No test files added** — expected for fidelity fixes where C source verification is the ground truth
3. **No accidental file changes** — only the 3 files listed, all relevant to the fixes
4. **Interface assertions safe** — all type assertions use the `if x, ok := ...` two-value form
5. **Zero-division guarded** — `GET_INT(victim)` duration calculation checks `vi.GetInt() > 0`
6. **Consistent patterns** — flamestrike outdoor check follows existing meteor swarm pattern (`HasFlag(3) && Sector == 0`)
