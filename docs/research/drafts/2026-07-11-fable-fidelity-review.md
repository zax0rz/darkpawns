# C→Go Port Fidelity Review — 2026-07-11 (Fable)

Method: every finding below was verified by reading the C source in `src/*.c` directly and
confirming the current Go behavior in `pkg/`. Live-path analysis: melee combat runs through
`pkg/combat/engine.go processCombatPair`; skills run through `pkg/game/damage_stubs.go
doDamage/DoSpellDamage`; spells run through `pkg/spells/damage_spells.go inflictDamage →
combat.TakeDamage`; movement runs through `pkg/game/world.go MovePlayer` (session `cmdMove`),
NOT `act_movement.go doSimpleMove`. Several fight.c ports in `pkg/combat/fight_core.go`
(the full `damage()` port) are **unreachable dead code** on the paths noted below.

Excluded per instructions: DP-1008 (animate dead), DP-1015 (charm-skip mask), 07-05 audit
streams 1–5. Known-intentional divergence not reported: position damage multiplier uses float
math per DP-515 (`pkg/combat/formulas.go:533`).

---

### F1. Wounded states and death threshold eliminated — HP clamps at 0, death at HP<=0  [HIGH]
- **C:** src/fight.c:186-201 `update_pos()` — `GET_HIT <= -11 → POS_DEAD; <= -6 → POS_MORTALLYW; <= -3 → POS_INCAP; else POS_STUNNED`; fight.c:1484 `GET_HIT(victim) -= dam` (HP goes negative); src/limits.c:510-513 — incapacitated chars bleed 1/tick, mortally wounded 2/tick until dead.
- **Go:** pkg/game/player_affects.go:122-129 and pkg/game/mob.go:294-301 — `TakeDamage` clamps `Health/CurrentHP` at 0; pkg/combat/engine.go:367,429 — engine treats `GetHP() <= 0` as death immediately.
- **Divergence:** C has a 12-HP "dying" band (stunned/incap/mortally wounded) with bleed-out, rescue-the-dying gameplay, and attackers getting the ×2–×3 position multiplier on downed victims. Go kills instantly at 0.
- **Player impact:** Characters die ~11 HP earlier; no stunned/incap/mortally-wounded states ever occur; healing a dying groupmate is impossible; PointUpdate's incap/mortally bleed branches (limits_condition.go:206-220) are dead code.
- **Fix sketch:** Allow negative HP down to -11 for players and mobs, derive position via the existing `GetPositionFromHP`, and only trigger death handling at `<= -11` (or POS_DEAD).

### F2. Spell kills bypass the entire kill pipeline (no XP, no kill credit, wrong death penalty)  [HIGH]
- **C:** src/magic.c:827 — `mag_damage()` calls the same `damage(ch, victim, dam, spellnum)` as weapon hits; src/fight.c:1634-1712 — that function awards XP/group gain, autogold/autoloot, kill counters, alignment change, and calls `die_with_killer()` (exp/37 penalty, killer-aware corpse messages).
- **Go:** pkg/spells/damage_spells.go:255-268 — `inflictDamage` → `combat.TakeDamage`, then on `GetHP() <= 0` calls `HandleSpellDeath` → pkg/game/death.go:230-236 `HandleNonCombatDeath(victim)` with **killer=nil, attackType=-1**. The death-award block inside `combat.TakeDamage` (fight_core.go:412-500) is unreachable because HP clamps at 0 so `GetPositionFromHP` never returns `PosDead` (needs HP <= -11).
- **Divergence:** A mob killed by a spell yields zero XP, no gold split, no kill counter, no autoloot, no MobKilled event, and a generic corpse description (fire spells never make "charred corpse"). A player killed by a spell suffers the **non-combat** penalty exp/3 (fight.c:628 `die()`) instead of exp/37 (fight.c:589 `die_with_killer()`), with no PK bookkeeping.
- **Player impact:** Casters cannot level from kills at all; spell PK is 12× more punishing to the victim than C; kill-milestone counters never advance for casters.
- **Fix sketch:** Have `inflictDamage` call `world.HandleDeath(victim, killer, spellNum)` (the same DeathFunc path the melee engine uses) instead of `HandleSpellDeath`/`HandleNonCombatDeath`.

### F3. Kill XP bypasses gain_exp: no level-ups from combat, no max_exp_gain cap, no one-level clamp  [HIGH]
- **C:** src/limits.c:297-315 `gain_exp()` — `gain = MIN(max_exp_gain, gain)` (config.c:81 `max_exp_gain = 100000`), `gain = MIN(max_exp-1, gain)` ("can only level one time!"), then auto-advances level with "You advance to level %d!".
- **Go:** pkg/game/party.go:199,253 — `AwardMobKillXP` calls `p.AddExp(xp)` (pkg/game/player_identity.go:91-98), which only adds and floors at 0. The faithful port `World.GainExp` (pkg/game/limits_exp.go:131-205, with cap + auto-level) exists but its only production caller is item_donate.go:156. Nothing else calls `AdvanceLevel` except character creation (player.go:317).
- **Divergence:** Combat XP never triggers level advancement, is never capped at 100k per kill, and can bank multiple levels' worth of XP silently.
- **Player impact:** Players killing mobs never gain levels (must rely on whatever non-combat path exists); one huge kill can exceed the C per-kill cap.
- **Fix sketch:** Route `AwardMobKillXP` member payouts through `World.GainExp(p, xp)` instead of `p.AddExp`.

### F4. XP share formula is fabricated — proportional scaling replaces C's tiered penalty, and fighting up grants a bonus multiplier  [HIGH]
- **C:** src/fight.c:659-685 `calc_level_diff()` — `share = MIN(max_exp_gain, MAX(1, base))`; if attacker is higher level: solo gets `level_diff -= 2` slack, then `>15 → -70%`, `>10 → -50%`, `>5 → -30%` (full XP within 5 levels); plus flat `-20%` for anyone over level 20. Never a bonus for fighting up.
- **Go:** pkg/game/party.go:186-198 (solo) and 236-252 (group) — `xp = xp * victimLevel / killerLevel` when higher level, and `xp = xp * (2*victimLevel - killerLevel) / victimLevel` when LOWER level (a bonus up to ~2× victim exp). No cap, no over-20 penalty, no solo slack. The comment attributes this to "fight.c perform_group_gain()" but no such formula exists anywhere in src/fight.c.
- **Divergence:** Entire XP curve differs; C gives full XP within 5 levels then steps down; Go scales continuously and *inflates* XP when fighting higher-level mobs.
- **Player impact:** Leveling economy is wrong across the board; "punch up for 1.5-2× XP" exploit that doesn't exist in C.
- **Fix sketch:** Replace with a port of `calc_level_diff()` (it already exists correctly as `combat.CalcLevelDiff`, fight_core.go:856-885 — call that).

### F5. Melee and skill damage skip every damage() modifier: sanctuary, protect evil/good, race-hate, immortal immunity, 3000 cap, peaceful rooms  [HIGH]
- **C:** src/fight.c:1466-1483 — inside `damage()`: race-hate `dam += GET_LEVEL(ch)`, `AFF_SANCTUARY → dam /= 2`, `AFF_PROTECT_EVIL/GOOD → dam -= GET_LEVEL(victim)/4`, immortal victims `dam = 0`, `dam = MAX(MIN(dam, 3000), 0)`; plus fight.c:1336-1341 peaceful-room block, 1410-1440 charmed-pet retarget & high-level mob target switch, 1370-1401 jail-guard intercept. ALL weapon/skill/spell damage funnels through this one function.
- **Go:** pkg/combat/engine.go:409-415 — `processCombatPair` computes `CalculateDamage` (no such modifiers, see formulas.go:512-553) and calls `defender.TakeDamage(damage)` directly; pkg/game/damage_stubs.go:129-146 — `doDamage`/`DoSpellDamage` (bash, kick, backstab, spec procs) likewise call `TakeDamage` raw. Only the spell path (`combat.TakeDamage`, fight_core.go:298-328) applies them.
- **Divergence:** Sanctuary halves only spell damage; melee (the dominant damage source) and all skills ignore sanctuary, protection auras, race-hate weapons, the 3000 cap, immortal invulnerability, and peaceful-room protection mid-fight.
- **Player impact:** Sanctuary — the signature cleric buff — does nothing against melee; immortals can be melee'd to death; protect-from-evil is inert; jail guards/charm retarget behaviors never happen.
- **Fix sketch:** Make `processCombatPair` and `doDamage` route the computed damage through `combat.TakeDamage` (or extract its modifier block into a shared function applied on all three paths).

### F6. Player parry/dodge never fire in melee (GetSkill wiring bug), and the Go mechanic diverges from C anyway  [HIGH]
- **C:** src/fight.c:1949-1963 — parry is a once-per-round check `number(0,10000) <= GET_SKILL(ch, SKILL_PARRY)` (~1% at skill 100) that sets `IS_PARRIED(FIGHTING(ch))`; fight.c:1999-2004 — being parried reduces the opponent's **attack count** (`attacks += dex_app[...].defensive` or `attacks--`). Dodge (1965-1973) is NPC-only via `AFF_DODGE`, `number(0,100) < GET_LEVEL(ch)`.
- **Go:** pkg/game/combat_wire.go:72-77 — `cb.GetSkill = func(name string, skillNum int) int { ... return p.GetSkill(name) }` passes the **player's own name** as the skill name, so it always returns 0; therefore `CheckParry`/`CheckDodge` (pkg/combat/formulas.go:652-722) always fail and the engine's parry/dodge branches (engine.go:377-407) are dead. If the wiring were fixed, the Go formula (`number(1,101) > skill` **per attack**, negating the hit) would make skill-100 parry deflect ~99% of individual attacks vs C's ~1% per round.
- **Divergence:** Parry/dodge skills are currently nonfunctional in melee; the intended Go mechanic is also a different (vastly stronger) model than C. The same broken callback zeroes the wimpy retreat/escape skill check in fight_core.go:397-399.
- **Player impact:** Training parry/dodge is wasted practice sessions; fixing only the callback would swing to near-immunity.
- **Fix sketch:** Fix the callback to map `skillNum` → skill name and look that up; then re-model parry per C (per-round roll vs 10000, reduce attacker's attack count).

### F7. Death traps don't kill  [HIGH]
- **C:** src/act.movement.c:289-302 — entering a `ROOM_DEATH` room: `log_death_trap(ch); death_cry(ch); extract_char(ch);` (character and mount destroyed).
- **Go:** live path `World.MovePlayer` (pkg/game/world.go:881-965) has **no ROOM_DEATH check at all**; secondary path pkg/game/act_movement.go:339-343 does `ch.TakeDamage(ch.GetHP() + 1)` (clamped to 0 HP by player_affects.go:126-128) and returns — no death handler is invoked, so the player stands at 0 HP and regens back on the next PointUpdate tick (limits_condition.go:150-160).
- **Divergence:** DT rooms are harmless on the live path and a no-op scratch on the dead path.
- **Player impact:** A core hazard of the world (builder-placed death traps) does nothing.
- **Fix sketch:** In `MovePlayer`, after a successful move, check `roomFlagDeath` and call `w.HandleNonCombatDeath(p)` (plus the DT mudlog).

### F8. Mob spell affects never expire — permanent sleep/blind/curse/poison on mobs  [HIGH]
- **C:** src/magic.c:431-457 `affect_update()` — iterates `character_list` (players **and** NPCs), decrements every affect duration each mud hour, removes expired ones with wear-off messages.
- **Go:** pkg/game/affect_update.go:35-77 — `AffectUpdate` iterates `w.players` only. Mob affects are parked in `CustomData["affect_<spell>"]` with a comment "the affect tick system will handle duration-based removal" (pkg/game/mob.go:901-920), but no ticker ever reads those keys (only AddAffect/RemoveAffectBySpell touch them).
- **Divergence:** Any duration-limited debuff landed on a mob (blindness, curse, poison, sleep, slow) lasts until the mob dies or is explicitly dispelled.
- **Player impact:** Exploit: blind or sleep a hard mob once and it is permanently crippled; poison ticks on a mob forever.
- **Fix sketch:** Extend `AffectUpdate` to iterate active mobs, decrement the `engine.Affect` durations stored in CustomData, and call `RemoveAffectBySpell` on expiry.

### F9. Live movement-cost table is wrong for nearly every sector; immortals pay movement  [MED]
- **C:** src/constants.c:1345-1363 `movement_loss[]` = {2,2,3,4,5,7,5,6,2,6,8,6,6,6,6,4} indexed by src/structs.h:93-108 (`SECT_UNDERWATER=8`, `SECT_FLYING=9` — note the C comments at constants.c:1355-1356 are swapped relative to the enum; the *actual* runtime costs are UNDERWATER=2, FLYING=6). Cost = average of src+dst (act.movement.c:135-136); immortals exempt (act.movement.c:210-211).
- **Go:** live path pkg/game/world.go:967-993 `sectorMoveCost` = 1,1,2,3,4,6,4,4,4,1 with `default: 1` — every value differs from C; DESERT (C:8), SWAMP (C:4) and the elemental planes (C:6) all fall through to 1; `MovePlayer` (world.go:941-942) charges everyone including immortals. Secondary table pkg/game/act_movement.go:74-91 is close to C but encodes UNDERWATER=6/FLYING=2, i.e. it follows the erroneous C *comments* instead of the C *behavior*.
- **Divergence:** All terrain movement costs are roughly halved and flattened; deserts cost 1 instead of 8.
- **Player impact:** Movement points barely matter; terrain strategy (fly over mountains/desert) is gone.
- **Fix sketch:** Single shared `movementLoss` table matching constants.c actual indices; use `(src+dst)>>1`; skip deduction for level >= LVL_IMMORT.

### F10. PK bookkeeping missing on the live death path: no OUTLAW flag, no PKs/Deaths counters, no PK log  [MED]
- **C:** src/fight.c:1671-1689 — on player death: PK mudlog, `SET_BIT_AR(PLR_FLAGS(ch), PLR_OUTLAW)` on the killer if victim wasn't outlaw, `GET_PKS(ch)++`, `GET_DEATHS(victim)++`, `GET_LAST_DEATH(victim) = time(0)`.
- **Go:** pkg/game/death.go:377-518 `handlePlayerDeath` — none of this happens. The port of it exists only in the unreachable `combat.TakeDamage` death block (fight_core.go:464-482, see F2) wired via combat_wire.go:126-289.
- **Divergence:** Player-killers are never flagged OUTLAW; PKs/Deaths/LastDeath stats never change.
- **Player impact:** The entire outlaw system (peaceful-room exemptions at fight.c:1336, low-level protection bypass at fight.c:1351, jail guards) is inert; score stats frozen.
- **Fix sketch:** Move the fight.c:1671-1689 block into `HandleDeath`'s player branch using killer identity already in hand.

### F11. Alignment never changes from kills  [MED]
- **C:** src/fight.c:484-502 `change_alignment()` — `GET_ALIGNMENT(ch) += (-GET_ALIGNMENT(victim) - GET_ALIGNMENT(ch)) >> 4` on every non-neutral kill, called from the death block (fight.c:1667, 704).
- **Go:** `combat.ChangeAlignment` (fight_core.go:152-172) is correct but only called from the unreachable TakeDamage death block / legacy GroupGain. The live path (`HandleDeath` → `AwardMobKillXP`, pkg/game/party.go:85) never touches alignment.
- **Divergence:** Killing good/evil mobs has no alignment consequence.
- **Player impact:** Alignment-gated equipment, protection spells, dispel evil/good, and align-aggro mobs all interact with a permanently static alignment.
- **Fix sketch:** Call `combat.ChangeAlignment(killer, victim)` (or a game-layer equivalent) from `HandleDeath`/`AwardMobKillXP` per member.

### F12. Bash: mobs take no wait-state, keep attacking, and stay knocked down for the whole fight  [MED]
- **C:** src/act.offensive.c:489-495 — successful bash: `GET_POS(vict) = POS_SITTING; WAIT_STATE(vict, PULSE_VIOLENCE*2)`; src/fight.c:1975-1987 — a waiting mob gets `attacks = 0` while `GET_MOB_WAIT > 0`, and stands back to POS_FIGHTING once the wait expires (mid-combat).
- **Go:** pkg/command/skill_commands.go:1622-1626 — `WaitTarget` is only applied when the target is a `*game.Player`; mobs get no wait. pkg/combat/engine.go:354-437 — `processCombatPair` never checks the attacker's position or wait state, so a sitting mob swings at full attack count; engine.go:142-160 — `StartMobPositionRecovery` explicitly skips mobs that are fighting, so a bashed mob never stands up during combat.
- **Divergence:** Opposite-direction errors: bash doesn't cost the mob its attacks (weaker than C), but the mob stays sitting for the entire fight eating the position damage multiplier every round (stronger than C).
- **Player impact:** Bash-once = permanent 1.33× damage amplifier on the mob for the whole fight, while the "stun" effect players expect does nothing.
- **Fix sketch:** Add a mob wait-state field consumed by `GetAttacksPerRound`/`processCombatPair` (attacks=0 while waiting), and stand fighting mobs back up when it expires, per fight.c:1982-1987.

### F13. Backstab: multiplier not truncated to int, and no to-hit roll on success path  [MED]
- **C:** src/class.c:719-728 — `int backstab_mult(int level) { return ((level*.2)+1); }` — returns **int**, so the multiplier truncates (level 14 → 3, level 19 → 4); src/act.offensive.c:224-230 — on a passed skill roll C calls `hit(ch, vict, SKILL_BACKSTAB)`, which still runs the full THAC0 d20 to-hit roll (fight.c:1825-1830) and can miss.
- **Go:** pkg/game/skill_combat.go:97-101 — `mult := combat.BackstabMult(ch.GetLevel())` (float64, fight_core.go:825-833) and `dam = int(float64(dam) * mult)` — fractional multiplier applied; no to-hit roll — success on the skill roll always lands.
- **Divergence:** Backstab damage is up to ~27% higher than C at mid levels (e.g. level 14: ×3.8 vs ×3), and never whiffs on the d20 like C.
- **Player impact:** Thief burst damage consistently overtuned relative to the original.
- **Fix sketch:** Truncate the multiplier to int before applying; optionally add the `CalculateHitChance` roll before damage.

### F14. Aggressive/memory/AGGR24 mobs ignore visibility, NOHASSLE, sneak, and peaceful rooms  [MED]
- **C:** src/mobact.c:205-215 — aggro loop skips victims when `!CAN_SEE(ch, vict)`, `PRF_NOHASSLE`, or `AFF_SNEAK && !number(0,3)` (75% skip); mobact.c:267-273 — memory attacks also gated by CAN_SEE/NOHASSLE and peaceful-room; mobact.c:304-311 — AGGR24 requires CAN_SEE, !NOHASSLE, !ROOM_PEACEFUL.
- **Go:** pkg/game/mobact.go:217-260 — aggressive block checks only wimpy and protect-evil/good (no canSee, no NOHASSLE, no sneak skip); mobact.go:314-330 (memory) and mobact.go:367-379 (AGGR24) have none of the gates. (The race-hate block at 268-308 *does* implement them — the others don't.)
- **Divergence:** Invisible/sneaking players and NOHASSLE immortals are attacked by aggro mobs; AGGR24 fires in peaceful rooms.
- **Player impact:** Invisibility and sneak lose their primary defensive purpose; immortals get mobbed.
- **Fix sketch:** Apply the same canSee/NOHASSLE/sneak/peaceful gates used in the race-hate block to the aggressive, memory, and AGGR24 blocks.

### F15. Game-hour ticks run 2.1× fast (30s vs 63s), and mob AI runs on two overlapping tickers  [MED]
- **C:** src/utils.h:135 `SECS_PER_MUD_HOUR 63`; src/comm.c:825-828 — `affect_update()` and `point_update()` run once per 63 real seconds; src/structs.h:633 `PULSE_MOBILE (4 RL_SEC)`.
- **Go:** pkg/game/world.go:194 — `StartPointUpdateTicker(30 * time.Second)`; pkg/engine/gameloop.go:31 `PULSE_TICK = 30s` drives `AffectUpdate` (cmd/server/main.go:239-241). Mob AI: gameLoop `OnMobileActivity` every 4s (matches C) **plus** `StartAITicker` every 10s (pkg/game/ai.go:183, called at main.go:225) both invoke the per-mob activity, ~1.25× the C rate.
- **Divergence:** Regen, hunger/thirst decay, spell durations, corpse decay, and idle timers all run 2.1× faster than C; mob wander/aggro checks fire from two independent loops.
- **Player impact:** Buffs wear off in half the wall-clock time; hunger nags twice as often; double-sourced AI makes wander rates and aggro latency diverge from C.
- **Fix sketch:** Set the tick to 63s (or make both C-relative), and remove one of the two mob-AI drivers.

### F16. Object/corpse decay is driven per-mob-in-room: N mobs = N× decay, zero decay in mob-less rooms  [MED]
- **C:** src/limits.c:529-686 — `point_update()` iterates the global `object_list`, decrementing each corpse/dust/field-object timer exactly once per tick regardless of location (including corpses carried by players, limits.c:598).
- **Go:** pkg/game/limits_condition.go:280 — `w.decayObjectsInRoom(roomVNum)` is called once **per active mob** inside the NPC loop, with no room dedup. A corpse in a room with 5 mobs loses 5 timer ticks per PointUpdate; a corpse in a room with no mobs never decays; carried corpses never decay.
- **Divergence:** Decay rate is a function of local mob population instead of time.
- **Player impact:** Player corpses (and their gear) evaporate in a fraction of the intended 10 ticks in busy rooms, or persist forever in empty ones.
- **Fix sketch:** Hoist decay out of the mob loop: iterate all object-bearing rooms (or a global object list) exactly once per PointUpdate.

### F17. Flee XP penalty: base loss gated to level>10, and no max_exp_loss cap anywhere  [MED]
- **C:** src/act.offensive.c:362-406 — `loss = (GET_MAX_HIT(FIGHTING) - GET_HIT(FIGHTING)) * GET_LEVEL(FIGHTING)` applies at **all** levels on a successful flee; only the extra `500*(level/2.6)` is level>10; src/limits.c:319 — `gain = MAX(-max_exp_loss, gain)` caps any single loss at 500000 (config.c:82).
- **Go:** pkg/session/movement_cmds.go:248-256 — the whole deduction (base + bonus) sits inside `if level > 10`, so fleeing below level 11 is free; `LoseExp` (pkg/game/player_affects.go:186-192) has no cap. Death exp loss (death.go:413-425, exp/3 or exp/37) is likewise uncapped.
- **Divergence:** Low-level flee penalty missing; high-exp characters can lose far more than 500k exp per flee/death.
- **Player impact:** Wrong flee economics at both ends of the level range.
- **Fix sketch:** Apply the base loss unconditionally, gate only the +500*(level/2.6) term, and clamp all exp losses at 500000.

### F18. Carry weight (CAN_CARRY_W) not enforced — only item count  [MED]
- **C:** src/utils.h:448 `CAN_CARRY_W(ch) = str_app[STRENGTH_APPLY_INDEX(ch)].carry_w`; utils.h:543-545 `CAN_GET_OBJ` requires both weight and count headroom.
- **Go:** pkg/game/inventory.go:141-150 — `SetCapacity` implements only `CAN_CARRY_N` (count); comment admits "Weight tracking (CAN_CARRY_W) requires str_app table — implement item count limit only for now". `GetWeight()` exists (inventory.go:129) but nothing checks it on pickup.
- **Divergence:** Strength has no effect on carrying capacity; heavy items are free to hoard up to the count cap.
- **Player impact:** STR loses one of its two core benefits; weight-based puzzles/loot balance void.
- **Fix sketch:** Add carry_w to the str_app table (already in pkg/combat/formulas.go) and check `GetWeight()+obj <= CAN_CARRY_W` in the get/take paths.

### F19. Attack count floor: slowed characters still get 1 attack (C allows 0)  [LOW]
- **C:** src/fight.c:1945-1946,2006-2007 — `AFF_SLOW` does `attacks--` and the sanity check is `if (attacks < 0) attacks = 0` — a slowed level-1..24 player or low-level mob can have **0** attacks in a round.
- **Go:** pkg/combat/formulas.go:630-634 — `if attacks < 1 { attacks = 1 }`.
- **Divergence:** SLOW can never fully deny a round in Go.
- **Player impact:** The slow spell is weaker than C against low-attack targets.
- **Fix sketch:** Clamp at 0, not 1, and let the round loop no-op.

### F20. counter_procs milestone rewards don't reproduce the C fallthrough  [LOW]
- **C:** src/fight.c:1280-1290 — `switch(number(1,3))` with **no breaks**: roll 1 → +2 max_hit, +1 mana, +1 move; roll 2 → +1 hit/+1 mana/+1 move; roll 3 → +1 move, +1 hit (no mana).
- **Go:** pkg/game/death.go:943-951 — always exactly +1 hit, +1 mana, +1 move (matches only roll 2); pkg/combat/fight_core.go:1000-1013 duplicates the same simplification with a comment misdescribing the C bug.
- **Divergence:** Expected value differs slightly (C averages +1.33 hit, +0.67 mana, +1 move).
- **Player impact:** Marginal — kill-milestone stat rewards slightly off.
- **Fix sketch:** Reproduce the three fallthrough outcomes with a 1-in-3 roll.

### F21. Immortal instakill threshold too low and missing equal-level guard  [LOW]
- **C:** src/act.offensive.c:138-154 — `do_kill` instakill requires `GET_LEVEL(ch) >= LVL_IMPL-1` (implementor tier) and refuses when `GET_LEVEL(vict) == GET_LEVEL(ch)`.
- **Go:** pkg/session/combat_cmds.go:21-40 — any `GET_LEVEL >= LVL_IMMORT` (31) instakills anyone, including same-level immortals; victim gets no "chops you to pieces" message.
- **Divergence:** Every immortal has implementor-grade instakill.
- **Player impact:** Staff-tier ability granted 8+ levels early; imm-vs-imm griefing possible.
- **Fix sketch:** Gate on LVL_IMPL-1 and add the equal-level refusal.

### F22. Scavenger mobs ignore CAN_GET_OBJ and the cost>1 floor — can pick up corpses and no-take items  [LOW]
- **C:** src/mobact.c:103-117 — scavengers take the best object only if `CAN_GET_OBJ(ch, obj)` (ITEM_WEAR_TAKE + carry limits) and `GET_OBJ_COST(obj) > max` with `max = 1`; corpses have no TAKE flag (fight.c:386-388) so mobs never take them.
- **Go:** pkg/game/mobact.go:176-190 — picks the highest-cost item in the room with no takeable check and no cost floor.
- **Divergence:** Scavengers can vacuum up player corpses, fountains, and zero-cost scenery objects.
- **Player impact:** A wandering scavenger can steal your corpse (with all your gear inside).
- **Fix sketch:** Filter on `item.IsTakeable()` and `GetCost() > 1`.

### F23. Weapon damage message tiers re-bucketed and reworded vs C  [LOW]
- **C:** src/fight.c:895-992 — 12 fixed tiers with boundaries 0/2/4/6/10/14/19/23/33/43/53+ and exact texts ("massacres... to small fragments" at 20-23, "OBLITERATES" at 24-33, "ROCKS THE HELL OUT OF" at >53).
- **Go:** pkg/combat/fight_core.go:533-767 — 14 tiers with boundaries 26/36/48/60/80/101/10000 and randomized invented variants per tier (labeled CRIT-010).
- **Divergence:** At a given damage number, players see different (often milder) messages than C; the C ">53 = ROCKS" cap now requires 10000 damage.
- **Player impact:** Cosmetic, but veterans use these strings to gauge damage; the signal is recalibrated.
- **Fix sketch:** Restore C tier boundaries; keep variants if desired but anchor tier 0-11 texts to the C table.

### F24. Mob HP regen rounds up for odd levels  [LOW]
- **C:** src/limits.c:133-137 — `gain = 2.5*GET_LEVEL(ch)` assigned to int truncates: level 9 → 22.
- **Go:** pkg/game/limits_gain.go:151-157 — `(lvl*5 + 1) / 2` rounds up: level 9 → 23.
- **Divergence:** +1 HP/tick for every odd-level mob under 23.
- **Player impact:** Negligible.
- **Fix sketch:** Use `lvl * 5 / 2`.

---

## Subsystems checked and found substantially clean
- **Spell damage dice** (magic.c:602-819 ↔ pkg/spells/damage_spells.go magDamageFormula): all formulas, reagent bonuses, backfire chances, dispel evil/good self-target, and soul-leech heal match. (Two nits, not filed: Go clamps saved damage to min 1 where C's `dam >>= 1` can reach 0; soul-leech heals even when damage was blocked, where C gates on `damage()` returning true.)
- **Saving throws** — golden-tested (pkg/spells/saving_throws_golden_test.go) against the magic.c tables; `mag_savingthrow` semantics match.
- **THAC0/hit calc** (fight.c:1783-1830 ↔ pkg/combat/formulas.go CalculateHitChance): class tables, str/int/wis bonuses, bless, drunk, dex-AC, nat-1/nat-20 semantics all match. get_minusdam table matches.
- **Regen gain formulas** (limits.c mana/hit/move_gain ↔ pkg/game/limits_gain.go): positions, class shifts, poison/flaming/cutthroat, hunger, regen rooms all match (F24 aside). PointUpdate structure matches (F16 aside).
- **Attacks-per-round schedule** (fight.c:1904-1947 ↔ formulas.go GetAttacksPerRound): mob level bands and player class/level chances match (F19 aside).
- **Corpse creation** (fight.c:258-428 ↔ pkg/game/death.go makeCorpse/makeDust): container flags, NODONATE, val[3]=1, timers (5/10), inventory+equipment+gold transfer, money-pile descriptions, attack-type corpse descriptions all match.
- **Group XP base** (fight.c:708-736 ↔ party.go): per-member split and the 1% >100 group penalty match (the level-diff step after it is F4).
- **Mob wander mechanics** (mobact.c:119-143 ↔ ai.go wanderMob): single number(0,18) draw semantics, STAY_ZONE, DEATH/NOMOB gates match (cadence issue is F15).
- **CON-loss on death** (fight.c:598-607 ↔ death.go): 1-in-4 / extra 1-in-6 over level 20 match.
