# 2026-09-24: one C function, five Go seams

Field notes from DP-1327, where skills skipped C `damage()`'s protections.
Claims here are backed by the ledger rows named in each section.

## 1. A chokepoint the port split into five pieces

C has one `damage()` (`src/fight.c:1312`). Every skill, spell, mob special and
melee swing reaches its protection block: peaceful rooms, the level-10
player-killing protections, and shopkeepers (`fight.c:1318-1368`). The port
had five stand-ins for that function, and no two applied the same checks:

- `combat.takeDamage` had all four checks, but it refused silently, without
  C's refusal line.
- `World.DoSpellDamage`, which the skill command tail used for most skills,
  had none of them.
- `spells.inflictDamage` had the two level checks with their messages, and
  no peaceful or shopkeeper check.
- The melee engine had only the shopkeeper check, silently, and before
  `hit()`'s dice rather than after them.
- The mob-special seams had one peaceful check between them.

The `hit` and `assist` commands each carried a complete, correct copy at the
command layer, which is why player-versus-player melee looked right in play.
Any skill that didn't route through one of those commands let a level-40
character hurt a level-1 player that C protects. The fix is one gate,
`World.DamageRefused`, which every seam asks (PF-016).

The pattern is worth naming. A single C function with many callers is a
chokepoint. When a port reimplements it per caller, the subsets drift, and
every caller that works is a copy someone happened to get right. The oracle
found it one caller at a time. Only the class sweep (R5c) found all five.

## 2. A delegation that pointed at the wrong proof

The groinrip manifest had a `delegated` row, `groinrip.shared-damage-gates`,
saying the damage seam's peaceful, newbie and shopkeeper behaviour was
covered by the "verified combat-entry matrix". That matrix covered `do_hit`'s
command-layer gates. Groinrip never passed through them. The row was
well-formed and validated, and it cited a real proof, but not a proof of this
path (PF-017).

`delegated` means the case is proven somewhere else. That only holds if the
delegate is on the case's actual call path, which R5e asks for and nothing
in the manifest tooling checks. The coincidence-green pattern is agreement
that proves nothing. This is the manifest version of it: a citation that
proves something else.

## 3. Re-proving the fix found the next invention

The ledger asked for the sleeping-victim path to be re-proved with a victim
above level 10, since a level-1 victim now goes to the protection instead.
The obvious scenario uses `advance` to raise the victim to level 11. In C the
victim then died from a 40-point groinrip. In the port it survived. The
cause was unrelated to damage: the Go `AdvanceLevel` refills hit points,
mana and move to the new maximum, and C's `advance_level` never does
(`src/class.c:698-712`). Every player who levels in the port gets a free
full heal (DP-1329, PF-018).

The heal had survived the census because nothing checked current hit points
straight after a level-up. `stat` shows maximums, and the normalizer hides
the prompt. Writing the proof a fix asks for is itself a probe: the scenario
has to reach states that no earlier scenario reached.

## 4. A name passed as an argument

The first census on the fix failed one scenario, `spec-proc-cityguard-breed`.
A city guard attacks Kane the weaponsmith, a shopkeeper, and the port had
Kane slap the guard. C's `ok_damage_shopkeeper` calls
`do_action(victim, GET_NAME(ch), cmd_slap)`, which hands the attacker's name
to the slap social as if someone had typed it. For a player that finds the
player. For a mob named "a Kir-Oshi guard" it looks for someone called "a",
finds nobody, and prints nothing. The port's first copy of the prelude only
ran for player attackers, so this behaviour of C's never showed. Once the
gate served mob attackers too, the port slapped the guard directly and the
oracle caught it (PF-019).

The rescuer special had already met the same shape (`mobRescueVictim`).
When C passes a name through a command handler, the port has to reproduce
the parse, not the intent.

## 5. The proof chain kept going

With the level-up heal fixed (DP-1329), the level-11 victim died on both
servers, and the victim's own view showed three more inventions on the
player-death path. The victim was shown the room's "GripDozer is dead!
R.I.P." line, which C's `TO_ROOM` never sends it. It also got "You lose N
experience points." and a constitution-loss line, where C's penalties are
silent. None of these had been caught, because no earlier scenario watched a
player die from the victim's side (PF-020).

The last difference is the prompt. C sends the dead victim a vitals prompt
with negative hit points. The normalizer rewrites vitals to `<PROMPT>` and
drops prompt-only lines, but the minus sign survived as `-<PROMPT>`. That's
the first prompt in the census the normalizer failed to hide, and it
exposed that the port sends no prompt after output caused by another
player's command (DP-1307, which had been scoped to the DP_CLOCK pump
alone). Section 2 of the browser-terminal note says a normalizer is part of
the claim (RO-011). Here a negative number punched through it.

One fix asked for one proof. Writing that proof turned up the level-up
heal, then the death bytes, then a wider prompt gap. Each fix let the
scenario reach a state that no scenario had reached before.
