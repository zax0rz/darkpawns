# Modernization Phase 5.1 — Shared element teleport spine

Date: 2026-09-06  
Status: implementation prepared locally

## Scope

Pure refactor only. The player-only element vehicles now share one transfer
spine for room ordering, direct text, departure `Act`, checked
`PlayerTransfer`, optional destination look, and arrival `Act`:

- `specElementsMasterColumn`
- `specElementsPlatforms`
- `specElementsGaleruColumn`

`specElementsGaleruAlive` reuses the same player ordering helper before it
appends the separately ordered mob portion of its all-character scan. Its
player/NPC type switch and Galeru-specific destination framing remain local.
The talisman state machine, exact messages, destination VNums, audience flags,
and error-log labels are unchanged. No player-facing bytes, RNG draws, or
state transitions are intentionally changed (R1/R3/R4).

## Call-path evidence

- `src/spec_procs3.c:936-1024` — master-column and platform vehicles.
- `src/spec_procs3.c:1137-1216` — Galeru-column and Galeru-alive vehicles.
- `src/handler.c:514-557` — canonical `char_from_room`/`char_to_room`
  relocation path represented by Go `PlayerTransfer`/`MobTransfer`.
- `src/act.informative.c:725-840` — destination room-look and occupant
  rendering that must remain between transfer and arrival `act()`.

The actual Go dispatch path was checked before extraction (R5e). Existing
per-procedure unit tests cover the direct messages, ordering, state gates,
relocation, and audience behavior.

## Verification

- `go test ./pkg/game` — pass.
- Focused oracle: `spec-proc-elements-master-column-none`,
  `spec-proc-elements-master-column-all`,
  `spec-proc-elements-master-column-stale-state`,
  `spec-proc-elements-platforms`, `spec-proc-elements-galeru-column`,
  `spec-proc-elements-galeru-alive`, and
  `spec-proc-elements-galeru-alive-dead` — 7 passed, 0 failed, 0 infra,
  0 unpinnable, 0 timed out.

The next step is the normal repository gates, then a PR based on merged Phase
4.7 (`38c4f1b35`).
