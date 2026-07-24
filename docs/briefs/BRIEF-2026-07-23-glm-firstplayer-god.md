# BRIEF (glm) — port the first-player-is-God bootstrap (fidelity fix + skill-gate enabler)

**Owner:** glm-5.2. **Gate:** unit tests now (byte/stat-exact vs C `init_char`);
Claude wires the oracle scenario once the harness `empty-players` support lands
(Claude is building that in parallel). CI green.
**Git:** branch off `main` as `glm/firstplayer-god`, commit, push, open a PR. Do
NOT merge. Sized to one PR (S).
**Why:** C makes the very first character created on a fresh MUD an Implementor
(the classic DikuMUD convention). Go does **not** replicate this — a real
player-observable fidelity gap (R1). It's also the enabler for oracle-gating the
combat skill layer: a bootstrapped God can run `skillset` to grant a fixture
mortal a skill (see the `skillset` brief), all through faithful game code.
**Cite:** `src/db.c:3030-3090` (`init_char`, the `top_of_p_table == 0` block);
`pkg/game/limits.go:23` (`LVL_IMPL = 40`); rules **R1**, **R3**
(`docs/fidelity/RULEBOOK.md`).

## The C truth — init_char first-player block

Inside `init_char` (runs for every new character), C special-cases the first one:

```c
if (top_of_p_table == 0) {          /* first player ever → God */
    GET_EXP(ch)        = 7000000;
    GET_LEVEL(ch)      = LVL_IMPL;   /* 40 */
    ch->points.max_hit  = 500;
    ch->points.max_mana = 100;
    ch->points.max_move = 82;
}
...
/* skills loop, ALL characters: */
for (i = 1; i <= MAX_SKILLS; i++)
    SET_SKILL(ch, i, (GET_LEVEL(ch) < LVL_IMPL) ? 0 : 100);  /* God → every skill 100 */
...
/* conditions, ALL characters: */
for (i = 0; i < 3; i++)
    GET_COND(ch, i) = (GET_LEVEL(ch) == LVL_IMPL ? -1 : 24); /* God → -1 (no hunger/thirst/drunk) */
```

So the first character becomes: **level `LVL_IMPL` (40), exp 7,000,000, max_hit
500 / max_mana 100 / max_move 82, every skill at 100, all three conditions -1.**
Non-first characters: the same loops set skills to 0 and conditions to 24 — which
is presumably what Go already does today; **do not change the non-first path**
except to gate it behind the new condition.

The first-player block has **no `number()` draws** — it's pure assignment — so it
cannot perturb the shared RNG stream (R3). (The sex-based weight/height `number()`
draws in `init_char` already happen for every char and are unchanged.)

## The trigger — "fresh MUD" (two paths: production DB-empty, harness env)

C's `top_of_p_table == 0` means "no players exist yet." The Go bootstrap fires
when the MUD is **fresh** (no players). Implement a single predicate,
`World.IsFreshMUD()` (name yours), used at char-creation finalization (around
`pkg/session/char_creation.go:497` where `CreatePlayer` is called, and/or
`pkg/game/character.go DoStart`) — apply the God block iff it returns true for
the character being created:

1. **Production path (faithful):** the player store is empty — add a
   `CountPlayers()` on the DB layer (`pkg/db/player.go`, `players` table) and
   return true when it reports 0 rows (only the very first char).
2. **Harness path (required):** the oracle harness runs Go with a **deliberately
   dead DB** (`-db` = an unreachable Postgres DSN — see
   `cmd/dp-oracle-diff/main.go:32`), so `CountPlayers()` can't be used there. When
   the env var **`DP_FRESH_MUD`** is set (non-empty), treat the MUD as fresh and
   bootstrap the **first character created this process** (an in-process "have I
   crowned anyone yet this boot" latch — crown exactly one, then behave normally).

> ⚠️ **Interface contract with the harness (Claude's parallel work):** the harness
> sets `DP_FRESH_MUD=1` **only** for God-fixture scenarios, and passes the C
> oracle an empty players file for the same scenarios — so both servers crown
> their first character. For every existing/normal scenario, `DP_FRESH_MUD` is
> UNSET and the C players file is non-empty, so the first char is an ordinary
> mortal exactly as today. **Do NOT bootstrap the first char in the harness
> without `DP_FRESH_MUD`** — that would wrongly crown the primary actor in
> combat-death/backstab-opener and desync against C. The env gates it.
>
> `DP_FRESH_MUD` is a Go-side test control over *initial store state* (the
> DP_SEED/DP_CLOCK category — an external input), not a gameplay-output injection;
> it's fine to add (Go is ours; the read-only rule is C-oracle-only).

## Ordering with do_start

C calls `init_char` (this block) and then `do_start` for a new character.
`do_start` grants starting equipment, sets the class's starting skills, and the
start room — for the God too. Keep Go's existing `DoStart` running after the
bootstrap, unchanged; where `do_start` sets specific class skills they overwrite
the God's 100 for those skills, exactly as in C. Do not special-case `DoStart`
for the God. (For our use the God only needs `LVL_IMPL` to run `skillset`; but
port the whole block faithfully — a "first login becomes God" scenario is itself
gate-worthy.)

## Tests

- **fresh MUD, first char → God:** with `DP_FRESH_MUD` set (harness path) OR
  `CountPlayers()==0` (production path, with a test DB), a newly created character
  has level `LVL_IMPL`, exp 7000000, max_hit 500, max_mana 100, max_move 82,
  `GetSkill` == 100 for a representative set of skills, and conditions -1.
- **fresh MUD, second char → mortal:** with `DP_FRESH_MUD` set, the *second*
  character created this process is an ordinary level-1 of its class (conditions
  24, skills 0 except its `DoStart` class grants, e.g. thief backstab 10) — the
  first-crown latch is consumed exactly once.
- **not fresh → mortal:** with `DP_FRESH_MUD` unset and no DB (harness default),
  the first char is an ordinary mortal — the regression guard that existing
  scenarios (combat-death, backstab-opener) still get mortals, unchanged.
- **no RNG perturbation:** the God block consumes zero draws (assert the shared
  stream position is unchanged across the block, mirroring the cast_cmds draw
  tests) — the only creation draws remain the sex weight/height `number()` calls.

## Oracle gate (Claude, in parallel — informational)

Once the harness `empty-players` fixture + probe-on-peer support lands, Claude
authors: (a) a first-login-God creation scenario, and (b) the R5c skill openers
that use a bootstrapped God to `skillset` a mortal fixture. Until then, the unit
tests above are the gate. Design so nothing depends on test-only state.

## Guardrails

- **Never** edit `src/` or `darkpawns-c-oracle/` — reference only.
- `make reachability` zero regressions; `go test -race`; **run `golangci-lint`**;
  `gofumpt -w` every file you touch.
- Don't stage `website/static/map/world-sphere.json` or `docs/reports/reek/*`.

## Deliverable

A faithful first-player-God bootstrap in Go char creation, keyed strictly on the
player store being empty, with the exact `init_char` God stats/skills/conditions,
`DoStart` unchanged after it, and the three unit tests. Claude integrates the
harness side and the oracle scenarios.
