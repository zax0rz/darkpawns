# BRIEF — Domain 7c: `consider` target-eval — O15

**For:** codex (frontier). **Owner of gate:** Claude (oracle red→green + review vs C).
**Branch:** `refactor/domain-consider` off `main`.
**Findings:** DP-1104 / O15 (not-found text, private-eval broadcast leak, pronoun/sex mapping).
**Part 3 of Character-view.** **Method rules:** read `src/act.informative.c` `do_consider`
(2330-2431) directly. Gated by an **oracle red→green run**.

---

## 1. Oracle-PROVEN RED (`scenarios/character-view.txt`, verified 2026-07-15)
Actor + co-located observer (both level-1 K Warriors) at 8162. Diffs (C `-`, Go `+`):

| # | probe | audience | C oracle | Go port | finding |
|---|---|---|---|---|---|
| 1 | `consider Cviewobs` | **observer** | (silent) | `Cviewactor considers Cviewobs.` | O15 — **private eval leaked to the room**; C is `TO_CHAR` only |
| 2 | `consider Nobody` | actor | `Consider killing who?` | `They aren't here.` | O15 — wrong not-found text |

**Note (NOT a consider bug):** `consider Cviewobs [actor]` also differs on the "physical shape"
line (`in about the same` vs `in a little worse`) — that's purely because the two chars' HP
differ (DP-1161 model delta), feeding a different `hitdiff`. The **thresholds match**; do not
"fix" the wording. Once HP is aligned (DP-1161) this line matches. At level 1 consider is
otherwise **deterministic** (bare-hand dam `number(0, level/3)` = 0, no dice), so the damdiff
("fair fight") and leveldiff ("about as confident") lines already MATCH.

## 2. Root cause
Session `consider` (pkg/session/consider.go): not-found prints `"They aren't here."` then
**broadcasts** the result to the room (`:210-216`, `:283-298`); its pronoun mapping assumes
`0 neutral / 1 male / 2 female` (`:252-273`) but player sex is `0 male / 1 female / 2 neutral`
(pkg/game/player.go:68-72) → wrong `$E/$S`. C emits the whole thing via a single
`act(buf, TRUE, ch, 0, victim, TO_CHAR)` with proper $-code sex resolution.

## 3. Faithful C reference (`do_consider`, act.informative.c:2330-2431)
- `one_argument`; `!get_char_room_vis` → **`"Consider killing who?\r\n"`** (RED #2).
- `victim == ch` → `"Easy!  Very easy indeed!\r\n"`.
- Compute `chardam`/`victdam` (str todam + damroll + wielded dice **or** bare hands
  `number(0, GET_LEVEL/3)`), `hitdiff = GET_HIT(vict)-GET_HIT(ch)`, `damdiff = victdam-chardam`,
  `leveldiff = GET_LEVEL(vict)-GET_LEVEL(ch)`. Port the **exact three threshold ladders**:
  - damdiff: `>20` eat-for-lunch / `>10` tear-you-up / `>5` hurt-you / `>-3` **fair fight** /
    `>-5` easy kill / `>-10` very easy kill / else not-worth-the-effort. (each `"$N looks like …, "`)
  - hitdiff (vs multiples of `GET_HIT(ch)`): `>4x` much better / `>2x` a lot better / `>1x` better /
    `>=0` **about the same** / `>-25` a little worse / `>-50` worse / else a lot worse.
    (each `"looks to be\r\nin … physical shape …, "`)
  - leveldiff: `>lvl` many won battles / `>lvl/2` knows $S opponent / `>-1` **about as confident** /
    `>-3` less confident / `>-5` much less / `>-7` ready to run / else never been in battle.
- Emit once: **`act(buf, TRUE, ch, 0, victim, TO_CHAR)`** — `TO_CHAR` ONLY (RED #1: no room
  broadcast). `$N`/`$E`/`$S` resolve against the **victim's** sex via the canonical act() pronoun
  tables — use `Act()` (F0a) so sex mapping is shared/correct, not a local table.

## 4. Session adoption
Route `consider` to a game-owned `DoConsider(ch, arg)` over `Act(... ToChar)`; delete the
session-local pronoun table and the room broadcast. Preserve the exact strings above.

## 5. Acceptance gate
1. **Oracle red→green:** `--scenario character-view` → `consider Cviewobs [observer]` clean (no
   leak) and `consider Nobody [actor]` = `Consider killing who?`. The `consider Cviewobs [actor]`
   "physical shape" line stays divergent until DP-1161 (HP) — call that out, don't mask it.
2. **Unit tests** (exact C strings): not-found, self ("Easy!  Very easy indeed!"), each of the
   three threshold ladders at representative boundaries, `TO_CHAR`-only (no observer message),
   and pronoun correctness for sex 0/1/2 (male/female/neutral) via a fixed fixture.
3. `make check-fmt vet` + `go test ./...` green; no WS schema break.

## 6. Gotchas
- **Sex constants:** DP is `0 male / 1 female / 2 neutral` — the old local table had it backwards.
  Use the shared act() pronoun path; assert $E/$S for all three.
- **`number()`/`dice()` are RNG** — at level 1 the bare-hand term is `number(0,0)=0` so it's
  deterministic; at higher levels consider becomes Tier-2-only. Keep the ladders exact regardless.
- **HP delta (DP-1161)** drives the only residual actor-line diff — not this PR's concern.
