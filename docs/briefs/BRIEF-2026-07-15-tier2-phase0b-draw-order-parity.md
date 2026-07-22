# BRIEF — Tier-2 Phase 0b: first draw-order parity (character creation + combat round)

**For:** codex (frontier). **Owner of gate:** Claude (oracle red→green + review vs C; Claude builds
the combat scenario). **Branch:** `refactor/tier2-draw-order-parity` off `main`.
**Depends on:** PR #348 (Phase 0a, merged — one CMWC stream + `DP_SEED` on both servers).
**Method rules:** read `src/class.c` `advance_level`, `src/db.c` newbie init, `src/fight.c` `one_hit`
directly. Gated by **oracle red→green**, not a green build.

---

## 0. Why this phase / the key insight
Phase 0a proved the *generator* is byte-exact and consolidated every roll onto one seeded stream.
It did NOT prove that Go *consumes* that stream in C's **draw order** inside any real subsystem —
if Go draws a different count/sequence than C at a given code path, outputs desync even with a
perfect generator. Phase 0b establishes draw-order parity on the first two subsystems and is the
gate the whole spell/skill effort rests on.

**First evidence (I ran `--scenario character-view` on merged main, `DP_SEED=1` both servers):**
`score [actor]` now shows **HP 22(22) and Mana 100(100) MATCHING** — earlier (pre-seed-wiring) HP
was 22 vs 24. That alignment *proves the creation RNG draws are already in parity through HP/mana*.
Only two `score` residuals remain, and Phase 0a turned them from "un-provable Tier-2 model deltas"
into **deterministic, oracle-provable-now** bugs:
```
-Hit points: 22(22)  Mana points: 100(100)  Movement points: 85(85)     (C)
+Hit points: 22(22)  Mana points: 100(100)  Movement points: 86(86)     (Go)
-You are naked, have you no shame?                                        (C)
+You are well armored.                                                    (Go)
```
These block combat parity (Part B) because a to-hit roll's outcome depends on the victim's AC, and
a fair combat transcript needs both fighters stat-identical. So Part A comes first.

---

## PART A — Character-creation stat parity (close the `score` gap) → DP-1161

### A1. Movement `85` vs `86` (the first draw-order/formula hunt)
C newbie move is set at creation, warrior path (`src/class.c` `advance_level`, CLASS_WARRIOR):
```c
case CLASS_WARRIOR:
  add_hp   += number(11, 14);     /* DRAW 1 */
  add_move  = number(1, 4);       /* DRAW 2 */
  GET_PRACTICES(ch) += MIN(2, MAX(1, wis_app[GET_WIS(ch)].bonus));   /* NOT random */
  break;
...
ch->points.max_hit  += MAX(1, add_hp);
ch->points.max_move += MAX(1, add_move);   /* base max_move = 82  (db.c:3053) */
```
So C: `max_move = 82 + MAX(1, number(1,4))`. Oracle rolled 3 → 85. **Warrior draws exactly TWO
numbers here, in order: `add_hp` then `add_move`. There is NO `add_mana` draw for warrior** (the
warrior case has no `add_mana = number(...)` line). Because HP already matches, the stream is
aligned *through* DRAW 1 — so the move divergence is one of:
1. **A phantom/missing draw** between `add_hp` and `add_move` in Go (e.g. Go rolls an `add_mana`
   for warriors, or draws practices, shifting `add_move` onto the wrong stream value), OR
2. **A wrong range** for `add_move` (must be `number(1,4)`; ranger is `number(2,4)`), OR
3. **A wrong base/formula** (base must be 82; the add is `MAX(1, add_move)`).
Root-cause the Go newbie/`advance_level` path against `class.c` for **every class** (each class's
`add_hp`/`add_move`/`add_mana` draw set and ORDER must match the C `switch` exactly — psionic/mystic
draw `add_mana`, warrior/thief/ranger do not; ranger move is `number(2,4)`). The warrior off-by-one
is the visible RED; fixing the draw set/order/ranges to match C class-for-class is the real task.

### A2. AC `naked` vs `well armored` (deterministic model bug — no RNG)
C newbie armor is a constant: `GET_AC(ch) = 100` (`db.c:2986`; also `points.armor = 100` at
`db.c:2461/2608/3055`). AC 100 renders "You are naked, have you no shame?" via the already-faithful
`do_score` AC prose (act.informative.c:1233-1260). Go shows "well armored" → Go's newbie AC is far
below 100. Almost certainly Go **auto-equips the starter gear** (C hands newbies their kit in
*inventory*, carried not worn — class.c starter-eq path) **or** initializes AC below 100. Fix: a
freshly created newbie has **AC = 100 and starter gear CARRIED, not worn**. No RNG here; purely the
starting model. (This is the AC half of DP-1161.)

### Part A gate
`--scenario character-view` → `score [actor]` **fully clean** (HP/mana already are; move + AC now
match). Unit tests: newbie warrior `max_move == 82 + MAX(1, number(1,4))` with a scripted/seeded
roller pinned so the value is deterministic; per-class `advance_level` draw-set/order/range golden
(assert warrior draws {add_hp} then {add_move}, no add_mana; psionic draws add_mana; ranger move
range 2..4); newbie `GET_AC == 100`, starter gear unworn.

---

## PART B — First combat-round draw-order parity (the payoff)

**Claude builds the scenario** (`scenarios/combat-round.txt`); codex's job is to make Go's `one_hit`
consume the CMWC stream in C's exact order/count. With Part A landed, two co-located level-1 K
warriors (the character-view pair) are **stat-identical**, so a barehand swing is a fair test.

### C draw map — `one_hit` (`src/fight.c:1766+`), level-1 barehand
```c
calc_thaco = thaco[class][level];              /* L1 warrior = 20; then subtract */
calc_thaco -= str_app[..].tohit;               /* deterministic */
calc_thaco -= GET_HITROLL(ch);                 /* deterministic */
calc_thaco -= (GET_INT(ch)-13)/1.5; ... WIS;   /* deterministic */
diceroll = number(1, 20);                       /* DRAW 1 — the to-hit roll */
victim_ac = GET_AC(victim) / 10;
/* hit iff (diceroll<20 && AWAKE) && (diceroll==1 || (calc_thaco - diceroll) > victim_ac) */
   /* on HIT, barehand (no wielded weapon, PC): */
   dam += GET_DAMROLL(ch);                       /* deterministic */
   dam += number(0, GET_LEVEL(ch)/3);            /* DRAW 2 — at L1 = number(0,0) = 0 but STILL A DRAW */
```
**Per barehand swing: 1 draw (to-hit), then IFF it lands, 1 more draw (bare-hand dam, value 0 at
L1 but it consumes the stream).** The Go swing must draw in exactly this order and count.

### Conditional draws — MUST be accounted for (these are the desync traps)
- **`dam_message` (fight.c:1174):** `if (!IS_MOB(victim) || !number(0,20))` — for a **mob** victim
  this DRAWS `number(0,20)`; for a **player** victim it short-circuits (no draw). → Prefer a
  **player-vs-player** first scenario so this draw doesn't exist (if DP permits the PK; if not,
  Claude will use a fixed mob and add the `number(0,20)` to the expected map).
- Skill-proc draws at fight.c:603/606 are `GET_LEVEL>5 / >20` gated → **skipped at L1**.
- Crit/wither/parry draws (fight.c:1917+, 1949+) are level/skill gated → **skipped at L1**.
- Mount draws (fight.c:1548/1552) require a mount → **absent**.
Codex: audit Go's combat round for any draw NOT in the C map above (an extra `number()` anywhere in
the L1 barehand path desyncs everything downstream). The Go round must reproduce C's draw sequence
exactly; where Go's combat structure draws in a different order, reorder to match `one_hit`.

### Part B gate
`--scenario combat-round` (Claude authors) → the swing transcript (hit/miss + damage lines the
actor sees) is **clean** under `DP_SEED`. Unit tests: a seeded/scripted round asserts draw order
(to-hit before damage; no phantom draws) and that `number(0,0)` bare-hand damage still consumes one
draw. Keep `ScriptedRoller` assertions on the exact roll sequence.

---

## PART C — consolidation stragglers (light)
Phase 0a's import guard already blocks new `math/rand`. Confirm: (a) no production `math/rand`
remains (guard green); (b) document the handful of Go-only RNG consumers that ride the shared
stream but have no C counterpart (they must still draw from `dprng` so the stream stays coherent —
list them so Claude can mark their oracle status for Phase 2). Nothing else new here.

## Acceptance gate (whole PR / or split A then B)
1. **Oracle:** `score [actor]` fully clean (Part A); `combat-round` swing clean under `DP_SEED`
   (Part B, Claude's scenario).
2. **Unit tests:** per-class `advance_level` draw-set/order/range; newbie AC=100 + gear unworn;
   combat swing draw-order + zero-damage-still-draws.
3. `make check-fmt vet` + `go test ./...` green; import guard green; no WS schema break.
4. **May land as two PRs** (0b-1 creation parity, 0b-2 combat parity) if cleaner to review — Part B
   depends on Part A being in.

## Gotchas
- **Never touch the oracle** ([[darkpawns-oracle-proof-gate]]). C is the reference; Go conforms.
- **Draw order is everything.** HP matching but move not = a draw seam issue between them; hunt the
  phantom/missing/mis-ranged draw, don't just tweak the move constant to 85 (that would hide a real
  ordering bug that resurfaces in Phase 2). Prove it via the seeded per-class draw golden.
- **AC is NOT RNG** — don't chase it in the stream; it's the starting model (AC 100 / gear unworn).
- **`number(0,0)` still draws.** The L1 bare-hand damage term consumes a stream value even though
  it's always 0 — omitting it desyncs every subsequent draw in the fight.
- This is the template for Phase 2: every future spell/skill scenario is "map C's draws → make Go
  draw the same → oracle-prove under a shared seed."
