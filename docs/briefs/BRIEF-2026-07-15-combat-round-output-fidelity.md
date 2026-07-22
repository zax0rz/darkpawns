# BRIEF — Combat round output fidelity (synchronous first swing + message system)

**For:** codex (frontier) — with a Kimi-able mechanical sub-part (§3). **Owner of gate:** Claude
(oracle red→green + review vs C; Claude owns the combat-swing scenario).
**Branch:** `refactor/combat-round-output` off `main`. **Finding:** DP-1165.
**This is B-2 (Tier-2 combat), first slice.** **Method rules:** read `src/fight.c` `hit`/`one_hit`/
`damage`/`dam_message`/`skill_message`/`load_messages`, the `attack_hit_text[]`/`fight_messages[]`
tables, and `lib/misc/messages` directly. Gated by an **oracle red→green run**.

---

## 0. Context (why this is bigger than "draw order")
The B-2 scenario (`scenarios/combat-swing.txt`, Claude-built: L1 K warrior walks 8162→8105
non-peaceful street, `hit trainee` vs sentinel #16303) proved combat capture is **deterministic**
(3/3 identical). But Go's first-swing output diverges from C on three entangled axes — and you can't
compare the `number(1,20)` to-hit draw until the message layer matches, because **the message layer
itself consumes RNG.** Fix the message system + swing cadence first; draw-value parity falls out.

Observed (DP_SEED=1, `hit trainee`):
```
C : You try to hit a guard trainee who easily avoids the blow.
Go: You attack a guard trainee!
    You scratch a guard trainee as you hit a guard trainee.
    a guard trainee tries to hit you, but misses.
```

## 1. Synchronous first swing + kill the invented line (codex — the cadence fix)
- C `do_hit` (act.offensive.c) → `hit(ch, vict, TYPE_UNDEFINED)` performs the **first attack
  synchronously**: the swing result prints immediately, then `WAIT_STATE(ch, PULSE_VIOLENCE+2)`.
- Go `cmdHit` enrolls in the combat engine (`StartCombat`) and defers the first swing to the next
  ~2s tick, printing an invented `"You attack a guard trainee!"` first. **Delete that line** and make
  `cmdHit` resolve round one inline (call the same round path the engine tick uses, once, now), so
  the first swing's output matches C's timing. Subsequent rounds continue on the engine pulse (both
  servers) — only the first swing needs to be synchronous for a clean aligned comparison point.
- **⚠ Harness reality:** the quiescence capture (300ms) only sees the first burst; continuous rounds
  land in later windows and won't align between servers. The synchronous first swing IS the gate
  point — don't try to diff the whole fight (Claude will handle multi-round capture strategy later).

## 2. Message selection must match C's draw count EXACTLY (codex — the draw crux)
Per swing, C draws for message selection; Go must draw identically or everything downstream desyncs:
- **`skill_message` path** (used when `fight_messages[]` has the weapon type): C picks a random
  variant → `nr = dice(1, fight_messages[i].number_of_attacks)` (fight.c:1035) — **one draw per
  swing**. Barehand (TYPE_HIT) HAS entries (see §3), so C takes this path and draws.
- **`dam_message` fallback** (no fight_messages for the type): C is **deterministic** — fixed message
  per severity tier, **no draw**. Go's `DamMessage` currently calls `randPick` on room/char/victim
  (fight_core.go:731-733) = up to 3 draws where C makes 0. **Remove the randPick draws**; select the
  fixed tier message like C.
- Wire it as C does: `hit()` calls `skill_message()`; if it returns "handled", stop; else
  `dam_message()`. Go already has this shape (cbSkillMessage→DamMessage, fight_core.go:435) — the bug
  is the data (§3) + the draw mismatches above. Route all draws through `dprng` (Phase 0a stream).

## 3. Load `lib/misc/messages` → fight_messages, TYPE_HIT coverage (KIMI-ABLE — mechanical)
Go does **not** load `lib/misc/messages` (46KB data file); its hardcoded skill-message table lacks
barehand, so it falls through to `DamMessage`. Mechanical port:
- Parse `lib/misc/messages` into `fight_messages[]` (per weapon-type: N variants, each with
  hit/miss/god/death sub-messages + `number_of_attacks`). Match C's `load_messages()` format exactly.
- Ensure **TYPE_HIT (barehand)** entries are present — the exact flavor C emits lives there, e.g.
  `lib/misc/messages:392` "You swing your fist at $N, but miss $M!", `:420` "You try to hit $N who
  easily avoids the blow." Transcribe the whole file, don't cherry-pick.
- Also transcribe the **`dam_message` severity ladder** faithfully (fight.c): dam==0→miss tier;
  ≤2→"scratch"; ≤4; ≤6; ≤10; ≤14; ≤19; ≤23; ≤33; ≤43; ≤53; … (copy every threshold + all 3 audience
  strings per tier with `#w`/`#W` verb tokens), and the **`attack_hit_text[]`** verb table
  (TYPE_HIT→"hit"/"hits", etc.). These are static tables — verify against source, exact strings,
  including capitalization ("A guard trainee" — C capitalizes the room/victim sentence start).
This sub-part is pure data transcription + a parser; hand it to Kimi to run ahead while codex does §1/§2.

## 4. Acceptance gate
1. **Oracle red→green:** `--scenario combat-swing` → `hit trainee` first-swing block matches C under
   `DP_SEED` (wording AND the swing result/damage — which proves the to-hit `number(1,20)` and the
   message-selection draws are all in C's order). No "You attack X!" line.
2. **Unit tests:** `skill_message` selection draw count (=1) for a type with fight_messages;
   `dam_message` determinism (0 draws) + exact tier strings at each severity boundary; TYPE_HIT
   flavor loaded from the file; verb-token (`#w`/`#W`) + `$n/$N/$m/$e/$M` substitution via Act().
3. `make check-fmt vet` + `go test ./...` green; import guard (dprng) green; no WS schema break.

## 5. Gotchas
- **Message selection is RNG** — the whole point: C's per-swing `dice(1, number_of_attacks)` draw
  must be reproduced, and `dam_message`'s must-NOT-draw must be honored. Draw count is law.
- **Never touch the oracle / `lib/misc/messages` is C's data** — port it, don't edit it.
- **Route every draw through `dprng`** (Phase 0a) — no `math/rand` (import guard enforces).
- First swing only for the oracle gate; multi-round capture is a later Claude harness task.
- Whitespace/caps are fidelity (see the earlier message-parity brief).
