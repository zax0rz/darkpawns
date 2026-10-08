# Spell message ordering repair (R1, R3, R5e, R5h)

Stop tier: changes shared spell damage state and player-visible message selection.

C subtracts damage and calls update_pos at src/fight.c:1484-1493, then
skill_message at 1534-1543. Wound/death notices follow at 1560-1583.
Go formerly selected the message before applying damage, so lethal flame arrow
printed its hit text. The retained seed-1 oracle now prints the C death text.

The repair subtracts positive adjusted damage and silently classifies position
before EmitSkillMessage. UpdatePositionAfterDamage remains afterward: its wound
messages and fighting cleanup must not move before the skill message. The
existing World.HandleDeath boundary still runs exactly once. Damage gates,
modifiers, zero-damage return, breath enrollment and the legacy non-Combatant
fallback are unchanged. No callback registration or global RNG change.

Readers: the only inflictDamage callers in damage_spells.go are the spoken
mag_damage path and its breath damage(0) bridge. The separate melee damage tail
already updates HP/position before its skill message. File-backed skill message
selection reads victim HP to distinguish die/hit, then routes room, attacker,
and victim bytes. Tests cover all three audiences and exactly one selector draw
for lethal, wounded, zero, immortal and breath-zero cases. State/order tests
also cover mortally wounded, incapacitated and stunned bands, prove wound output
has not happened at selection, and retain the fighting body until later cleanup.

Locks: no new lock acquisition. TakeDamage, GetHP, GetPosition and SetPosition
use each body's existing accessors, releasing their internal locks before message
callbacks. No world, combat-engine or lifecycle lock is introduced or retained
across EmitSkillMessage or World.HandleDeath.

Reproduce the compiling green/revert/restore control from repository root:

    python3 docs/fidelity/depth/handoff/2026-10-07-spell-message-order-control.py

Focused race proof:

    go test -race ./pkg/spells -run 'TestSpellMessage|TestInflictDamage|TestSpellDamage|TestCombatBodyBreath' -count=1

Oracle vehicle: spell-lethal-message-order. Reference seed 1 output and controls
are retained under ~/Archives/darkpawns/oracle-runs/2026-10-07/dp-1371-spell-message-order-proofs/.
Full combined census must cover the committed tip before review is complete.
