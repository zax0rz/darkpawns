# BRIEF (glm) — derive combat message attack-type from the wielded weapon (DP-1204)

**Owner:** glm-5.2. **Gate:** Claude runs the differential oracle red→green on
`combat-backstab-opener` (goes FULLY green) + `combat-death` stays green + reviews
unit tests; CI green.
**Git:** branch off `main` as `glm/weapon-attacktype`, commit, push, open a PR. Do
NOT merge. Sized to one PR (S/M).
**Closes:** DP-1204. **Related:** DP-1203 (merged #447 — fixed backstab; this is
the *other* half of the same red scenario).
**Cite:** `src/fight.c:1792-1806` (one_hit w_type derivation), `src/fight.c:889-1020`
(dam_message / attack_hit_text), rules **R1**, **R3** (`docs/fidelity/RULEBOOK.md`).

## The bug (one sentence)

Every player weapon attack renders the generic verb **"hit"** instead of the
weapon's verb (**"pierce"** for a dagger, "slash", "whip", …), because
`performOneHit` hardcodes the message attack-type to `AttackNormal` and never
looks at the wielded weapon.

Observed (thief with a dagger, post-backstab rounds):
```
C : You scratch a guard trainee as you pierce him.   /  You barely pierce a guard trainee.
Go: You scratch a guard trainee as you hit him.      /  You barely hit a guard trainee.
```
Invisible until now only because every prior combat fixture fought **barehand**.

## The C truth (fight.c one_hit)

```c
if (wielded && GET_OBJ_TYPE(wielded) == ITEM_WEAPON)
    w_type = GET_OBJ_VAL(wielded, 3) + TYPE_HIT;       // dagger val3=11 → TYPE_PIERCE
else if (IS_NPC(ch) && ch->mob_specials.attack_type != 0)
    w_type = ch->mob_specials.attack_type + TYPE_HIT;  // mob's natural attack
else
    w_type = TYPE_HIT;                                  // barehand
```
`w_type` feeds **only the message** (`dam_message`/`skill_message` →
`attack_hit_text[w_type - TYPE_HIT]`). It does **not** change the damage number.

## How Go's message path already works (do NOT rebuild it)

The real messages come from `MessageFunc` → `combat.SendWeaponMessage(dam, ch,
victim, attackType)` (`fight_core.go:774`), which treats `attackType` as the
**0-based offset** (i.e. C's `val3` = `w_type - TYPE_HIT`):
```go
func SendWeaponMessage(dam int, ch, victim Combatant, attackType int) {
    if dam == 0 || victim.GetHP() <= -11 {
        if cbSkillMessage(dam, ch.GetName(), victim.GetName(), TYPE_HIT+attackType, ...) { return }
    }
    DamMessage(dam, ch, victim, attackType)
}
```
So **offset 0 → "hit", offset 11 → "pierce"**. `performOneHit`
(`engine.go`) passes `pair.LastAttackType = int(AttackNormal)`. `AttackNormal`
is `0` (`formulas.go:25`), which *coincidentally* equals the "hit" offset — which
is why barehand is correct and everything else is wrong.

## The fix

`performOneHit` (and the miss branch) must pass the **weapon-derived offset** to
`sendHitMessage`/`sendMissMessage` — i.e. C's `val3`:
- **player** wielding a weapon → the wield slot's `Values[3]` (e.g. 11 for the
  dagger);
- **mob** with a natural `attack_type` → that value;
- **barehand / no special** → `0` (renders "hit", unchanged).

### Three traps — get these exactly right (they won't all fail loudly):

1. **Offset scheme, not TYPE_\*.** `SendWeaponMessage` **adds `TYPE_HIT` itself**.
   Pass `val3` (11), NOT `TYPE_PIERCE` (311) and NOT `TYPE_HIT+val3`. Passing a
   `TYPE_*` constant double-adds and misses the set.
2. **Decouple damage from message (R3/damage-correctness).** `CalculateDamage`
   branches on `attackType == AttackNormal` to apply AC reduction (`getMinusDam`,
   `formulas.go:564`). **Keep passing `AttackNormal` to `CalculateDamage`** — the
   damage math must not change. Only the value handed to the *message* senders
   changes. Use two distinct locals (e.g. keep `AttackNormal` for damage, compute
   `msgAttackType` for the message).
3. **Miss branch staleness.** `pair.LastAttackType` is only assigned *after* a
   hit; on a miss the message sender currently reads a stale/zero value. Compute
   `msgAttackType` fresh each round and pass it to **both** `sendMissMessage` and
   `sendHitMessage`.

### Getting the weapon offset into the engine

The engine works on the `Combatant` interface (no equipment access). There is a
**latent, unwired** callback for exactly this: `GameCallbacks.GetWeaponInfo(chName)
(wType, damDice, damSize, isBlessed int/bool)` (`callbacks.go:52`) — currently
defined but never assigned or consumed. Wire it (game-layer impl) to return the
attacker's **`val3` offset** as `wType` (wielded weapon's `Values[3]`; else mob
`attack_type`; else `0`), and call it from `performOneHit`. (Alternatively add a
focused `Combatant` accessor — your call — but `GetWeaponInfo` is the intended
slot; if you wire it, confirm `wType` is the **offset**, not `TYPE_HIT+offset`,
and don't disturb the existing `GetDamageRoll` dice path.)

## Tests

- **verb from weapon:** a player wielding a piercing weapon (`Values[3]==11`)
  produces a **"pierce"** hit/miss message (offset 11 reaches `SendWeaponMessage`);
  barehand → "hit" (offset 0). Assert via the message callback capture (see
  `pkg/game/backstab_skillmsg_test.go` `wireBackstabMessages` for the pattern).
- **damage unchanged:** a wielded-weapon hit deals the same damage as before the
  change (AC reduction still applies — `CalculateDamage` still gets `AttackNormal`).
- **mob path:** a mob with a natural `attack_type` renders its verb; a plain mob
  → "hit".

## Oracle gate (Claude authors/runs)

- **`combat-backstab-opener` FULLY GREEN** — with DP-1203 already merged, the only
  remaining divergence is this pierce/hit one; fixing it closes the scenario.
- **`combat-death` stays GREEN** — barehand warrior, offset 0, `TYPE_HIT` path
  unchanged (the regression anchor).

## Guardrails

- **Never** edit `src/`, `darkpawns-c-oracle/`, or `lib/misc/messages` — reference only.
- `make reachability` zero regressions; `go test -race`; **run `golangci-lint`**;
  `gofumpt -w` every file you touch (worktree pushes bypass the hook).
- Don't stage `website/static/map/world-sphere.json` or `docs/reports/reek/*`.

## Deliverable

`performOneHit` derives the message attack-type offset from the wielded weapon
(via a wired `GetWeaponInfo` or a `Combatant` accessor), passes it to
`sendHitMessage`/`sendMissMessage` only (damage keeps `AttackNormal`), with the
miss branch fixed and unit tests. Claude runs the oracle gate.
