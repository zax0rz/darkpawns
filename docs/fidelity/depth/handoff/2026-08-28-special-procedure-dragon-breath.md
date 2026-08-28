# Depth-fidelity handoff: `dragon_breath`

Date: 2026-08-28
Branch: `glm/spec-dragon-breath`
Starting main: `af726eaf3` (`docs: record unassigned mayor special (#704)`)

## Frontier

Before this slice, `make fidelity-depth` reported 587 total cases, 569
proven/delegated, 5 blocked, and 13 excluded: 569/574 actionable, or 99.1%.

The manifest now records eight proven/delegated dragon-breath cases and one
blocked downstream transcript. The current frontier is 596 total cases, 577
proven/delegated, 6 blocked, and 13 excluded: 577/583 actionable, or 99.0%.

## Queue position and C inventory

`dragon_breath` is the next source-order special after the unassigned `mayor`,
at `src/spec_procs.c:926-983`. The active registration census in
`src/spec_assign.c` is:

| registration order | mob VNUM | prototype |
| --- | ---: | --- |
| 1 | 4209 | Kaerdein, Ice Dragon |
| 2 | 4705 | dragon prototype |
| 3 | 11000 | dragon prototype |
| 4 | 11001 | dragon prototype |
| 5 | 11002 | dragon prototype |
| 6 | 20027 | Strabo |

The C procedure selects frost for 4209/4705, acid for 11000, lightning for
11001/20027, and fire for 11002 plus the default branch
(`src/spec_procs.c:937-954`). The first live vehicle uses registered mob 4209
with its world script stripped; no C or oracle files were edited.

## C call path and branch matrix

The commandless autonomous path is `mobact.c:68-93` →
`spec_procs.c:926-983`. The fighting path is reached after the ordinary NPC
attack loop in `fight.c:1898-2032`. The C procedure's relevant branches are:

- command, asleep, and negative-HP gates return false;
- a fighting dragon stands when between sleeping and fighting, otherwise it
  consumes `number(0,3)`; a failed roll returns true without bytes, while a
  successful roll calls `call_magic(..., CAST_BREATH)` and then the shared
  `magic_user` path;
- a non-fighting dragon scans room occupants in list order, selecting the
  first visible non-NPC without `PRF_NOHASSLE`, emits the exact three threat
  lines, and calls `call_magic(..., CAST_BREATH)`;
- `CAST_BREATH` maps through `spell_parser.c:452-467` to the generic area
  path. `magic.c:1559-1630` skips NPCs and level-31-or-higher players, then
  deliberately calls `mag_damage(..., 1)`. The breath spells have no
  `mag_damage` formula, but `magic.c:615-827` still reaches saving throw and
  `damage(0)`, which enrolls combatants and emits the loaded breath skill
  message (`fight.c:1314-1480`).

The Go changes implement only those confirmed C branches: the exact VNUM
spell switch and `CAST_BREATH`, the noncombat threat scan, canonical standing
output, C's literal area saving type, the level-31 immortal gate, and the
zero-damage breath tail. The shared combat transcript remains outside this
slice.

## Proof and manifest result

Added these rows to `docs/fidelity/depth/spec-procs.tsv`:

- `mob.dragon-breath-entry-gates` — D1, focused unit proof;
- `mob.dragon-breath-spell-selection` — D2, focused unit proof;
- `mob.dragon-breath-noncombat-threat` — D3, live oracle proof;
- `mob.dragon-breath-pulse-dispatch` — D5, live autonomous-pulse proof;
- `mob.dragon-breath-standing-recovery` — D4, focused state/output proof;
- `mob.dragon-breath-combat-roll` — D5, focused RNG/return proof;
- `mob.dragon-breath-call-magic` — D4, focused shared spell-path proof;
- `mob.dragon-breath-magic-user-delegation` — D5, delegated to
  `mob.magic-user-combat-cast`;
- `mob.dragon-breath-combat-transcript` — D5, blocked.

The scenario `cmd/dp-oracle-diff/scenarios/spec-proc-dragon-breath.txt`
matches the exact C threat transcript on seeds 1, 2, 3, 5, and 8. The seed-1
`--show-oracle` run showed:

```text
Kaerdein the Ice Dragon looks at you.
Kaerdein the Ice Dragon growls, 'So, you have found my lair...'
Kaerdein the Ice Dragon exclaims, 'For that you must die!'
```

The separate diagnostic scenario
`cmd/dp-oracle-diff/scenarios/spec-proc-dragon-breath-combat.txt` uses a
level-1 outlaw peer to reach the direct breath vehicle. On seed 1, C and Go
both emit the exact frost-breath bytes:

```text
Kaerdein the Ice Dragon breathes frost at you, but you avoid it.
Dragonpeer avoids Kaerdein the Ice Dragon's frost breath.
```

The remaining divergence is the pre-existing shared melee/death/loot and
subsequent pulse transcript: C continues attacking and looting while Go does
not. Two honest live attempts were made (clean main and the scoped
outlaw/reagent-style diagnostic after the breath fix); both reached the
breath boundary but did not prove the shared combat transcript. Keep
`mob.dragon-breath-combat-transcript` blocked under R4/R5b/R5c/R5e rather than
fixing unrelated combat behavior in this slice.

Focused tests pass:

```text
go test ./pkg/game ./pkg/spells -run 'TestSpecDragonBreath|TestDragonBreathSpell|TestMagAreas_BreathUsesCImmortalGateAndZeroDamageTail|TestMagAreas_SkipsCharmedNPC|TestMagDamage_DoesNotFabricateRemovedSpells|TestSpellInfo_GoldenAgainstCSource' -count=1
ok   github.com/zax0rz/darkpawns/pkg/game
ok   github.com/zax0rz/darkpawns/pkg/spells
```

`make fidelity-depth` passes at the frontier above. The full repository gates
also pass on this branch: `go build ./...`, `go vet ./...`, `go test ./...`,
`golangci-lint run ./...` (0 issues), and `gofumpt -l .` (no output). The PR
check is still pending.

## Next queue

After this special-procedure slice is merged or left open according to the CI
rule, the next queue item is the one blocked row
`objmagic.sleep-entry-gates`. Attempt it once with the cast-sleep vehicle,
including outlaw and reagent arms; convert only the reachable proof and leave
the remainder blocked with a sharp note. Then continue the remaining
un-manifested command families in `src/interpreter.c` table order.

This handoff follows R1 (exact player-facing bytes), R2 (registered command and
autonomous surface), R3 (C RNG and draw-sensitive spell path), R4 (no invented
behavior), R5b/R5c (shared behavior ownership), and R5e (actual C call-path
verification). The unrelated untracked brief
`docs/briefs/BRIEF-2026-08-28-economy-specproc-cluster.md` was preserved.
