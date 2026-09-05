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

The slice used the standard two-seed protocol with a 300-second per-scenario
timeout. No C or oracle source was modified, and no offensive branch was
silently absorbed into a broad green claim.

Initial coverage is green for `combat-entry-gates`,
`combat-backstab-opener`, `combat-bash-opener`, `disembowel-depth`,
`order-depth`, `flee-audience-success`, `rescue-roll`, `dragon-depth`, and
`shoot-entry-depth` at seeds 1 and 2. Direct checks of `ambush-depth` at seeds
1 and 2 initially exposed a real transport-ordering red: the Go lethal path
inserted one blank line between the room death line and the XP message. C's
`process_output` owns one descriptor buffer for combat and ordinary text, but
the Go combat callbacks bypassed the prompt-invalidation boundary. The fix in
`pkg/session/manager.go` routes combat frames through the same per-session
enqueue boundary as `World.MessageSink`, consuming the DP_CLOCK interruption
CRLF on the first combat frame and retaining output accounting for the trailing
prompt. `TestCombatMessageConsumesPulseInterruptionPrefix` locks the seam.

After the fix, `ambush-depth`, `sleeper-outcome-depth`, and `neckbreak-depth`
all report `no normalized divergence` at seeds 1 and 2. The existing
`ambush.tsv` multi-seed claim is therefore current again. The 138-callsite
inventory classification is promoted to `proven-already`; its remaining
shoot/flee entries are explicit C-reason exclusions already recorded in their
focused manifests.
