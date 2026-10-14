# DP-1371 backlog batch — 2026-10-14

Four issues closed in four reviewable trains, each one commit, each rebased onto
`origin/main` immediately before its final combined census. All four branches are
off this round's base; the census lines are at the end.

| Issue | Branch | Commit |
|---|---|---|
| DP-1435 | `dp1371/dp-1435-mobact-room-acts` | `b3c6a2b97` |
| DP-1433 | `dp1371/dp-1433-lua-char-tables` | `c4b05c68c` |
| DP-1429 | `dp1371/dp-1429-death-floors` | `7796f1327` |
| DP-1430 | `dp1371/dp-1430-score-about-gear` | `90466082e` |

Work happened in a worktree (`/home/zach/darkpawns-dp1371-batch`) because the
main checkout carried another issue's uncommitted WIP. The reference oracle was
`~/darkpawns-c-oracle/bin/circle`, sha256
`bd6c548c6fbc41d35eede384e4ce2c5a858ff2ce0f568a379d7fb366a9051b07`, matching
`cmd/dp-oracle-diff/reference-oracle.sha256`.

**Stop tier: not reached.** DP-1429's login-gate question resolved without a
login-flow change (see below), so nothing in this batch was stopped for Zach.

---

## DP-1435 — mobile_activity room acts

### C trace (grepped in `src/` this session)

`mobile_activity()` (src/mobact.c:24-340) emits five room acts the port never
sent. Reading the arms in source order:

- **Race hate** (src/mobact.c:246-257): `act("'Come to destroy my kin? Die!',
  exclaims $n.", ..., TO_ROOM)` at `:252-253`. The port already sent this text
  through a hand-rolled `RoomEcho`; left as is here (see *Residual* below).
- **Mob memory** (src/mobact.c:261-282): the victim predicate is
  `IS_NPC(victim) || !CAN_SEE(ch, victim) || PRF_FLAGGED(victim, PRF_NOHASSLE)`
  (`:267-269`), the match is on `names->id == GET_IDNUM(vict)`
  (`:271`), and the two warning acts are the halves of one
  `!ROOM_FLAGGED(ch->in_room, ROOM_PEACEFUL)` gate — the exclaim when
  `can_speak(ch)` (`:273-275`) and the growl otherwise (`:276-278`). `hit()`
  follows unconditionally (`:279`).
- **Helper mobs** (src/mobact.c:284-302): a charmed mobile whose master is
  fighting calls `do_assist(ch, GET_NAME(ch->master), 0, 0)` and returns
  (`:287-291`); otherwise it joins the first mobile in room order with
  `IS_NPC(victim) && FIGHTING(victim) && CAN_SEE(ch, FIGHTING(victim)) &&
  !IS_NPC(FIGHTING(victim)) && ch != FIGHTING(victim)` (`:295-297`), sends
  `act("$n jumps to the aid of $N!", FALSE, ch, 0, victim, TO_ROOM)` (`:298`)
  and hits `FIGHTING(victim)` (`:299`).
- **AGGR24 players** (src/mobact.c:304-322): selects the first
  `!IS_NPC && CAN_SEE && !PRF_NOHASSLE && !ROOM_PEACEFUL` player of
  `GET_LEVEL >= 24` (`:305-311`), then the grin (`:313`), the speak pair gated
  on `can_speak(ch)` (`:314-319`) and `hit(ch, vict, 0)` (`:320`).
- **AGGR24 + AGGRESSIVE + full health** (src/mobact.c:324-336): selects the
  first visible mobile with `GET_LEVEL(tmp_ch)+3 < GET_LEVEL(ch)` (`:327-331`),
  then `act("$n grins evilly.", FALSE, ch, 0, 0, TO_ROOM)` (`:333`) and
  `hit(ch, vict, 0)` (`:334`).

`do_assist` is src/act.offensive.c:54-98; `can_speak` is `is_intelligent` at
src/utils.c:685-687 with `intelligent_races[]` at src/constants.c:353-373.

### What changed

`pkg/game/mobact.go`:

- the memory arm's two acts, in C's order and inside one peaceful gate;
- the helper arm's aid act and C's `CAN_SEE` gate on the defended player;
- the helper arm's charmed-master branch, through a new mobile-side
  `World.mobAssist` porting `do_assist`;
- the AGGR24 player branch's three acts at C's point;
- the AGGR24+AGGRESSIVE grin at C's point.

`NpcRescue` is no longer called from the helper arm: C's `npc_rescue`
(src/mobprog.c:435-462) is defined but never called anywhere in `src/`, and the
port's version sent unsubstituted `$N`/`$n` text (R4). No RNG draw was added,
moved or removed.

### Proof and R5h controls

Six real-pulse tests in `pkg/game/mobact_test.go` drive
`World.MobileActivity()` and read the room bytes through the message sink:

`TestMobileActivityMemoryWarningActs`, `TestMobileActivityHelperJumpsToAid`,
`TestMobileActivityAggr24WeakerMobileGrin`,
`TestMobileActivityAggr24PlayerRoomActs`,
`TestMobileActivityCharmedHelperAssistsMaster`, `TestMobAssistGuards`.

One-at-a-time reverts, each run alone:

```
memory-warning acts removed: RED (as required)
helper aid act removed: RED (as required)
helper CAN_SEE gate removed: RED (as required)
AGGR24 weaker-mobile grin removed: RED (as required)
AGGR24 player room acts removed: RED (as required)
charmed-master assist arm removed: RED (as required)
```


### Oracle fixtures withdrawn, and the finding they exposed

Two observer fixtures were built for the AGGR24 arms
(`mobact-aggr24-weaker-mobile-grin`, `mobact-aggr24-player-room-acts`) and both
diverged for a reason outside this train, so they were withdrawn rather than
kept red. Raw transcripts, seed 1:

```
scenario mobact-aggr24-weaker-mobile-grin (room 8162), probe ~dpclock pulse 40
  c-oracle: Overlord Mischra Drath grins evilly.
  go-port : The daemon courier stings Overlord Mischra Drath very hard.   (x3)
            Overlord Mischra Drath EVISCERATES the daemon courier with his incredible hit!!
            ...
            The daemon courier is dead!  R.I.P.
  look: c-oracle "A small daemon pushes a cart here." /
        go-port "The corpse of the daemon courier is lying here."

scenario mobact-aggr24-player-room-acts (same room, observer without NOHASSLE)
  c-oracle: (empty block)
  go-port : You swing your fist at Overlord Mischra Drath, but miss him!   (+10)
  look: c-oracle "The daemon lord of the Fortress stands before you." /
        go-port "Overlord Mischra Drath is here, fighting YOU!"
```

Room 8162 is `ROOM_PEACEFUL` — `lib/world/wld/80.wld` gives it flags `28`
(`ROOM_NOMOB|ROOM_INDOORS|ROOM_PEACEFUL`), and the port parses it as
`flags=[28 0 0 0]`, `peaceful=true`. In C, `hit()` never calls `set_fighting`
(src/fight.c:1762-1900); `damage()` enrols the pair and its peaceful arm is
gated on `FIGHTING(victim) != ch` (src/fight.c:1336-1342), so the **first**
swing in a peaceful room is refused and nothing is enrolled. The port's `hit()`
equivalent is `combatEngine.StartCombat`, which enrols the pair; the refusal
runs later inside the combat pulse (pkg/combat/engine.go:764 ->
pkg/game/damage_gate.go:45), and by then `FIGHTING(victim) == ch`, so C's own
second-hit exemption applies and the damage lands.

This predates DP-1435 — the pre-existing arms called `StartCombat` the same way
— and is filed as a `blocked` depth row (`mobact.peaceful-first-hit-enrolment`)
rather than fixed here. The arms themselves are proven by the six real-pulse
tests above; the blocked `mobact.aggr24-room-grin-act` rows in
`combat-entry.tsv` and `surface-inventory.tsv` are closed.

### Residual (same class, not fixed here)

- The race-hate act at src/mobact.c:252-253 still goes out through the port's
  hand-rolled `RoomEcho` rather than `act()`. Same rendered text for a visible
  mobile, but a different audience/substitution path. Not one of the five
  missing acts.
- The memory match is by player name (`MobInstance.Memory []string`) where C
  stores `GET_IDNUM`. Equivalent while names are unique; noted for the memory
  model's owner.
- `mobact.peaceful-first-hit-enrolment` above.

---

## DP-1433 — Lua character tables

### R5e finding: the cited builder is not the production builder

DP-1433 cites `pkg/scripting/engine.go` ("the char-table code") as the builder
that omits C's fields. Verified call path this session:

- `RunScript` selects the bridge whenever `ctx.MeRef != nil || ctx.OwnerType != ""`
  (pkg/scripting/engine.go:458-465) and says so itself: "Every production run
  uses the bridge. The non-bridged path is only used by legacy engine tests."
- Every production `ScriptContext` literal sets one of those:
  `MobInstance.CreateScriptContext` always sets `MeRef`
  (pkg/game/scripts.go:135), `CreateSelfScriptContext` copies it to `ChRef`
  (`:148-152`), and `room_obj_scripts.go:99,147,187,228` set `MeRef` or
  `OwnerType`.
- The production builder is therefore `bridge.go charToTableDepth`
  (`:300-349`) fed by `WorldScriptableAdapter.CharFields`
  (pkg/game/world_bridge.go:85-150), and it already emitted C's full field set,
  a real `wear` list and no `skills` table. It has been that way since
  `814d8c5ee` (2026-09-24), three weeks before the issue was filed.

The issue's three complaints are true only of the legacy builder, which no
server run reaches. Fixing it is still the right call — a divergent builder left
in the tree is a landmine — but the *reachable* surface was already faithful,

### What changed

C's `char_to_table` is one shape for both kinds (src/scripts.c:1823-1891):
`name, align, gold, level, hp, maxhp, pos, alias`, `objs` when `ch->carrying`
(`:1841-1847`), `wear` over `GET_EQ(ch, 0..NUM_WEARS-1)` when any slot is filled
(`:1849-1863`), `evil` as `IS_EVIL` = `GET_ALIGNMENT <= -350` (`:1865-1868`),
`leader` recursing into `ch->master` (`:1870-1873`), then the player-only
`id, exp, bank, mana, move, cha, jail, tattoo, followers` (`:1875-1884`) or the
mobile-only `vnum, timer` (`:1885-1888`), and `struct` (`:1890`).

- `pkg/scripting/types.go`: `ScriptablePlayer` / `ScriptableMob` gained the
  accessors the missing fields need, plus a `ScriptableChar` union for the
  leader chain, plus `ScriptableWorld.GetScriptableChar`.
- `pkg/game/scriptable_char.go` (new): the `*Player` / `*MobInstance`
  implementations, each citing the C macro it mirrors.
- `pkg/game/world_scriptable.go`: `GetScriptableChar` resolves the follow
  target through the world's existing follower resolver.
- `pkg/scripting/engine.go`: `charToTableLocked` / `mobToTableLocked` now emit
  C's field set through one `legacyCharTable`; the invented `skills` table is
  gone and the mobile `wear` list is real.

### Proof and R5h controls

Real Lua dispatch on **both** paths:

- production: `TestLuaCharTablePlayerFields` / `TestLuaCharTableMobileFields`
  (pkg/game/lua_char_table_depth_test.go) run a script through
  `WorldScriptableAdapter` and read the table back out of the script's own
  `say()` line, asserting every field, the two object lists and
  `tostring(ch.skills) == "nil"`.
- legacy: `TestLegacyCharTableMatchesCFieldSet` /
  `TestLegacyCharTableEvilThreshold` (pkg/scripting/legacy_char_table_test.go)
  drive `RunScript` with a `ScriptableWorld` that is not a `Bridge`.

```
player-only fields removed: RED (as required)
invented skills table restored: RED (as required)
player wear list emptied: RED (as required)
mobile wear list emptied: RED (as required)
IS_EVIL threshold reverted: RED (as required)
```

One note on the first attempt at the IS_EVIL control: it came back GREEN because
the assertion looked for `|0|` anywhere in a joined line that also carried
`ch.id = 0`. The test was tightened to read only the `evil` field before the
control went red, which is the R5h check doing its job.

---

## DP-1429 — the invented death floors

### C trace

`die_with_killer` (src/fight.c:585-621):

```c
if (GET_HIT(ch)<0) {
  if (!IS_NPC(ch)) {
    if (GET_LEVEL(ch)>5 && !number(0,3)) {
      (ch)->real_abils.con--;            /* :605 — no floor */
      if (GET_LEVEL(ch)>20 && !number(0,5))
        (ch)->real_abils.con--;          /* :607 — no floor */
      affect_total(ch);                  /* :608 */
    }
  }
  ...
}
```

`affect_total` is what bounds the *effective* value:
`GET_CON(ch) = MAX(0, MIN(GET_CON(ch), i))` (src/handler.c:356).

`damage()` applies the hit with no floor: `dam = MAX(MIN(dam, 3000), 0);
GET_HIT(victim) -= dam;` (src/fight.c:1483-1484). `update_pos` is what turns
`hp <= -11` into `POS_DEAD` (src/fight.c:192-193).

### What changed

- `pkg/game/death.go`: both `if player.Stats.Con < 1 { = 1 }` clamps removed.
  A `conLossNumber` seam (`var conLossNumber = number`) was added so a focused
  test can pin both gates; it is the same shared stream in production.
- `pkg/game/player_affects.go` and `pkg/game/mob.go`: the `if HP < -11 { = -11 }`
  clamps removed from both `TakeDamage`s.
- Existing tests that asserted the floors now assert C's values:
  `TestTakeDamage_FloorsAtNegative11` became `TestTakeDamageMatchesCHitPoints`,
  `TestReplayCrashLoadBadStatsLethalStopsAtRawKill` and the two
  `pkg/session` crash-load-lethal tests now assert `-30`.


### Reader audit (R5c) — every reader of real CON and post-death HP

- **Login gate.** `Session.prepareWorldEntry` refuses when
  `p.LoginConstitution() == 0` (pkg/session/menu.go:524). That method reads
  `p.Stats.Con` (real) plus tattoo and active-affect bonuses, and clamps to
  0..18 only when `len(p.ActiveAffects) > 0`
  (pkg/game/effective_attributes.go:199-217). That is exactly C: the gate reads
  `GET_CON` right after `reset_char` (src/interpreter.c:2173-2178), and
  `store_to_char` copies the saved abilities straight into `real_abils` **and**
  `aff_abils` (src/db.c:2441-2442) and reaches `affect_total` only through
  `affect_to_char` for the loaded spells (`:2476-2480` → src/handler.c:395).
  `tattoo_af` does not call it (src/tattoo.c). So with no saved spell affects
  C's `GET_CON` is the raw saved value, negative included, and its `!GET_CON`
  test refuses only on exactly 0 — the same as the port. **No login-flow change
  needed; the stop tier was not reached.**
- **score / save / restore.** `score` prints `GET_HIT` verbatim, and the save
  record stores the stat block as `Stats CharStats` with no clamp on either
  side (pkg/game/save.go:55, :325-326), so negative CON and HP round-trip. No
  save-format change.
- **update_pos and its callers.** `combat.GetPositionFromHP` tests
  `hp <= -11` (pkg/combat/fight_core.go:144-145), and the wounded-band
  broadcasts, the point_update death ticks (pkg/game/limits_condition.go) and
  `Player.IsAlive` (pkg/game/player_combat.go:18) all use `<=`, so deeper
  negatives change nothing there.
- **The croak.** `room_activity.go`'s CON<=0 arm reads the effective CON, which
  `affect_total` clamps to 0, so a real CON of 0 or below now reaches it —
  covered by the DP-1426 tests (`TestRoomActivityConCroakUsesEffectiveCon`).

### Proof and R5h controls

`TestTakeDamageMatchesCHitPoints`, `TestDieWithKillerConLossHasNoFloor`,
`TestDieWithKillerBothConLossDrawsAreUnfloored` and
`TestPlayerDeathCountersStillRunWithNegativeCon`
(pkg/game/death_floors_depth_test.go), driving the real `HandleDeath` /
`die_with_killer` path on a level-21 PC with a seeded stream.

```
player HP floor restored: RED (as required)
mob HP floor restored: RED (as required)
first CON floor restored: RED (as required)
second CON floor restored: RED (as required)
```

The second CON control needed the draw seam to fail: without it the first
decrement is unfloored and can carry CON below any value the second floor could
have clamped, so the floor is unobservable from the final value alone.

---

## DP-1430 — score's clothing and weight lines

### C trace

- Pack line: `if (IS_CARRYING_W(ch)) weight = CAN_CARRY_W(ch)/IS_CARRYING_W(ch);
  else weight = -1; weight = MAX(-1, weight);` (src/act.informative.c:1304-1315).
  `IS_CARRYING_W` is `ch->char_specials.carry_weight` (src/utils.h:322), which
  `obj_to_char` / `obj_from_char` maintain (src/handler.c:559-601) and which
  `do_wear` leaves before `equip_char`, so worn gear never counts. Contents do
  count, because `obj_to_obj` adds them to the top-level container's own weight
  (src/handler.c:940-960).
- Clothing line: the AC bands at src/act.informative.c:1240-1265 start
  `if (ac == 100) "are naked..."` — an exact test.
- AC itself: `apply_ac` is `factor * GET_OBJ_VAL(obj, 0)` with factor 3 for
  `WEAR_BODY`, 2 for `WEAR_HEAD`/`WEAR_LEGS` and 1 for everything else,
  `WEAR_ABOUT` included (src/handler.c:610-635), applied as
  `GET_AC(ch) -= apply_ac(ch, pos)` (`:734-735`); then `equip_char` runs
  `affect_modify_ar` over every object affect (`:743-746`), and `APPLY_AC` adds
  to `GET_AC` (`:182`).

### What changed

- `Player.CarriedWeight` counts the carrying list only (pkg/game/player.go).
- `Equipment.GetArmorClass` applies C's per-slot factors and reads
  `GetValue(0)` (honouring overrides) instead of `Prototype.Values[0]`;
  new `Equipment.GetWornAffectAC` returns the worn `APPLY_AC` total
  (pkg/game/equipment.go).
- `Player.GetAC` subtracts the apply_ac value and adds the worn affect
  (pkg/game/player_stats.go).
- `acText`'s first band is `ac == 100` (pkg/session/cmd_info.go).

### Proof and R5h controls

The oracle vehicle `score-about-worn-gear` spawns stock object 4372 (the
tattered rags of a Holy Martyr: `ITEM_WORN`, `WEAR_ABOUT`, weight 1,
`APPLY_AC +10`, `APPLY_CON -5`), gets and wears it through the real commands,
and runs `score`. Before the fix it diverged on both lines:

```
-You are lightly clothed.            (c-oracle, ac 100 + 10)
+You are naked, have you no shame?   (go-port, ac 100)
-Your pack is empty.                 (c-oracle)
+Your pack is light.                 (go-port)
```

After the fix it reports `result: no normalized divergence` on seeds 1, 2, 3, 5
and 8. Unit vehicles:

```
CarriedWeight counts worn gear again: RED (as required)
apply_ac slot factors removed: RED (as required)
worn APPLY_AC dropped: RED (as required)
acText naked band widened back to >=100: RED (as required)
```

and that is recorded here rather than silently assumed.


---

## Gates and censuses

Each gate was run on its own with its exit status checked. Census lines are
filled in below as each train's final combined census completes.

<!-- CENSUS-LINES -->

## Notes for review

- **R5e, twice.** DP-1433's cited builder is unreachable (see above). DP-1435's
  blocked `mobact.aggr24-room-grin-act` row was deferred to an ownership
  boundary that the issue says is gone — it is, and the row is closed.
- **R5h, twice.** The IS_EVIL control and the second CON control each came back
  green on the first attempt; both were fixed by tightening the test, not by
  weakening the control.
- **No governing document was touched.** No rulebook amendment is proposed for
  this batch: every change is a direct C-mismatch fix, and the two findings
  above are new defects rather than new rules.
