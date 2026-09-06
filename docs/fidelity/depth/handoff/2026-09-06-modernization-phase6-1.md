# Modernization Phase 6.1 — mechanical lookup tables

Date: 2026-09-06  
Scope: mechanical subset of roadmap item 6.1

## Change

Converted three proven, player-facing lookup switches to data tables without
changing their values or call paths:

- `pkg/game/limits_exp.go`: `FindExp` class modifiers and fixed levels 0–12
  now use tables; invalid classes retain C's default modifier of `1.0`, and
  non-positive levels retain the existing level-0 result.
- `pkg/game/equipment.go`: `EquipmentSlot.String` and
  `ParseEquipmentSlot` now use canonical-name and accepted-input tables. The
  existing case-insensitive parsing and aliases are unchanged.
- `pkg/game/look.go`: observation equipment labels now use a Go-slot to C
  `where[]` position table. Shared C positions retain their exact fixed-width
  labels and the unknown-slot fallback remains `<used>               `.

This slice deliberately does not touch the `wiz_set` side-effect switch or
spell dispatch. Those are separate modernization surfaces requiring their own
call-path and state proofs; no new behavior is inferred here (R4).

## Fidelity basis

- `FindExp` is transcribed from `src/class.c:1089-1166`; fixed values and all
  class modifiers remain covered by the existing golden tests.
- Equipment output uses C's `src/constants.c:727-749` `where[]` table and the
  `do_equipment` loop at `src/act.informative.c:1470-1495`.
- Observation equipment labels retain the same C positions used by the
  existing look path (`src/act.informative.c` equipment observation path).

The change is data-structure-only: player-facing bytes, command aliases,
fallbacks, and arithmetic are intended to remain identical (R1/R2/R3).
Call paths were checked before and after the edit (R5e).

## Verification

Focused unit tests:

```text
/usr/local/go/bin/go test ./pkg/game ./pkg/session       PASS
```

Focused oracle matrix:

```text
levels-depth             PASS
levels-arguments-depth   PASS
levels-npc-depth         PASS
equipment-glance-depth   PASS
```

Oracle summary: 4 scenarios, 4 passed, 0 failed, 0 infrastructure failures,
0 unpinnable, 0 timed out.

Full repository gates remain required before merge:

```text
/usr/local/go/bin/go build ./...
/usr/local/go/bin/go vet ./...
/usr/local/go/bin/go test ./...
PATH=/usr/local/go/bin:$PATH golangci-lint run ./...
make fidelity-depth
python3 scripts/gen_expected_divergences.py --check-pins
git diff --check
```
