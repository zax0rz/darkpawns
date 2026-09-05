# Dated Handoff: 2026-08-27 (blocked-row burn-down wave)

Three self-merged rounds (PRs #665, #666, #667) converted **16 blocked rows**
across the vehicle groups named in the wave brief; the blocked count went
**37 → 22** and actionable completion 93.5% → 95.5% (`make fidelity-depth`,
476 cases).

## Round 1 — mounted vehicle (glm/depth-mounted, PR #665)

- Pure coverage (port already faithful): sit/rest/sleep.mounted-gate
  (position-mounted-gates — all three share C's "You can't rest while
  mounted." line, sit/sleep borrow rest's text), assist.mounted-gate
  (arg-independent, proven arg-less from the saddle), rescue.mounted-gate
  (combat-entry-mounted-rescue: God `skillset 'rescue' 75` warmup, the horse
  doubles as rescue target).
- hit.dismount-branch RED: C damage() runs stop_follower(victim) when the
  attacker IS the victim's master (fight.c:1457-1458) — the charm branch's
  "A horse hates your guts!" (utils.c:411-414) landed before the swing
  message on BOTH the miss and hit arms. Fixed via cbStopFollowerOfMaster
  wired from performOneHit + combat.TakeDamage, a faithful StopFollowerMob
  (charm trio, plain-unfollow room lines, unmount), and the player
  StopFollower's ToVict call fixed to pass the leader (the line previously
  rendered to a nil vict). C quirk kept: the AFF_CHARM bit SURVIVES
  stop_follower for ride-charm (affect_from_char is a no-op there), which
  is what keeps an attacked mount unrideable afterwards.
- quaff.wear-hold-priority RED: do_use resolves WEAR_HOLD before the
  carrying list (act.other.c:897-910). World.HeldItemVis now feeds cmdQuaff
  AND cmdRecite (same C block, R5c), the held copy unequips+extracts, and
  mag_objectmagic's WAIT_STATE(PULSE_VIOLENCE) — missing for both — now
  lands (unit: TestCmdQuaffHoldPriority / TestCmdReciteHoldPriority; new
  unit-green row recite.wear-hold-priority — no holdable scroll vehicle
  exists in indexed obj files).

## Round 2 — combat-state vehicle (glm/depth-fightpos, PR #666)

- Pure coverage, no source change: sit/rest/sleep/stand.while-fighting
  (position-fighting) + rescue.fighting-victim
  (combat-entry-rescue-fighting).
- Vehicle pattern worth reusing: `hit trainee` in drained setup + a warmup
  pulse pad of exactly PULSE_VIOLENCE+2 pulses spends do_hit's WAIT_STATE on
  BOTH engines (C drains wait on wall-clock main-loop passes; Go on pumped
  pulses), so probe commands execute immediately on both. No RNG runs in
  the probe (frozen clock holds the fight mid-round).
- Harness footguns: C's nanny rejects DIGITS in character names — a "2" in
  a God name desynced the whole creation dialogue ("Invalid name" cascade →
  EOF). Go's cmdForce has a 3-second per-target rate limiter (and a
  denylist) that C lacks — multi-force scenarios need spaced steps or
  multiple peers; the pulse-pad pattern avoids force entirely.

## Round 3 — cheap-vehicle group (glm/depth-cheaparms, PR #667)

- tell.npc-nodesc pure coverage: Go's player-only tell resolution already
  answers NOPERSON for a plain spawned mob.
- hit.shopkeeper-gate RED: Go had NO .shp loader — no mob was ever a
  shopkeeper. New parser.ParseAllShopFiles (boot_the_shops field order,
  sscanf %d semantics for "10wheat" buy-words; unit:
  TestParseAllShopFiles_FieldOrder) → World.ShopBitvectorForKeeper;
  isShopkeeper carries C's full set (shp keepers ∪ guild/guild_guard/
  butler/clerk specs ∪ vnums 8003-8011,8078); the gate renders
  ok_damage_shopkeeper's prelude (slap + "Get out of here before I call the
  guards!" tell, shop.c:1006-1023) before "Ha ha... Don't think so.".
  NOTE: the shop ECONOMY (buy/sell/list, shop spec procs) is still unported
  — only keeper membership + the damage gate are live.
- kill.immortal-instinstakill RED: the old blocker was STALE — under
  empty-players both Gods start at 1204 and a `goto` warmup puts the God in
  the mortal start room. Instakill is now a faithful raw_kill (death cry to
  room+adjacent, corpse, no invented victim bytes). NEW blocked row
  kill.immortal-postdeath-menu: C's deferred extraction (heartbeat) returns
  the dead PC to CON_MENU with the menu text — Go needs a deferred-
  extraction arm + a session menu-return path; its own round.
- echo.immort RED ×2: Manager.BroadcastToRoom silently DROPPED every
  non-JSON payload (telnet writeLoop json.Unmarshals each send) — raw
  payloads are now wrapped at the sink, repairing the whole raw-text caller
  class (door knocks, wiz_object gestures, teleport/home); and cmdEcho's
  broadcast included the echoer (C's TO_ROOM excludes ch).

## Remaining blocked rows (22), grouped by the obvious next vehicles

- god-set extensions (the goal's round 4, NOT reached this wave): verify
  the C oracle's `set` supports stat/position fields (wis 0, position
  stunned), fix Go cmdSet stat clamps if divergent → say.stupid-gate, the
  ×.stunned-or-worse arms (stand/sit/rest/sleep), wake.target-bad-shape.
- pulse/time vehicles: time.clock-variants, weather.sky-variants.
- charmed/board/linkless vehicles: hit.charm-master, tell.linkless,
  tell.writing, assist.cant-see, assist.mob-helpee-pers, reply.no-arg.
- still no vehicle found: wake.cant-wake-aff-sleep (save-gated sleep),
  objmagic.sleep-entry-gates, recite.wrong-type, use.effect, door
  .lock-unlock-pick, where.immort-zone-arg, score.state-variants, and the
  new kill.immortal-postdeath-menu.

## Watch-list from the round-3 sweep (unproven, not rows)

- cmdEcho has an invented 500-char cap; cmdForce's denylist/limiter are
  R4 inventions; cmdGoto/cmdAt are vnum-only (C accepts character names).
- castTeleport still carries the never-matching transferWorld assertion
  (carried forward from the objindex round).
- Pre-existing REDs on main stand: guild-practice, backstab-aware-trio.
