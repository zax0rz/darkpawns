# Modernization Phase 6.2 — prompt renderer builder

Date: 2026-09-06

## Scope

Converted the proven `Session.promptText` playing-branch buffer in
`pkg/session/session_send.go` from repeated formatted-string concatenation to a
`strings.Builder`. The invisibility prefix, C vital-field order, and infobar
suppression are unchanged. Writing, AFK, and inactive branches remain explicit
early returns because C suppresses the playing-branch fields for those states.

## Evidence

- `TestInvisPromptIncludesCLevel`
- `TestPromptTextPreservesCFieldOrdering`
- `TestInfobarUpdateUsesCLayoutAndBitOrder`
- `TestCmdPromptMatchesCDoDisplay`

The added matrix pins the C-visible byte order for the three vital fields and
the invisibility/infobar combinations. No C/oracle source was modified.

## Gate result

The focused session tests and repository gates were run on the branch before
handoff. The change is a pure buffer-construction refactor; no command gate,
state transition, or player-facing literal changed.
