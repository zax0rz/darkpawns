---
tags: [active]
---

# Dark Pawns — World Restoration Swarm Plan

> **Goal:** Restore 92 RESTORE + 41 CANDIDATE scripts using parallel K2.6 agent swarms, QA agents filing bugs, GLM-5.1 fixing them. BRENDA and BRENDA's memory work proceeds in parallel on a separate track.  
> **Model:** K2.6 for restoration (bounded, mechanical, clear spec). QA agents for testing. GLM-5.1 for bug fixes.  
> **Prime Directive:** Every agent reads [[CLAUDE]] first. Every script ported against original C + Lua source.

---

## Current State

**Already ported (12):** bane, banker, cityguard, cleric, clerk, creation, dracula, fighter, magic_user, pyros, sorcery, valoran

**Engine functions — IMPLEMENTED:** act, do_damage, say, emote, action, oload, mload, extobj, extchar, number, send_to_room, strlower, strfind, strsub, gsub, getn, tonumber, tostring, log, raw_kill, spell, tport, is_fighting, dofile, call, round, aff_flagged, plr_flags, obj_list

**Engine functions — STUBBED (blocking some scripts):** has_item, obj_in_room, objfrom, objto, obj_extra, create_event, tell, plr_flagged, cansee, isnpc

> Stubs return false/nil. Scripts that depend heavily on them will partially work but need engine fixes. Flag these in PRs.

---

## Priority Tiers

### Tier 1 — Engine Unlocks (implement stubs first, unblocks everything else)
These are Go engine changes, not Lua ports. K2.6 handles these too.

| Stub | Used by | Complexity |
|---|---|---|
| `create_event` | bane, valoran, beholder, brain_eater, many others | Medium — needs timer/event queue |
| `tell` | many quest/social scripts | Low — just route to session |
| `plr_flagged` | cityguard, breed_killer, dracula, werewolf | Low — check player flags |
| `cansee` | cityguard, most mob AI | Low — distance/visibility check |
| `isnpc` | nearly everything | Trivial — check mob vs player |
| `has_item` | clerk, shopkeeper, crafting chain | Medium — inventory search |
| `obj_in_room` | donation, scavenger mobs | Low — room item check |
| `objfrom` / `objto` | crafting, donation, shopkeeper | Medium — item transfer |

**Assign:** 2 K2.6 agents, each taking 4 stubs. PRs against `feat/engine-stubs-1` and `feat/engine-stubs-2`.

---

### Tier 2 — Combat AI (every mob fight gets better)
No dependencies on stubs. Port now, works immediately.

| Script | Source | Notes |
|---|---|---|
| `dragon_breath.lua` | lib/scripts/mob/archive/ | Central handler for all dragon mobs |
| `anhkheg.lua` | lib/scripts/mob/archive/ | Acid blast in combat |
| `drake.lua` | lib/scripts/mob/archive/ | Fireball in combat |
| `bradle.lua` | lib/scripts/mob/archive/ | Poison on bite |
| `caerroil.lua` | lib/scripts/mob/archive/ | Self-heals in combat |
| `ettin.lua` | lib/scripts/mob/archive/ | Boulder throw, heavy damage |
| `snake.lua` | lib/scripts/mob/archive/ | Poison attack |
| `troll.lua` | lib/scripts/mob/archive/ | Regeneration |
| `mindflayer.lua` | lib/scripts/mob/archive/ | Psionic attacks |
| `paladin.lua` | lib/scripts/mob/archive/ | Holy combat spells |

**Assign:** 2 K2.6 agents, 5 scripts each. Independent — no shared dependencies.

---

### Tier 3 — Economy & NPC Services (world feels alive)
Depends on: `has_item`, `objfrom`, `objto` (Tier 1 stubs). Start after Tier 1 lands.

| Script | Source | Notes |
|---|---|---|
| `shopkeeper.lua` | lib/scripts/mob/archive/ | Buy/sell items |
| `shop_give.lua` | lib/scripts/mob/archive/ | Shop give handler |
| `identifier.lua` | lib/scripts/mob/archive/ | Identifies items for gold |
| `stable.lua` | lib/scripts/mob/archive/ | Mount rental |
| `merchant_inn.lua` | lib/scripts/mob/archive/ | Inn room rental |
| `merchant_walk.lua` | lib/scripts/mob/archive/ | Wandering merchant |
| `teacher.lua` | lib/scripts/mob/archive/ | Skill training |
| `recruiter.lua` | lib/scripts/mob/archive/ | Guild recruitment |
| `pet_store.lua` | lib/scripts/mob/archive/ | Pet purchase |
| `remove_curse.lua` | lib/scripts/mob/archive/ | Curse removal service |

**Assign:** 2 K2.6 agents, 5 scripts each.

---

### Tier 4 — Environmental & Ambient (world feels dangerous and textured)
Depends on: `create_event` (Tier 1). Start after Tier 1 lands.

| Script | Source | Notes |
|---|---|---|
| `donation.lua` | lib/scripts/mob/archive/ | Donation room cleanup |
| `eq_thief.lua` | lib/scripts/mob/archive/ | Steals from inventory |
| `aurumvorax.lua` | lib/scripts/mob/archive/ | Eats gold items |
| `brain_eater.lua` | lib/scripts/mob/archive/ | Beheads corpses, gains levels |
| `beholder.lua` | lib/scripts/mob/archive/ | Multi-ray attacks |
| `memory_moss.lua` | lib/scripts/mob/archive/ | Copies player spells |
| `medusa.lua` | lib/scripts/mob/archive/ | Petrification gaze |
| `sandstorm.lua` | lib/scripts/mob/archive/ | Area damage effect |
| `phoenix.lua` | lib/scripts/mob/archive/ | Resurrects on death |
| `souleater.lua` | lib/scripts/mob/archive/ | Drains levels |

**Assign:** 2 K2.6 agents, 5 scripts each.

---

### Tier 5 — Crafting & Quest Chains (economy depth)
Depends on: `objfrom`, `objto`, `has_item`, `create_event`.

| Chain | Scripts |
|---|---|
| Wheat→Flour→Dough→Bread | farmer_wheat, miller, baker_flour, baker_dough |
| Crystal armor | crystal_forger |
| Dragon scale armor | dragon_forger |
| Enchanting | enchanter |
| Golem assembly | golem_from_crate, golem_miner, golem_to_crate |
| Tattoo parlor | tattoo |

**Assign:** 2 K2.6 agents by chain — each agent owns one complete crafting chain end-to-end.

---

### Tier 6 — Ambient / Flavor (CANDIDATE scripts)
Low priority. Safe to do in parallel with anything. Pure flavor.

| Script | Notes |
|---|---|
| `beggar.lua` | Ambient begging strings |
| `citizen.lua` | Reacts to bribes |
| `carpenter.lua` | Ambient strings |
| `towncrier.lua` | Zone announcements |
| `minstrel.lua` | Songs and performance |
| `mime.lua` | Silent emote reactions |
| `singingdrunk.lua` | Drunk ambient behavior |
| `bearcub.lua` | Finds mama bear |
| `dog.lua` | Already active — verify |

**Assign:** 1 K2.6 agent, batch of 8.

---

## QA Agent Layer

After each Tier lands, 2 QA agents run:

**QA Agent 1 — Automated test runner:**
- Runs `go test ./pkg/scripting/...` 
- Triggers each restored script against known test fixtures
- Reports pass/fail with error output

**QA Agent 2 — Behavioral reviewer:**
- Reads the original Lua source
- Reads the ported version  
- Compares behavior descriptions
- Files GitHub issues for any divergence from original intent

**Bug fix:** GLM-5.1 as the fix agent. QA files issue → GLM-5.1 assigned → fixes → PR. GLM-5.1 is slow but thorough — right model for "read the bug report, trace the code, fix it properly."

---

## Execution Model

```
Session 1 (now):
  - 2 agents: Tier 1 engine stubs (parallel, split 4 stubs each)

Session 2 (after Tier 1 merges):
  - 2 agents: Tier 2 combat AI (10 scripts, parallel)
  - 2 QA agents: verify Tier 1

Session 3:
  - 2 agents: Tier 3 economy (parallel, after Tier 1 confirmed)
  - 2 agents: Tier 4 environmental (parallel, after Tier 1 confirmed)
  - 2 QA agents: verify Tier 2

Session 4:
  - 2 agents: Tier 5 crafting chains
  - 1 agent: Tier 6 ambient batch
  - GLM-5.1: fix any open bugs from QA

Session 5+:
  - Remaining RESTORE scripts (50+ remaining)
  - GLM-5.1 bug fixing pipeline
  - Narrative memory work (separate track, BRENDA + BRENDA)
```

---

## Branch Strategy

```
main
├── feat/engine-stubs-1       (Tier 1, agent A)
├── feat/engine-stubs-2       (Tier 1, agent B)  
├── feat/combat-ai-1          (Tier 2, agent A)
├── feat/combat-ai-2          (Tier 2, agent B)
├── feat/economy-scripts-1    (Tier 3, agent A)
├── feat/economy-scripts-2    (Tier 3, agent B)
... etc
```

One branch per agent. No shared branches. QA creates issues, not PRs.

---

## Agent Prompt Template

Every restoration agent gets:

```
Read CLAUDE.md first. Then read the original source at lib/scripts/mob/archive/<script>.lua
from the rparet/darkpawns upstream (git show origin/master:lib/scripts/mob/archive/<script>.lua).

Port <script>.lua to test_scripts/mob/archive/<script>.lua for the Go scripting engine.

Rules:
- Port faithfully. Do not invent behavior not in the original.
- Document the source line for every non-trivial piece of logic.
- If a function you need is stubbed in pkg/scripting/engine.go, use it anyway and note it.
- Write an integration test in pkg/scripting/integration_test.go for your script.
- go build ./... must pass.

Commit: 'feat(scripts): port <script>.lua'
Push and open PR against zax0rz/darkpawns main.
```

---

## Parallel Track: BRENDA & Memory

While the swarm runs, BRENDA's work continues independently:
- `agent_narrative_memory` Postgres schema
- Memory bootstrap in auth response
- Valence-based salience scoring
- Session consolidation
- Research log entries

These never touch the Lua scripts. Zero conflict.

---

*Plan written 2026-04-21. Start with Tier 1 engine stubs — they unlock everything else.*
