# Modernization Phase 6.2 — infobar renderer builders

Date: 2026-09-06

## Scope

Converted the three proven infobar output buffers in
`pkg/session/display_cmds.go`—the changed-field update, enable frame, and
disable frame—from repeated string concatenation to `strings.Builder`.
The C field order, VT100 bytes, labels, level-dependent fields, and state
remembering remain unchanged.

## Evidence

- `TestInfobarUpdateUsesCLayoutAndBitOrder`
- `TestInfobarUnknownStateResetsToOff`
- `infobar-depth@1,2,3,5,8`
- `infobar-mortal-depth@1,2,3,5,8`

The focused tests and oracle scenarios cover the raw frame, update save/restore
bytes, C render order, immortal/mortal layouts, and off transition. No C/oracle
source was modified.

## Gate result

The focused tests, full repository gates, fidelity ledger checks, and focused
infobar oracle scenarios were run on the branch before handoff. This is a pure
buffer-construction refactor; no player-facing literal or state transition
changed.
