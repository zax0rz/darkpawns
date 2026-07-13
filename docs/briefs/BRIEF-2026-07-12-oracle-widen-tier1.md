# BRIEF 2026-07-12 — Widen Tier-1 oracle scenarios (+ navigate-to-shared-room baseline, + hunger/thirst)

**Executor:** Kimi or GLM (scenario authoring + reading C `src/*.c` to derive expected behavior —
worker-friendly, no game-logic changes). **Prove every claim with pasted transcript output.**
Claude reviews against `origin/main`.
**Branch:** `feat/oracle-tier1-widen` off current `main` (after the scenario-hardening PR lands).
**One PR** (or a small series if it gets large — coordinate with Zach).
**Read first:** the now-landed harness — `cmd/dp-oracle-diff/main.go`,
`internal/oraclediff/{scenario,normalize,conn,diff,report}.go`, and the migrated
`cmd/dp-oracle-diff/scenarios/look-start-room.txt` (per-server `[setup:*]` + shared `[probe]`,
block-aligned diff). Background: `docs/research/drafts/2026-07-12-c-oracle-differential-testing.md`.

---

## Why / context

The hardened harness (setup/probe split, block-aligned diff) already produced 4 verified fidelity
findings on scenario #1: **DP-1083** (telnet leaks room vnum), **DP-1084** (examine dumps a stat
block), **DP-1085** (new-char start room ignores hometown), **DP-1086** (`look <room-feature>`
skips room extra-descs). This brief **widens Tier-1 coverage** — all deterministic, no RNG parity
needed (Tier-2 PRNG port is still separate/out of scope).

**Key constraint discovered in DP-1085:** C and Go fundamentally disagree on where a *new* character
starts (C routes by hometown via the `start_room` spec at room 8099; Go always uses 8004). So you
**cannot** get both servers into the same room just by finishing character creation. Every
same-room text comparison therefore needs an explicit **navigation to a shared room** first.

## Part A — Navigate-to-shared-room baseline (do this FIRST; it unblocks the rest)

Add a reusable probe pattern: after per-server `[setup:*]` finishes creating a char, the shared
`[probe]` phase **walks both characters to one fixed, deterministic room**, then runs the actual
look/examine assertions there. This removes the DP-1085 start-room confound from every other
scenario.

- **Pick a target room reachable from both start points by a short, fixed path.** Room **8004
  (At the Temple Altar)** is a good anchor: it has a lookable room extra-desc (`sign` →
  "It reads: Please don't leave your pets in the Temple area."), multiple exits, and objects
  nearby. Derive the walk from each server's actual start room (C: hometown temple, e.g. KD →
  8162 Temple Infirmary; Go: 8004). Because the start rooms differ, the **movement steps may
  differ per server** — so put the navigation in a *per-server setup extension* (or a
  `[setup:*]`-style block) whose output is NOT diffed, and only diff the look/examine probe once
  both are confirmed in the anchor room. (Alternatively, if a shared path exists from both, keep
  it in `[probe]` but understand movement output itself may diverge — that's a *finding*, see
  Part B movement.)
- **First assertion in the anchor room:** `look` (room name + desc) and `look sign` (exercises the
  DP-1086 room-extra-desc path — expect a divergence until DP-1086 is fixed; that's fine, the
  block-aligned diff localizes it). This becomes the clean text-fidelity baseline.
- ⚠️ Do NOT pre-seed a saved character or edit the oracle clone's `lib/` to force a start room —
  observe real behavior only. (This was explicitly declined in the DP-1085 discussion.)

## Part B — Widen deterministic Tier-1 scenarios

Each is a new scenario file + expected-divergence note. All are RNG-free / fixed-input, so
`DP_SEED=1` is enough. For each, read the cited C source to know the *correct* behavior, then let
the harness show where Go diverges. Bias toward **under-normalizing** (see normalize.go's
`volatileLine` — don't let it eat real lines).

1. **Room descriptions & exits** — in the anchor room and 2-3 neighbors: `look`, autoexits, and
   `look <direction>`. Pure world data. C: `look_at_room` / `look_in_direction`
   (act.informative.c). Catches world-data + exit-rendering drift.
2. **Room features (extra-descs)** — `look`/`examine`/`read <keyword>` for several room E-entries.
   C: `look_at_target` room-exdesc branch (act.informative.c:1033). Regression coverage for
   DP-1086 across more rooms.
3. **Movement & sector cost** — walk a fixed multi-room path; observe movement messages and (via
   `score`/prompt) movement-point drain per sector. C: `do_simple_move` / `move_cost`
   (movement in act.movement.c + sector `movement_loss[]`). Deterministic per sector type.
4. **Object interactions** — `get`/`drop`/`wear`/`remove`/`inventory`/`equipment` on the starting
   gear (`do_start` grants class kit — warrior: small sword 8037, tunic 8019, a pack with bread/
   waterskin). C: act.item.c + act.informative.c `do_inventory`/`do_equipment`. Fixed-input.
5. **Score / condition display** — `score` (and any `cond`/attributes command). C: `do_score`
   (act.informative.c). Compares stat/vitals layout & labels (mask RNG'd roll values, which the
   normalizer already does). Also the natural place to observe hunger/thirst (Part C).
6. **Shops** — `list`/`value`/`buy`/`sell` at a fixed shopkeeper near the anchor. C: shop.c.
   Prices are fixed-input deterministic (profit_buy/profit_sell), good Tier-1 signal.

Author them incrementally; land what's solid. Each scenario's PR note should paste the block diff
and say which hunks are real findings (hand back for triage → `[Fidelity]` issues) vs clean.

## Part C — Hunger / thirst / drunk (the condition system) — Tier-1 slice

Zach flagged this as the most characteristic (and most annoying) DP mechanic — worth oracle
coverage. **Scope it to the DETERMINISTIC, non-tick parts** (the decay itself is tick-driven and
timing-sensitive, and the tick *rate* is already a known divergence — see **DP-1035/F15**, game-hour
ticks run ~2.1× fast. Don't try to time-test decay over a live run here; that belongs in a golden/
unit test).

Deterministic, immediately observable Tier-1 targets (C: `do_eat`/`do_drink`/`do_drink`-fill logic
in act.item.c, `GET_COND`/`gain_condition` in limits.c, `SECS_PER_MUD_HOUR`/COND constants in
utils.h/structs.h):

1. **Starting condition values** — a fresh char's initial hunger/thirst/drunk (via `score`/`cond`
   or the absence of hunger/thirst messages). C sets these at creation — confirm Go matches.
2. **`eat <food>` / `drink <container>` responses** — how much a food item fills you and the exact
   response text ("You eat the …", "You are full.", refusal when full: "You are too full to eat
   more."). Fixed from the object's food/drink values — no RNG, no wait.
3. **Threshold messages** — the "You are hungry."/"You are thirsty." warnings and their wording/
   trigger points. Compare text and the boundary at which they fire.
4. **Drink-container depletion & `pour`/`fill`** if reachable with starting/nearby items.

Use the starting pack's bread (obj 8010) and water skin (obj 8063) from `do_start` so the scenario
needs no shopping. Paste the eat/drink response blocks; any divergence in fill amounts, thresholds,
or wording is a finding.

## ⛔ Guardrails
1. NO changes to game logic/messages/tables/creation in **either** codebase to force matches.
2. Do NOT touch `src/*.c` in the C clone or the `DP_SEED` patch; do NOT commit C source/binary.
3. Do NOT pre-seed player saves or edit the oracle `lib/` to fake a start state (Part A note).
4. Keep the build gate green and the clean skip when `DP_ORACLE_BIN` is unset.

## Success criteria (paste proof for each)
1. Part A navigation lands both servers in the anchor room (paste both `look` room-name lines
   showing the same room) — the DP-1085 confound is gone from downstream probes.
2. At least the room-desc, object-interaction, and hunger/thirst(eat/drink) scenarios exist, run,
   and produce block-aligned reports — pasted. Real divergences are captured for triage, not
   chased in this PR.
3. New scenario files parse (extend `ParseScenario` tests if the format grew); `go build/vet/test`
   green; clean skip when `DP_ORACLE_BIN` unset — shown.
4. ⛔ honored.

## Out of scope
- Fixing any divergence (O1-O4 or new) — those are separate `[Fidelity]` issues.
- Tier-2 RNG parity (porting `random.c` into Go) — combat/skills/spells wait on that.
- Tick-timing tests of condition decay (DP-1035 territory; use golden/unit tests, not the live rig).

## Wrap-up
`go build ./... && go vet ./... && go test ./...` green; commit; push; open PR with pasted
proof-of-life (anchor-room `look` from both servers + at least one hunger/thirst eat/drink block);
list the candidate divergences found; then STOP — Claude reviews against `origin/main` and triages
findings into issues.
