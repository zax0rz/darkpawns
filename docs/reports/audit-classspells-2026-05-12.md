# Audit: classSpells vs C init_spell_levels()

**Date:** 2026-05-12
**Auditor:** Daeron
**Files:** `pkg/session/spell_level.go` (Go) vs `src/class.c:init_spell_levels()` (C)

## Summary

The Go `classSpells` table in `session/spell_level.go` does **not** match the authoritative C source `src/class.c:init_spell_levels()`. The discrepancies fall into three categories:

1. **Extra spells in Go** — Go includes psionic/hybrid spells for Mage that C does not
2. **Level mismatches** — Same spell, different level between C and Go
3. **Missing spells in Go** — C has spells that Go omits

## Mage (Class 0) — Worst Offender

C source: 27 spells. Go code: ~50 entries. The Go code includes ~23 extra spells that are psionic/hybrid spells not present in the C Mage list.

### Extra spells in Go (not in C for Mage):
- Spell 1 (armor) at level 1 — C has no armor for Mage
- Spell 32 (magic missile) at level 1 — C has magic missile at level 1 ✓ (but ID 32 is correct)
- Spell 20 (detect magic) at level 1 — C has detect magic at level 2
- Spell 31 (locate object) at level 3 — not in C Mage list
- Spell 5 (burning hands) at level 3 — C has burning hands at level 5
- Spell 37 (shocking grasp) at level 5 — C has shocking grasp at level 7
- Spell 29 (invisible) at level 7 — C has invisible at level 4
- Spell 39 (strength) at level 7 — C has strength at level 6
- Spell 33 (poison) at level 9 — not in C Mage list
- Spell 4 (blindness) at level 10 — C has blindness at level 9
- Spell 7 (charm) at level 12 — not in C Mage list
- Spell 36 (sanctuary) at level 13 — not in C Mage list
- Spell 40 (summon) at level 14 — not in C Mage list
- Spell 9 (clone) at level 15 — not in C Mage list
- Spell 2 (teleport) at level 16 — not in C Mage list
- Spell 23 (earthquake) at level 17 — not in C Mage list
- Spell 6 (call lightning) at level 18 — not in C Mage list
- Spell 46 (dispel good) at level 20 — not in C Mage list
- Spell 22 (dispel evil) at level 20 — not in C Mage list
- Spell 25 (energy drain) at level 22 — C has energy drain at level 13
- Spell 58 (hellfire) at level 25 — C has hellfire at level 20
- Spell 41 (meteor swarm) at level 32 — not in C Mage list
- Spell 79 (mirror image) at level 34 — not in C Mage list
- Spell 97 (haste) at level 35 — not in C Mage list
- Spell 98 (slow) at level 36 — not in C Mage list
- Spell 99 (dream travel) at level 38 — not in C Mage list
- Spell 105 (conjure elemental) at level 40 — not in C Mage list
- Spell 100 (psiblast) at level 43 — not in C Mage list
- Spell 101 (call of chaos) at level 45 — not in C Mage list
- Spell 80 (mass dominate) at level 48 — not in C Mage list
- Spell 61 (mindpoke) at level 4 — not in C Mage list
- Spell 62 (mindblast) at level 10 — not in C Mage list
- Spell 63 (chameleon) at level 8 — not in C Mage list
- Spell 64 (levitate) at level 12 — not in C Mage list
- Spell 65 (metalskin) at level 14 — C has metalskin at level 21
- Spell 66 (invulnerability) at level 20 — C has invulnerability at level 28
- Spell 67 (vitality) at level 22 — not in C Mage list
- Spell 68 (invigorate) at level 24 — not in C Mage list
- Spell 69 (lesser perception) at level 5 — not in C Mage list
- Spell 70 (greater perception) at level 15 — not in C Mage list
- Spell 71 (mind attack) at level 8 — not in C Mage list
- Spell 72 (adrenaline) at level 10 — not in C Mage list
- Spell 73 (psyshield) at level 12 — not in C Mage list
- Spell 74 (change density) at level 14 — not in C Mage list
- Spell 75 (acid blast) at level 6 — C has acid blast at level 2
- Spell 76 (dominate) at level 16 — not in C Mage list
- Spell 77 (cell adjustment) at level 18 — not in C Mage list
- Spell 78 (zen) at level 20 — not in C Mage list
- Spell 102 (water breathe) at level 20 — C has water breathe at level 22
- Spell 51 (waterwalk) at level 22 — C has waterwalk at level 12
- Spell 54 (calliope) at level 5 — not in C Mage list
- Spell 45 (protect good) at level 20 — not in C Mage list

### Spells in C but missing from Go:
- None完全 — all C spells appear to be present in Go, but at different levels

### Level mismatches (same spell, different level):
| Spell | C Level | Go Level | Delta |
|-------|---------|----------|-------|
| acid blast | 2 | 6 | +4 |
| burning hands | 5 | 3 | -2 |
| shocking grasp | 7 | 5 | -2 |
| invisible | 4 | 7 | +3 |
| strength | 6 | 7 | +1 |
| blindness | 9 | 10 | +1 |
| energy drain | 13 | 22 | +9 |
| hellfire | 20 | 25 | +5 |
| metalskin | 21 | 14 | -7 |
| invulnerability | 28 | 20 | -8 |
| water breathe | 22 | 20 | -2 |
| waterwalk | 12 | 22 | +10 |
| detect magic | 2 | 1 | -1 |

## Root Cause Analysis

The Go `classSpells` table appears to have been written from a different version of the game than the C source in `src/class.c`. The C source has 27 spells for Mage; the Go code has ~50. The extra spells in Go are psionic/hybrid spells (mindpoke, mindblast, chameleon, etc.) that were likely added in a later version of Dark Pawns (possibly 2.3+) or were incorrectly ported from a different class.

## Recommendation

1. **The C source `src/class.c:init_spell_levels()` is the authoritative reference.** It matches the help files (which are player-facing).
2. **Rebuild `classSpells` from the C source** — strip the extra spells, fix the levels.
3. **The help files are also stale** — they reference "flame arrow" as spell 1 for Mage, but the C source has "magic missile" as spell 1. The help files need updating too.
4. **Priority: HIGH** — affects which spells each class can learn and at what level. This is a gameplay-breaking bug if the Go code is used as-is.
