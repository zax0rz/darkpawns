# 2026-08-30 — depth-fidelity handoff: SPECIAL(roach)

## Starting frontier

- Starting main: `d976004dc` (merged `glm/spec-prostitute`, PR #809).
- Queue slice consumed: the next active special in source-definition order,
  `SPECIAL(roach)`, registered as `ASSIGNMOB(23, roach)` at
  `src/spec_assign.c:185` and defined at `src/spec_procs3.c:707-804`.
- The source inventory and newest prior handoff were read after the required
  `main` checkout/pull and `make fidelity-depth` confirmation. No C source or
  `darkpawns-c-oracle/` file was edited.

## C call-path audit

The player-command path is `special()` in `src/interpreter.c:1407-1456`: it
walks mobiles in the actor's room and invokes the registered procedure with
the player as `ch`, the roach as `me`, the numeric command, and the argument.
The autonomous path is `mobile_activity()` in `src/mobact.c:68-93`: after the
fighting/awake gate it invokes the procedure with the mob as both C character
arguments and `cmd == 0`. The Go adapter's autonomous call has `ch == nil`, so
the port uses `me` for all mob state and room acts; the player-command path is
still command-nonempty fallthrough for this pulse-only procedure.

Reachable branches, in C order:

- nonzero command, missing character, or a sleeping actor: `FALSE`, silent;
- two nested `number(0,10000)` starvation checks (the second draw occurs only
  when the first is zero), then `GET_MAX_HIT < 11`: room act, pending
  `extract_char`, and `TRUE`, with no corpse/death pipeline;
- first room object satisfying `CAN_WEAR(i, ITEM_WEAR_TAKE)`: feed room act,
  then either growth or burp; growth adds integer `cost/2`, independently
  increments `damnodice` and `damsizedice`, and either emits stretching or
  rumbling text, or resets the original and silently reads a new vnum-23 roach
  when max HP exceeds 400; the food object is always extracted and the proc
  returns `TRUE`;
- no food: `number(0,10)` cases 0–3 emit the four idle room acts and return
  `FALSE`; case 4 has a nested `number(0,5)` gate, then chooses a room by C
  RNUM, rejects PRIVATE/GODROOM/NOMAGIC/DEATH with the fade-in-place act and
  `FALSE`, or emits hide-invisible origin/destination fade acts, calls
  `look_at_room`, and returns `TRUE`; cases 5–10 are silent fallthrough.

## RED, fix, and proof

The first RED on main used the registered vnum-23 fixture and seed 1. The
oracle emitted `A large roach ...` while the port's direct room formatter
emitted lowercase `a large roach ...`. A food vehicle then showed the oracle
consuming the takeable bread and following the C food branch while the port
left the bread in the room, followed a different idle path, and used the
wrong room-audience bytes. The final vehicle moves the warrior's authored
bread out of its starter pack during the probe; this avoids the wizard
`random_load_msg()` draw and isolates roach behavior.

The confirmed Go changes are:

- route every roach player-facing write through canonical `Act`, preserving C
  capitalization, `$n`/`$p` substitution, audience, and hide-invisible flags;
- use `IsTakeable()` for the C `ITEM_WEAR_TAKE` predicate and keep the
  unconditional object extraction;
- preserve nested starvation and case-4 teleport draw order through a local
  `roachNumber` seam;
- replace `HandleDeath` with corpse-free extraction and spawner bookkeeping;
- preserve per-instance C damage-dice mutations in typed mob runtime state;
- reset the original roach and use silent `read_mobile`-equivalent spawning on
  reproduction, including C's final `ch` assignment quirk for the new roach.

Proof is GREEN for seeds 1, 2, 3, 5, and 8 with
`cmd/dp-oracle-diff/scenarios/spec-proc-roach.txt`. Focused tests are
`TestSpecRoachEntryGates`, `TestSpecRoachFoodSuccessGrowthAndExtraction`,
`TestSpecRoachFoodFailureStillExtractsObject`,
`TestSpecRoachSplitResetsOriginalAndSpawnsQuietRoach`,
`TestSpecRoachTeleportSubgateAndAudience`, and
`TestSpecRoachStarvationExtractsWithoutCorpse`.

## Frontier counts and handoff

After the slice, `make fidelity-depth` reports:

- 1,323 total cases;
- 1,269 proven/delegated;
- 14 blocked;
- 40 excluded;
- actionable completion: 1,269/1,283 (98.9%).

The special-procedure inventory still contains the single blocked
`objmagic.sleep-entry-gates` row; the blocked row and interpreter command
family sweep remain later queue phases. The next special-definition position
after roach is `mortician`, but its active vnum-8095 procedure is already
claimed in the manifest; continue scanning source/registration order to the
next unclaimed active registration rather than repicking it. Rules applied:
R1/R2 for exact bytes and dispatch surface, R3 for nested draw order and
five-seed proof, R4 for no invented branches, R5e for the verified C call
path, and R5b/R5c for the typed state/audience class audit.
