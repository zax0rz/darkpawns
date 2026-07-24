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

## The trigger — key on the player store being EMPTY

C's `top_of_p_table == 0` means "no players exist yet." The faithful Go
equivalent is **"the player store is empty at creation time."** Implement a
`CountPlayers()` (or equivalent existence check) on the DB layer
(`pkg/db/player.go` — there is a `players` table) and, at char-creation
finalization (around `pkg/session/char_creation.go:497` where `CreatePlayer` is
called, and/or `pkg/game/character.go DoStart`), apply the God block **iff the
player table has 0 rows** before this character is inserted.

> ⚠️ **Interface contract with the harness (Claude's parallel work):** the
> bootstrap MUST key on this "player store empty" signal and nothing else. The
> oracle harness will control emptiness per scenario — empty for God-fixture
> scenarios (first char → God), pre-seeded (≥1 row) for all normal scenarios
> (first char → mortal, exactly as today). Do NOT key on an in-process
> "first-connection-this-boot" counter — that would wrongly crown the primary
> actor in every existing scenario (combat-death, backstab-opener) and desync
> against C. Empty-store is the one correct signal.

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

- **first player → God:** with an empty player table, a newly created character
  has level `LVL_IMPL`, exp 7000000, max_hit 500, max_mana 100, max_move 82,
  `GetSkill` == 100 for a representative set of skills, and conditions -1.
- **second player → mortal:** with the table non-empty (one row present), a new
  character is an ordinary level-1 of its class — conditions 24, skills 0 except
  its `DoStart` class grants (e.g. thief backstab 10). This is the regression
  guard that existing scenarios still get mortals.
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
