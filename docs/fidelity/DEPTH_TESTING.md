# Command Depth Fidelity

This is the handoff guide for finishing the Dark Pawns port by behavior rather
than by file count. The Go implementation may exist and a command may have one
green oracle scenario while substantial player-visible branches remain wrong.

## Strategy: Breadth, Then Depth

Breadth answers: **does every registered command have at least one live C-vs-Go
probe?** It catches missing handlers, obvious messages, and broad command-surface
gaps. PR #598 completed the registered-command breadth backlog.

Depth answers: **does one command match across its reachable behavior tree?** A
depth pass maps the actual C call path, enumerates gates and outcomes, proves
each audience and state transition, checks RNG draw parity, and identifies
shared behavior that belongs to another manifest.

Do not infer completion from a breadth scenario. Do not duplicate every shared
callee branch under every caller: prove the caller/callee boundary, then delegate
the callee's full matrix to its own depth manifest.

## Proof Levels

Use D1-D5 as a practical review order, not as a claim that complexity is linear:

- **D1 — entry gates:** position, argument, authorization, and early-return bytes.
- **D2 — direct outcomes:** success/failure branches and terminal messages.
- **D3 — audiences and topology:** actor, victim, origin room, destination room,
  object/room state, and failures returned by direct callees.
- **D4 — state transitions:** combat teardown, XP/resources, affects, cooldowns,
  ordering, and exact arithmetic. Use focused unit tests when state is not visible.
- **D5 — hidden/re-entrant behavior:** RNG draw parity, automatic combat calls,
  NPC/specproc entry, wait-state interactions, and other non-command call sites.

The C source is the case inventory. Follow the real call path (R5e); summaries,
Go tests, and the existing Go implementation are not behavioral authority.

## Per-Command Workflow

1. Locate the registered Go handler and the C handler/call sites.
2. Enumerate reachable branches and player audiences before changing Go.
3. Create `docs/fidelity/depth/<command>.tsv`, one row per case.
4. Prefer an oracle scenario for observable bytes. Add named peers when room or
   victim output matters. Use unit tests for exact hidden state/arithmetic.
5. Add `# depth-case: <case-id>` to each proving scenario.
6. For RNG, run several `--seed` values and expose the selected draw in output;
   equal final outcomes alone do not establish R3.
7. Run with `--show-oracle` at least once to verify the intended C block executed.
8. Fix only confirmed divergences, checking the entire behavior class under R5c.
9. Run `make fidelity-depth`, the relevant oracle matrix, and all repository gates.

Manifest statuses are deliberately explicit:

- `oracle-green` / `oracle-green-multiseed`: live transcript proof.
- `unit-green`: focused state-level proof tied to a real test symbol.
- `delegated`: shared callee behavior owned by another named manifest.
- `excluded`: unreachable from this surface; state the owning surface.
- `blocked`: a real proof gap. Do not relabel inconvenience as exclusion.

`scripts/gen_fidelity_depth.py` validates scenario annotations and unit-test
symbols, then prints the actionable completion report. The manifest is the
durable evidence; prose percentages are only snapshots.

## Harness Lessons From the `flee` Pilot

- World topology is test input. Disposable `replace-room-exits` and
  `set-room-flag` fixtures make movement branches deterministic without touching
  `src/`, the C checkout, or checked-in world data.
- Recipient bytes matter. A primary actor plus origin/destination peers exposed
  canonical leave/arrival text that an actor-only probe missed.
- Seed matching is necessary but insufficient. All six exits in the success
  fixture led to one destination, while observer arrival text revealed which
  direction the RNG selected.
- Successful movement must use the canonical movement path. Low-level relocation
  can match destination state while inventing or omitting player-facing bytes.
- Some descriptor-issued timing states are unreachable because command input is
  drained only after wait reaches zero. A synchronous `force` or focused unit
  test can exercise the same handler call path without pretending queued input did.
- `--show-oracle` is a development requirement for timing-sensitive probes; a
  no-diff report is meaningless if the intended block never ran.
- State such as exact XP deltas may be silent in C. Prove the arithmetic in a
  unit test and separately prove that no invented player message appears (R4).

## Dated Handoff: 2026-08-23

- Registered-command breadth coverage was completed and merged in PR #598.
- `flee` is the first depth pilot, implemented in PR #601.
- Its manifest is `docs/fidelity/depth/flee.tsv`: 14 mapped cases, 13 actionable
  cases proven/delegated, zero blocked, and one NPC-only case excluded to the
  future mob/specproc surface.
- Transitive boat, charm, tunnel, mount, special, and greet failures belong in a
  movement depth manifest; `flee` proves the callee false-return edge.
- The movement depth pass is now captured in `docs/fidelity/depth/movement.tsv`.
  It exposed and fixed destination-look ordering during follower recursion and
  added disposable exit-keyword and room-sector fixtures. Ordinary movement,
  closed exits, boats, tunnels, vertical audiences, follower state, resource
  costs, and death traps are proven.
- Mounted movement is now implemented and proven as a vertical slice: spawned
  mobs have C's 50-point movement pool, rider/mount pairs transfer together,
  only mounts pay movement cost, failure gates match, and room observations
  represent the pair once. Focused script recording now proves the final shared
  movement ordering boundary: destination look, then mob greet, then room-enter
  script. The movement depth manifest has no remaining actionable gaps.

## Dated Handoff: 2026-08-25

- Object-magic effect messages (round 1 of the depth-fidelity loop) are captured
  in `docs/fidelity/depth/object-magic.tsv`: every `mag_affects` switch arm was
  audited against `src/magic.c`'s to_vict/to_room/to_self strings and the
  magic.c:1414-1421 send block (to_self gated on `ch != victim`, then to_vict,
  then to_room). Missing lines were added across the whole class (R5c),
  invented curse/poison/sleep lines were removed (R4), and a `$M`/`$m`
  objective-pronoun helper now backs the substitutions.
- Nine single-spell, zero-RNG potions/scrolls prove the live bytes (one
  wait-setting item command per scenario, quaff-effect pattern), including
  peer-proven to_room lines and targeted-recite to_self lines.
- Save-gated spells (chill touch, blindness, curse, poison, sleep, flamestrike)
  are unit-proven with the save outcome fixed; live oracle parity stays blocked
  pending call_magic RNG-stream work.
- Harness: quaff/recite set WAIT_STATE PULSE_VIOLENCE, so post-item commands
  stall behind the frozen clock — keep the item command last and pad with
  `~dpclock pulse 20` steps. Zone-reset loads roll `percent_load`, so
  probabilistic prototypes need the new `force-load <vnum>` fixture; boot-loaded
  vnums need spawn max headroom.

## Dated Handoff: 2026-08-25 (disposal round)

- The object-disposal family (`junk`, `donate` — both `do_drop` subcommands;
  `sacrifice` does not exist in the C command table) is captured in
  `docs/fidelity/depth/disposal.tsv`, 27/28 proven/delegated. Junk is fully
  oracle-green; donate's `number(0,3)` routing draw was proven to take the same
  branch on both servers across six seeds (R3), with a peer standing in
  donation room 1 via a `replace-room-exits` fixture.
- That peer exposed a real R1 divergence: C `act()` uppercases the assembled
  message (`CAP(lbuf)`, comm.c:2477), so donation-room appear lines render
  "A loaf of bread suddenly appears..." — Go emitted them lowercase. Fixed for
  the item and gold appear lines; the gold line is unit-proven (newbie
  characters hold no gold — the `drop.gold-success` unit-test precedent).
- A near-miss worth recording: the donate draw-0 arm leaves `RDR = 0`, and
  `NOWHERE` is -1, so C's "Sorry, you can't donate anything right now." gate
  never fires mid-draw; Go's mode-gated room check matches C exactly. Read the
  defines before "fixing" gates (R5e).
## Dated Handoff: 2026-08-25 (position round)

- The position-command family (`stand`, `sit`, `rest`, `sleep`, `wake`) is a
  pure-coverage depth round: `docs/fidelity/depth/position.tsv` walks the
  do_stand/do_sit/do_rest/do_sleep/do_wake state machine (src/act.movement.c:
  696-880) with three GREEN scenarios — actor-only transitions, room-audience
  pairs with a male peer, and targeted wake against a peer whose setup ends in
  `sleep` (proving the TO_VICT|TO_SLEEP delivery resolves the waker's name and
  lands the victim in POS_SITTING). No source change: the port was already
  faithful, including C's per-arm act() hide flags, do_rest's default arm
  landing in POS_SITTING, and the do_sleep room-line "lie down" typo.
- The POS_FIGHTING arms are blocked pending a combat-entry round, which must
  also port do_hit's WAIT_STATE(PULSE_VIOLENCE+2) (act.offensive.c:127) — Go
  currently executes commands immediately after `kill`. Mounted gates await a
  mount manifest; the magical-sleep and bad-shape wake arms await
  RNG/stunned vehicles.

## Dated Handoff: 2026-08-26 (combat-entry round)

- The combat-entry family (`hit`/`murder`, `kill` routing, `assist`, `rescue`)
  is captured in `docs/fidelity/depth/combat-entry.tsv`, 27/38 proven/delegated.
  Five of six scenarios were RED on pre-fix main; the fixes: do_hit's
  WAIT_STATE(PULSE_VIOLENCE+2) now fires even when damage()'s peaceful/newbie
  gates block the swing; StartCombat stands the ATTACKER to POS_FIGHTING at
  entry (C set_fighting, fight.c:223), which do_hit's "You do the best you
  can!" branch depends on; assist's gates were rewritten (C's two-space
  already-fighting text, NOPERSON, self gate, $M pronoun, TO_NOTVICT room
  exclusion, and the immediate hit() swing with its no-wait quirk); rescue's
  resolution failure answers with the same prompt as no-arg and the
  nobody-fighting line uses $M.
- The defender-side POS_FIGHTING stand is deliberately deferred to a
  retaliation/damage-matrix round — standing the defender at entry interlocks
  with the damage path's position model (AWAKE/wimpy gates) and several
  engine tests model downed combatants via enrollment.
- Scenario shape notes: a gate-blocked swing still costs the attacker the
  round on both servers, so queued commands lag symmetrically behind the
  frozen clock — keep the wait-setting command last or accept identical lag;
  NEVER pump pulses in a room with a live fight unless you want violence
  rounds in the transcript.

## Dated Handoff: 2026-08-26 (info round)

- The info-display family (`time`, `weather`, `where`, `score`, `who`) is
  captured in `docs/fidelity/depth/info.tsv`, 11/15 proven. The family was
  largely already faithful (the act-informative sweep stays GREEN); the round's
  one real find was `where` listing players in Go's random sessions-map order —
  intermittently RED — where C walks descriptor_list newest-connection-first.
  `cmdWhere` now sorts by connectedAt descending, matching the existing
  `cmdWho` pattern.
- The immortal `where` proved comparable after all: under empty-players both
  servers' Gods start at 1204, so the listing (peer + God) is a live oracle
  case rather than blocked on the DP-1205 start-room note.

## Dated Handoff: 2026-08-26 (comm round)

- The communication family (`say`, `tell`, `whisper`, `ask`, `reply`, `emote`)
  is captured in `docs/fidelity/depth/comm.tsv`, 21/34 proven: pure coverage —
  the port was already faithful across every reachable branch, including say's
  four punctuation variants, the whisper/ask vict/self/others trios, tell's
  gates plus the soundproof-room arm (ROOM_SOUNDPROOF bit 5 via set-room-flag),
  reply's no-target and success arms, and emote's pair.
- say.drunk-speech stays blocked: the drink fullness gate (FULL>20 && THIRST>0)
  blocks repeated drinking for fresh characters and pulse-driven thirst decay
  (1800 pulses ≈ 24 MUD hours) does not clear it — a condition-control fixture
  is needed.
- Harness discovery: Guinness (18007) lives in 180.obj, which is in NEITHER
  tree's obj index — the game never loads that file, so consumables.txt's
  guinness steps have been vacuously failing identically on both servers.
  When a scenario's C transcript shows "You don't see X here", believe it and
  check the obj index before trusting GREEN.

Update this dated section when the frontier materially changes. Keep the rest
of this document stable unless the methodology itself changes.
