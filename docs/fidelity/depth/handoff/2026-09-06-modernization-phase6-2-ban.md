# Modernization Phase 6.2 — ban list header builder

Date: 2026-09-06

## Scope

Folded the two fixed `ListBans` header rows directly into its existing
`strings.Builder` in `pkg/game/bans.go`. The C column widths, separator text,
CRLF framing, entry order, and date formatting are unchanged.

## Evidence

- `ban-depth@1,2,3,5,8`
- `TestIsBannedLiteralAsterisk`

The oracle scenario covers the empty list, add/list formatting, newest-first
order, duplicate handling, invalid flags, and C's fixed-column date/banned-by
fields. No C/oracle source was modified.

## Gate result

The focused tests, full repository gates, fidelity ledger checks, and the
focused ban oracle scenario were run on the branch before handoff. This is a
pure buffer-construction refactor; no player-facing literal or ban state
transition changed.
