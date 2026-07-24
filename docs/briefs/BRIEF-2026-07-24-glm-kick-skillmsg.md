# BRIEF (glm) — DP-1207: reroute DoKick through skill_message + start combat

**Owner:** glm-5.2. **Gate:** byte/draw-exact unit tests now (mirror the backstab
skill_message tests); Claude runs the `combat-kick-opener` oracle gate after merge.
CI green.
**Git:** branch off `main` as `glm/kick-skillmsg`. Edit → commit → push → open a PR.
Do NOT merge. Sized to one PR (S).
**Finding:** DP-1207 (High). **This is the DP-1203 backstab pattern, applied to
kick** — the machinery already exists; you are wiring kick into it and deleting
invented strings. **Cite:** `src/act.offensive.c:587-634` (do_kick),
`src/fight.c` (damage()→skill_message + set_fighting), `src/spells.h:181`
(`SKILL_KICK 134`); Go `pkg/game/skill_combat.go:229-278` (DoKick),
`:25-140` (DoBackstab — the reference), `pkg/command/skill_commands.go:1511-1548`
(sendSkillResult); rules **R1/R3/R4/R5c/R5e**.

---

## The bug

C `do_kick` (act.offensive.c:622-633): rolls `percent`, then
- miss → `damage(ch, vict, 0, SKILL_KICK)`
- hit → `damage(ch, vict, GET_LEVEL(ch) >> 1, SKILL_KICK); improve_skill(...)`
- `WAIT_STATE(ch, PULSE_VIOLENCE + 2)` (both branches).

`damage()` does two things Go's `DoKick` skips: (a) emits the message via
**skill_message** — the `SKILL_KICK` set (134) from `lib/misc/messages`, drawing
`dice(1, number_of_attacks)` — and (b) **starts combat** (`set_fighting`).

Go's `DoKick` (skill_combat.go:257-277) instead returns **hardcoded**
`MessageToCh/Vict/Room` (`"You try to kick $N, but miss!"` /
`"You kick $N square in the chest!"`) and **never enrolls combat** — the oracle
shows Go's post-kick pulses empty while C's fight proceeds:

```
[kick trainee]
  C : You miss your kick at a guard trainee's groin, much to his relief...  (skill_message, set 134)
  Go: You try to kick a guard trainee, but miss!                            (invented string)
[~dpclock pulse 20]  C: combat rounds proceed   Go: (empty)
```

Two faults, R4 (invented message, skips the `dice(1,N)` draw — R3) + combat-start gap.

**Note — this is message-only, NOT an outcome flip.** Both C and Go *miss* here,
so the roll/AC math is fine for kick (unlike bash — DP-1210 — where the outcome
itself diverges). Do **not** touch the formula. If the opener still diverges on
*outcome* after this reroute, stop and flag it (that would be a separate kick
AC/draw finding); current data says it won't.

## The fix — mirror DoBackstab exactly

Everything you need already exists — do not add constants or plumbing:
- **`SkillKickNum = 134`** is already defined (`pkg/game/death.go`) and already in
  the corpse-attack switch. **Use it.** (It equals C `SKILL_KICK`, spells.h:181 —
  the messages-file key. Do NOT use any `combat.SKILL_*` enum; the messages-file
  number is the one skill_message keys on — same trap noted for backstab's 131.)
- **`sendSkillResult`** already emits via `SkillMessage(result.Damage, ch, target,
  result.SkillMsgType, room)` when `SkillMsgType != 0` (skill_commands.go:1520-1522),
  and calls `StartCombat` when `result.StartCombat && Damage <= 0` (1544-1547).
- **Set 134** exists in `lib/misc/messages` with the exact kick strings (miss at
  line 276, hit at 273-274) — the same loader backstab (131) uses.

In `DoKick`, replace the two hardcoded return blocks:

```go
// MISS — C: damage(ch, vict, 0, SKILL_KICK): skill_message miss + set_fighting.
if percent > prob {
    return SkillResult{
        Success:      false,
        SkillMsgType: SkillKickNum, // 134 — lib/misc/messages Kick set
        StartCombat:  true,
        WaitCh:       3, // PULSE_VIOLENCE + 2 (act.offensive.c:633)
    }
}

// HIT — C: damage(ch, vict, GET_LEVEL>>1, SKILL_KICK): skill_message hit + damage pipe.
dam := ch.GetLevel() >> 1
improveSkill(ch, SkillKick)
return SkillResult{
    Success:      true,
    Damage:       dam,
    SkillMsgType: SkillKickNum,
    WaitCh:       3, // PULSE_VIOLENCE + 2
}
```

- **Keep** the skill-known gate (`GetSkill(SkillKick)==0` → bare message — DP-1206,
  do not touch), the self/mounted checks, and the `percent`/`prob` formula
  (skill_combat.go:249-250) **exactly as-is**.
- **Delete** the six hardcoded kick strings.
- Follow DoBackstab (skill_combat.go:92-140) for the field pattern — miss sets
  `SkillMsgType + StartCombat`; hit sets `Success + Damage + SkillMsgType` (the
  caller routes `dam>0` through the damage/death pipeline and emits via
  `SkillMessage`, applying HP once — same as backstab).

## Draw parity (R3 — get this right)
C order per kick: **`number(1,101)`** (the percent roll) **then** the
`skill_message` **`dice(1, N)`** (message selection). Go already draws
`dprng.Number(1,101)` in `DoKick` (line 249); the `dice(1,N)` draw happens inside
`SkillMessage` in `sendSkillResult`. So the shared-stream order is `Number(1,101)`
→ `dice(1,N)`, matching backstab's documented order (skill_combat.go:88-91).
Assert this in a test — no extra or missing draws.

## Tests (`pkg/game/kick_skillmsg_test.go` — mirror `backstab_skillmsg_test.go`)
- **miss:** `percent > prob` → result has `SkillMsgType == SkillKickNum` (134),
  `Damage == 0`, `StartCombat == true`, and **empty** `MessageToCh/Vict/Room`
  (no hardcoded string). `messages.Variants(SkillKickNum)` resolves the Kick set.
- **hit:** `SkillMsgType == SkillKickNum`, `Damage == GetLevel()>>1`, `Success`.
- **draw count:** the kick path consumes exactly `Number(1,101)` then the
  `dice(1,N)` from SkillMessage — assert the shared-stream position advances by
  the same count as backstab's miss (mirror the backstab draw-parity test).
- **skill-known gate unchanged** (DP-1206 regression): `GetSkill(kick)==0` still
  returns the bare `"You'd better leave all the martial arts to fighters."`.

## Oracle gate (Claude, after merge — informational)
I run `combat-kick-opener` (committed RED) → expect **GREEN** (skill_message miss
line + combat proceeds under the pulses). `combat-backstab-opener` and the bash
scenarios stay as-is (bash still RED on DP-1210, unrelated).

## Guardrails
- **Never** edit `src/`, `darkpawns-c-oracle/`, or `lib/misc/messages` — read-only.
- `go build ./...`, `go vet ./...`, `go test ./... -race`; **`golangci-lint run`**;
  `gofumpt -w` every file you touch. Don't stage `website/static/map/world-sphere.json`,
  `docs/reports/reek/*`, `.zcode/`, or generated reachability reports.

## Deliverable
`DoKick` rerouted through `skill_message` (`SkillMsgType = SkillKickNum`) with
combat-start on both branches, hardcoded strings deleted, formula + DP-1206 gate
untouched, draw order `Number(1,101)`→`dice(1,N)` preserved, and the mirror tests.
Claude greens `combat-kick-opener`.
