# BRIEF (codex) — DP_CLOCK settle-pump (DP-1162, follow-up to PR #384)

**Owner:** codex. **Gate:** Claude runs `combat-swing` red→green + full sweep under `DP_CLOCK`. **Branch off `main`, one PR.**
Background + gate result: `docs/briefs/DESIGN-2026-07-17-dp-clock-pulse-sync.md`. PR #384's freeze is correct and stays; this adds the deterministic settle-pump that freeze-only lacked. All five decisions below are answered from the C source — implement them exactly; do not re-infer.

## Why (one paragraph)
A new char enters at **room 8099** (`interpreter.c:2241`, direct `look_at_room` — does NOT trigger the room special). It sits there until **`room_activity` calls the room special** `GET_ROOM_SPEC(ch->in_room)(ch,room,0,0)` (`comm.c:736-738`), which runs `SPECIAL(start_room)` (`spec_procs.c:2204`): prints the entry "dream" and teleports the char to their hometown infirmary (8162 for HOME_KD). `room_activity` is bundled with `mobile_activity`+`object_activity` at **`PULSE_MOBILE`** (`comm.c:828-831`, = 40 passes). Frozen clock ⇒ birth never fires ⇒ the first real command is consumed by the birth ⇒ navigation desyncs vs the Go port (which births at creation). The pump makes birth deterministic.

## The five decisions (answers)

1. **Settle count = 40 pulses (one `PULSE_MOBILE`).** `pulse` inits to 0 and the frozen boot never advances it, so pumping `heartbeat(++pulse)` for 1..40 hits `pulse % PULSE_MOBILE == 0` exactly once at 40 → fires `mobile_activity`+`room_activity`+`object_activity` once (births the char; ticks the fixture mob once). `perform_violence` fires at 20 and 40 but draws nothing with no combatants. Nothing else crosses a boundary in 40 (`PULSE_ZONE`=100; `point_update`/weather ≫ 40; idle-pw at 150). Pump **exactly 40 per entry**; the counter accumulates across entries (char2 → 41..80, fires once at 80). `start_room`'s birth path draws **no RNG** (verified) — the only settle draws are the one `mobile_activity` tick (fixture mob) + `room_activity`'s `number(0,1)` iff a char stands in a `ROOM_FLOWS` room (8099/8162 are not flow rooms → none).

2. **Pump after each character's entry; NOT after boot.** Each newbie sits at 8099 independently, so pump 40 immediately after each `1` (enter-game) step — primary and every peer. **No boot pump:** C populates zones synchronously at boot (`db.c:387-392` resets every zone), so the fixture mob exists pre-pump; the Go port must match (decision 5).

3. **Append settle output to the creation transcript before diffing — for `[creation:*]` only; drain it for `[setup:*]`.** C emits the dream *during* the settle-pump; the Go port emits it at creation. Appending the pump's output to the creation block reconciles the timing so the dream is still diffed (keeps `character-creation` green). For `[setup:*]` scenarios the pump output is drained like the rest of setup (consumed to keep the stream aligned; not diffed).

4. **Minimum settle-capable heartbeat only — do NOT unify point_update/zone_reset now.** The pumped `heartbeat(pulse)` must dispatch, in C's order (`comm.c:817-833`): `mobile_activity` → `room_activity` → `object_activity`, plus `perform_violence` as a structural no-op (empty combat list ⇒ zero draws). Draw-order among these must byte-match C. `zone_update`, `point_update`, weather, idle-pw do not fire within a 40-pump, so leave them out of the pumped path for now (they return to the deferred combat-death/round-pump work, where violence and point_update actually matter).

5. **Go must populate initial zones synchronously at boot, before its readiness marker.** Under `DP_CLOCK` the async reset ticker is off, so do the initial zone population inline during boot (mirroring `db.c` boot reset) and only then emit `"World state restored"`. The harness already gates setup on that marker (`main.go:221 waitForLog`), so no boot pump and no extra handshake are needed. (Empirically the Go port already had the fixture mob at 8105 under `DP_CLOCK`, so this likely already holds — just make it a guaranteed pre-marker invariant, not ticker-dependent.)

## Reserved pump command
A control line the harness sends over telnet, intercepted **before** `command_interpreter`/`ExecuteCommand`, active **only when `DP_CLOCK` is set** (inert otherwise, so production is untouched):

- **Syntax:** `~dpclock pulse <n>` (leading `~` = never a real command; pick any unambiguous sentinel but keep C and Go identical).
- **Behavior:** fire `heartbeat(++pulse)` exactly `n` times, then return. Must be **draw-neutral itself**: do NOT route through `command_interpreter` (skips the per-command `number(0,3)` AFF_HIDE-clear draw from #375) and do NOT touch wait-state. Only the heartbeats it fires may draw. Heartbeat side-effect output (the dream) flows to the socket normally; the harness drains it via quiescence — no ack needed.
- **Harness (`cmd/dp-oracle-diff`):** after each character's enter-game step, send `~dpclock pulse 40` to that engine's connection and drain (append-to-creation or drain per decision 3). Same bytes to both engines.

## Acceptance (Claude-gated, from a PR-branch worktree)
1. `--scenario combat-swing` → `no normalized divergence` (Go already emits C's faithful "You swing your fist at a guard trainee, but miss him!"; the pump must let C navigate to 8105 so both swing).
2. Full committed sweep under `DP_CLOCK` stays green — especially `character-creation` (decision 3 keeps the dream diffed) and every fresh-char scenario whose probe is near entry.
3. `DP_CLOCK` unset ⇒ byte-identical to today on both engines.
