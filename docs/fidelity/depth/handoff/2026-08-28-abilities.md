# Depth-fidelity handoff: `abilities`

Date: 2026-08-28
Branch: `glm/depth-abilities`
Starting main: `e9a27583d` (`fix: deepen dragon breath special procedure (#705)`)

## Frontier and queue position

The required baseline checkout/pull reached `main` at `e9a27583d`. The initial
`make fidelity-depth` invocation saw only the three unmanifested cases in the
new `abilities-depth` vehicle; after this slice's manifest was added, the
frontier is **602 total cases, 582 proven/delegated, 6 blocked, and 14
excluded**: **582/588 actionable (99.0%)**.

The special-procedure inventory and the one blocked
`objmagic.sleep-entry-gates` attempt are already claimed by earlier dated
handoffs. The first remaining un-manifested command family in
`src/interpreter.c` table order is `at`, after `abilities` at line 322.

## C call path and branch inventory

The registered command is `{ "abilities", POS_SLEEPING, do_abils, 0, 0 }` at
`src/interpreter.c:322`; normal dispatch runs through
`src/interpreter.c:1407-1456` into `src/act.informative.c:1077-1095`.
The handler's reachable branches are:

- no descriptor: immediate silent return, unreachable from the registered
  descriptor-backed player surface;
- otherwise, six fixed `stc` lines, with no argument read, using
  `abil_names[GET_STR/DEX/INT/WIS/CON/CHA]`;
- the POS_SLEEPING command gate still permits the handler;
- the literal C command is `abilities`; `abils` is not registered and reaches
  C's `Huh?!?` fallback;
- the `GET_*` values are effective `aff_abils` values. C's spell/equipment
  application path (`src/magic.c:1251-1264`, `src/handler.c:314-373`) clamps
  player STR/DEX/INT/WIS/CON to the [0,18] display range after affects, while
  CHA follows C's uncapped path.

## Proof and confirmed fixes

`abilities-depth` is a pure-coverage vehicle for exact output, ignored
arguments, the non-C alias, and the sleeping entry. Its seed-1
`--show-oracle` run is GREEN and shows both the six-line display and the C
`Huh?!?` response for `abils`.

`abilities-active-stats` uses the established empty-players God-to-mortal-peer
vehicle: the God casts `SPELL_STRENGTH` on the peer, then the peer runs
`abilities`. The pre-fix run was RED on clean main (seed 1: C `(very good)` vs
Go `(good)`) because Go read raw `Stats`. After switching to effective getters,
seed 8 exposed the C [0,18] cap (C `(excellent)` vs Go `(superior)`). The
branch now matches the C transcript on seeds **1, 2, 3, 5, and 8**; seed 8
`--show-oracle` confirms C `(excellent)` and a no-divergence result.

The Go-only fixes are:

- `cmdAbils` reads `Player.GetStr/GetDex/GetInt/GetWis/GetCon/GetCha`;
- `cDisplayAbility` mirrors C's `affect_total` cap for the five capped player
  stats without inventing a CHA cap;
- the invented Go `abils` alias is removed from command registration.

Focused tests `TestCmdAbils` and
`TestCmdAbilsClampsCEffectiveStatCeilings` pin effective modifiers, exact
bytes, the five C ceilings, and uncapped CHA behavior. No `src/` or oracle tree
was edited.

## Gates

All required gates passed on this branch:

- `make fidelity-depth` — exit 0;
- `go build ./...` — exit 0;
- `go vet ./...` — exit 0;
- `go test ./...` — exit 0;
- `golangci-lint run ./...` — `0 issues`;
- `gofumpt -l .` — no output.

The unrelated untracked brief
`docs/briefs/BRIEF-2026-08-28-economy-specproc-cluster.md` remains preserved.

This handoff follows R1 (exact bytes), R2 (registered command surface), R4 (no
invented alias or behavior), R5b/R5c (shared effective-stat ownership), and
R5e (actual C call-path verification).
