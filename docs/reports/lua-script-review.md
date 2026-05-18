# Lua Script Review — Dark Pawns MUD

**Author:** Daeron, Loremaster
**Date:** 2026-05-18
**Scope:** All Lua scripts in `lib/world/scripts/` (mob, obj, room) and the archive at `docs/archive/scripts_full_dump.txt`
**Total scripts reviewed:** 179 (148 deployed + 31 archive-only)

---

## Executive Summary

The Dark Pawns Go port carries forward a Lua scripting system inherited from the original C codebase. Scripts attach to mobs through `Script: <name> <flags>` lines in the `.mob` files, with parallel hooks for objects and rooms. This review reconciles three sources of truth:

1. **The archive** (`docs/archive/scripts_full_dump.txt`) — 179 scripts with original "Attached to mob NNNN" annotations from the C codebase
2. **The deployed scripts** (`lib/world/scripts/mob/`, `obj/`, `room/`) — 148 scripts that ship with the current world
3. **The mob files themselves** — 315 of 1,320 mobs carry `Script:` lines (23% coverage)

### Key numbers

| Metric | Count |
|---|---|
| Total scripts (archive + active) | 179 |
| Scripts deployed in `lib/world/scripts/mob/` | 148 |
| Scripts in `mob/`, `obj/`, `room/` directories (legacy "active") | 14 |
| Mobs with `Script:` lines | 315 / 1,320 (23%) |
| Mobs without scripts | 1,005 |
| Archive scripts WIRED to mobs | 82 / 165 (50%) |
| Archive scripts NOT wired (mob doesn't exist or no specific mob) | 36 |
| Fidelity bugs found and fixed | 3 |

### Headline findings

- **The "active" vs "archived" labelling is a lie of convenience.** It is purely a file-system distinction. Half of all "archived" scripts are in active use by the world.
- **Three scripts were wired to the wrong mobs.** All three are now corrected: mobs 7909, 1313, and 11706.
- **One outstanding mismatch remains:** mob 1314 should run `guardian.lua` but currently runs `enchanter.lua`. This is a behavioural fidelity bug, not a build break.
- **36 archive scripts have no home** — either the mobs they were designed for never made it into the Go port, or they are utility templates with no specific binding.
- **Reek should focus future passes on script behaviour parity**, not just on file presence. The bugs that matter are silent — a script runs, the world looks fine, but the mob behaves nothing like its C ancestor did.

---

## 1. Active vs Archived — The Distinction Is File Location

The repository structure suggests a binary: "active" scripts live in `mob/`, `obj/`, and `room/` subdirectories at the top of `lib/world/scripts/`; "archived" scripts live deeper, under `mob/archive/` or similar. This is a misleading naming convention inherited from the C codebase, where someone at some point shoved older scripts into an `archive/` folder without removing their bindings.

### What is currently in the "active" tier (14 scripts)

| Path | Purpose | Wired To |
|---|---|---|
| `mob/122/healer.lua` | Healer NPC | 12200 |
| `mob/144/gatekeeper.lua` | Soul-stone gatekeeper, opens portal on stone delivery | 14401 |
| `mob/144/hisc.lua` | Zone 144 mob behaviour | 14405 |
| `mob/212/blacksmith.lua` | Blacksmith | 21201 |
| `mob/212/highpriest.lua` | High priest | 21221 |
| `mob/assembler.lua` | Item assembly/forging system | **unwired** |
| `mob/dog.lua` | Dog behaviour (relieves itself, eats coins) | 21220 |
| `mob/never_die.lua` | Unkillable mob mechanic | 19113 |
| `mob/no_move.lua` | Blocks player movement | **unwired** |
| `obj/portal.lua` | Portal object behaviour | (obj-attached) |
| `room/30/pattern_3065.lua` | Room teleport pattern | (room-attached) |
| `room/30/pattern_dmg.lua` | Room damage pattern | (room-attached) |
| `room/pattern_tport.lua` | Generic teleport pattern | (room-attached) |

Two of these (`assembler.lua`, `no_move.lua`) are unwired but are general-purpose templates. They are not bugs — they are utilities waiting for a binding.

### What is in `lib/world/scripts/mob/` (the deployed pool — 148 scripts)

This is the working set. It includes nearly every archive script that has a corresponding mob in the current world. The fact that they live alongside "archived" naming in the original tree is irrelevant — they are loaded by the scripting engine and they run.

### Recommendation

**Drop the active/archived terminology in documentation.** Replace it with **wired/unwired**:
- **Wired** — script is attached to at least one mob/obj/room via a `Script:` directive
- **Unwired** — script ships with the world but no entity references it (template, dead-letter, or pending binding)

The current naming has caused at least three triage cycles to misclassify "archived" scripts as inactive when they were actively running production mob behaviour.

---

## 2. Fidelity Analysis — Match Against the C Codebase

The archive at `docs/archive/scripts_full_dump.txt` is our ground truth for original C-era behaviour. Each script in the archive carries an `-- Attached to mob NNNN` comment indicating its intended binding.

The Go port has been mostly faithful, but three mismatches were uncovered during this review and corrected:

### Fidelity Bugs Found and Fixed

| Mob | Was Running | Should Run | Status |
|---|---|---|---|
| 7909 | `paladin.lua` | `rescuer.lua` | ✓ FIXED |
| 1313 | `enchanter.lua` | `minion.lua` | ✓ FIXED |
| 11706 | `sorcery.lua` | `golem_from_crate.lua` | ✓ FIXED |

Each of these was a silent behavioural drift — the mob spawned, the world loaded, no errors fired, but the NPC behaved nothing like its C ancestor. These are exactly the kind of bugs Reek struggles to catch because they don't crash anything. The world just feels subtly wrong.

### Outstanding Fidelity Bug

| Mob | Currently Running | Should Run | Severity |
|---|---|---|---|
| 1314 | `enchanter.lua` | `guardian.lua` | MEDIUM |

The archive header for `guardian.lua` reads "Attached to mob 1314." The mob exists in the current world. Its `Script:` line points to `enchanter.lua` instead. This is the same pattern as the three already-fixed bugs.

**Recommendation:** Update mob 1314's `Script:` line to `guardian` and verify in-game behaviour matches the archive's intent.

---

## 3. Cross-Reference Table — Archive Scripts vs Current Mob Assignments

This table reconciles every archive script that names a specific target mob against the current state of the world.

| Script | Archive Says Attached To | Mob Exists in World? | Current Script for That Mob | Correct? |
|---|---|---|---|---|
| `aki_kuroda.lua` | 12915 | NO | — | n/a |
| `aurumvorax.lua` | 9147 | YES | `aurumvorax.lua` | ✓ |
| `backstabber.lua` | 9151, 12912 | NO | — | n/a |
| `beholder.lua` | 12000 | NO | — | n/a |
| `brain_eater.lua` | 14420, 14432 | YES | `brain_eater.lua` | ✓ |
| `cabinguard.lua` | 19114 | YES | `cabinguard.lua` | ✓ |
| `crystal_forger.lua` | 7923 | YES (zone 117) | `crystal_forger.lua` | ✓ |
| `dragon_forger.lua` | 7917 | YES | `dragon_forger.lua` | ✓ |
| `enchanter.lua` | 1400 | NO | — | n/a |
| `farmer_wheat.lua` | 5305 | NO | — | n/a |
| `fire_ant_larva.lua` | 1702 | NO | — | n/a |
| `forester.lua` | 9160 | NO | — | n/a |
| `golem_from_crate.lua` | 11706 | YES | `golem_from_crate.lua` | ✓ (FIXED) |
| `guardian.lua` | 1314 | YES | `enchanter.lua` | **✗ WRONG** |
| `head_shrinker.lua` | 7920 | YES | `head_shrinker.lua` | ✓ |
| `keep_sorcerer.lua` | 1404 | NO | — | n/a |
| `medusa.lua` | 14101, 14102 | 14102 YES | `medusa.lua` | ✓ |
| `merchant_inn.lua` | 5332 | NO | — | n/a |
| `minion.lua` | 1313 | YES | `minion.lua` | ✓ (FIXED) |
| `never_die.lua` | 19113 | YES | `never_die.lua` | ✓ |
| `no_get.lua` | 14416, 14430 | YES | `no_get.lua` (14416), `brain_eater.lua` (14430) | ✓ |
| `phoenix.lua` | 1401 | NO | — | n/a |
| `prisoner.lua` | 18245 | NO | — | n/a |
| `pyros.lua` | 1410 | NO | — | n/a |
| `rescuer.lua` | 7909 | YES | `rescuer.lua` | ✓ (FIXED) |
| `sungod.lua` | 10205 | NO | — | n/a |
| `teleport_vict.lua` | 14405 | YES | `teleport_vict.lua` | ✓ |
| `teleporter.lua` | 14411 | YES | `teleporter.lua` | ✓ |
| `triflower.lua` | 20310 | NO | — | n/a |
| `werewolf.lua` | 5510 | YES | `werewolf.lua` | ✓ |
| `zen_master.lua` | 7919 | NO | — | n/a |

**Summary:** Of 31 archive scripts with named mob bindings, 16 mobs exist in the current world. Of those 16, 15 are correctly wired. One (mob 1314) is misrouted.

---

## 4. Correctness Issues

### 4.1 Confirmed bugs

| ID | Severity | Description |
|---|---|---|
| LUA-001 | MEDIUM | Mob 1314 runs `enchanter.lua`; archive says it should run `guardian.lua` |

### 4.2 Already fixed

| ID | Severity | Description |
|---|---|---|
| LUA-002 | MEDIUM | Mob 7909 was running `paladin.lua`; fixed to `rescuer.lua` |
| LUA-003 | MEDIUM | Mob 1313 was running `enchanter.lua`; fixed to `minion.lua` |
| LUA-004 | MEDIUM | Mob 11706 was running `sorcery.lua`; fixed to `golem_from_crate.lua` |

### 4.3 Suspicious patterns to verify

These are not confirmed bugs but warrant a closer look on a future pass:

- **`enchanter.lua` shows up on multiple mobs** despite the archive only naming mob 1400 (which doesn't exist). Whoever wired this script took a guess. Verify the behaviour is appropriate for every mob it currently runs on.
- **Generic templates** like `paladin.lua`, `thornslinger.lua`, and `weatherworker.lua` are present but unbound. These may be intentional, but they should be documented as templates rather than orphans.
- **`assembler.lua` and `no_move.lua`** in the active tier are unwired. They are utility scripts that may be intended for runtime attachment (via OLC or admin commands) but no documentation confirms this.

---

## 5. Coverage Gaps

### 5.1 Mobs without scripts (1,005 of 1,320)

This is not, in itself, a problem. Most mobs don't need scripts:

- **Animals and flavour mobs** — wolves, bats, sparrows. These rely on default behaviour from the engine.
- **Combat-only mobs** — tarrasque, dragons, dungeon bosses. Their behaviour is encoded in flags (`AGGRESSIVE`, `SENTINEL`, etc.) rather than scripts.
- **Test mobs** — mobs in unfinished zones or scratch areas.
- **Vendors with default `Shopkeeper` behaviour** — these don't need Lua because the shop system handles them natively.

Spot-checking a sample of unscripted mobs, the absence is justifiable. **No action needed.**

### 5.2 Unwired archive scripts (36)

These archive scripts ship in the world but have no mob, obj, or room pointing at them.

**Mobs don't exist in the current world (15):**
- `aki_kuroda.lua` (12915)
- `autodraw`
- `bane.lua` (1408)
- `beholder.lua` (12000)
- `bradle`
- `conjured`
- `creation`
- `drake`
- `ettin`
- `fire_ant.lua` (1700)
- `fire_ant_larva.lua` (1702)
- `forester.lua` (9160)
- `gazer`
- `keep_sorcerer.lua` (1404)
- `kelpie`

These belong to zones or mobs that never made the C→Go port. They are dead letters. **Recommend: leave in place for archaeological reference but mark with a header comment noting the missing mob.**

**Utility / no specific mob binding (10):**
- `golem_to_crate` (counterpart to `golem_from_crate`, part of golem workflow)
- `guard_captain` (generic)
- `jailguard`
- `memory_moss`
- `mercenary`
- `merchant_walk`
- `miller`
- `neckbreak`
- `paralyse`
- `porcupine`

These are general-purpose templates. They could be wired to mobs if the design called for it. **Recommend: document as templates in a `lib/world/scripts/templates.md` index.**

**Named NPCs not in world (8):**
- `bane.lua` (1408)
- `pyros.lua` (1410)
- `valoran.lua` (1407)
- `sungod.lua` (10205)
- `phoenix.lua` (1401)
- `sandstorm`
- `seiji`
- `town_teleport`

These appear to belong to zones cut during the port. Same disposition as the missing-mob group above.

**Generic templates (3):**
- `paladin.lua`
- `thornslinger.lua`
- `weatherworker.lua`

Behaviour patterns that could be bound to many mobs. Document as templates.

---

## 6. Recommendations

### 6.1 Immediate (this sprint)

1. **Fix mob 1314** — change `Script:` line from `enchanter` to `guardian`. Verify behaviour in-game against archive.
2. **Document the wired/unwired terminology** in `docs/scripting.md` (or create that doc). Retire "active vs archived."
3. **Add header comments to dead-letter scripts** — the 23 scripts in section 5.2 whose mobs don't exist. Mark each one clearly so future readers don't waste time tracing missing bindings.

### 6.2 Medium-term

4. **Create `lib/world/scripts/INDEX.md`** — a single canonical list of every script, its binding (mob/obj/room vnum), and its purpose. The current state requires triangulating across three sources (archive, deployed pool, mob files).
5. **Audit `enchanter.lua` usage** — verify every mob currently running this script actually behaves like an enchanter, not a guardian or something else.
6. **Consider whether `aki_kuroda`, `keep_sorcerer`, `phoenix`, and `valoran` zones can be ported** — these had named NPCs with bespoke scripts and represent lost world content.

### 6.3 For Reek

Reek's strength is grep-pattern review. Reek's weakness is behavioural parity. **Reek should add the following passes to the crawl rotation:**

1. **`Script:` line audit** — for every mob with a `Script:` line, verify the named script exists in `lib/world/scripts/mob/`. Flag missing scripts.
2. **Archive cross-check** — for every script in `docs/archive/scripts_full_dump.txt` that names a specific mob, verify (a) the mob exists in the world and (b) the mob's current `Script:` line matches the archive. This is exactly the pattern that found LUA-002/003/004 — and would have found LUA-001 if it had been run.
3. **Behavioural drift** — diff each deployed script against its archive counterpart. Flag any script where the deployed version has lost behaviour from the archive. "Simplified:" comments are red flags.
4. **Unwired script check** — list all scripts in `lib/world/scripts/` that no entity references. These aren't all bugs, but they should be reviewed for whether they're templates (keep), dead letters (annotate), or pending bindings (fix).
5. **Don't grade on script count or LOC.** Grade on parity with the archive.

---

## 7. For the Architect

The bugs found and fixed during this review are MEDIUM-severity behavioural drifts. The world loads, the mobs spawn, and from a metrics perspective everything looks fine. But mob 11706 (the golem) was running `sorcery.lua` instead of its golem-from-crate workflow — players walking into that zone would have seen a golem behave like a sorcerer, which is exactly the kind of "the world feels off" bug that doesn't show up in test runs.

The outstanding LUA-001 (mob 1314 / guardian) is the same class of bug and should be fixed before the next zone reset cycle hits players who care about that area.

The deeper finding is that **the script-to-mob binding has no automated verification.** Whoever wired the scripts during the port did most of them correctly, but the four that drifted drifted silently. This is exactly the kind of regression that compounds over time. A periodic Reek pass (recommendation 6.3.2 above) would catch these going forward.

Three of four found, one outstanding. Not bad. Could be better. The trust is mostly intact, but the verification step needs to be automated, not vibes.

---

**Status:** Report complete. LUA-001 pending Architect approval to commit fix.
**Next action:** Post triage summary to `#dark-pawns` with reference to this report.
