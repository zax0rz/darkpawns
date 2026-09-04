# Offensive activity-surface audit — 2026-09-04

## Scope

This slice audits the 138 literal `act()`/`send_to_char()` call sites in
`src/act.offensive.c`: assist, hit/kill, backstab, disembowel, order, flee,
bash, rescue, kick/dragon-kick, tiger punch, shoot, retreat, subdue, sleeper,
and ambush/neckbreak.

## Existing ownership

The ordinary player-visible branches are already represented by
`combat-entry.tsv`, `backstab.tsv`, `disembowel.tsv`, `order.tsv`, `flee.tsv`,
`bash.tsv`, `combat-entry.tsv` (rescue and hit/kill), `kick.tsv`, `dragon.tsv`,
`shoot.tsv`, `escape.tsv`, `sleeper.tsv`, `ambush.tsv`, and `neckbreak.tsv`.
Native special-procedure callers are owned by `spec-procs.tsv`; shared combat
message and damage seams are delegated to their canonical combat rows. The
existing manifests also record the explicit unreachable NPC shoot-flee path
and other C-reason exclusions.

## Protocol and decision

The slice uses the standard two-seed protocol with a 300-second per-scenario
timeout. The file-level decision remains open until every assigned branch is
green or has a sharply bounded C-reason exclusion. No C or oracle source is
modified, and no offensive branch is silently absorbed into a broad green
claim.

Initial coverage is green for `combat-entry-gates`,
`combat-backstab-opener`, `combat-bash-opener`, `disembowel-depth`,
`order-depth`, `flee-audience-success`, `rescue-roll`, `dragon-depth`, and
`shoot-entry-depth` at seeds 1 and 2. Direct checks of `ambush-depth` at seeds
1 and 2 are red: the Go lethal path inserts one blank line between the room
death line and the XP message. The existing `ambush.tsv` green claim is stale
on this tip and must be fixed or replaced with a sharp blocked note before
this family can be promoted.
