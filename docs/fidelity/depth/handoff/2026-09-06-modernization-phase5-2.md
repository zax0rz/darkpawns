# Modernization Phase 5.2 — Shared special-procedure family gates

Date: 2026-09-06  
Status: implementation prepared locally

## Scope

Pure refactor only. The four tattoo shops now share list rendering and numeric
choice parsing while retaining each shop's original gate order, owner text,
price text, offer table, and `giveTattoo` state path. The three directional
castle guards share their command gate and autonomous second-guard scan, with
the down/up/north owner offsets and registration names passed explicitly. The
fighter and paladin specials share their combat gate and fighting-target
picker; their random arm ranges, return conventions, and native skill/spell
dispatch remain local. The two unassigned undead-knight procedures share only
their gate, taunt-vs-hunt split, and opposing-VNum scan; their taunt tables
remain local.

No player-facing bytes, RNG draw order, state transitions, or return behavior
are intentionally changed (R1/R3/R4). The C-derived procedure boundaries are
preserved rather than generalized across unrelated specials.

## Call-path evidence

- `src/spec_procs2.c:927-943,945-1137,1282-1340` — tattoo list/buy gates and
  shared `give_tat` success path.
- `src/spec_procs2.c:2134-2258` — castle guard command and autonomous paths.
- `src/spec_procs.c:509-568` — fighter and paladin combat gates and pickers.
- `src/spec_procs.c:1147-1295` — black/red undead source procedures.
- `src/spec_assign.c:108-109` — undead procedures are declared but have no
  `ASSIGNMOB` entry for their source VNums, so no registered oracle vehicle
  exists for those two unreachable procedures (R2/R4).

The actual Go registration and dispatcher paths were checked before
extraction (R5e). The existing tattoo, castle-guard, fighter, and paladin
focused unit tests remain the local proof boundary.

## Verification

- Focused unit tests for tattoo, castle-guard, fighter, and paladin families —
  pass.
- Focused oracle: 21 tattoo, castle-guard, fighter, and paladin scenarios —
  21 passed, 0 failed, 0 infra, 0 unpinnable, 0 timed out.

The next step is the normal repository gates, then a PR based on merged Phase
5.1 (`ecafde5302596e29`, PR #1399).

## 2026-09-11 status correction

The historical `implementation prepared locally` status above is stale. PR
#1400 merged this work as `ad3a9e7db70169fd02e4be51183b6f33ab97976f`, and the
current `origin/main` audit candidate is `18b5a911ce75702e5ab71df4aa46e87baa00c076`.
The dated audit is recorded in
[`2026-09-11-modernization-phase5-2-audit.md`](2026-09-11-modernization-phase5-2-audit.md).
It preserves the historical evidence, verifies the active registered families
within their named boundaries, and blocks a whole-Phase 5.2 verification claim
on the unassigned undead-knight source/coverage and vnum/probability mismatch.
