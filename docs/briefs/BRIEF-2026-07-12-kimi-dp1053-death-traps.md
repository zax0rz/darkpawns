# BRIEF 2026-07-12 — Kimi: DP-1053 death traps still don't kill (QA Q1)

**Executor:** Kimi k2.7-code. **Branch:** `fix/dp1053-death-traps` (fresh off current `main`, your own clone/worktree — never share a HEAD with another executor).
**One PR.** Claude read the authoritative C (`src/act.movement.c`) and every Go anchor below against `origin/main` @ `9613e2e`.
**Repo:** `git@github-darkpawns:zax0rz/darkpawns.git`
**Build gate:** `go build ./... && go vet ./... && go test ./...` — ALL THREE MUST PASS.

This is the one genuinely design-shaped fix in the QA batch. Read this whole brief before writing code. The correct respawn machinery already exists in `handlePlayerDeath` — you are building a *stripped-down* variant of it, not inventing a death system.

**Scope lock — you own exactly two files:**
- `pkg/game/world.go` (the DT check on the player-move path)
- `pkg/game/death.go` (add the new DT handler here, next to `handlePlayerDeath`)
- plus `pkg/game/batch_b_test.go` (the existing DT tests live here — rewrite them)

Do NOT touch `pkg/game/party.go`, `pkg/game/mobact.go`, `pkg/game/limits_exp.go`, `pkg/db/`, or `pkg/session/`. Do NOT modify any `src/*.c` file (reference only). Do NOT refactor `handlePlayerDeath` itself — add a sibling function.

---

## The bug

`pkg/game/world.go:966-976` — after a mortal moves into a `ROOM_DEATH` room:
```go
if result != nil && roomHasFlagBit(result.Flags, 1) && p.Level < LVL_IMMORT {
    slog.Info("death trap", "player", p.GetName(), "room", result.VNum)
    p.TakeDamage(p.GetHP() + 1)   // <-- BUG
    p.SendMessage("You have entered a death trap!\r\n")
    w.roomMessage(result.VNum, fmt.Sprintf("The sound of a death cry is heard as %s enters the room!\r\n", p.GetName()))
}
```
`TakeDamage(GetHP()+1)` was written assuming HP clamps at 0. Post-DP-1021 (F1), HP is allowed to go negative, so this leaves the player at **HP = −1, which is merely POS_STUNNED**. No death handler runs, nothing extracts the player — they lie stunned in the DT room and recover. The player-facing effect: death traps are harmless.

The existing test `TestMovePlayer_DeathTrapKillsMortal` (batch_b_test.go:97) only asserts `HP <= 0`, so it passes against a stunned-not-dead player. That assertion is wrong and must change (see Regression Tests).

---

## Authoritative C — `src/act.movement.c:288-301` (do_simple_move, DT block)

Claude read this directly:
```c
if ( (ROOM_FLAGGED(ch->in_room, ROOM_DEATH)) &&
     (GET_LEVEL(ch) < LVL_IMMORT || IS_NPC(ch))  )
{
    log_death_trap(ch);
    death_cry(ch);
    extract_char(ch);
    if (mount)
    {
        log_death_trap(mount);
        death_cry(mount);
        extract_char(mount);
    }
    return 0;
}
```
**What this means, precisely:**
1. **Gate:** fires for `GET_LEVEL(ch) < LVL_IMMORT` mortals **OR any NPC**. Immortals are exempt. (Go's current gate `p.Level < LVL_IMMORT` already matches the mortal half; the NPC half is discussed under "Mob path" below.)
2. `extract_char` is an **immediate, corpse-less, penalty-free** removal. There is **no XP loss, no CON loss, no corpse, no gold drop, no equipment scatter** — none of the `die()`/`die_with_killer()` penalty machinery runs. It is *not* a normal death. (`extract_char`, `src/handler.c`.)
3. **The mount dies too.** If the character was riding a mount, the mount gets the identical treatment (`log_death_trap`/`death_cry`/`extract_char`).
4. `log_death_trap` just writes a mudlog line (`src/utils.c:141`) — a `slog.Info` is the faithful Go equivalent.
5. `return 0` aborts the move — the character does not "stay" in the DT room; they're extracted out of it.

**Cite:** `src/act.movement.c:288-301` (do_simple_move); `src/utils.c:141` (log_death_trap); `src/handler.c` (extract_char — no penalty/corpse).

---

## The fix

### Part A — add a dedicated DT handler in `death.go`

Add next to `handlePlayerDeath`:
```go
// deathTrap performs an immortal-exempt ROOM_DEATH extraction, faithful to
// src/act.movement.c:288-301. Unlike handlePlayerDeath this is corpse-less and
// penalty-free: no XP loss, no CON loss, no gold drop, no equipment/inventory
// scatter — extract_char() removes the char and respawns them at the temple.
// The player keeps everything they carried. Called from MovePlayer, OUTSIDE w.mu.
func (w *World) deathTrap(player *Player) {
    // log_death_trap (src/utils.c:141) — mudlog line only.
    slog.Info("death trap", "player", player.GetName(), "room", player.GetRoom())

    // death_cry to the room the player is dying in.
    w.roomMessage(player.GetRoom(),
        fmt.Sprintf("The sound of a death cry is heard as %s enters the room!\r\n", player.GetName()))
    player.SendMessage("You have entered a death trap!\r\n")

    // If mounted, the mount dies too (C extracts it identically). At minimum the
    // player must be dismounted so they don't ride a ghost after respawn. See Part C.
    w.deathTrapMount(player)

    // Respawn tail — the SAME machinery handlePlayerDeath uses at death.go:550-564,
    // MINUS every penalty/corpse block. Copy exactly these lines and nothing else:
    player.SetRoom(LoginStartRoom(player))
    player.SetPosition(combat.PosStanding)
    player.Heal(9999)
    player.StopFighting()
    if player.IsAffected(affWerewolf) {
        player.SetAffect(affWerewolf, false)
    }
    player.SendMessage("\r\nYou feel your soul wrenched from your body...\r\n")
    player.SendMessage("\r\nYou awaken in the temple.\r\n\r\n")
}
```
**Do NOT copy** from `handlePlayerDeath`: the PK bookkeeping (433-454), the EXP loss (456-472), the CON loss (474-500), the inventory/equipment scatter (502-519), the gold-zeroing (521-523), or the corpse creation (532-540). Those are exactly what make a DT *not* a normal death.

**Concurrency note:** `handlePlayerDeath` guards against double-processing with the `player.dying` CAS latch + HP>0 short-circuit (DP-943, death.go:415-431). The DT path runs on the movement goroutine with `w.mu` already released (world.go:964). It is far less racy than combat death (a player in a DT room isn't normally in a combat tick), and it applies no penalties, so a double-fire is at worst a double-respawn. **You do not need the CAS latch here** — but if you add it for consistency, mirror the exact pattern (CAS true, `defer Store(false)`) and do not gate on HP>0 (the player's HP is whatever the move left it at, not necessarily ≤0). Simplest correct choice: no latch. State whichever you chose in the PR body.

### Part B — wire it into `world.go`

Replace the buggy block at world.go:966-976 with a call to the new handler (keep the immortal-exempt gate — it matches C's `GET_LEVEL(ch) < LVL_IMMORT`):
```go
if result != nil && roomHasFlagBit(result.Flags, 1) && p.Level < LVL_IMMORT {
    // Death trap — act.movement.c:288-301. Corpse-less, penalty-free extraction
    // to the temple; performed outside the world lock because roomMessage/
    // SendMessage/SetRoom acquire their own locks.
    w.deathTrap(p)
}
```
The gate must stay OUTSIDE `w.mu` (it already is — `w.mu.Unlock()` is at world.go:964). Keep the existing comment about why.

### Part C — the mount (`deathTrapMount`)

C kills the mount too. Implement a helper that:
1. If the player is not mounted (`!player.IsMounted()`), return immediately.
2. Find the mount mob in the player's current room (the mob whose `GetMountRider() == player.GetName()` — see the existing pattern in `combat_wire.go:189-203` `Dismount`).
3. Send the mount's death cry to the room, then **remove the mount mob from the world** (grep for an existing mob-extraction/removal helper — e.g. `RemoveMob`, `extractMob`, or how `ExtractPendingChars` handles mobs; use the established path, do not hand-roll map deletion that skips locks).
4. Clear the player's mount state: `player.SetAffect(affMounted, false)`, clear the mob's rider, `player.SetFollowing("")` — mirror `combat_wire.go`'s Dismount teardown.

**If you cannot find a safe, existing mob-removal helper**, do NOT invent one that risks a lock/consistency bug. Instead: dismount the player (step 4 only — this is mandatory so they don't ride a ghost), leave the mount mob alive in the DT room, and clearly document in a code comment + the PR body that "mount extraction is deferred; C also extracts the mount (act.movement.c:296-300) — follow-up if a mob-extract helper is added." Player dismount is required; mount *death* is best-effort this PR.

### Mob path (NPC death traps) — investigate, likely a no-op

C's gate includes `|| IS_NPC(ch)` — NPCs die in DT rooms too. **But** Go's `wanderMob` (`pkg/game/ai.go:143`) already *refuses to enter* `ROOM_DEATH` rooms, so a wandering mob can never be inside one. Before adding anything: grep for any OTHER path that moves a mob into an arbitrary room (charmed-follower movement, forced move, flee). 
- If **no** such path routes a mob into a DT room, add a one-line comment in `wanderMob` noting that the DT skip already subsumes C's NPC-DT death, and do nothing else. (Do not touch mobact.go — it's out of scope and another brief's territory.)
- If youdo find such a path **inside world.go or death.go** (your owned files), note it in the PR body and propose the fix — but do NOT expand scope into other files without flagging it first.

State your finding either way in the PR body.

---

## Regression Tests — `pkg/game/batch_b_test.go`

The existing `TestMovePlayer_DeathTrapImmortalSurvives` (line 127) should still pass unchanged (immortal stays in room 1002 with HP > 0) — verify it does.

**Rewrite `TestMovePlayer_DeathTrapKillsMortal` (line 97)** — its `HP <= 0` assertion is now wrong (a DT respawns the player to full HP). Rename to `TestMovePlayer_DeathTrapExtractsMortal` and assert the real DT outcome. After `MovePlayer(p, "north")` into the DT (room 1002):
- `p.GetRoom() == MortalStartRoom` (8004) — extracted to the temple, NOT left in 1002.
- `p.GetHP() > 0` — respawned and healed, not lying stunned. (Assert full HP if `GetMaxHP` is available.)
- `p.GetPosition() == combat.PosStanding`.
- The player's EXP is **unchanged** from before the move (penalty-free). Set `p.SetExp(50000)` before the move and assert it's still 50000 after.
- **No corpse** was created in room 1002 (assert `w.GetObjectsInRoom(1002)` / `w.roomItems[1002]` is empty — use whatever accessor the other tests in this file use).

**Add `TestMovePlayer_DeathTrapKeepsInventory`:** give the player an object in inventory, walk them into a DT, assert after respawn the item is **still in their inventory** (corpse-less — they didn't drop it). This is the load-bearing distinction from a normal death.

**Add `TestMovePlayer_DeathTrapMountDismounts` (if the mount API makes it feasible):** mount the player on a mob, walk into a DT, assert the player is no longer mounted (`!p.IsMounted()`) after respawn. If setting up a mount in a unit test is impractical, skip this test but say so in the PR body.

Note in the test world you'll want room 1002 flagged `ROOM_DEATH` (`Flags: []string{"2"}`, as the existing test already does). Room 8004 need not exist as a `parser.Room` for the vnum assertion to hold (`SetRoom` just sets the field), but adding it makes the test more realistic — your call.

---

## Execution order

1. Part A (`deathTrap` in death.go) + Part C (`deathTrapMount`).
2. Part B (wire into world.go, delete the buggy `TakeDamage` block).
3. Mob-path investigation (comment or note).
4. Rewrite + add tests.
5. Build gate.

## After all fixes

```bash
git checkout -b fix/dp1053-death-traps main
# ... implement ...
go build ./... && go vet ./... && go test ./...
git add -A
git commit -m "fix: death traps extract instead of stun (DP-1053)"
git push -u origin fix/dp1053-death-traps
gh pr create --title "fix: death traps extract instead of stun (DP-1053)" \
  --body "DT rooms left mortals stunned at -1 HP instead of extracting them (QA Q1). Adds a corpse-less, penalty-free extraction faithful to act.movement.c:288-301, including mount handling. See docs/briefs/BRIEF-2026-07-12-kimi-dp1053-death-traps.md. Fixes DP-1053."
```

Then STOP. Do not merge. Claude reviews against `origin/main`, runs the build gate, checks the C fidelity (esp. that NO penalty/corpse machinery leaked in), verifies the tests actually assert extraction, and merges.
