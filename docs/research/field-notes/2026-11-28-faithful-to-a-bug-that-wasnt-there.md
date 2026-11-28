# Faithful to a bug that wasn't there, and how often the port chose

2026-11-28, Claude, for Zach. Two results from Neon's C-bug decisions
inventory (darkpawns#3111,
`docs/fidelity/audits/2026-11-c-bug-decisions/`), which Zach asked for after
the C errata list started.

## 1. How often the port reproduced a C bug, and who decided

Neon inventoried 91 known bugs in the C original and what the port does about
each. Counts checked with `awk` against the TSV:

| decision | rows |
|---|---|
| fixed (the port departs on purpose) | 45 |
| reproduced (Classic shows what players saw) | 37 |
| oracle-patched (R1a) | 7 |
| accidental (departs, nobody decided) | 2 |

Who decided: an agent 32, a rule 30, Zach 29. **40 of the 91 decisions were
recorded only in code comments**, in no ledger, ruling or manifest. Before
the C errata list (2026-11-28) there was no single place where "we kept this C
bug on purpose" was written down, so nobody had counted. Neon proposed 32
errata rows from the reproduced set (E-002 to E-033), for Zach to rule on.
One standout: C's `slow` spell sets `AFF_HASTE` (`src/magic.c:1049-1051`),
so in the game players knew, slowing an enemy sped it up.

## 2. A hallucinated C bug, defended as fidelity (CB-090)

`pkg/game/skill_berserk_kuji.go`, `checkKkSuccess`, carries this comment from
the original kuji-kiri port (2026-06-18, `85186a560`, BRENDA69/OpenClaw):

> NOTE: the original C switch has no case for SKILL_KK_TOH, so prob stays 0
> for that seal and the 1-101 roll always exceeds it — the Toh seal's success
> check always fails. Ported faithfully; do not "fix".

C has the case (`src/new_cmds.c:1513-1515`:
`case SKILL_KK_TOH: prob = GET_SKILL(ch, SKILL_KK_TOH); break;`). The Go switch
omits it, so every ninja's Toh seal has failed for five months, and the
comment told every later reader not to touch it.

This is a new shape, distinct from both port slop (a paraphrase) and the
never-connected class (PF-190): **fabricated fidelity**. The agent invented a
fact about the source, and then used the project's own fidelity rule
("reproduce C's bugs") to protect the invention. The rule against fixing C
bugs is only as good as the claim that the bug exists. R5g (cite C from lines
grepped in the session) would have caught it: the claim cites no line, and
any grep of `check_kk_success` shows the case.

Detector: every Go comment that asserts what C does ("C has no …", "C never
…", "C's own bug") is a claim to check against `src/`. Neon's sweep found
this one by reading the comments the brief listed.
