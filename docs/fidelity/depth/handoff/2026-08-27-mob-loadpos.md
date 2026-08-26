# Dated Handoff: 2026-08-27 (mob loadpos + entry-aggro round)

- Round 9's "mob 2109 spawn failure" was mislabeled — the mob spawned fine and
  the parser was innocent (unit-probed: 21.mob parses, zone fixture parses,
  world table holds 2109). What actually happened: Go ignored the mob file's
  loadpos (line-11 first field), so the sleeping orc spawned STANDING and the
  setup settle pulses' mobile_activity let it wander away before the probe;
  C's sleeper never passes the wander gate. `NewMob` now applies
  `proto.Position` (C read_mobile's `*mob = mob_proto` copy, db.c:1757);
  unit-pinned across all loadpos values.
- Second fix in the same scenario: the invented `OnPlayerEnterRoom` aggro hook
  is removed (R4) — C has no room-entry aggro path at all; aggressive mobs
  attack from their own mobile_activity tick, gated on AWAKE. The hook even
  engaged sleeping mobs. Production aggro still runs via the pulse-driven
  block, which already had C's awake gate.
- NEW finding, blocked with owner: the kill opener's miss-variant selection
  (lib/misc/messages) diverged on seed 1 ("wildly punch at the air" vs "swing
  your fist") while seeds 2-3 agreed — a draw-offset class for the combat
  message round.
- Harness additions: `--show-go-log` flag dumps the Go port's server log after
  the report; the spawner now has no per-spawn logging, so failures are the
  only signal — temporary INFO debug lines (spawn/wander/extract) found the
  wander this round and were removed after.
