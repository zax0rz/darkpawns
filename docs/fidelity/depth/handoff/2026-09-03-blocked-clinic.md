# Blocked-clinic handoff — 2026-09-03

## Scope and baseline

This clinic used the refreshed `origin/main` at `79c5a5a32` after the
exclusion audit. The depth report was 4,111 total cases: 4,013
proven/delegated, 47 blocked, and 51 excluded. C/oracle files were not
edited. Each live vehicle used a 300-second timeout and `--show-oracle` for
the diagnostic runs; empty pulse blocks are recorded as unproven dispatch,
not as green proof.

The objective's bounded-attempt rule is satisfied by the existing evidence
for force and sleep, by the prior two-attempt records for the combat clinics,
and by the refreshed runs below. No row is retried merely to seek a green
transcript.

## Clinic results

### Force NPC command interpreter

`force-mob` already has two current-main attempts (seeds 1 and 2) in
`2026-09-03-triage-force.md`. Both resolve player targets but fail to resolve
the visible NPC target in Go, while C executes the generic
`command_interpreter` and emits the sit/stand room action. The shared NPC
interpreter remains an architectural gap; both force NPC rows stay blocked.

### Object-magic sleep entry

The one permitted `objmagic.sleep-entry-gates` attempt is already recorded in
`2026-08-28-objectmagic-sleep-entry-gates.md`. C's potion self-target route is
rejected by `TAR_NOT_SELF`, while the separate cast-sleep vehicle is already
green on seeds 1, 2, 3, 5, and 8. No second object-magic attempt was made.

### Janitor pulse dispatch

The refreshed `spec-proc-janitor` vehicle was run with `DP_SEED=1` and `2`.
Both runs completed normally with no normalized divergence but had empty C
and Go pulse blocks: the registered janitor was not dispatched in the
vehicle. This is absence of an executed proof block, not a parity claim;
`mob.janitor-pulse-dispatch` remains blocked under R2/R5b/R5c/R5e.

### Cityguard pulse dispatch and breed-killer arm

The refreshed `spec-proc-cityguard` vehicle was run with seeds 1 and 2. Both
reach the exact outlaw warning and synchronous punch opener, then diverge in
the shared death/respawn transcript: C emits the native death cry and prompt,
while Go emits its alternate death and temple sequence. The refreshed
`spec-proc-cityguard-breed` vehicle was also run with seeds 1 and 2; C emits
`A Kir-Oshi guard exclaims, 'Die, nightbreed!!'` while Go is silent. Both
cityguard rows remain blocked, with the downstream shared combat and
`breed_killer` owners named rather than duplicated.

### Paladin combat action — promoted

The refreshed `spec-proc-paladin` vehicle produced no normalized divergence
on seeds 1 and 2 with `--show-oracle`, and no normalized divergence on seeds
3, 5, and 8. The C blocks include the combat action and score snapshots on
each seed; the Go transcript matches them. This is a stable multiseed result,
not a one-seed inference, so `mob.paladin-combat-action` is promoted from
blocked to `oracle-green-multiseed` with proof
`spec-proc-paladin@1,2,3,5,8`.

The promotion is limited to the registered paladin vehicle and its existing
focused special proofs. It does not claim unrelated combat classes green.

### Teleport-victim combat transcript

The refreshed `spec-proc-teleport-victim` vehicle was run with seeds 1 and 2.
Seed 1 retains the extra Go blank line before the special output. Seed 2 also
diverges in the pre-special hit result and in the random landing room, while
both runs reach the exact scoff, speech, teleport, and landing-look boundary.
The row remains blocked at the shared combat/RNG transcript; no special-only
change is justified by these results.

## Result

After the paladin promotion, the expected accounting is 4,111 total, 4,014
proven/delegated, 46 blocked, and 51 excluded. The remaining blockers are
sharp, bounded findings rather than unverified exclusions. The evidence
applies R1/R2/R3/R4/R5b/R5c/R5e: preserve exact bytes and draws, keep the
configured autonomous surface, do not invent an NPC interpreter or combat
transcript, delegate shared owners, and verify the actual call path.

