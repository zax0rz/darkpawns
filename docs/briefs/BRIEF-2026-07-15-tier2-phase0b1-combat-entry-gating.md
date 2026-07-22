# BRIEF — Tier-2 Phase 0b-1: combat-entry gating (root cause: room-flag lookup)

**For:** codex (frontier). **Owner of gate:** Claude (oracle red→green + review vs C).
**Branch:** `refactor/tier2-combat-gating` off `main`.
**Finding:** DP-1163 (name-based room-flag lookups can't read static world flags).
**Depends on:** PR #349 (Phase 0b Part A, merged). **Scenario (Claude-built, on main):**
`cmd/dp-oracle-diff/scenarios/combat-round.txt`. **Method rules:** read `src/fight.c` `damage()`
gate ladder + `src/act.offensive.c` `do_hit`/`do_kill` directly. Gated by **oracle red→green**.

---

## 0. The discovery (read this first — it changes what you fix)
The combat-round scenario looked like "Go is missing the peaceful-room combat gate." It is NOT.
Go's `cmdHit` (pkg/session/combat_cmds.go:122) **already has** a faithful peaceful gate — plus the
newbie-PK and self gates — with the exact C strings. The gate is **dead code** because the room-flag
lookup it calls always returns false:

- `wld.go:216` stores `room.Flags` as the **raw decimal bitvector words** (room 8162 → `["28","0","0","0"]`;
  PEACEFUL = bit 4 = 16, which IS set in 28).
- `RoomHasFlag`/`hasRoomFlag` (other_helpers.go:90) + `roomHasFlag` (limits.go:106) do **pure name
  comparison** (`strings.EqualFold(f, "peaceful")`) against those numeric strings → never matches.
  There is no name→bit table.
- The correct decoder already exists: `roomHasFlagBit(flags, bit)` (room_flags.go) and the bit-first
  `movementRoomHasFlag(room, bit, legacyName)` — which is why movement/weather flag checks DO work.

**So Part B-1 is a room-flag-lookup fix (DP-1163), not a combat rewrite.** Fixing the lookup lights
up the already-correct combat peaceful gate — and several other silently-dead gates.

## 1. Oracle-PROVEN RED (`--scenario combat-round`, verified 2026-07-15)
Level-1 K warrior does `hit cub` at 8162 (ROOM_PEACEFUL; non-aggressive mob #11035 spawned there):
```
-This room just has such a peaceful, easy feeling...      (C — combat blocked, fight.c:1337)
+You attack a tiger cub!                                    (Go — combat proceeds)
```
(PvP is impossible here anyway — fight.c:1345 blocks a level≤10 attacker — so the scenario uses a
mob target; that's also why the peaceful gate, not the PK gate, is the visible RED.)

## 2. The fix — make the name-based room-flag API bit-aware (DP-1163)
Give `hasRoomFlag`/`roomHasFlag`/`RoomHasFlag` a name→bit resolution: map the flag name to its
canonical `Room*` constant (room_flags.go has the bit list; look.go:986 has the display-name order
DARK/DEATH/!MOB/INDOORS/PEACEFUL/…) and delegate to `roomHasFlagBit(room.Flags, bit)`. **Preserve the
dynamic-name path** — houses.go appends literal `"house"`/`"atrium"` name strings to `room.Flags` at
runtime, so keep matching those too (mirror `movementRoomHasFlag`: try the bit, then the legacy
name). Prefer one canonical resolver that all three wrappers call.

Do NOT change `wld.go`'s storage format or the bit constants; just make the name lookups resolve
through the bits. Do NOT re-add a competing flag representation.

## 3. Faithful C reference — `damage()` gate ladder (fight.c ~1330-1360)
For completeness, confirm Go's `cmdHit` gates match C's order/strings once the flag lookup works
(they appear faithful today, but verify against source):
1. peaceful (only if `!IS_OUTLAW(victim) && FIGHTING(victim) != ch`): `"This room just has such a
   peaceful, easy feeling...\r\n"` → block.
2. attacker newbie (`!IS_NPC(ch) && !IS_NPC(victim) && GET_LEVEL(ch) <= 10`): `"You are not
   experienced enough to attack $N!"` → block.
3. victim newbie (`… GET_LEVEL(victim) <= 10 && !PLR_OUTLAW`): `"Ancient forces protect $N from your
   wrath!"` → block.
4. shopkeeper protection (`is_shopkeeper(victim)` / `!ok_damage_shopkeeper`): `"Ha ha... Don't think
   so."`.
Plus `do_hit` (act.offensive.c:101): self → `"You hit yourself...OUCH!"` + `$n hits $mself…` to room;
charm-friend (`master == vict`) → `"$N is just such a good friend, you simply can't hit $M."`.
Fix any string/order drift you find; the peaceful case is the oracle RED.

## 4. ⚠ Blast radius — validate, don't just fix-and-ship
The lookup fix changes behavior everywhere the name API reads a static flag (all currently dead):
- mob aggression peaceful (mobact.go:449), hunt avoidance (graph.go:100/148), skill peaceful
  (skill_combat.go:143), `isDark`/`isOutside` (other_helpers.go:117/122), ai.go death/no_mob,
  limits_gain regenroom, other_utility no_recall/bfr/death/tunnel.
Audit each: turning these ON is the *intended* fix, but confirm none now double-fires with a
bit-based path that already worked (e.g. `isOutside` vs the bit-based `World.IsOutside`; if the
name-based one was dead and something relied on its always-"outside" answer, that surfaces now).
Add unit tests: `RoomHasFlag(8162,"peaceful")==true`, a non-peaceful room ==false, a house-flag
(dynamic name) still matches, and `roomHasFlagBit` decimal-not-hex behavior is preserved.

## 5. Acceptance gate
1. **Oracle red→green:** `--scenario combat-round` → `hit cub` clean (`"This room just has such a
   peaceful, easy feeling..."`). Also confirm `--scenario character-view` stays fully clean (only the
   `time` DP-1162 residual) — the lookup fix must not regress score/where/who/consider.
2. **Unit tests:** name→bit resolution for peaceful/dark/death/indoors/etc.; dynamic house-name still
   matches; the combat gate ladder strings/order vs C (§3).
3. `make check-fmt vet` + `go test ./...` green; import guard green; no WS schema break.

## 6. Gotchas
- **Never touch the oracle** ([[darkpawns-oracle-proof-gate]]). 8162 is genuinely ROOM_PEACEFUL in C.
- **The combat gate code is already right** — resist rewriting it; fix the lookup underneath it.
- **Blast radius is the risk, not the gate.** Lighting up long-dead flag checks can surface *other*
  latent divergences (mob aggro, light). Surface them to Claude rather than silencing — some may be
  their own findings.
- **This is Part B-1 only** (deterministic gating). Combat *draw-order* parity (one_hit RNG) is
  Part B-2 — Claude is building that scenario separately; don't fold RNG draw work in here.
