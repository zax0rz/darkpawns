# Ported faithfully, never connected: the corpse wording

2026-11-28, Claude, for Zach. A field note on a defect Luna's lockstep hunt
found, and on Zach's classification of it. The root cause of the remaining
path isn't confirmed yet (DeepSeek is on it), so the claim is
needs-verification.

## What players saw

C's `make_corpse()` picks one of 15 long descriptions from the attack that
killed the victim (`src/fight.c:280-361`): "The hacked up, bloody corpse"
for slashes, "well-ventilated" for piercing, "charred … still smoking" for
fire spells, and so on, with "The corpse of X is lying here." as the default.
In the Go port, a new warrior killing recruits with a sword in the newbie
training area (rooms 16301/16302) got the plain default. Luna's lockstep hunt
found it in three sessions, and two repeated in a second census
(`~/Archives/darkpawns/play-windows/2026-11-27-luna-hunt-1/divergences/class-corpse-attacktype-wording.md`).
Neither Zach nor Egon, a returning player, had noticed: a missing flavor line
reads as normal.

## The history

- **2026-04-20, `e490ef05f` (BRENDA69/OpenClaw):** "spell-specific corpse
  descriptions". The whole table was ported faithfully, with unit tests on the
  mapping (`pkg/game/death.go`, `attackTypeToCorpseAttack`,
  `corpseAttackLongDesc`; `death_test.go`).
- **Until 2026-10-29:** the combat engine handed the death callback
  `AttackNormal` (0), which isn't a C `TYPE_*` value, so every weapon kill fell
  through to the default. The tests checked the table, never a real kill, so
  they passed for six months.
- **2026-10-29, `7accde6e6` (DeepSeek):** fixed one path ("pass one_hit's
  w_type to the death callback"), with an accurate comment at
  `pkg/combat/engine.go:1160-1166`. Nobody checked the other kill paths.
- **2026-11-27:** Luna's run on `a3f111900`, which includes that fix, still
  shows the default corpse for newbie-zone sword kills. So at least one more
  path loses the value.

## Zach's classification

Zach separates this from **port slop**, the plausible paraphrase of C that the
slop-squash thesis is about (the `yank` mis-port). Here nothing was
paraphrased: the leaf is C's table, byte for byte. His name for it is
**standard slop: ported faithfully, never connected.** It can happen in any
project built by AI agents, port or not, because each session starts without
memory of what the others built or still need to connect.

He places it as a cousin of **amnesiac clones** (PF-114). Both come from the
same missing memory:

| | Amnesiac clone (PF-114) | Never connected (this note) |
|---|---|---|
| What exists | two ports of one C function | one correct port |
| What's missing | knowledge that the first port exists | the wiring that feeds it real input |
| How it looks | two helpers that disagree | a helper that's right and unused, or fed a constant |
| What tests show | each clone green on its own | the leaf green, the live path never tested |
| Detector | C citations, not code similarity | live comparison (lockstep, census), or caller and argument audits |

Earlier members of the never-connected family in this repo: PF-007 (ten
implemented-but-unwired commands), PF-061 (`OnHuntItems` declared, never
wired, closed as Done with no commit), PF-069 (DP-1440, unwired mob-speech
sinks), and the "registered but never called" subsystems of 2026-09-17
(`/metrics` with every collector at zero, the admin audit trail, `pkg/privacy`).

Two traits make this instance a good teaching case:
1. **Green at the wrong layer.** The tests proved the mapping and couldn't
   fail for the bug that mattered (R5h).
2. **A fix that stopped at the first path.** `7accde6e6` was right, and it
   turned a total loss into a partial one, which is harder to notice (R5c:
   find one, find the class).

## How to test the claim

- Confirm the remaining path when DeepSeek's PR lands, and record which kill
  paths dropped the attack type.
- Candidate measurement: for every Go function that ports a C function taking
  a discriminating parameter (an attack type, a subcommand, a message number),
  check that some live caller passes a non-constant value. A port whose
  callers only ever pass one constant is a never-connected candidate.
