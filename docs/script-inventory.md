# Dark Pawns Script Inventory
_Reviewed via Gemini 2026-04-20. Source: `lib/scripts/` (179 total)._

---

## Summary

| Status | Count | Description |
|--------|-------|-------------|
| ACTIVE | 14 | Live production scripts running at server shutdown |
| DEPENDENCY | 16 | Core engine logic, combat AI matrices, shared helpers required by other scripts |
| RESTORE | 92 | High-value mechanics: crafting economy, multi-stage quests, guild/city systems, environmental threats |
| CANDIDATE | 41 | Ambient behaviors, basic combat spell loops, cosmetic flavor mobs. Safe to implement, low priority |
| INCOMPLETE | 5 | Broken logic loops or messy global state (e.g. autodraw.lua) |
| SHELVE | 11 | Deprecated iterations or one-off gag scripts |

---

## Phase 3 Restoration Priority

### 1. Core Engine (nothing else works without these)
- `globals.lua`
- `mob/no_move.lua`
- `mob/assembler.lua`

### 2. Combat AI Matrices (prevents mobs from being lifeless punching bags)
- `mob/archive/fighter.lua`
- `mob/archive/magic_user.lua`
- `mob/archive/cleric.lua`
- `mob/archive/sorcery.lua`

### 3. Newbie & Economy Pipeline
- `mob/archive/creation.lua` — character creation quiz, stat/lifestyle determination
- `mob/archive/clerk.lua` — starting gear for new characters
- `mob/archive/banker.lua` — starting gold via lifestyle bond certificates

### 4. Law & Order (keeps starting cities functional)
- `mob/archive/take_jail.lua`
- `mob/archive/guard_captain.lua`
- `mob/archive/breed_killer.lua`
- `mob/archive/cityguard.lua`

### 5. Crafting Chain (closed-loop economy proof)
- `mob/archive/farmer_wheat.lua`
- `mob/archive/miller.lua`
- `mob/archive/baker_flour.lua`
- `mob/archive/baker_dough.lua`

---

## Notable RESTORE Scripts

| Script | Triggers | Description |
|--------|----------|-------------|
| `mob/archive/bane.lua` | fight, death, greet, onpulse_pc | Daemon summoning ceremony Part 1 — requires Valoran + 5 players |
| `mob/archive/valoran.lua` | fight, death, greet, onpulse_pc | Daemon summoning ceremony Part 2 |
| `mob/archive/dracula.lua` | fight, oncmd | Turns players into vampires via eye contact |
| `mob/archive/dragon_breath.lua` | fight, greet | Central handler mapping dragon VNums to breath attacks |
| `mob/archive/brain_eater.lua` | onpulse_all | Beheads corpses to increase its own level and HP |
| `mob/archive/eq_thief.lua` | onpulse_pc | Silently steals items from player inventories |
| `mob/archive/aurumvorax.lua` | onpulse_all | Eats gold items or attacks players carrying gold |
| `mob/archive/beholder.lua` | oncmd, onpulse_pc, fight | Disrupts casting, fires sleep/charm/curse rays |
| `mob/archive/enchanter.lua` | sound, ongive | Enchants weapons/armor for elemental dust |
| `mob/archive/donation.lua` | onpulse_all | Keeps donation rooms clean — sorts or junks items |
| `mob/archive/crystal_forger.lua` | oncmd | Exchanges crystalline chunks for crystal armor |
| `mob/archive/dragon_forger.lua` | oncmd | Exchanges dragon scales for dragon scale armor |
| `mob/archive/blacksmith.lua` (archive) | sound, ongive | Old armor assembler — superseded by `mob/212/blacksmith.lua` → SHELVE |
| `mob/archive/cabinguard.lua` | oncmd | Attacks if captain's door opened without key |
| `mob/archive/aki_kuroda.lua` | sound, oncmd | Attacks players who try to open a specific painting |

## Notable CANDIDATE Scripts

| Script | Triggers | Description |
|--------|----------|-------------|
| `mob/archive/elven_prostitute.lua` | sound, bribe | Pulls player into shadows for coins |
| `mob/archive/beggar.lua` | sound | Periodic begging ambient strings |
| `mob/archive/citizen.lua` | sound, bribe | Emotes and reacts to bribes |
| `mob/archive/carpenter.lua` | sound | Carpenter ambient strings |
| `mob/archive/bearcub.lua` | onpulse_all | Wanders until it finds mama bear |
| `mob/archive/cuchi.lua` | oncmd | Oro's pet — grants gold/stats if patted |
| `mob/archive/anhkheg.lua` | fight | Casts acid blast during combat |
| `mob/archive/drake.lua` | fight | Casts fireball in combat |
| `mob/archive/bradle.lua` | fight | Poisons player on bite |
| `mob/archive/caerroil.lua` | fight | Self-heals during combat |
| `mob/archive/ettin.lua` | fight | Hurls boulders for heavy damage |

## SHELVE

- `mob/archive/autodraw.lua` — overly complex lottery system, messy global state
- `mob/archive/bhang.lua` — one-off gag script
- `mob/archive/blacksmith.lua` (archive) — replaced by active `mob/212/blacksmith.lua`
- 8 others (deprecated iterations, redundant scripts)

---

## Trigger Types Used Across Active Scripts

| Trigger | Scripts Using It |
|---------|-----------------|
| `sound` | healer, blacksmith, highpriest, gatekeeper, dog, + many RESTORE/CANDIDATE |
| `ongive` | healer, blacksmith, highpriest, gatekeeper, + banker, clerk, enchanter, forgers |
| `oncmd` | hisc (no_move), pattern_3065 (tport), portal, + beholder, cabinguard, etc. |
| `onpulse` | pattern_dmg |
| `onpulse_all` | never_die, + donation, brain_eater, aurumvorax, bearcub |
| `onpulse_pc` | cityguard, take_jail, eq_thief, + others |
| `fight` | hisc (via cleric.lua), + all combat AI matrix scripts |
| `ondeath` | bane, valoran |
| `onenter` | archive only — not in active set |
| `ongive` | assembler pattern — used by all crafting/quest mobs |
| `bribe` | dog, citizen, elven_prostitute |
| `greet` | dracula, dragon_breath |

---

_Full script dump: `docs/scripts_full_dump.txt`_
