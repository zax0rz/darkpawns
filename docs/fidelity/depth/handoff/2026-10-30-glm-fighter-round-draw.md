# Fighter-round draw divergence — diagnosis handoff (2026-10-30, GLM)

Item 3 of the 2026-10-30 retaliation brief. Not fixed this round; the vehicle
is retained and reproduces deterministically on unmodified main.

## The vehicle

`cmd/dp-oracle-diff/scenarios/fighter-wimpy-window-depth.txt` — the God
(bare hands, maxhit 400, hit 100) against mob 20020 (SPECIAL(fighter),
src/spec_procs.c:509-532), six padded violence pulses. **Seeds 5, 11 and 17
diverge deterministically; 1, 3, 8, 13, 21 are green** (three runs each,
stable). The wimpy `set` line is a timing red herring: dropping it shifts the
warmup by one turn and the fight lands on a different alignment; the wimpy
value itself is never read (the guard never lands a hit on the implementor).

## What the divergence is NOT

- `Dice(1,4)` is not weather and not a weapon. It is **skill_message's
  variant selection** (`nr = dice(1, fight_messages[i].number_of_attacks)`,
  src/fight.c:1035) — the draw that picks which flavor line ("You scratch…",
  "…easily avoids the blow") a swing prints.
- `Number(0,13)` is the God's bare-hand damage bound (level 40/3,
  src/fight.c:1851).

So at the first mismatching draw (e.g. stream 4111, seed 5) C is completing
the God's swing (damage draw) while Go is drawing a message variant — the
God's swing **hit in C and missed in Go on the same to-hit roll value**, and
C's damage bands run higher ("hit hard" vs "hit"/"scratch") as if C's prone
multiplier applied. The transcript confirms: every diff line is a God swing
outcome flip; the guard only ever misses the God.

## What was audited and matches (static, R5e)

- `specFighter` action table: number(0,10), cases 1-4, subcmd=1
  (src/spec_procs.c:509-532 vs pkg/game/spec_procs.go).
- `mobHeadbutt` incl. the mortal-immortal sit-down gate before the draw
  (new_cmds.c:446-452).
- `mobBash` gates and order (act.offensive.c:419-477): peaceful, position,
  move spend, percent formula, NOBASH/sleeping/level zeroing, 131 threshold,
  both sitting arms.
- `mobBerserk` draw/gate order incl. the already-active gate AFTER the
  percent draw (new_cmds.c:2134-2160) and the failed-attempt zero-modifier
  affects; curse's spacing-hidden `aggro=TRUE` sibling lesson applied.
- `mobParry` incl. IS_PARRIED(FIGHTING(ch)) → MarkParried(vict) and the
  engine's dex_app/attacks-- consumption (fight.c:1999-2007).
- PC round pre-draws (Number(0,10000) parry check) and the round order.

## The narrowed root

Within one fighter-special round the engines' shared stream splits at a God
swing whose outcome flips on identical preceding draws — a victim-AC or
thaco delta applied by the special's action state in one engine only. The
berserk AC affect (+25, src/new_cmds.c:2146-2156) is the only AC-touching
action in the set; the head-to-head audit above matches, so the residual is
in the affect application/lifetime (e.g. duration-1 expiry ordering across
the pulses, or affect_total recomputation timing for mobs), not the gate
order. Next step: dump both engines' guard-AC series across the six pulses
(seed 5) — the first pulse where they differ names the site.

## Reproduction

    DP_ORACLE_BIN=… go run ./cmd/dp-oracle-diff \
      --scenario fighter-wimpy-window-depth --seed 5 --draw-parity

    draw-parity MISMATCH stream=4178 c=Number(0,13) go=Dice(1,4)
