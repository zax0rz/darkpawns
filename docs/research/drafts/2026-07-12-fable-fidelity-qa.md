# Fable Fidelity QA — 2026-07-12

QA pass over all 25 issues in Linear project **Fable Fidelity Audit 2026-07-11** (DP-1021→DP-1046, plus DP-1047/DP-1048), verified against the original C in `src/`. Method per the audit rules: read the C directly, never trust Go comments, check the live paths. Build state at QA time: `origin/main @ 4e5fa33`, `go build ./... && go test ./...` both clean.

**Bottom line: 21 of 25 land faithfully — genuinely good work.** Two fixes don't actually work (F7 death traps, DP-1046 jail guard), one opens a progression hole (F3's level cap), and one hands mortals an immortal command (mold). Details below, worst first.

---

## Verdict summary

| Issue | Finding | Verdict |
|---|---|---|
| DP-1027 F7 | Death traps | ❌ **Still doesn't kill** |
| DP-1046 | Jail-guard intercept | ❌ **Never fires (wrong vnums)**; other 2 redirects ✅ |
| DP-1023 F3 | Kill XP → gain_exp | ⚠️ Works, but level cap wrong (30→31 possible) |
| DP-1038 F18 | Carry weight | ⚠️ Enforced only for brand-new characters |
| DP-1035 F15 | Tick rates | ⚠️ PointUpdate 63s ✅; AffectUpdate still 75s |
| DP-1034 F14 | Aggro gates | ✅ gates correct; protect-roll inversion found in same block |
| DP-1048 | mold command | ⚠️ Missing LVL_IMMORT + POS_RESTING gates |
| DP-1047 | detect command | ⚠️ C's command word is `search`, not `detect` |
| DP-1031 F11 | Alignment from kills | ✅ solo exact; group members missed |
| DP-1036 F16 | Decay dedup | ✅ N× fixed; mob-less rooms still never decay |
| DP-1021/1022/1025 F1/F2/F5 | Damage pipeline | ✅ (verified in prior sessions) |
| DP-1024 F4 | XP share formula | ✅ exact (incl. float-truncation order) — 4 nits |
| DP-1026 F6 | Parry/dodge | ✅ excellent — matches ticket plan and C |
| DP-1029 F9 | Movement costs | ✅ exact (index-8/9 trap handled) |
| DP-1030 F10 | PK bookkeeping | ✅ — 2 nits |
| DP-1032 F12 | Bash wait-state | ✅ |
| DP-1033 F13 | Backstab | ✅ truncation + to-hit both right |
| DP-1037 F17 | Flee XP | ✅ on the live `cmdFlee` path |
| DP-1039 F19 | Slow → 0 attacks | ✅ |
| DP-1040 F20 | counter_procs | ✅ exact (incl. the `default:` clause) |
| DP-1041 F21 | Instakill gate | ✅ — 1 nit |
| DP-1042 F22 | Scavenger | ✅ — 2 nits |
| DP-1043 F23 | Damage tiers | ✅ boundaries/texts; invented variants remain |
| DP-1044 F24 | Mob HP regen | ✅ |
| DP-1045 | Entry gates | ✅ (reviewed at PR #185) |

---

## HIGH — fixes that don't do what the ticket says

### Q1. DP-1027 (F7): death traps still don't kill — the fix is ineffective

`pkg/game/world.go:973` does `p.TakeDamage(p.GetHP() + 1)`. Pre-F1 that clamped to 0; **post-F1 it leaves the player at HP = −1, which is merely POS_STUNNED**. No death handler runs, nothing extracts the player — they lie stunned in the DT room and recover. The test (`TestMovePlayer_DeathTrapKillsMortal`) only asserts `HP <= 0`, so it passes against a stunned-not-dead player.

C (`src/act.movement.c:288-301`): `log_death_trap(ch); death_cry(ch); extract_char(ch);` — an immediate, corpse-less, penalty-free extraction, **plus the mount**, and the gate is `GET_LEVEL(ch) < LVL_IMMORT || IS_NPC(ch)` — **NPCs die in DTs too** (Go's check is player-move only; `wanderMob` has no DT check either).

Fix sketch: dedicated DT path — death cry to room, extract/respawn the player with no corpse and no XP/CON penalty (C's extract_char sends to menu; nearest Go equivalent is the respawn tail of `handlePlayerDeath` without the penalty/corpse blocks), log it, handle mounts, and add the same check to mob movement.

### Q2. DP-1046: jail-guard intercept gates on the wrong mob vnums — it can never fire

`pkg/combat/engine.go:509` checks `HasMobVNum(attacker, 8102) || 8103`. But `take_to_jail` is assigned to mobs **8001, 8002, 8020, 8027, 8059** and `wall_guard_ns` to **8060** — identically in C (`src/spec_assign.c:285-291`) and Go (`pkg/game/spec_assign.go:110-115`). Neither 8102 nor 8103 carries the spec in either codebase (the numbers came from the dead `fight_core.go:352` port, which was already wrong). The subdue path, messages, room 8118, and `max(2, level/2)` jail timer are all faithful — they're just unreachable.

Fix sketch: gate on the spec assignment (`MobSpecAssign[vnum] == "take_to_jail" || "wall_guard_ns"`) via a callback instead of hardcoded vnums. Also missing C's `CAN_SEE(ch, victim)` condition (minor).

The other two redirects verify clean: charm-retarget conditions/odds exact (`!number(0,10)`, master in room, both NPC), switcheroo exact (`level > 20`, per-occupant `!number(0,80)`, victim fighting the mob). One shared nuance: C transfers the *current* blow to the new target via `hit()`; Go aborts the round and retargets for the next tick — one round of lag, acceptable.

---

## MED — behavioral divergences worth fixing

### Q3. DP-1023 (F3): level-up cap lets level-30 mortals advance to level 31 (immortal)

`pkg/game/limits_exp.go:182`: `if p.Level < LVL_IMPL-1 && ...` (i.e. < 39). C (`src/limits.c:303`): `GET_LEVEL(ch) < LVL_IMMORT-1` (i.e. < 30) — XP can never take a mortal past level 30. In Go a level-30 player who crosses the threshold auto-advances to 31 = LVL_IMMORT, acquiring immortal gates (instakill routing, DT immunity, aggro immunity…). One-token fix: `LVL_IMMORT-1`.

Nits while there: C's condition is `GET_EXP > exp_needed` (strict); Go uses `>=`. And Go floors the capped gain at 1 where C would apply a negative gain if exp somehow exceeds the next-level threshold — both nano.

### Q4. DP-1038 (F18): carry weight only enforced for brand-new characters

`SetCapacity(str, dex, level)` is called from exactly two places, both at creation. The live load paths defeat it:

- `pkg/db/convert.go:55-67` (`RecordToPlayer`): calls `NewCharacter` (capacity computed from **freshly rolled** stats at level 1), then overwrites `p.Stats`/`p.Level` with saved values **without recomputing** — a str-18 level-25 warrior gets whatever the roll said, e.g. 100 lb / ~10 items.
- `pkg/game/save.go:329`: builds the Player literal with bare `NewInventory()` → `MaxWeight = 0` = weight unenforced entirely on that path.
- Level-ups/stat changes never recompute either (C computes `CAN_CARRY_W/N` live from current stats on every check).

Fix sketch: recompute capacity from final stats at the end of both load paths and in `AdvanceLevel`; or better, compute limits at check time from live stats like C instead of caching. Nit: `CarryWeight()` ignores `StrAdd`, so 18/xx players get 255 instead of 280–480 (`str_app[26..30]`), and str 19–25 clamps to 255 instead of 640–1750 (mostly affects high-str mobs).

### Q5. DP-1035 (F15) residual: AffectUpdate/Weather still run a 75s mud hour, PointUpdate runs 63s

The fix moved `StartPointUpdateTicker` to 63s ✅ and removed the double AI driver ✅. But the gameloop constant `pkg/engine/gameloop.go:33` is still `SECS_PER_MUD_HOUR = 75` (stock Circle), which drives `OnAffectUpdate` + `OnWeatherAndTime` (gameloop.go:274-276). Dark Pawns' `src/utils.h:135` is **63**. Net: spell durations tick ~19% slow vs C, and the game's two "mud hour" drivers now disagree with each other (63 vs 75). One-token fix: set the constant to 63.

### Q6. Aggressive-mob protect-evil/good roll inverted (found during DP-1034 QA)

`pkg/game/mobact.go:294,299`: `if vict.IsAffected(12) && mobIsEvil(ch) && rand.IntN(6) != 0 { continue }` — skips the victim **5/6** of the time. C's aggressive block (`src/mobact.c:210-213`) is `... && !number(0,5)` — skips only **1/6**. (Confusingly, C's *race-hate* block really does make protection work 5/6 of the time, and Go's race-hate port at mobact.go:361 is exactly right — the aggressive block copied the wrong block's odds.) Everything DP-1034 itself added verifies clean: canSee/NOHASSLE on all three blocks, sneak skip 1-in-4, AGGR24 peaceful gate, memory attack correctly *not* peaceful-gated (C only gates the growl message).

### Q7. DP-1048: `mold` registered for mortals

`pkg/session/commands.go:277` registers mold at level 0, POS_STANDING. C (`src/interpreter.c:551`): `{ "mold", POS_RESTING, do_mold, LVL_IMMORT, 0 }`. An immortal object-creation command is now available to every mortal. Fix: `LVL_IMMORT`, `PosResting`.

### Q8. DP-1047: the command word should be `search`, not `detect`

C has no `detect` command; `do_detect` is bound to **`search`** (`src/interpreter.c:411`, POS_STANDING, level 0). Go registered the right handler under a word C players never typed, and `search` still doesn't exist. Fix: register as `search` (keep `detect` as an alias if you like, but `search` must work).

### Q9. DP-1031 (F11): group kills only shift the killer's alignment

The solo-path shift in `pkg/game/death.go:178-198` is exact (formula, `>>4` arithmetic shift, ±1000 clamp, ±350 neutral gate all match `src/fight.c:484-502`). But C's group path calls `change_alignment(member, victim)` for **every** group member inside `perform_group_gain` (`src/fight.c:688-705`); Go's group loop in `pkg/game/party.go:259-267` awards XP only. Grouped non-killer members never shift. Fix: apply the same shift in the `AwardMobKillXP` group loop (and skip the killer-only shift for group kills to avoid double-shifting — cleanest is to move the shift into the award paths wholesale).

---

## LOW — nits and accepted divergences

- **DP-1036 (F16) residual:** dedup is correct, but decay still only runs for rooms containing a live mob. C's `point_update` iterates the *global* object list (`src/limits.c:529+`) — corpses in mob-less rooms (i.e. every room the player just cleared) and corpses *carried by players* never decay in Go.
- **DP-1041 (F21) nit:** Go instakill routes through `HandleDeath(vict, killer)` → killer gets XP, PK counters, OUTLAW flag, victim takes exp/37. C's `do_kill` calls `raw_kill` — no XP, no PK bookkeeping, no penalty. Also C's self-kill message differs, but the equal-level guard happens to block self-kills anyway.
- **DP-1030 (F10) nits:** C increments `GET_KILLS(ch)`/`counter_procs` on **all** kills including player victims (`src/fight.c:1689-1690` sits outside the NPC-victim branch); Go only does it in the mob branch — PK kills don't count toward milestones. And `GET_DEATHS(victim)++` is unconditional in C; Go skips it when killer is self/empty.
- **DP-1043 (F23) residual:** boundaries and C-verbatim primary texts are exact, but each tier still carries invented "flavor variants" that fire ~50% of the time; C is deterministic. If full fidelity is the goal, drop the variants.
- **DP-1042 (F22) nits:** pickup is silent — C sends `act("$n gets $p.", ...)` to the room (`src/mobact.c:115`). `CAN_SEE_OBJ` omitted (documented). Str>18 carry clamp as in Q4.
- **DP-1024 (F4) nits:** (1) group base uses `base -= base/100` (int div); C truncates `base - base*.01` as a double — off-by-one for many values (e.g. 250 → Go 248, C 247). (2) Solo 1-XP message says "measly little" — C's solo string is "one lousy experience point." (group is "measly"). (3) Zero-exp mobs: Go returns before awarding; C gives the MAX(1,…) lousy point + message. (4) A grouped player whose groupmates are elsewhere takes the group path (no −2 solo slack); C's `is_in_group` is same-room and would grant it. The core formula, including the float-truncation order (`int(share − share*0.7)`), is exact — nice.
- **DP-1028 (F8) nits:** decrement/expiry semantics match `magic.c:431-457` exactly, but Go broadcasts mob wear-off messages to the room — C's `send_to_char` to a mob shows nothing to anyone. And expiry cadence inherits the 75s bug (Q5).
- **DP-1034 nit:** memory block can attack multiple remembered players in one tick; C stops at the first (`for (vict; vict && !found; ...)`).
- **DP-1026 nano:** Go keeps an awake-position guard on parry that C doesn't have; C's parry block has no position check at all. Harmless. The `game.CheckParry`/`DoParry` vestiges in `skill_c10_combat.go` are still present (dead code the ticket suggested retiring).

---

## What verified clean (spot-check details)

- **DP-1026 (F6):** `SKILL_RETREAT=149/ESCAPE=157/PARRY=172` match `src/spells.h` exactly; `combatSkillName` mapping un-breaks GetSkill (and wimpy-retreat); parry is once/round `Number(0,10000) <= skill` with the mutual-fighting gate; reduction uses the *defender's* `dex_app[].defensive` (negative → add, else −1, floor 0) — exactly `fight.c:1999-2007`; dodge is NPC-only AFF_DODGE(17), `number(0,100) < level`. The old per-hit negation, weapon requirement, MOB_AWARE check, and roundPenalty were all correctly removed (none exist in C).
- **DP-1040 (F20):** the C switch really does have a `default:` after case 3 (`src/fight.c:1280-1290`), so roll 1 = +2hit/+1mana/+1move, roll 2 = +1/+1/+1, roll 3 = +1move/+1hit — Go reproduces all three outcomes plus the heal-to-full.
- **DP-1029 (F9):** table matches `movement_loss[]` value-for-value; the index-8/9 comment trap (C comments swap Flying/Underwater vs the enum) is handled and documented; cost = avg(src,dst) = C's `>>1`; immortals exempt.
- **DP-1033 (F13):** `BackstabMult` now truncates exactly like C's int return (level 14→3, 19→4; ≤0→1, immort→20 all match `class.c:719-728`); passed skill roll now runs `CalculateHitChance` (C's `hit()` THAC0 d20) and can miss.
- **DP-1044/1039/1032/1037/1030/1035(core)/1036(core)/1042(core)/1041(core)/1043(core)/1023(core)/1024(core)/1031(solo):** verified against cited C lines; details above.

## Suggested ticket order

1. **DP-1027 reopen** (Q1) — death traps, HIGH.
2. **DP-1046 reopen** (Q2) — jail vnums, one-line gate fix + spec callback.
3. **Q3** limits_exp.go level cap — one token, progression-breaking.
4. **Q7** mold gates — one line, exploit-adjacent.
5. **Q4** carry-weight recompute on load — small, player-visible.
6. **Q5** SECS_PER_MUD_HOUR 75→63 — one token.
7. **Q6** protect-roll inversion — two tokens.
8. **Q8/Q9** and the LOW pile as a batch.
