# BRIEF (glm) — DP-1215: melee combat-round parity — stand-up round draws, NPC wait semantics, PC stand-up, act() capitalization

**Owner:** glm-5.2. **Gate:** draw-exact unit tests in `pkg/combat` (recording
roller) now; orchestrator runs the `combat-trip-opener` +
`combat-headbutt-opener` oracle gates red→green after merge. CI green.
**Git:** branch off `main` as `glm/melee-round-parity`. Edit → commit → push →
open a PR. Do NOT merge. Sized to one PR (M — one function reordered + one
message class, all in `pkg/combat`).
**Finding:** DP-1215 — post-knockdown melee rounds diverge from C: scramble not
capitalized, stand-up round skips a draw AND the mob's attack, no PC stand-up
branch, attacker position gate too strict. Rules **R1/R3a/R3b/R3d/R4/R5c/R5e**.
**Verified NOT bugs (do not "fix" these — already faithful, R5e):**
- **Unarmed miss/hit message routing.** Go already routes melee misses through
  `lib/misc/messages` set 300 (`SendWeaponMessage` → `cbSkillMessage`,
  `pkg/combat/fight_core.go:774-781`). "You try to hit $N who easily avoids the
  blow." (messages:420) and "You wildly punch at the air, missing $N."
  (messages:406) are both set-300 miss variants — the oracle's wording
  difference was the `Dice(1,4)` landing on a different variant because of the
  draw-stream offset (Fix 2), not a missing/wrong message. **No change.**
- **Round order.** Go's `combatOrder` head-prepends (`engine.go:319-335`) and
  iterates head-first (`engine.go:383-408`) — mob acts before the tripping
  player, same LIFO as C's `combat_list` (`src/fight.c:214-215`, `:1906`).
  **No change.**

**Cite (R5e — verified against C, read them yourself before editing):**
`src/fight.c` (perform_violence:1906-2025; hit:1762-1892; set_fighting:205-225;
damage combat-start:1403-1445; skill_message:1022-1095), `src/comm.c:2397-2477`
(perform_act + `CAP(lbuf)` at :2477), `src/utils.h:166` (`CAP`), `:462-464`
(`WAIT_STATE`), `:468` (`CHECK_WAIT`); Go `pkg/combat/engine.go:443-560`
(processCombatPair), `pkg/combat/fight_core.go:762-769`
(`capitalizeFightMessage`); `lib/world/mob/163.mob:44-57` (guard trainee:
attack_type 0, `1d4+0`).

---

## C truth: perform_violence's per-combatant sequence (src/fight.c:1906-2025)

For each combatant, in this EXACT order:

1. **Attacks-count computation** (:1910-1947). NPC: level-banded base, then one
   unconditional draw `if (number(0, 900)<GET_LEVEL(ch)) attacks++;` (:1917).
   PC: `attacks=1`, class-gated `number(1,100)` draws (:1928-1937), then
   `number(0,500)` if level ≤ 30 (:1939). **These draws happen even if the
   combatant is downed, waited, or about to stop fighting.**
2. **Parry/dodge pre-check** (:1949-1973). PC: `number(0,10000)` drawn for
   EVERY PC in combat_list EVERY round, before the FIGHTING checks (:1949).
   NPC: `number(0,100)` only if `IS_NPC && AFF_DODGE` (:1966). On success C
   emits act() messages (parry :1957-1961, dodge :1969-1971) and sets
   `IS_PARRIED(FIGHTING(ch))`.
3. **NPC wait + stand-up** (:1975-1988):
   ```c
   if (GET_MOB_WAIT(ch) > 0) { GET_MOB_WAIT(ch) -= PULSE_VIOLENCE; attacks = 0; }
   if ((GET_POS(ch) < POS_FIGHTING) && !GET_MOB_WAIT(ch)) {
       GET_POS(ch) = POS_FIGHTING;
       act("$n scrambles to $s feet!", TRUE, ch, 0, 0, TO_ROOM);
       send_to_char("You drag yourself to your feet.\r\n", ch);
   }
   ```
   **Critical:** `GET_MOB_WAIT` is `mob_specials.wait_state`. `WAIT_STATE(ch,
   cycle)` (`src/utils.h:462-464`) writes `ch->wait` for ANY non-NULL ch —
   including mobs — and **never** `GET_MOB_WAIT`. The only `GET_MOB_WAIT` writer
   in C is the Lua bridge (`src/scripts.c:2017`). So a tripped/bashed mob
   (`WAIT_STATE(victim, PULSE_VIOLENCE)`, `src/new_cmds.c:813`) arrives with
   `GET_MOB_WAIT == 0`: attacks NOT zeroed, stands up, **and attacks in the
   same round**. The `attacks = 0` path is effectively Lua-only.
4. **PC stand-up** (:1990-1998): `if (!IS_NPC(ch)) if ((GET_POS(ch) <
   POS_FIGHTING) && !CHECK_WAIT(ch))` → stand, same scramble act(), same
   `"You drag yourself to your feet.\r\n"`. `CHECK_WAIT(ch)` = `(ch)->wait > 1`
   (`src/utils.h:468`), read-only here; PC wait decrements once per game-loop
   pass in the descriptor loop (`src/comm.c:597`), NOT per violence pulse.
5. **IS_PARRIED attack adjustment + clamp** (:1999-2007).
6. **Attack loop** (:2009-2025): `hit(ch, FIGHTING(ch), TYPE_UNDEFINED)` gated
   ONLY on `AWAKE(ch)` (= `GET_POS > POS_SLEEPING`) and same-room; otherwise
   `stop_fighting`. **A sitting/resting attacker still swings** — position
   below POS_FIGHTING does NOT stop combat or skip the attack. There is no
   position-≺-FIGHTING early-exit.

`act()` capitalization: `perform_act` (`src/comm.c:2397`) uppercases the first
byte of the fully-expanded string for EVERY audience via `CAP(lbuf)`
(`src/comm.c:2477`, `#define CAP(st) (*(st) = UPPER(*(st)), st)` at
`src/utils.h:166`). The mob's name is lowercase in the data
(`a guard trainee~`, `lib/world/mob/163.mob:46`); C prints `A guard trainee…`
because of CAP, always.

`hit()` draws (both paths): `number(1,20)` FIRST (`src/fight.c:1815`), then hit
path rolls damage dice (NPC: `dice(damnodice, damsizedice)` :1848; barehand PC:
`number(0, GET_LEVEL/3)` :1851), miss path calls `damage(…,0,w_type)` →
`skill_message` draws `dice(1, number_of_attacks)` (`src/fight.c:1035`).

## Go today (verified, pkg/combat/engine.go:443-560)

`processCombatPair` runs the wait/stand block FIRST (`:454-481`), then validity
checks (`:485-501`), redirects (`:503`), THEN `GetAttacksPerRound` (`:514`) and
`prepareRoundDefense` (`:520`). Divergences:

1. **Stand-up round draw skip (R3a/R3b).** When a waited NPC stands
   (`waitedThisRound`), the early `return` at `:479` skips
   `GetAttacksPerRound` — the NPC's `Number(0,900)` draw
   (`pkg/combat/formulas.go:598`) never happens. C draws it at :1917 BEFORE the
   wait block. One-draw stream shift → downstream hit/miss flips (the DP-1215
   outcome flip; same class as DP-1212).
2. **NPC wait semantics wrong (R4/R3d).** Go zeroes NPC attacks from the
   generic wait state (`wc.GetWaitState() > 0` → `waitedThisRound = true`,
   `:457-462`). C zeroes only on `GET_MOB_WAIT`, which nothing in normal
   gameplay writes — `WAIT_STATE` on a mob writes the OTHER field
   (`utils.h:462-464`). Go has exactly one wait field, written for mobs by
   `sendSkillResult` (`pkg/command/skill_commands.go:1619-1621`, `WaitTarget`)
   — the C `ch->wait` analogue, which perform_violence ignores for NPCs.
   Result: Go's tripped mob skips its stand-up-round attack; C's mob stands
   AND swings (matches the C transcript: scramble → mob swing → player swing,
   all in one round).
3. **No PC stand-up branch (R1/R4).** The whole block is `if attacker.IsNPC()`.
   C stands PCs too (`fight.c:1990-1998`). A knocked-down Go player never
   scrambles back via the combat engine.
4. **Position gate too strict (R4).** `if attacker.GetPosition() < PosFighting
   || defender.GetPosition() == PosDead { StopCombat }` (`:486-491`). C stops
   only when the attacker is NOT AWAKE (`GET_POS <= POS_SLEEPING`) or in a
   different room (`fight.c:2010-2016`) — a sitting (tripped, still-waited)
   attacker keeps swinging. Verify Go's position constants against
   `src/structs.h` POS_* before editing.
5. **Scramble + parry/dodge not capitalized (R1).** `engine.go:475` uses raw
   `GetName()` in a hand-rolled `fmt.Sprintf`. The parry/dodge messages at
   `engine.go:563/567/572/576` have verbatim C text (`fight.c:1957-1971`) but
   are emitted via `SendMessage`/observer paths with no capitalization; C
   routes them through `act()` → CAP.

## The fix (pkg/combat, one PR)

### Fix 1 — capitalize name-initial combat messages (R1, R5c class audit)

C capitalize rule: uppercase the FIRST BYTE of the fully-composed message
(`CAP(lbuf)`), for every audience. Go already has
`capitalizeFightMessage` (`pkg/combat/fight_core.go:762-769`) — same package,
use it.

- **engine.go:475 (scramble)** — capitalize the composed string. (Cite:
  `src/comm.c:2477`, `src/fight.c:1984/1994`.)
- **engine.go:563, 567, 572, 576 (parry/dodge)** — text already matches C
  byte-for-byte; apply the same capitalization. (Cite: `src/fight.c:1957-1971`.)
- **Class audit, do NOT expand scope:** the remaining name-initial sites
  (`engine.go:678` scream, `:798`/`:816`/`:820` hit-miss fallbacks, `:856`
  killed-by; `fight_core.go:176-186`, `:359-360`, `:405-406`, `:428-429`,
  `:455-467`) are fallback/dead paths in production (`MessageFunc` bypasses
  them). Verify reachability per site (R5e); for any that IS reachable and has
  a C `act()` counterpart, capitalize; for the rest add
  `// TODO(port): fallback path — no verified C counterpart, message unverified`
  and move on. **Do not invent or reword strings (R4).**

### Fix 2 — stand-up round parity (the core; R3a/R3b/R3d)

Reorder `processCombatPair` to C's per-combatant sequence:

1. **Move the draw-bearing blocks to the top**, before ALL early exits:
   `GetAttacksPerRound` (the NPC `Number(0,900)` / PC `Number(0,500)` draws)
   and `prepareRoundDefense` (the PC `Number(0,10000)` / dodge draws) run
   FIRST, matching C :1910-1973 preceding :1975+. Keep them per-combatant;
   C draws these even when the attack loop later no-ops.
   - **Do NOT** move `applyMobCombatRedirects` in this PR. Its draws are
     short-circuited off in the gated scenarios (defender is a PC). If you find
     a reachable draw-order divergence there, flag it in the PR — don't fold it
     in.
2. **NPC wait: GET_MOB_WAIT-only.** Delete the `waitedThisRound` early-return.
   C's `attacks = 0` fires only for `GET_MOB_WAIT > 0`, which only Lua writes
   (`src/scripts.c:2017`); Go's scripting layer writes no mob wait (verified:
   no `SetWaitState` in `pkg/scripting`). So in Go, NPC attacks are NEVER
   zeroed by wait. Keep `DecrementWaitState` only if another system reads the
   value (audit readers of mob `GetWaitState` — report what you find in the
   PR); otherwise drop the decrement too, with a comment citing
   `utils.h:462-464` (C never decrements a mob's `ch->wait` either).
3. **NPC stand-up** stays (pos < PosFighting → stand + scramble), now reachable
   every round since wait never blocks it — matching C (mob-wait always 0).
4. **Add the PC stand-up branch** (`src/fight.c:1990-1998`): `!IsNPC &&
   GetPosition() < PosFighting && wait <= 1` (C `!CHECK_WAIT`, `utils.h:468`)
   → `SetPosition(PosFighting)` + the scramble BroadcastFunc (capitalized) +
   `"You drag yourself to your feet.\r\n"` to the stander. Read the player's
   wait via the same `waitStateHolder` interface; do NOT decrement it here
   (PC wait drains in the heartbeat, `pkg/session/manager.go:580-590` — C
   drains it in the descriptor loop, `comm.c:597`).
5. **Fix the position gate** (`:486-491`): stop combat only when the attacker
   is not AWAKE (C: `GET_POS <= POS_SLEEPING` → verify the Go constant
   equivalents against `src/structs.h:210-220`) or out of room; keep the
   `defender == PosDead` extraction. A downed-but-awake attacker (sitting,
   waited) continues to the attack loop and swings from the ground
   (`fight.c:2010-2025`).

**Leave untouched (R3):** every draw's parameters and the draws inside
`CalculateHitChance`/`CalculateDamage`/`SendWeaponMessage` (d20, damage dice,
`Dice(1,4)` message select) — byte/draw-identical. `WaitTarget` writes in
`pkg/command/skill_commands.go` stay as-is (they mirror C's `WAIT_STATE`;
only the engine's *interpretation* changes).

## Tests (`pkg/combat/engine_standup_test.go` — new)

Use a recording roller (wrap `GetRoller()` to log each Number/Dice call with
args) and the existing engine test harness patterns.

- **Stand-up round, full draw sequence:** enroll player + mob (mob downed,
  `SetWaitState(1)`), run one round. Assert the draw log equals C's sequence:
  mob `Number(0,900)` → mob `Number(1,20)` → (miss: `Dice(1,4)` | hit:
  `Dice(1,4)` damage) → player `Number(0,500)` → player `Number(0,10000)` →
  player `Number(1,20)` → message/damage draw. (Adjust to each side's actual
  gated draws per `formulas.go`; the point is the mob's `0,900` is present and
  FIRST for the mob.)
- **Mob attacks on stand-up round:** after the round, assert the mob both
  stood (position PosFighting) AND produced an attack (hit or miss message);
  scramble broadcast emitted exactly once.
- **Capitalization:** scramble for a mob named `a guard trainee` (sex male)
  emits `A guard trainee scrambles to his feet!`; parry/dodge strings start
  uppercase.
- **PC stand-up:** downed player with wait 0/1 stands, scramble + `You drag
  yourself to your feet.\r\n` delivered to the player; downed player with
  wait 2 stays sitting but STILL attacks (AWAKE gate), no scramble.
- **Regression anchors:** existing combat tests stay green; the trip/bash
  skill tests (`pkg/game/*_skillmsg_test.go`) untouched.

## Oracle gate (orchestrator, after merge — informational)

`combat-trip-opener` + `combat-headbutt-opener` must run RED on pre-fix `main`
and GREEN on the branch. Anchors that must stay green: `combat-death`,
`combat-backstab-opener`, `combat-kick-opener`, `combat-bash-opener`,
`combat-unlearned-bash-opener`. Per R5a, outcome parity (the hit/miss flip) is
verified by the paired draw-trace, not a single threshold green. **Expected
side effect:** the bash outcome flip (DP-1210) is the same stand-up-round
class — if `combat-bash-opener` greens, DP-1210 closes.

## Guardrails

- **Never** edit `src/`, `darkpawns-c-oracle/`, or `lib/misc/messages` — read-only.
- All gates (AGENTS.md §Build & Verify): build, vet, `test ./... -race`,
  `golangci-lint run`, `gofumpt -l .` empty, `make reachability`. Don't stage
  `.zcode/`, generated reachability reports, `website/static/map/world-sphere.json`,
  or `docs/reports/reek/*`.

## Deliverable

`processCombatPair` reordered to C's per-combatant sequence (draws first);
NPC wait no longer zeroes attacks (GET_MOB_WAIT semantics, cited); PC stand-up
branch ported with both messages; position gate relaxed to AWAKE; scramble +
parry/dodge messages capitalized via `capitalizeFightMessage`; dead fallback
sites marked `TODO(port)`; the recording-roller tests. Orchestrator greens the
trip + headbutt openers and re-checks DP-1210.
