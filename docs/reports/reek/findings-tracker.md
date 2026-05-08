# Reek Findings Tracker

Maintained by Daeron. Updated per triage cycle.

## CRITICAL

| ID | Finding | File | Status | Fixed In |
|---|---|---|---|---|
| CRIT-001 | Graceless shutdown | cmd/server/main.go, main_web.go | FIXED | upstream (8c574d2) |
| CRIT-002 | HandleNonCombatDeath stub | pkg/game/limits_condition.go | FIXED | 052945a |
| CRIT-003 | Spawner.StartZoneResets no-op | pkg/game/spawner.go:57 | DOWNCLOSED | not a bug (dead code, not broken path) |

## HIGH

| ID | Finding | File | Status | Notes |
|---|---|---|---|---|
| HIGH-001 | Lock ordering undocumented | multiple | OPEN | Needs comment, not code fix |
| HIGH-002 | DefaultServeMux exposed | cmd/server/main_web.go | FIXED | upstream (8c574d2) |
| HIGH-003 | Duplicated entry points | main.go vs main_web.go | OPEN | Architecture refactor |
| HIGH-004 | Doors don't reset on zone reset | pkg/game/world.go | FIXED | upstream (8c574d2) |
| HIGH-005 | Non-TLS default | cmd/server/main_web.go | OPEN | Configuration decision |

## MEDIUM (selected — full list in triage reports)

| ID | Finding | File | Status |
|---|---|---|---|
| MED-001 | SA4004 unconditionally terminated loop | equipment.go:235 | REJECTED (intentional) |
| MED-002 | SA4000 identical && expressions | spec_procs3.go:903 | REJECTED (intentional) |
| MED-003 | 268 U1000 unused code (ported C handlers) | pkg/game/*, pkg/session/* | KNOWN (migration backlog) |
| MED-004 | SA1019 deprecated strings.Title | multiple | OPEN |
| MED-005 | SA6003 range over []rune | multiple | OPEN |
| MED-006 | SA6005 strings.EqualFold | multiple | OPEN |
| MED-007 | S1039 unnecessary fmt.Sprintf | multiple | OPEN |
| MED-008 | SA4006 assigned and not used | multiple | OPEN |

## Cycle History

| Date | Reek Report | Confirmed | Rejected | False Positive Rate |
|---|---|---|---|---|
| 2026-05-07 | Deep dive (server/) | 122 | 2 | 1.6% — Good reek |
