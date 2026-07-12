# BRIEF 2026-07-12 — GLM: DP-1045 (partial) peaceful-room + low-level PC protection gates

**Executor:** GLM. **Branch:** `glm/dp1045-peaceful-lowlevel-gates` (fresh off
current `main`, your own clone/worktree — never share a HEAD). **One PR.** Claude
read the C source and every Go anchor below against `main` and made the placement
decisions for you. Implement exactly; do not roam.

## Scope — READ THIS FIRST

DP-1045 lists **five** `damage()` combat-entry gates. This brief covers **only the
two clean command-layer gates**:

1. **Peaceful room** (fight.c:1336-1342)
2. **Low-level PC protection** (fight.c:1344-1357)

The other three — **jail-guard intercept** (fight.c:1370-1401), **charmed-pet
retarget** (1410-1418), **high-level mob switcheroo** (1420-1440) — are **OUT OF
SCOPE**. They are mob-initiated combat-flow *redirects* (teleport-to-jail, hit-the-
master-instead, retarget-to-random-attacker) that need combat-engine retargeting,
not a command-layer guard. Claude will take those in a follow-up. **Do not attempt
them.** Leave DP-1045 open after your PR; note in the PR body "partial: peaceful +
low-level gates only; mob redirects (jail-guard/charm-retarget/switcheroo) still
open on DP-1045."

## Why the command layer (not the per-hit funnel)

In C these gates live in `damage()`, which is technically per-hit — but combat
effectively can't *start* in a peaceful room because the attack path returns
`FALSE` before any swing. In Go the faithful, non-spammy home is the **combat-
initiation command**: `cmdHit` in `pkg/session/combat_cmds.go`. That's the mortal
melee entry point, and `cmdKill` already delegates mortals to it (combat_cmds.go:62).
The skill path already has its own peaceful check (pkg/game/skill_combat.go:472);
mob-initiated aggro peaceful gating is handled separately (F14/DP-1034, done). So
`cmdHit` is the one remaining gap for player-initiated melee.

**Do NOT touch `StartCombat` (pkg/combat/engine.go)** — it's a shared funnel hit by
mob aggro, scripts, and arena code; adding refusal logic there risks side effects.
Keep the gates in `cmdHit`.

## Exact placement

In `pkg/session/combat_cmds.go`, function `cmdHit`, **after** target resolution
(the `if !found { … }` block, ~line 91-94) and **before** the `if tgt.Mob != nil`
branch (~line 96). Insert one block that runs for both target types.

## The C source (authoritative — Claude read this directly)

```c
/* peaceful rooms — fight.c:1336 */
if (!IS_OUTLAW(victim) && FIGHTING(victim) != ch)   /* outlaws get no protection */
   if (ch != victim && ROOM_FLAGGED(ch->in_room, ROOM_PEACEFUL)) {
      send_to_char("This room just has such a peaceful, easy feeling...\r\n", ch);
      return FALSE;
   }

/* low-level PC protection — fight.c:1344 (PC vs PC only) */
if (victim != ch) {
  if (!IS_NPC(ch) && !IS_NPC(victim) && GET_LEVEL(ch) <= 10) {
     act("You are not experienced enough to attack $N!", FALSE, ch, 0, victim, TO_CHAR);
     return FALSE;
  }
  if (!IS_NPC(ch) && !IS_NPC(victim) && GET_LEVEL(victim) <= 10 &&
      !PLR_FLAGGED(victim, PLR_OUTLAW)) {
     act("Ancient forces protect $N from your wrath!", FALSE, ch, 0, victim, TO_CHAR);
     return FALSE;
  }
}
```

Notes on the semantics you must preserve:
- **Peaceful applies to BOTH mob and player targets** (a mob victim `IS_OUTLAW` is
  always false → mobs are protected in peaceful rooms). Two exemptions: victim is an
  **outlaw** (players only), or the victim **is already fighting the attacker**
  (`FIGHTING(victim) == ch` → retaliation allowed).
- **Low-level protection is PC-vs-PC only** — applies only when the target is a
  player (`tgt.Player != nil`). Attacker level ≤ 10 blocks; victim level ≤ 10 blocks
  unless the victim is an outlaw. `GET_LEVEL <= 10` uses the attacker/victim
  character level.

## Go APIs to use (all confirmed present & reachable from `pkg/session`)

- Room peaceful: `s.manager.world.RoomHasFlag(s.player.GetRoom(), "peaceful")`
  — exported (`pkg/game/act_other_bridge.go:75`), already used in session at
  `cmd_inventory.go:19` (for "death").
- Outlaw flag: `game.PlrOutlaw` (bit index, `pkg/game/other_helpers.go:14`), tested as
  `tgt.Player.GetFlags()&(1<<uint(game.PlrOutlaw)) != 0`. Reachable — session already
  uses `game.PrfSummonable`, `game.PLR_INVISIBLE`, etc.
- Victim already-fighting-me: `tgt.Player.GetFighting() == s.player.Name` (players)
  or `tgt.Mob.GetFighting() == s.player.Name` (mobs). `GetFighting()` returns the
  target's current opponent name.
- Levels: `s.player.GetLevel()`, `tgt.Player.GetLevel()`, `tgt.Mob.GetLevel()`.
- Attacker is always a player here (`cmdHit` is a player command), so `IS_NPC(ch)` is
  false — the low-level gate only needs the `tgt.Player != nil` (victim-is-PC) test.

## Reference implementation (write it in the file's own idiom; this is the shape)

```go
// Resolve victim's combat/identity facts uniformly across mob/player targets.
victimFighting := ""
victimIsOutlaw := false
victimIsPlayer := tgt.Player != nil
victimLevel := 0
switch {
case tgt.Player != nil:
    victimFighting = tgt.Player.GetFighting()
    victimIsOutlaw = tgt.Player.GetFlags()&(1<<uint(game.PlrOutlaw)) != 0
    victimLevel = tgt.Player.GetLevel()
    if tgt.Player == s.player {
        s.Send("You hit yourself...OUCH!.\r\n") // or existing self-attack handling
        return nil
    }
case tgt.Mob != nil:
    victimFighting = tgt.Mob.GetFighting()
    victimLevel = tgt.Mob.GetLevel()
}

// Peaceful room — fight.c:1336. Outlaws and already-engaged retaliation exempt.
if !victimIsOutlaw && victimFighting != s.player.Name &&
    s.manager.world.RoomHasFlag(s.player.GetRoom(), "peaceful") {
    s.Send("This room just has such a peaceful, easy feeling...\r\n")
    return nil
}

// Low-level PC protection — fight.c:1344 (PC vs PC only).
if victimIsPlayer {
    if s.player.GetLevel() <= 10 {
        s.Send(fmt.Sprintf("You are not experienced enough to attack %s!\r\n", tgt.Player.Name))
        return nil
    }
    if victimLevel <= 10 && !victimIsOutlaw {
        s.Send(fmt.Sprintf("Ancient forces protect %s from your wrath!\r\n", tgt.Player.Name))
        return nil
    }
}
```

Adapt names to match the file (check the exact type/field names on `tgt`, `s.player`,
and whether a self-attack guard already exists earlier in `cmdHit` — if so, drop the
self-attack lines above and keep the `ch != victim` intent via the existing guard).
Confirm `game` is already imported in combat_cmds.go (it uses `LVL_IMPL`; verify the
import path/alias and reuse it).

## Tests (pkg/session — combat_cmds_test.go exists)

Follow the existing `combat_cmds_test.go` scaffolding for building a session + world
+ room + target. Add cases:
1. Peaceful room blocks attacking a mob → no combat started, "peaceful, easy feeling"
   message, mob not fighting.
2. Peaceful room blocks attacking a player (same).
3. Peaceful room does NOT block when the victim is already fighting the attacker
   (retaliation) — combat proceeds.
4. Peaceful room does NOT block when the victim player is an outlaw — combat proceeds.
5. Attacker level ≤ 10 vs a player → "not experienced enough", no combat.
6. Victim player level ≤ 10 (non-outlaw) → "Ancient forces protect", no combat.
7. Victim player level ≤ 10 AND outlaw → combat proceeds (protection waived).
8. Low-level gate does NOT fire vs a mob target (PC-vs-PC only): a level-5 player can
   still attack a mob.
9. Positive control: level-20 player attacks a mob in a normal room → combat starts
   (regression guard that the gates don't over-block).

Use whatever mechanism existing tests use to set room flags and player level; if a
peaceful-room helper isn't already in the test util, set the room's flag directly on
the parsed room the test world is built from.

## Definition of done

- Peaceful + low-level gates enforced in `cmdHit`; the three mob redirects untouched.
- `go build ./...`, `go test ./pkg/session/...`, `-race` on pkg/session, `gofumpt -l`
  (empty), `go vet ./pkg/session/...` all clean.
- Commit trailer `Co-Authored-By: <your executor id>`. PR body: "Partial DP-1045 —
  peaceful-room + low-level PC protection gates in cmdHit. Mob redirects (jail-guard,
  charm-retarget, high-level switcheroo) still open on DP-1045 (Claude follow-up)."

## Scope guard

Touch ONLY `pkg/session/combat_cmds.go` and `pkg/session/combat_cmds_test.go`. Do
NOT touch `pkg/combat/engine.go` (`StartCombat`), `pkg/game/death.go`,
`pkg/game/party.go`, `pkg/spells/`, or `pkg/game/damage_stubs.go` — a Claude task
(F2/DP-1022, the spell-kill pipeline) is live on the death/XP/spell files and will
conflict. Staying inside `combat_cmds.go` keeps every in-flight branch clean.
```
