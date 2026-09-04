# Depth-fidelity triage handoff — dragon-breath combat transcript

Date: 2026-09-03

Branch: `glm/depth-dragon-breath-triage`

Base: `origin/main` at `9070a56ee`

## Verdict

`mob.dragon-breath-combat-transcript` remains blocked after two honest live
attempts. The dragon special itself is faithful; the remaining red is the
pre-existing shared melee/death/loot transcript. No Go behavior change was
made.

The required attempts were run against current main with `--show-oracle`:

```text
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle \
  /usr/local/go/bin/go run ./cmd/dp-oracle-diff \
  --scenario spec-proc-dragon-breath-combat --show-oracle --seed 1
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle \
  /usr/local/go/bin/go run ./cmd/dp-oracle-diff \
  --scenario spec-proc-dragon-breath-combat --show-oracle --seed 2
```

Both runs matched through the exact special output:

```text
Kaerdein the Ice Dragon breathes frost at you, but you avoid it.
Dragonpeer avoids Kaerdein the Ice Dragon's frost breath.
```

After that boundary C continued with the normal fight loop. Its transcript
included the player's miss, the dragon's claw messages, player death, corpse
loot/junk/equipment, and the dragon's next threat. Go stopped after the
breath output. The combat messages vary by seed, but the same boundary and
missing shared transcript occurred in both runs. This is stable content
evidence, not a timeout or contention failure.

## Call-path audit

The scriptless registered dragon is dispatched from `src/mobact.c:68-93` into
`src/spec_procs.c:926-983`. In the fighting branch, `dragon_breath` recovers a
sitting/resting mob, consumes C's `number(0,3)`, and on success calls
`call_magic(..., CAST_BREATH)` at `src/spec_procs.c:958-963`.

`CAST_BREATH` enters the generic area spell path in `src/spell_parser.c` and
`src/magic.c`, whose zero-damage breath tail enrolls combatants and emits the
verified breath skill message. The following autonomous combat is owned by
`src/fight.c:1898-2032` (`perform_violence`), including ordinary attack rolls,
death teardown, corpse loot, and the later special pulse.

The existing focused dragon proofs cover the special's entry gates, spell
selection, standing recovery, RNG return, `CAST_BREATH` mapping, and exact
breath bytes. The live red begins after those proven arms, at the shared
melee/death/loot path. Fixing it here would violate R5b/R5c by changing an
unrelated shared combat class; keep `mob.dragon-breath-combat-transcript`
blocked until that combat manifest is intentionally addressed.

## Checks

`make fidelity-depth` passed on the base before this triage, reporting 4,111
cases: 4,013 proven/delegated, 45 blocked, and 53 excluded. No `src/` or
`darkpawns-c-oracle/` file was edited.

This handoff advances after the two honest attempts required by the objective
and cites R1/R3/R4/R5b/R5c/R5e.
