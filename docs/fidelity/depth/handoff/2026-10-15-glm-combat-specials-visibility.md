# 2026-10-15: combat-round specials audit, the get_rider finding, and DP-1295 (GLM train 2)

Worktree `~/wt/glm-combat-specials`. Brief:
`~/GLM-BRIEF-2026-10-15-combat-specials-and-visibility.md`. Three PRs, each
stacked directly on `origin/main` (`d62179115`) per the brief:

1. `glm/combat-round-specials` — the specials class audit (Part 1).
2. `glm/mount-follow-drag` — the handed-forward `get_rider` finding, corrected.
3. `glm/visibility-light-ok` — DP-1295: CAN_SEE's LIGHT_OK.

## 1. Part 1 — specials that act during combat (R5c)

**The call sites.** `perform_violence()` calls an assigned mobile special
every round *after* the mob's ordinary attack (`src/fight.c:2030-2031`);
`mobile_activity()` skips fighting mobs (`src/mobact.c:68`). Go ports the
seam (`pkg/combat/engine.go` `MobSpecialFunc`, wired by
`Manager.SetMobSpecialFunc`, whose flag gates are session-proven) — the audit
checked the *spec bodies* it reaches.

**The class inventory** (every `ASSIGNMOB` in `src/spec_assign.c` whose body
acts while `FIGHTING(ch)`, verified by grep + reading each body):

- Direct actors: snake, magic_user, cleric, fighter, paladin, dragon_breath,
  troll, werewolf, mindflayer, teleporter, teleport_victim, citizen.
- Delegators: dracula and medusa → magic_user; cityguard, take_to_jail,
  breed_killer, guild_guard → fighter.
- Correctly *skipping* while fighting (mobile_activity-only): fido,
  backstabber, brain_eater, clerk, con_seller, ira, little_boy, prostitute,
  mayor, normal_checker, tipster, pissedalchemist, rescuer (its fighting arm
  only declines when its own target fights back).
- **Declared but never assigned in C — dead code**: fearface, conductor,
  ninelives, mickey, mallory (their Go stubs match; Go assigns none of them).

**Three gaps found and fixed** (`glm/combat-round-specials`):

1. `specSnake`: the `number(0,32-level)` draw now short-circuits behind the
   `FIGHTING` and same-room gates (`src/spec_procs.c:338-339`), and the
   victim is `FIGHTING(ch)` — not the mobile's separate hunt target.
2. `specMagicUser`: the victim walk covers every body in the room in C
   people order with the per-candidate draw (`src/spec_procs.c:417-420`);
   the fallback is `FIGHTING(ch)` of any type — a caster fighting another
   mobile now casts instead of silently declining.
3. `specBreedKiller`: the `FIGHTING` arm delegates to `fighter()`
   (`src/spec_procs2.c:1688-1690`) instead of returning false.

**Proofs**: engine-round tests on the seam (`combat_round_specials_test.go`)
— the snake's bite after the ordinary attack with poison landing and the
draw-gate's zero-draw refusal; twin-fixture return/draw-tail equivalence for
the three delegators; the drawless teleporter for seam order. R5h revert
controls observed failing for each fix. Rows: `mob.snake-unreachable`,
`mob.cityguard-fighting-branch`, `mob.take-to-jail-fighting-branch` flipped
excluded → unit-green (the exclusions had read only mobile_activity's
fighting skip); `mob.breed-killer-fighting-delegation` added.

**Rows audit (brief item 4):** the only wrongly-excluded rows citing
mobile_activity for a combat-time special were those three; every other
excluded/unassigned row (fearface, mayor, elemental_room, eviltrade,
evillead, little_boy, ira, rescuer-15808, pissedalchemist-15814) was
re-verified against `spec_assign.c` and is correct. Luna's
`docs/fidelity/depth/surface/*.tsv` maps do not exist on this main; if they
land on a branch that reads these rows, the status flips carry.

## 2. The handed-forward get_rider finding — corrected

The blocked-cases handoff claimed the player-path follower drag was missing
C's `get_rider(k->follower) != ch` check (`src/act.movement.c:347`) as a
one-line fix. That reading was wrong: `get_rider(mount)` returns a mounted
NPC's *rider* (`src/utils.c:387-393`), and C's check exists because
`ch->followers` **includes the mount mobile** — the mover's own mount —
which must not be dragged a second time beside `do_simple_move`'s mount arm.
Go's player-path follower enumeration is players-only, so the mount never
enters the drag and the literal comparison is structurally vacuous there.
`glm/mount-follow-drag` lands `TestRiderWalkMovesMountOnceNotTwice`: a
mounted rider's walk carries the mount with exactly one movement deduction.
The R5h control (a C-shaped mount drag double-spending movement) fails the
test; restored, it passes.

## 3. DP-1295 — CAN_SEE's LIGHT_OK (R1/R5b/R5c)

`glm/visibility-light-ok` ports C's light law (`src/utils.h:515-541`) into
the one shared predicate behind act()'s `hide_invisible` filter, PERS, look
and room targeting:

- `LIGHT_OK = !AFF_BLIND && (room lit || AFF_INFRAVISION)` — room light
  resolves through the observer's owning world (`Player.world`, set at
  `AddPlayer`; the mobile already carries it).
- `CAN_SEE_OBJ` gains the same room-light arm.
- The World-aware wrappers (`canSeeCharacterInRoom`, `canSeeObjectInWorld`)
  and the per-spec local copies (prostitute, con-seller) collapse into the
  shared predicate — one CAN_SEE, per R5c.

Tests corrected to C's truth: the dark prostitute's "If I could see you"
acts reach no one (act's hide filter, `comm.c:2537-2538`), and dark-room
targeting fails (get_char_room_vis flows through CAN_SEE — DP-1295's own
"kill witch → They aren't here." example).

**Proofs**: `TestCanSeeLightOKMatrix` (lit/dark × mortal/immortal ×
holylight × infravision, over act delivery, PERS and targeting),
`TestCanSeeObjectDarkRoom`, and three live scenarios, all green against the
reference with no divergence:
`visibility-dark-social` (hidden dark act; "You hear someone or something
moving around nearby."; lit control), `visibility-dark-holylight` (the
toggle flips target resolution), `visibility-dark-infravision` (skillset +
self-cast flips it back). R5h: removing the light arm fails the matrix.
`visibility.tsv` records the six rows.

**Session/login check (brief stop condition):** the change touches no
session, login or transport code; the only session-file edits are test
expectation corrections (`extchar_deferred_test.go`).

## 4. Evidence and gates

- Gates on the combined train (`glm/combat-specials`, which contains exactly
  these three commits plus a gofumpt style commit), each run on its own with
  its exit status checked: `make fmt`, `go build ./...`, `go vet ./...`,
  `go test ./...` (39 ok), `go test ./pkg/game/... -count=1`,
  `golangci-lint run ./...` (0), `make deadcode-ratchet` (0 new),
  `make fidelity-depth` (5593 cases, 11 blocked — unchanged from this base),
  `make fidelity-units` (1682 PASS), `make string-census`, `gosec
  -severity=high -quiet ./...`, `go test -race ./pkg/game/...`.
- Each PR branch was rebuilt standalone from `origin/main` (cherry-pick +
  fmt amend) and its own tests pass there, so each PR's CI runs green
  independent of the others (the brief's note on #1993).
- Targeted census `glm-combat-specials-visibility` (24 scenarios — the
  affected spec vehicles, the mount scenarios, the three visibility
  scenarios, and act-heavy controls, with `--draw-parity`): verdict CLEAN,
  `draw_divergent=1`. The one draw divergence is
  `spec-proc-dragon-breath-combat` — pre-existing on main (DRAW in the
  `batch-f-deploy-candidate` census at `e2d905876`, the finish-line
  baseline's known "Number(0,99) | log ends | ~dpclock pulse 40" cluster),
  not this train's.

## 5. Residuals handed forward

- The engine's `MobSpecialFunc` runs after the attack loop but the round
  order work (God-against-mob) is DeepSeek's this cycle; no overlap was
  found — this train kept to the special invocation and spec bodies.
- The 13 skip-while-fighting specials keep their mobile_activity-only rows;
  no action needed.
- `Player.world` is set at `AddPlayer`; any future code path creating
  players outside the world registry would silently fall back to the
  pre-DP-1295 light-blind CAN_SEE (documented in `visibilityWorld`).
