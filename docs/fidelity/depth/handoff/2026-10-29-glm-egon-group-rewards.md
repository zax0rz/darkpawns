# Mixed player/NPC group rewards — 2026-10-29 (GLM)

This round closes the mixed-group class gap #2215 left open: C's
`is_in_group`, `group_gain`, `perform_group_gain`, and the autogold/autosplit
tail count every in-room AFF_GROUP actor, mobiles included
(`src/fight.c:637-657, 708-848`). The Go helpers `combatIsInGroup`,
`combatGroupMembers`, and `combatFollowersInRoom` became actor-inclusive, and
`AwardMobKillXP` now divides and pays by the actor-inclusive count.

## C corrections verified against `src/` this round

- The parent brief's parenthetical "still runs their alignment update" for
  NPC recipients is not what the source does: `change_alignment` returns
  immediately for NPC ch (`src/fight.c:486-489`, "mobiles shouldn't change
  align"). NPC members receive the (descriptor-less, no-op) share send, skip
  `gain_exp`, and their alignment does not move. The Go path skips
  `changeAlignment` for mobile members, which is byte- and state-equivalent.
- `IS_NEUTRAL` is alignment in (-350, 350) (`src/utils.h:453-456`), matching
  the existing Go `changeAlignment` guard.
- `group_gain`'s autosplit pays grouped mobile followers real gold
  (`GET_GOLD(follower) += gold_per_member`, `src/fight.c:798`); the leader
  share block exists only while `gold_per_member > 0`, so with a zero share
  only follower-loop members see the none-to-split notices
  (`src/fight.c:768-823`). Both were divergences in the previous Go split and
  are now covered by unit proof.
- An NPC killer's preferences are its own (no PRF bits), so a mobile killer
  loots nothing and the victim's gold stays for the corpse — the Go corpse
  path already only suppresses corpse gold for player killers with autogold.

## Reachability boundary for NPC-led groups

C can form a group led by a mobile through the implementor surface
(`force <mob> group all` — `do_group` is in `cmd_info`, `src/interpreter.c:471`
— or `do_switch` re-pointing the descriptor at the mob). The Go port cannot
reach that state today: `force` on a mobile routes through the character
actor lane where `group` has no ActorHandler, and a switched implementor's
`cmdGroup` still operates on the implementor's own player. The reward math in
this round is leader-type-agnostic (the leader enters the divisor and
recipient list whatever body it is), but forming an NPC-led group in Go needs
the `group` NPC command lane (an R2e gap in the group command's own manifest,
out of this round's scope). No oracle fixture claims it.

## New evidence

Oracle scenarios (all green on this branch at seeds 1-3 with draw parity
except the disclosed case below):

- `death-mixed-group-pc-kill` — grouped player leader + grouped charmed
  mobile; player kill; share wording from the two-member divisor, autosplit
  pays the mobile ("You share 50 gold with a guard trainee.").
- `death-mixed-group-npc-kill` — the grouped charmed mobile (MOB_HELPER
  assist arm, `src/mobact.c:284-302`) lands the killing blow; the player
  leader receives "You receive your share of experience -- 12 points."
- `death-group-outofroom-solo` — an out-of-room grouped member leaves the
  killer on the solo path (solo wording, no share).

Unit proofs in `pkg/game/party_mixed_group_test.go` cover the exact
actor-inclusive divisor (fails when only NPC acceptance is added while the
divisor stays player-only), the NPC killer path, NPC alignment/XP skips,
autosplit gold paid to a mobile with conservation, the leader-silent
zero-share split, out-of-room exclusion, and follower-list message order.
R5h: reverting only `pkg/game/party.go` and `pkg/game/combat_wire.go` fails
four of these tests (`unit-red.log`) and makes the npc-kill scenario
transcript-red; restoring fails nothing (`unit-green.log`). Both are retained
under `/home/zach/Archives/darkpawns/oracle-runs/2026-10-29/glm-egon-group-rewards/gates/`.

## Disclosed baseline draw divergence (not caused by this round)

`death-mixed-group-npc-kill` matches transcripts but diverges draw streams at
the kill pulse (stream 4217: C continues the violence round's extra-attack
rolls `src/fight.c:1925-1949` while the Go port starts mobact draws). An
otherwise-identical scenario with the pet ungrouped — which executes none of
this round's changed code — shows the same mismatch at the same stream index
on both this branch and the #2215 base. Evidence:
`.../glm-egon-group-rewards/draw-baseline/{ungrouped-mybranch,ungrouped-base}.log`.
This is a pre-existing kill-pulse violence/mobact draw-ordering divergence in
the mob-killer path, kept red and disclosed rather than relabeled; it needs
its own investigation round.

## Incidental findings (recorded, not fixed here)

- C stores hitroll/damroll as `sbyte` (`src/structs.h:916-917`): a `set`
  value of 200 wraps to -56 in C while the Go `set` stores the wide int, so
  any implementor `set <victim> damroll 128..200` diverges combat outcomes.
  Found while building the npc-kill fixture (pet with damroll 200 missed
  everything in C). The fix belongs to the `set` command's manifest with the
  sbyte class audited (affects, equipment, stat display).
- The harness fixture vocabulary gained `set-mob-flag <vnum> HELPER` (bit 12,
  `src/structs.h:259`) so both engines' disposable mob files can enable the
  assist arm symmetrically.

## Census and gates

The targeted census and per-gate logs are linked in the PR description;
`make fidelity-depth` passes with the new rows registered.
