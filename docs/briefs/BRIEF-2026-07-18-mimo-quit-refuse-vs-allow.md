# BRIEF (mimo) — `quit` refuse-vs-allow divergence in hunger-thirst

**Mode:** measured investigation → fix (you have `DP_ORACLE_BIN`). **Owner:** mimo. **Gate:** Claude runs `hunger-thirst` red→green. **Branch off `main`, one PR.**

## The divergence (measured 2026-07-18, both engines deterministic)
End of `hunger-thirst`: the char types `quit`. Diffed `[quit]` block:
- **C** emits the *successful* quit: `Goodbye, friend.. Come back soon!`
- **Go** emits the *refuse* block: `Type REALLYQUIT to quit the game and lose your eq.` / `Return to the temple and QUIT to leave the game and keep your equipment.` / `You can type RECALL to return to your temple.`

So **Go refuses a quit that C allows.** C considers the char in a safe/quittable state; Go does not.

## What we know (don't re-derive)
- Go's quit logic is `pkg/game/quit.go` `World.DoQuit` (DP-1115, PR #377): safe-quit keys on `isokquit` (safe rooms 8004/8008, hometown-gated homes 18201/21202/21258, owned house). Refuse block fires when `!isokquit`.
- C safe-quit = `Goodbye, friend..` then descriptor returns to the login menu; the refuse text is DP's non-safe-room message.
- Known related gap in memory (DP-1115 follow-up): *safe* `quit` from 8004 — C returns to the **login menu**, Go historically **hard-disconnects**. That's a different symptom (disconnect vs menu) than this one (allow vs refuse), but same subsystem — check if they share a root.

## The question to pin
**Where is the char actually standing (room vnum) at quit time, in C vs Go?** The likely root is a **positioning divergence**: if C's char is in a safe room (e.g. 8162/8004/8008) but Go's char ended up elsewhere (or vice-versa), the quit gate legitimately diverges. Alternatively both are co-located and Go's `isokquit` table/logic is wrong for that room.
1. Instrument/inspect both engines' `in_room` for the char right before `quit` (C: `GET_ROOM_VNUM(ch->in_room)`; Go: the player's `RoomVNum`). Are they the same room?
2. **If different rooms** → find why the char's position drifted between engines through the hunger-thirst setup (open pack / drink / eat — none should move the char; is there a spec/teleport, a settle-pump effect, or a `recall`?). Pin the positioning divergence; that's the real bug.
3. **If same room** → compare that room's vnum against C's isokquit safe-room test vs Go's `isokquit`. Is Go missing a safe-room case C has (or applying a fighting/position/hunger gate C doesn't)? Cite the C quit gate (`act.other.c` / wherever `do_quit`/`SCMD_QUIT` lives) line-by-line vs `quit.go`.

## Hard rules
- Faithful to C: match C's actual safe-room set and quit gate. No normalization, no scenario-file hack to dodge it (unless the true root IS a `[setup]` asymmetry — if so, prove it the way the observation co-location bug was proven, and fix the scenario faithfully).
- Temp instrumentation reverted + C rebuilt clean (standard). Do NOT disturb the committed `dp-oracle-seam` (DP_SEED/DP_CLOCK) — edit only temp hunks.

## Deliverable / Acceptance (Claude-gated)
1. Report: the char's room vnum in each engine at quit; root cause (positioning vs isokquit-logic); which side is faithful to C.
2. Minimal faithful fix (files/lines).
3. `hunger-thirst` `[quit]` block → `no normalized divergence` (Go emits `Goodbye, friend.. Come back soon!`). *(The score `Movement points:` −1 in the same scenario is a SEPARATE brief — don't fix it here; the scenario may stay red on move until both land, that's expected.)*
4. Full sweep stays green; instrumentation reverted, C builds clean.
