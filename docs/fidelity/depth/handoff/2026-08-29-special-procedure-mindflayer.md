# Depth-fidelity handoff — mindflayer

Date: 2026-08-29  
Slice: special procedure `mindflayer`  
Branch/PR: `glm/spec-mindflayer`, PR #771  
Merged main: `d6474b64f`

## Queue position

The special-procedure inventory was refreshed from the registration tables in
`src/spec_assign.c` and the procedure bodies in `src/spec_procs2.c`. The next
unclaimed registered procedure after this slice is selected only after the
next session repeats the census and verifies its source-and-registration
position; no item from a prior handoff was repicked.

`mindflayer` is registered for mob vnum `14414` at
`src/spec_assign.c:383`, with its body at `src/spec_procs2.c:1972-2000`.

## C path and behavior proved

The C audit followed the real dispatch path: mobile activity skips fighting
mobs, then `perform_violence()` invokes the registered mobile special after a
combat turn (`src/mobact.c:68-93`, `src/fight.c:1898-2032`). Command dispatch
passes a nonzero command value, so the special is autonomous-only. The body
requires `cmd == 0`, an awake fighting mob, and then consumes exactly one
`number(0,15)` roll:

- rolls 0 or 5: emit the two tentacle audience messages, apply direct
  `SPELL_SOUL_LEECH` damage at the victim's level, and add the victim's level
  to the mindflayer's hit points;
- roll 15: emit the two psychic-battering audience messages and apply direct
  `SPELL_PSIBLAST` damage at the mindflayer's level;
- all other rolls: return false with no visible or stateful effect.

The Go port preserves those gates, RNG arms, player/room audience text,
damage inputs, and soul-leech healing. Direct mob-special damage now enters the
existing combat redirect seam before modifiers, matching the ordering of the C
`damage()` path. No files under `src/` or `darkpawns-c-oracle/` were edited.

## Proof and verification

Scenario: `cmd/dp-oracle-diff/scenarios/spec-proc-mindflayer.txt`  
Focused tests: `pkg/game/spec_mindflayer_test.go`

The controlled live vehicle is oracle-green for seeds 1, 2, and 3. Those
one-round probes establish the autonomous combat dispatch and no-op/fallthrough
vehicle behavior; the successful soul-leech and psiblast arms, exact audience
messages, damage inputs, healing, and entry gates are proved by focused unit
tests. The final scenario does not claim an unisolated multi-round psychic
message/state result.

Required gates passed on the slice branch:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .   # clean
```

At handoff, `make fidelity-depth` reports:

```text
1111 total; 1066 proven/delegated; 13 blocked; 32 excluded
Actionable completion: 1066/1079 = 98.8%
```

## Fidelity rulings

This slice follows R1 (player-facing bytes), R2 (command/dispatch surface),
R3 (deterministic RNG and state effects), R4 (no invented behavior), and R5e
(the verified C call path). The inventory and manifest review continue to
apply R5b/R5c: repeated evidence is audited at the procedure-class level, and
the next slice must begin with a fresh source/registration census.

## Next action

On the next session: check out and pull `main`, run `make fidelity-depth`, read
`docs/fidelity/DEPTH_TESTING.md` and this newest handoff, refresh the ordered
special-procedure inventory, and take the next unclaimed registered procedure
in file-and-registration order. Do not revisit `mindflayer`.
