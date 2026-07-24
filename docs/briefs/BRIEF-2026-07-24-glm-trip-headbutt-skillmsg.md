# BRIEF (glm) — DP-1207 class: reroute DoTrip + DoHeadbutt through skill_message

**Owner:** glm-5.2. **Gate:** byte/draw-exact unit tests now (mirror
`kick_skillmsg_test.go` / `backstab_skillmsg_test.go`); Claude authors + runs the
`combat-trip-opener` and `combat-headbutt-opener` oracle gates after merge. CI green.
**Git:** branch off `main` as `glm/trip-headbutt-skillmsg`. Edit → commit → push →
open a PR. Do NOT merge. Sized to one PR (M — two functions, same mechanism).
**Finding:** the DP-1203/DP-1207 skill_message class, extended to **trip** and
**headbutt** (R5c — "find one, find the class"). **This is exactly the kick fix
(#464) you just did, twice more.** The machinery all exists; you wire two more
handlers into it and delete invented strings.
**Cite (R5e — verified against C, read them yourself before copying):**
`src/new_cmds.c` (do_trip:735, do_headbutt:368 — NOT act.offensive.c),
`src/spells.h:192` (`SKILL_TRIP 144`), `:189` (`SKILL_HEADBUTT 141`),
`src/fight.c` (damage()→skill_message + set_fighting); Go
`pkg/game/skill_combat.go` (DoKick:229 — the reference you just wrote; DoTrip:282;
DoHeadbutt:366), `pkg/game/death.go:714` (`SkillHeadbuttNum = 141`); rules
**R1/R3/R4/R5c/R5e**.

> **Rescue is NOT in this batch.** `do_rescue` (new_cmds.c) fails with
> `"You fail the rescue!"` and swaps the fight via `set_fighting` — it never calls
> `damage(…, SKILL_RESCUE)`, so there is no skill_message to route. Different class;
> leave `DoRescue` alone. (Claude will check rescue separately.)

---

## The bug (same as kick)

C `do_trip` and `do_headbutt` both call `damage(ch, vict, dam, SKILL_X)`, which
routes the message through **skill_message** (the skill's set from
`lib/misc/messages`, drawing `dice(1, number_of_attacks)`) **and starts combat**
(`set_fighting`). Go's `DoTrip`/`DoHeadbutt` instead return **hardcoded**
`MessageToCh/Vict/Room` strings and never enroll combat — R4 (invented message,
skips the `dice(1,N)` draw — R3) + combat-start gap. Both are message-only, NOT
outcome flips (the roll math is already faithful — unlike bash/DP-1210).

Verified C truth (read each `do_` to confirm before copying):

| skill | C miss / hit damage | msg set | Go `Skill*Num` | Go hit dam today |
|---|---|---|---|---|
| **trip** | miss `damage(…,0,SKILL_TRIP)` (nc:800) · hit `damage(…,(GET_LEVEL/2)+1,SKILL_TRIP)` (nc:805) | **144** | **MISSING — add `SkillTripNum = 144`** | `(level/2)+1` ✓ |
| **headbutt** | miss `damage(…,0,SKILL_HEADBUTT)` (nc:440) · hit `damage(…,GET_LEVEL,SKILL_HEADBUTT)` (nc:449) | **141** | `SkillHeadbuttNum = 141` ✓ (death.go:714) | `GetLevel()` (full) ✓ |

Both message sets exist in `lib/misc/messages` (trip 144: "You trip $N, who lands
with a strange \*CRAAACK\*"; headbutt 141: "You bang heads with $N, crushing $S
skull!"). Confirm with `messages.Variants(SkillTripNum/SkillHeadbuttNum)`.

## The fix — mirror DoKick (skill_combat.go:249-278) exactly

For **each** function, replace the two hardcoded outcome-return blocks:
- **miss** (`percent > prob`): `SkillMsgType: Skill<X>Num`, `StartCombat: true`,
  keep the existing `WaitCh`. No `MessageToCh/Vict/Room`.
- **hit**: `Success: true`, `Damage: <the C amount above>`, `SkillMsgType:
  Skill<X>Num`, keep `WaitCh` + the existing `improveSkill` call.
- **Delete** the hardcoded strings and the now-unused `chPronouns`/`victPronouns`
  (the gate/self/mounted messages above don't use them — build will confirm).

**For trip:** add `SkillTripNum = 144` to the const block in `death.go` (next to
`SkillHeadbuttNum`), and add it to the corpse-attack switch alongside the other
bludgeon/trip-type skills (mirror how `SkillHeadbuttNum` is handled — verify the
right `AttackType` for trip; it's a knockdown, check `fight.c` case for
SKILL_TRIP). Headbutt's constant already exists — just use it.

### ⚠️ What to LEAVE UNTOUCHED (message + combat-start only — do NOT refactor)
- **All formula/gate logic and its draws.** Trip's `number(1,121) + max(vict−ch,0)`
  and immort/sleeping `percent=101`; headbutt's `number(1,121)`, the
  `number(1,200)` gate draw, the recoil (`level/4`|`/3`), the `"But that could
  kill you!"` HP gate, the god-headbutt case, and the peaceful-room-first ordering.
  These stay byte/draw-identical — the reroute only swaps message *emission*.
- **`WaitCh`** — keep each function's current value; don't normalize to kick's 3.

## Draw parity (R3 — headbutt is the risky one)
The reroute **adds** the `skill_message` `dice(1,N)` draw at the point C calls
`damage()` — i.e. after the existing percent/gate draws, before combat proceeds.
- **Trip (player path):** `number(1,121)` → `dice(1,N)`. Clean, like kick.
- **Headbutt (player path):** the existing draws (`number(1,121)`, and the
  `number(1,200)` gate the port already makes) must remain in their current order,
  with the `dice(1,N)` slotting where `damage()` is called. **Trace C `do_headbutt`
  and assert the exact draw sequence** — a mis-ordered or missing draw here is the
  invisible desync that reddens the oracle. If you find the existing headbutt draw
  count already diverges from C (the code comments hint at past draw trouble),
  STOP and flag it as a separate finding — don't fold a formula fix into this reroute.

## Tests (`pkg/game/trip_skillmsg_test.go`, `pkg/game/headbutt_skillmsg_test.go`)
Mirror `kick_skillmsg_test.go` for each skill:
- **miss:** `SkillMsgType == Skill<X>Num`, `Damage == 0`, `StartCombat == true`,
  **empty** `MessageToCh/Vict/Room`; `messages.Variants(Skill<X>Num)` resolves.
- **hit:** `SkillMsgType == Skill<X>Num`, `Damage == <C amount>`, `Success`.
- **draw count/order:** assert the shared-stream advance matches C's sequence
  (trip: 121-roll → dice; headbutt: mirror its full existing sequence + the new dice).
- **DP-1206 gate regression:** `GetSkill(trip/headbutt)==0` still returns the bare
  unlearned message unchanged.

## Oracle gate (Claude, after merge — informational)
I author `combat-trip-opener` + `combat-headbutt-opener` (skillset the skill 75%
onto the warrior peer, opener probe) and run them red→green. Backstab/kick/unlearned
openers stay green as anchors.

## Guardrails
- **Never** edit `src/`, `darkpawns-c-oracle/`, or `lib/misc/messages` — read-only.
- All gates (AGENTS.md §Build & Verify): build, vet, `test ./... -race`,
  `golangci-lint run`, `gofumpt -l .` empty, `make reachability`. Don't stage
  `.zcode/`, generated reachability reports, `website/static/map/world-sphere.json`,
  or `docs/reports/reek/*`.

## Deliverable
`DoTrip` + `DoHeadbutt` rerouted through `skill_message` (`SkillMsgType`) with
combat-start on miss, `SkillTripNum = 144` added, hardcoded strings + unused
pronoun vars deleted, all formula/gate/draw/WaitCh logic untouched, draw order
verified per skill, and the mirror tests. Claude greens the trip + headbutt openers.
