# Dated Handoff: 2026-08-27 (socials, channels, and harness closer)

This wave began from clean `main` at the spell-vehicle checkpoint of **499
total, 483 proven/delegated, 6 blocked, 10 excluded**. It completed three
family rounds, one PR per round, and self-merged each PR only after the branch
was rebased on current `main` and hosted test, lint, and security checks were
green under the 2026-08-27 amendment. The final checkpoint is **513 total,
498 proven/delegated, 5 blocked, 10 excluded**; actionable completion is
**498/503 = 99.0%**.

## Round 1 — socials (PR #680)

Branch `glm/depth-socials`; merged as `a0de8ced2`.

- C tracing covered social loading and `do_action()` in
  `src/act.social.c:102-151,220-275`, the `lib/misc/socials` records, and
  `perform_act()` substitution/capitalization in `src/comm.c`. The audit
  enumerated self-only, victim-trio, not-found, self-target, and shared
  no-argument/gate arms across the full 187-record table.
- The required RED on main was a self-only `bah` with typed input: C emitted
  the actor/room pair while Go treated the `#` sentinel as a missing target.
  Go now honors C's NULL sentinels and typed-argument behavior, preserves
  exact trailing spaces/empty lines, and applies the C visibility boundary.
- `socials.tsv` contains live representatives for each structure class,
  high-traffic `smile`, `laugh`, `nod`, `wave`, and `bow`, plus data-driven
  table and visibility proofs. Seeds 1, 2, 3, 5, and 8 all reported no
  normalized divergence.

## Round 2 — channels (PR #681)

Branch `glm/depth-channels`; merged as `05cc82f9f`.

- C tracing covered `do_gen_comm()` in `src/act.comm.c:1146+`, the channel
  flag table, soundproof and position gates, level gates, holler movement
  cost, and the zone/global recipient loops. Go's sender-versus-recipient
  deaf flag split, `grats` vocabulary, holler registration, leading-space
  parsing, level gates, and 20-point holler cost were aligned.
- The required RED on main was `holler`: C emitted `You holler, 'world
  hello'` and the peer audience line while Go returned `Huh?!?` and no
  audience output. The live `channels-depth` vehicle is now green, as are
  the sender-noshout and soundproof vehicles; the focused gate test covers
  recipient flags, holler cost/exhaustion, and level gates.
- `channels.tsv` records vocabulary, recipient gates, zone boundary,
  sender-noshout, soundproof, and level-gate cases. The zone fixture proves
  shout's same-zone audience versus gossip's world audience.

## Round 3 — harness closer (PR #682)

Branch `glm/depth-closer`; merged as `ca56488f4`.

- C tracing verified `close_socket()` in `src/comm.c:2086-2148` leaves a
  playing character in `character_list` with `desc == NULL`, and
  `do_tell()`'s linkless arm in `src/act.comm.c:921-922`. The new
  `[peer-drop]` fixture closes a named peer TCP connection after setup and
  before the probe. Go now retains that player as linkless, stops only the
  detached transport writer, and leaves extraction to the existing linkdead
  reaper; orderly quit still unregisters normally.
- The required RED was C `He's linkless at the moment.` versus Go `No-one by
  that name here.`. `tell-linkless-depth` is now green with the exact C
  pronoun substitution, and `comm.tsv:tell.linkless` is converted.
- The score vehicle follows C `do_score()` in
  `src/act.informative.c:1168-1405`: a first-player God walks through
  sleep/wake/rest/sit/stand and runs `set <god> wis 0`. The pre-fix RED was
  C's empty pack versus Go's starter gear; Go now skips mortal `do_start`
  gear for the first-player God, matching C's `init_char`/`do_start` gate.
  `score.state-variants.position` is green. The C `set` surface does not
  invent a position field; position is reached through the registered
  position commands.
- `info.tsv:score.state-variants` remains blocked for affect flags,
  mounts/pets, tattoo, and naked/armed state branches. `score-state-depth`
  is the durable proof for the reachable position subset.

All fixes are Go-only; `src/` and `darkpawns-c-oracle/` were not edited.
Rules applied: R1 player-facing bytes, R2 command surface, R3 deterministic
draw/state parity, R4 no invention, and R5/R5e call-path verification.

## Remaining blocked frontier and owners

These five blocked rows are intentional proof gaps, not exclusions:

- `combat-entry.tsv:hit.charm-master` — combat-entry owner; needs a charmed
  attacker/master relation and the friendship gate vehicle.
- `combat-entry.tsv:assist.mob-helpee-pers` — combat-entry owner; needs a mob
  helpee vehicle proving C `PERS` rendering versus Go's short description.
- `combat-entry.tsv:kill.immortal-postdeath-menu` — combat-entry owner; needs
  its own full deferred extraction/heartbeat and `CON_MENU` return session.
- `info.tsv:score.state-variants` — info owner; affect, mount/pet, tattoo,
  and naked/armed branches remain unproven after the position subset.
- `object-magic.tsv:objmagic.sleep-entry-gates` — object-magic owner; the
  quaff entry is unreachable because sleep is `TAR_NOT_SELF`; the reachable
  cast surface remains separately proven by
  `objmagic.sleep-entry-gates.cast`.

No deep-engine backlog item was attempted in this wave, and no owner was
removed from a blocked row. In particular, postdeath-menu remains deferred
as a standalone extraction/menu-return session.

## Verification

- `make fidelity-depth` — **513 total, 498 proven/delegated, 5 blocked, 10
  excluded**; exit 0.
- `go build ./...` — pass.
- `go vet ./...` — pass.
- `go test ./...` — pass.
- `golangci-lint run ./...` — **0 issues**.
- `gofumpt -l .` — no output.
- Current live vehicles `socials-depth`, `channels-depth`,
  `comm-noshout`, `comm-soundproof`, `tell-linkless-depth`, and
  `score-state-depth` — no normalized divergence on the final branch.
- Socials matrix seeds 1, 2, 3, 5, and 8 — no normalized divergence.
- PRs #680, #681, and #682 — hosted test, lint, and security checks passed;
  all three were self-merged.
- Final repository state: clean `main` at `ca56488f4`.
