# Dated Handoff: 2026-08-27 (spell-vehicle depth wave)

This wave continued from clean `main` after the prior depth checkpoint of
**498 total, 479 proven/delegated, 9 blocked, 10 excluded**. It ran three
rounds, one command family per PR, and self-merged each PR only after the
required hosted checks were green under the 2026-08-27 amendment. The final
checkpoint is **499 total, 483 proven/delegated, 6 blocked, 10 excluded**;
actionable completion is **483/489 = 98.8%**.

## Round 1 — sleep spell vehicle (PR #676)

Branch `glm/depth-sleep`; merged as `b1382a356`.

- `sleep-spell-depth` was run against C first and then across seeds 1, 2, 3,
  5, and 8. The vehicle proves the vnum-1226 reagent narration and
  consumption, outlaw gate, level-window gate, live save parity, failed-save
  `AFF_SLEEP` state, and targeted wake behavior.
- C tracing covered `src/magic.c:1199-1249`,
  `src/spell_parser.c:879-905`, and `src/act.movement.c:808-880`. Go changes
  are limited to reagent/state/message handling and the `cmdSet` outlaw field;
  no C or oracle files were edited (R1/R3/R4/R5e).
- The reachable cast surface is now represented by
  `objmagic.sleep-entry-gates.cast` in `object-magic.tsv`. The original
  `objmagic.sleep-entry-gates` row remains blocked because sleep is
  `TAR_NOT_SELF` and cannot be entered through the object-magic path.

## Round 2 — invisibility assist vehicle (PR #677)

Branch `glm/depth-invisibility`; merged as `fc6833f46`.

- `invisibility-assist-depth` was RED on pre-fix `main`: Go let a mortal
  helper assist a fight against an invisible opponent, while C emitted only
  `You can't see who is fighting him!`. After the Go-only visibility guard,
  seeds 1, 2, 3, 5, and 8 all produced no normalized divergence.
- C tracing covered `src/act.offensive.c:54-96` and the invisibility affect in
  `src/magic.c:1072-1084`. The fix uses the canonical Go `CanSee` boundary and
  preserves the C actor-only rejection with no audience leakage (R1/R5e).
- A manifest scan found no additional reachable visibility-gated depth row;
  the remaining visibility-related cases are already delegated or unit-pinned.

## Round 3 — wand/staff object-magic vehicle (PR #678)

Branch `glm/depth-wand-staff`; merged as `47517febf`.

- `wand-staff-use-depth` and its paired `staff-use-depth` were both RED on
  pre-fix `main`, exposing invented Go point/tap text, incorrect targeting,
  caster self-casting on staff use, and missing C cast-type behavior.
- C tracing covered `src/spell_parser.c:558-817`, including equipped lookup,
  castable wand/staff branches, charge decrement, `DEFAULT_WAND_LVL` /
  `DEFAULT_STAFF_LVL`, `CAST_WAND`, `CAST_STAFF`, and staff room fan-out.
- Go now matches the C equipped-only lookup and exact actor/audience message
  order, mutates charges on the object instance, defaults an unspecified item
  level to 12, dispatches the correct cast type, excludes the staff caster
  from ordinary room fan-out, and preserves object-target resolution. The
  focused `TestDoUseWandConsumesInstanceChargeAndUsesDefaultLevel` test pins
  instance charge mutation and the level-sensitive default.
- Both vehicles include `~dpclock pulse 20` and use existing vnums 8053/8054;
  seeds 1, 2, 3, 5, and 8 all produced no normalized divergence. The wand
  seed-1 run was also checked with `--show-oracle` and showed the intended C
  blocks (R1/R3/R4/R5e).

## Remaining blocked frontier and owners

These rows were intentionally not attempted in this wave and remain blocked;
the owner is the next depth vehicle, not a claim of completion:

- `combat-entry.tsv:assist.mob-helpee-pers` — combat-entry owner; next vehicle
  must put a mob in the helpee role and prove C `PERS` rendering versus Go's
  short description.
- `combat-entry.tsv:hit.charm-master` — combat-entry owner; next vehicle must
  create the charmed attacker/master relation and prove the friendship gate.
- `combat-entry.tsv:kill.immortal-postdeath-menu` — combat-entry owner; next
  vehicle must cover deferred extraction and session menu return.
- `info.tsv:score.state-variants` — info owner; next vehicle is the affect,
  position, mount/pet, tattoo, and naked/armed state matrix.
- `comm.tsv:tell.linkless` — comm owner; the optional peer-drop/linkless
  descriptor vehicle was not attempted in this capped wave.
- `object-magic.tsv:objmagic.sleep-entry-gates` — object-magic owner; the
  quaff/object entry remains unreachable because sleep is `TAR_NOT_SELF`. The
  reachable cast-surface proof is separate as
  `objmagic.sleep-entry-gates.cast`.

No deep-engine backlog item was changed, and no `src/` or
`darkpawns-c-oracle/` file was edited.

## Verification

- `make fidelity-depth` — **499 total, 483 proven/delegated, 6 blocked, 10
  excluded**; exit 0.
- `go build ./...` — pass.
- `go vet ./...` — pass.
- `go test ./...` — pass.
- `golangci-lint run ./...` — **0 issues**.
- `gofumpt -l .` — no output.
- PRs #676, #677, and #678 hosted test, lint, and security checks — pass.
- Final repository state: clean `main` at `47517febf`.
