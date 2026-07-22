# BRIEF — DP-1085 / O3: newbie start room ignores hometown (Go always 8004)

**For:** codex (frontier). **Owner of gate:** Claude (runs oracle red→green, reviews vs C).
**Branch:** `fix/dp1085-hometown-start-room` off current `main`.
**Finding:** DP-1085 / Fidelity **O3** (Medium). **This is the prerequisite that unblocks the
Domain-5 movement work** — until it lands, C and Go newbies stand in different rooms and every
room/navigation differential is confounded (this is the "the worlds are off" wall).
**Method rules:** read `src/interpreter.c` + `src/spec_procs.c` in the C oracle clone directly.
Gated by an **oracle red→green run** — a green build is NOT sign-off. **Do NOT modify the C
oracle world or src** (the two `lib/world` trees are already byte-identical where it matters;
room 8162 lives in the same `80.wld` as 8004 — C just never spawns newbies there).

---

## 1. The bug

**C:** a brand-new character is placed into room **8099**, which carries the `start_room`
SPECIAL (spec_assign.c:606). On the next pulse the spec prints the "ethereal image" intro and
teleports the mortal by `GET_HOME` (spec_procs.c:2239):

```
HOME_KD (1) -> 8162   (Temple Infirmary, Kir Drax'in)
HOME_KO (2) -> 18201  (altar, Kir-Oshi)
HOME_AZ (3) -> 21202  (altar, Alaozar)
default     -> 8004   (Temple Altar; UNREACHABLE via creation — menu only accepts K/O/A)
```

Creation only accepts hometowns k/o/a (interpreter.c:2096, `GET_HOME = 1|2|3`), so every C
newbie lands in 8162 / 18201 / 21202 — **never 8004**.

**Go:** `pkg/session/char_creation.go` gets this wrong twice:
1. Lines **488-497** map hometown→room but with wrong vnums: **K→8004** (should be **8162**) and
   **A→21258** (should be **21202**); only O (18201) is correct.
2. Line **546** then discards that entirely: `s.player.SetRoom(game.LoginStartRoom(s.player))`
   overwrites the room with the generic mortal start (`MortalStartRoom = 8004`, death.go:73) for
   **every** newbie. So all Go newbies end at 8004 regardless of hometown.

**Oracle-visible effect** (from the movement run): with hometown K, `look`/`south` from the
start room —
- C: `Temple Infirmary` (8162), no south exit → `south` = `Alas, you cannot go that way...`
- Go: `Temple Altar` (8004), has south → `south` = `Temple of the Cross`

## 2. The fix

1. **Correct the hometown→room mapping** to the C values. Factor a single source of truth in
   `pkg/game` mirroring the C `start_room` switch, e.g.:
   ```go
   // NewbieHometownRoom returns the room a brand-new character of the given
   // hometown starts in, mirroring C start_room (spec_procs.c:2239).
   // Home values: 1=KD, 2=KO, 3=AZ (char_creation hometown codes / C GET_HOME).
   func NewbieHometownRoom(home int) int {
       switch home {
       case 1: return 8162  // Kir Drax'in — Temple Infirmary
       case 2: return 18201 // Kir-Oshi — altar
       case 3: return 21202 // Alaozar — altar
       default: return MortalStartRoom // 8004
       }
   }
   ```
2. **Place the newbie at the hometown room, not `LoginStartRoom`.** At creation completion
   (char_creation.go ~546), the final `SetRoom` must use `game.NewbieHometownRoom(s.charHometown)`.
   Use the same value for the persisted `RoomVNum` (the DB `PlayerToRecord` at ~501 currently
   saves the buggy value from lines 490-496) so a returning player loads consistently.
3. **Do NOT change `LoginStartRoom` or respawn/death.** That path (login of an existing player,
   death respawn — death.go:128/578/614) is correct as the mortal start; C returning players load
   at their saved room, not their hometown. This fix is scoped to **brand-new character creation**.
4. Keep the 8099 newbie-intro beat (Go shows 8099 once via `sendCurrentRoomState` at ~545) — that
   mirrors C placing the char at 8099 before the spec fires. (The exact C "ethereal image" intro
   *text* differs from Go's 8099 room render — that's a separate cosmetic fidelity nit, NOT part
   of this PR; note it as a follow-up if you like, don't fix here.)

## 3. Oracle proof (red→green)

Add `cmd/dp-oracle-diff/scenarios/newbie-start-room.txt`: create a hometown-**K** newbie on both
servers, **no recall/navigation**, probe `look` (and optionally `south`). Model the per-server
creation prompt sequences on `look-start-room.txt` (they differ: C `Y/pass/pass/N`, Go
`y/pass/pass/Y/N`) but **drop the `recall`** (recall masks the divergence by sending both to 8004).

- **RED (pre-fix):** actor `look` → C `Temple Infirmary` vs Go `Temple Altar`.
- **GREEN (post-fix):** both `Temple Infirmary` (8162), byte-identical (8162 is in the shared,
  identical `80.wld`; Observation domain already made room rendering faithful).
- Optionally add O (→18201) and A (→21202) coverage — but note those zones: 18201 is in a file
  present in both trees; 21202 likewise — verify the target `.wld` exists in the Go tree before
  relying on it (the Go tree lacks `150.wld`/`165.wld`).

Also unit-test `NewbieHometownRoom` (1→8162, 2→18201, 3→21202, other→8004).

## 4. Acceptance gate

1. `--scenario newbie-start-room` red→green (Claude will re-run; run it yourself first).
2. `look-start-room` (the existing recall-aligned O3 regression scenario) stays green.
3. No regression to login/respawn: an existing player logging in still lands at their saved
   room / mortal start, not their hometown.
4. `make check-fmt vet` + `go test ./...` green. Instance-safe (no `obj.Prototype.*` writes).

## 5. Why this unblocks Domain 5 (movement)

Once Go routes hometown-K newbies to 8162, the C and Go actors/observers in
`scenarios/movement.txt` co-locate at 8162 (identical `80.wld`), so the leave/arrive/position/
follow probes finally isolate **movement-logic** divergence instead of start-room noise. After
this lands, rework `movement.txt` to navigate the shared 8162 newbie-zone graph (drop its stale
"Go 8004 exits" comments and the 8004 assumption). See
`docs/briefs/BRIEF-2026-07-14-domain5-movement.md`.
