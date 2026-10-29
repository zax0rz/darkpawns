# Egon equipment and item-affect handoff — 2026-10-29

This note records the class audit for PR #2216 and its four new depth cases. It
is an implementation handoff, not a rulebook amendment. The governing C→Go
rules remain in `docs/fidelity/RULEBOOK.md`.

## C call paths verified

- `src/act.informative.c:1470-1495` implements `do_equipment` by iterating
  `i = 0 .. NUM_WEARS-1`, emitting `where[i]` and the visible item.
- `src/act.informative.c:449-460` uses the same wear-position order while
  showing a character's equipment. `src/utils.h:397-400` defines the four
  covered-position relations; C omits covered gear from another mortal's view,
  while the wearer and immortals can see it.
- `src/handler.c:240-270` implements `APPLY_SPELL`: NPCs are skipped; positive
  modifiers set the AFF bit, negative modifiers clear it, and a timed affect
  whose bitvector equals the modifier suppresses that operation.
  `src/handler.c:332-346` replays each equipped object's affects in wear order
  during `affect_total`.
- `src/act.informative.c:534` emits the room-list sanctuary aura;
  `src/act.informative.c:1416` builds the equipment-spell portion of score.
  `src/fight.c:1470-1471` halves damage when the victim has sanctuary.
- `src/structs.h:499-528` defines APPLY locations 0 through 29.
  `src/handler.c:140-270` contains their `aff_apply_modify` behavior.

The four scenario annotations now have manifest rows: `equipment.look-all-c-slots`
in `equipment.tsv`; `equipment.look.covered-items` and
`equipment.look-character.c-slot-order` in `look.tsv`; and
`equipment.apply-spell.worn-bit` in `equipment.tsv`. The unit proof
`TestDoDamage_WornApplySpellProvidesSanctuary` is also registered as
`equipment.apply-spell.sanctuary-combat`.

## Object APPLY inventory and Go comparison

I scanned all `lib/world/obj/*.obj` records, counting each standalone `A` record
by the following numeric location. The observed locations and counts are:

| Location | Count | C behavior | Go comparison |
|---|---:|---|---|
| 1–6 | 67, 62, 36, 42, 37, 67 | STR, DEX, INT, WIS, CON, CHA | Worn stat pass handles these in `pkg/game/effective_attributes.go:27-40`; the engine table now uses C's CHA=6. |
| 7–8 | 0, 1 | CLASS and LEVEL are reserved/no-op (`src/handler.c:156-159`). | No worn-item mutation, matching the C no-op. |
| 9–11 | 6, 10, 3 | AGE adjusts `player.time.birth`; CHAR_WEIGHT and CHAR_HEIGHT change GET_WEIGHT/GET_HEIGHT (`src/handler.c:160-166`). | **Gap for Zach:** Go getters (`pkg/game/player_stats.go:179-203`) expose stored Height, Weight, and Birth-derived age without folding in worn-item affects. Examples: object #31 at `lib/world/obj/31.obj:257-262` (location 9); #19515 at `lib/world/obj/195.obj:154-159` (location 10); #20301 at `lib/world/obj/203.obj:9-14` (location 11). This PR does not add those mutations. |
| 12–14 | 56, 39, 30 | MAX_MANA, MAX_HIT, MAX_MOVE (`src/handler.c:168-173`). | Getters include worn modifiers in `pkg/game/player_stats.go:159-162,210-213,321-324`. |
| 15–16 | 0, 0 | GOLD and EXP are no-op (`src/handler.c:174,181`). | No worn-item mutation, matching C. |
| 17–19 | 105, 176, 201 | AC, HITROLL, DAMROLL (`src/handler.c:182-188`). | Worn modifiers are included by the AC, hitroll, and damroll paths (`pkg/game/player_stats.go:119-130`, `pkg/game/player_combat.go:90,115`). |
| 20–24 | 9, 7, 10, 27, 43 | Five saving throws (`src/handler.c:190-203`). | `GetSavingThrow` includes worn modifiers (`pkg/game/player_stats.go:482-489`). |
| 25 | 18 | Race-hate list add/remove/toggle (`src/handler.c:205-238`). | Rebuilt in C wear order by `pkg/game/race_hate.go:4-45`, matching the C update order. |
| 26–28 | 27, 34, 18 | HIT_REGEN, MANA_REGEN, MOVE_REGEN are empty cases in C (`src/handler.c:175-180`). | **Potential fidelity divergence for Zach:** Go currently applies these equipment modifiers in `pkg/game/limits_gain.go:35,111,188`. This PR leaves the behavior unchanged. Decide whether to remove it for C parity or keep it as an approved intentional extension. Examples: #145 at `lib/world/obj/145.obj:9-14` and `:89-94`; #144 at `lib/world/obj/144.obj:290-296`. |
| 29 | 58 | APPLY_SPELL set/clear behavior with timed-affect precedence (`src/handler.c:240-270`). | Fixed in this PR: `pkg/game/effective_attributes.go:64-109` rebuilds an equipment overlay at affect-total boundaries; `pkg/game/player_stats.go:503-511` and the `IsAffected` path expose it to callers. The overlay is runtime-only, follows worn C order, and is not serialized. |

For this class scan, location 0 (APPLY_NONE) was absent from object A records.
The counts describe current world object records, not the number of distinct
modifier values or a claim that every prototype is in a loaded zone.

## Proof and limits

The red proofs against current main, each with matching draw parity, are retained
under `/home/zach/Archives/darkpawns/oracle-runs/2026-10-29/egon-equipment-baseline-final/`:

- `look-order-red.log` / `.exit`: normalized divergence, exit 1, 4098 matching draws.
- `covered-red-v3.log` / `.exit`: normalized divergence, exit 1, 4104 matching draws.
- `apply-spell-red-v4.log` / `.exit`: normalized divergence, exit 1, 4100 matching draws.

Clean targeted runs with matching draw parity:

- `/home/zach/Archives/darkpawns/oracle-runs/2026-10-29/egon-equipment` — 11 scenarios.
- `/home/zach/Archives/darkpawns/oracle-runs/2026-10-29/egon-equipment-existing` — 8 existing look/equipment/wear/remove/score scenarios.
- `/home/zach/Archives/darkpawns/oracle-runs/2026-10-29/egon-equipment-populatable-slots` — final all-player-populatable-slot fixture.

The slot fixture uses player commands and deliberately does not claim
`WEAR_LIGHT` or `WEAR_THROW`: C's `find_eq_pos` reserves those positions and
`do_grab` always equips into `WEAR_HOLD` (`src/act.item.c:1521-1584,1685-1712`).
The C and Go transcript proves self `equipment`, self `look <character>`, and a
mortal peer's look, including paired and extended reachable slots.

On the final manifest tree, `make fidelity-depth` exited 0 and reported all
four annotated scenarios plus the registered sanctuary damage unit proof. Its
actual stdout, stderr, and exit status are retained at
`/home/zach/Archives/darkpawns/oracle-runs/2026-10-29/egon-equipment/gates/fidelity-depth-final.{stdout,stderr,exit}`.
The docs follow-up also passed `make fmt`, `go build ./...`, `go vet ./...`,
`go test ./...`, fresh-cache `golangci-lint run ./...`, and
`make deadcode-ratchet`; each command's output and status is in the same gates
directory.

The engine currently has no manifestation for the three physical APPLY gaps or
the regen behavior divergence in this PR. These are evidence for Zach's follow-up
decision; no behavior change or rulebook edit was made for them.
