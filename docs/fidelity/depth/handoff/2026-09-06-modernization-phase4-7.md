# Modernization Phase 4.7 — Shared mob-script dispatch

Date: 2026-09-06  
Status: implementation prepared locally

## Scope

Pure refactor only. `FireMobFightScript` and `FireMobDeathScript` now delegate
to one private `fireMobScript` spine. The extraction preserves the existing
`ScriptEngine` gate, room/name lookup, trigger-specific `HasScript` check,
player context wiring, room context, script execution, and trigger-specific
error text. No player-facing bytes, RNG draws, or state transitions are
intentionally changed (R1/R3/R4).

## Call-path evidence

- `src/mobact.c:68-93` — mobile activity reaches fight-trigger dispatch.
- `src/fight.c:534-633` — mob death/raw-kill path reaches death-trigger
  dispatch.
- `pkg/combat/engine.go:604-605,1004-1005` — Go callback seams that call the
  two world methods.

The actual Go callback path was checked before extraction (R5e). The new unit
test proves both trigger names use the shared context/actor path; existing
script integration tests remain green.

## Verification

- `go test ./pkg/game` — pass, including
  `TestFireMobScriptsShareDispatchContext`.
- `pkg/scripting` and `pkg/combat` tests — pass.
- Oracle: `spec-proc-fighter` and `combat-death` — 2 passed, 0 failed,
  0 infra, 0 timed out.

The next step is the normal repository gates, then a PR based on merged
Phase 4.6 (`ce2919ee6`).
