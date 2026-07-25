# BRIEF (mimo) — DP-1205: route immortals to the immortal start room at char-creation/login entry

**Owner:** mimo-v2.5 (mimo code). **Gate:** byte-exact unit tests on the
creation path; orchestrator re-runs the `god-harness-smoke` oracle scenario
red→green after merge. CI green.
**Git:** branch `mimo/immort-start-room` **from `main`** — never from whatever
HEAD happens to be checked out (other agents share the live repo; the
orchestrator pre-creates the branch or hands you a `git worktree` — see the
warning in `docs/briefs/README.md`). Edit → commit → push → open a PR. Do NOT
merge. When you're done (including after any revise/force-push round), leave
the repo back on `main` (`git checkout main`). Sized to one PR (S — one
routing decision + tests).
**Finding:** DP-1205 — C's login entry routes by level: immortals enter
`r_immort_start_room` (1204), mortals the mortal start. Go hardcodes EVERY new
character (including the first-player God) into the newbie intro room (8099),
so Go's God co-locates with mortals at spawn; C's does not. Rule **R1** (R5e
verified below).

**Cite:** `src/interpreter.c:2191-2243` (login entry block — read it),
`src/config.c:142,149,152` (`mortal_start_room = 8004`,
`immort_start_room = 1204`, `frozen_start_room = 1202`),
`src/spec_assign.c:606` (`ASSIGNROOM(8099, start_room)`); Go
`pkg/session/char_creation.go:480-545` (`completeCharCreation`),
`pkg/game/death.go:72-119` (`MortalStartRoom`, `NewbieStartRoom`,
`ImmortStartRoom`, `FrozenStartRoom`, `LoginStartRoom`).

---

## C truth (interpreter.c:2191-2243, verified — read the block yourself)

Login/creation entry (the "1) enter the game" path), in order:

1. `load_room = GET_LOADROOM(ch)`; if `!= NOWHERE` → `real_room(load_room)`
   (returning players restore their saved room).
2. If `load_room == NOWHERE` (fresh char, or unsaved): level branch —
   `GET_LEVEL >= LVL_IMMORT` → `r_immort_start_room` (**1204**); else
   `r_mortal_start_room` (**8004**).
3. `PLR_FROZEN` → `r_frozen_start_room` (**1202**) — overrides even a saved
   room.
4. `load_room < 0` (real_room failed) → `r_mortal_start_room`.
5. `char_to_room(ch, load_room)` + `act("$n has entered the game.", TO_ROOM)`.
6. `if (!GET_LEVEL(ch))` → `new_char = TRUE; do_start(ch)`. **The first-player
   God is already LVL_IMPL from init_char, so it is NOT new_char** (DP-1212).
7. `if (new_char)` → `char_from_room` + `char_to_room(real_room(8099))` +
   `look_at_room`. **Only brand-new mortals take the 8099 move.**

So the God enters at **1204 and stays there** — no 8099, no Burning Hut, no
hometown transition. A new mortal enters 8004 (step 5, with the entry
broadcast) then is moved to 8099 (step 7), where the `start_room` spec
(spec_assign.c:606) drives the intro.

## Go today (verified)

`completeCharCreation` (`pkg/session/char_creation.go:480-545`):

- `:489-490` — persists `RoomVNum = NewbieHometownRoom(hometown)` for everyone.
- `:497-503` — the first-player-God bootstrap (`BootstrapFirstPlayerGod`) sets
  the God to LVL_IMPL — correct, keep.
- `:541` — `s.player.SetRoom(game.NewbieStartRoom)` (**8099**) for everyone,
  God included. **This is the bug** (R1): the God should be at 1204.

Returning-player login restores the saved `RoomVNum` from the DB record —
matches C step 1 (do not touch).

## The fix

In `completeCharCreation`, after the bootstrap (level is final at that point):

- **God / immortal** (`s.player.GetLevel() >= game.LVL_IMMORT`): live room AND
  persisted `RoomVNum` = `game.ImmortStartRoom` (**1204**). No 8099, no
  `NewbieHometownRoom` persist. The God must skip the Burning Hut intro path
  entirely (check whatever consumes `NewbieStartRoom`/the start_room
  transition downstream — e.g. the birth-transition comment at `:583` — and
  make sure it does not fire for the God; C never routes the God through
  8099).
- **Mortal:** unchanged — 8099 live room, hometown room persisted.

Keep it minimal: one level branch at the room-selection point; reuse the
existing `game.ImmortStartRoom` constant (do NOT hardcode 1204). Do not touch
`LoginStartRoom` (death.go) — it serves the respawn path.

### Audit items (report in the PR; do NOT change without orchestrator sign-off)

1. **The mortal two-step.** C places a new mortal in 8004 FIRST and broadcasts
   `$n has entered the game.` there (step 5), then moves them to 8099
   (step 7). Go places mortals directly in 8099 — an observer standing in
   8004 would see the entry broadcast in C but not in Go. Report what Go
   currently broadcasts at creation entry, in which room, and whether 8004 is
   player-reachable (can a mortal quit/rent there and witness entries?). Do
   not change the mortal flow in this PR.
2. **Login fallback.** C falls back to the level-based room when a returning
   player's saved room is missing/invalid (`NOWHERE` or `real_room` < 0), and
   forces `r_frozen_start_room` for PLR_FROZEN. Audit Go's returning-login
   path: does an invalid/missing saved `RoomVNum` fall back to
   `LoginStartRoom`-style level routing, and does a frozen player get forced
   to 1202? Report findings; fix only if it's a trivial reuse of the existing
   helper, otherwise leave for a follow-up brief.

## Tests

Extend the char-creation test coverage (`pkg/session/char_creation*_test.go`
or nearest existing fixture — find the first-player-God test from the
DP-1212/#451 work and mirror it):

- **God creation:** first character on a fresh playerbase → after
  `completeCharCreation`, `GetRoom()`/`RoomVNum` == `game.ImmortStartRoom`
  (1204), and the persisted record's `RoomVNum` == 1204. Assert the Burning
  Hut / newbie-intro transition does not apply (no 8099 anywhere on the God
  path).
- **Mortal creation:** unchanged — live room `game.NewbieStartRoom` (8099),
  persisted `RoomVNum` == `NewbieHometownRoom(hometown)`. Regression anchor.

## Oracle gate (orchestrator, after merge — informational)

`god-harness-smoke` (the scenario that surfaced DP-1205): the mortal's `look`
must no longer show the God co-located — RED on pre-fix main → GREEN on the
branch. Anchors: the creation/login scenarios stay green.

## Guardrails

- **Never** edit `src/`, `darkpawns-c-oracle/`, or `lib/misc/messages` — read-only.
- All gates (AGENTS.md §Build & Verify): build, vet, `test ./... -race`,
  `golangci-lint run`, `gofumpt -l .` empty, `make reachability`.
- Do not change the save-file format. Do not touch the guest-login path
  (`session_login.go:97` — guests are mortals, out of scope).

## Deliverable

Level-based entry routing in `completeCharCreation` (God → 1204 live +
persisted, no 8099/intro transition; mortals unchanged), the two audit
reports in the PR body, byte-exact tests, all gates green. Orchestrator greens
`god-harness-smoke`.
