# Dated Handoff: 2026-08-27 (defender-stand round)

- The round-4 deferral landed: StartCombat now stands the DEFENDER to
  POS_FIGHTING at entry, mirroring C set_fighting's unconditional fight.c:223
  (both the AWAKE attack gate and CalculateHitChance's awake-defender AC read
  it). Unit-proven; TestHandleDeath_PassesAttackType now runs on a fixed dprng
  stream because an awake defender's dex AC plus natural-1 auto-miss turned
  its 10-swing retry loop into a ~0.5% flake.
- The live-oracle attempt found two NEW divergences that blocked a clean
  vehicle, both recorded as blocked rows with owners in combat-entry.tsv:
  Go engages aggro-flagged mobs on ROOM ENTRY (C's aggro fires only in
  mobile_activity on pulses — behind the frozen clock C never aggros
  pre-kill), and mob 2109 (index-listed, 21.mob) never spawns in the Go port
  while the C oracle loads it — likely a Go mob-parser gap on that record
  shape.
- Vehicle hunting lessons: mob-file position lives on the record's
  loadpos/defaultpos line (line 11 in the standard shape — line 10 is
  attack/gold; several records use 2- or 4-field variants); the obvious
  "sleeping" mobs are aggro-flagged or scripted, which entangles mob-AI
  timing with any position proof.
