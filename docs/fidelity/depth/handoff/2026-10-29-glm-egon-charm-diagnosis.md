# Charm-shopkeeper draw divergence — diagnosis — 2026-10-29 (GLM)

Scenario `spell-charm-shopkeeper-depth` matched transcripts but diverged draw
streams. This round reproduces and localizes the divergence; no game code
changed.

## What the evidence shows

Reproduced on clean `origin/main` at seed 1 (3/3 stable reruns), first
mismatch at stream index 63584:

- C: `Number(0,900) = 662`, then `Number(0,100)`, `Number(1,20)`,
  `Dice(20,4)`, `Dice(14,4)` — the `perform_violence` signature
  (`src/fight.c:1916-1949`: attack-count roll `number(0,900)`, to-hit
  `number(1,20)`, damage dice), from the fight the probe started.
- Go: `Number(0,3)`, `Number(0,3)`, `Number(0,81)` — the end of Go's stream.

Both engines then end with the same final triple of draw kinds; C simply
runs ~188 more violence-family draws before reaching its own end.

## The charm handler is exonerated

63,583 draws matched exactly — the entire boot, world load, character
creation, `goto 12133`, the probe's `cast 'charm' merchant` ("Okay." /
"Your victim resists!" — the save succeeded, and `spell_charm`'s resist arm
made the merchant `hit(victim, ch)`, `src/spells.c:428-433`), the immediate
retaliation, and dozens of shared violence rounds (the merchant's `20d4`
barehand appears in both streams). The divergence begins strictly after the
probe's matched output, inside the post-probe settle/drain window.

## Trigger condition

Seeds 2, 3, 4, and 5 are draw-clean — all of them also print "Your victim
resists!" and start the same fight. Only seed 1's fight course produces the
divergence. The distinguishing seed-1 event is a combat outcome inside the
drain window whose aftermath the two engines settle differently; the draw
signatures are consistent with the kill/extraction boundary: C's
`extract_char` only queues the body (`src/handler.c:1194-1209`) and
`extract_pending_chars` drains it at the next heartbeat
(`src/comm.c:805`), so C keeps drawing violence work around the not-yet-
extracted body, while the Go port marks the body dead and retires it
immediately (MED-011 in `pkg/game/death.go`). The group-rewards round
baseline-confirmed the same mechanism mid-probe (an ungrouped-pet kill
showed the identical mismatch at the kill pulse on both the #2215 base and
the fix branch — see `oracle-runs/2026-10-29/glm-egon-group-rewards/draw-baseline/`).

## Conclusion and proposal

This is not a charm-path, setup, world-load, or fixture defect, and the
transcript is green. The first causal difference is the engines' post-kill
drain semantics in the settle window (deferred extraction vs immediate
retirement), which is invisible to transcripts but shifts the shared stream
whenever a kill lands inside a drained window. Two manifestations are now
on record (mid-prose kill pulse; post-probe drain window), both
baseline-confirmed on `origin/main`.

Proposed follow-up (needs its own round): mirror C's deferred-extraction
window in the Go violence iteration — the retired-but-present body keeps
participating until the heartbeat's extraction pass — with a targeted census
over the combat/death scenarios. That change is engine surgery with a wide
blast radius and should not ride on a fix PR.

## Evidence

- Baseline (clean origin/main, 2026-10-29):
  `oracle-runs/2026-10-29/egon-shopkeeper-baseline-draw/` — verdict CLEAN
  with `draw_divergent=1`; per-engine draw logs under `initial-logs/`.
- This round's reproductions (new directories, reds preserved):
  `oracle-runs/2026-10-29/glm-egon-charm-diagnosis/` — seed-1 draw-parity
  mismatch ×3 plus seed 2-5 clean runs and the show-oracle transcripts.
- Sibling manifestation:
  `oracle-runs/2026-10-29/glm-egon-group-rewards/draw-baseline/`.
