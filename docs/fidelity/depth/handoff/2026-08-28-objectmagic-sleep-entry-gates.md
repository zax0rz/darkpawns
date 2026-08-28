# Depth-fidelity handoff: `objmagic.sleep-entry-gates`

Date: 2026-08-28
Branch: `glm/depth-sleep-entry-gates`
Starting main: `e9a27583d` (`fix: deepen dragon breath special procedure (#705)`)

## Frontier

The post-dragon refresh reported 596 total cases, 577 proven/delegated, 6
blocked, and 13 excluded: 577/583 actionable, or 99.0%.

This was the one permitted attempt against the remaining blocked sleep row.
The reachable cast proof was rechecked and its existing manifest row remains
green; the unreachable object-magic row remains blocked. The frontier is
therefore unchanged at 596 total, 577 proven/delegated, 6 blocked, and 13
excluded: 577/583 actionable, or 99.0%.

## C path and one vehicle

The C object-magic entry is `src/spell_parser.c:544-558` and its potion arm
calls `call_magic(ch, ch, NULL, ...)` at `src/spell_parser.c:685-714`. Sleep is
registered as `TAR_CHAR_ROOM | TAR_NOT_SELF` at `src/spell_parser.c:1395-1396`,
and the target parser rejects the caster as its own target at
`src/spell_parser.c:886-892`. Consequently the object/quaff path cannot reach
the sleep effect body at `src/magic.c:1199-1249` without inventing a target
route or changing the game.

The single cast-sleep vehicle is
`cmd/dp-oracle-diff/scenarios/sleep-spell-depth.txt`. It uses C's vnum-1226
reagent, gives the level-8 caster the sleep skill, marks the caster outlaw,
casts at a same-level peer, and then exercises the sleep-save and targeted
magical-sleep wake gate. The seed-1 `--show-oracle` run reached the intended
blocks and matched Go, including:

```text
Pulling a bit of sand from a pocket, you cast it about the room...
Sleeptarget goes to sleep.
You feel very sleepy...  Zzzz......
You can't wake him up!
```

The same vehicle returned `result: no normalized divergence` on seeds 2, 3,
5, and 8. This reconfirms the existing
`objmagic.sleep-entry-gates.cast` row as `oracle-green-multiseed`; no new row
was needed. The object-magic `objmagic.sleep-entry-gates` row is deliberately
still `blocked`, not `excluded`: the attempted cast vehicle proves the
reachable sibling surface, while the C object entry itself remains
unreachable (R2/R4/R5e).

No `src/` or `darkpawns-c-oracle/` file was edited. The unrelated untracked
brief `docs/briefs/BRIEF-2026-08-28-economy-specproc-cluster.md` was preserved.

## Verification and next queue

`make fidelity-depth` passes at the unchanged frontier. The full repository
gates are run for this docs-only branch before its PR is opened.

After this handoff, the special-procedure and blocked-row queues are
exhausted. The next queue is the remaining un-manifested command families:
sweep the command table in `src/interpreter.c` against
`docs/fidelity/depth/*.tsv`, then take the first unmanifested family in table
order using one `glm/depth-<family>` PR per family.

This handoff follows R1 (the exact reachable bytes), R2 (the configured command
and object surface), R4 (no invented self-target route), R5b/R5c (do not
duplicate the already-owned cast proof), and R5e (the actual C parser and
call-path audit).
