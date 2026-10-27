# Command dispatch concurrency — the H6 decision

**Status:** accepted architectural divergence. **Date:** 2026-10-26.
**Origin:** the `BUG-R2-C2-A1-H6` finding in the 2026-10-26 OpenVuln triage
(`docs/triage/2026-10-26-openvuln-16-finding-triage.md`).

## The decision

**Command dispatch stays synchronous, exactly as C's `command_interpreter` is.**

A player's command can execute on two different goroutines: the session's own
readPump/heartbeat lane, and a *foreign* lane when another actor drives them —
`order` (a charmed follower), `force`, or a wizard `at`. The `order` path is the
clearest case: `cmdOrder` → `executeOrder` → `world.CommandExecFunc` →
`Manager.SetCommandExecFunc`'s closure → `ExecuteCommand(sess, command, nil)`
runs the ordered command **on the orderer's goroutine** while the victim's own
connection can be executing a command at the same time. Nothing serializes the
two.

We are **not** fixing that by serializing the dispatch layer. The fix belongs in
the individual handlers: each check-then-act sequence on a resource is made
atomic where the resource lives. This note records why, and the audit that
replaced the dispatch-layer fix.

## Why the dispatch layer is not the right place

Three blockers, each verified in the tree:

1. **`executeCommandRaw` recurses into itself for alias expansion**
   (`pkg/session/commands.go`, the `executeCommandRaw(s, nextCmd, nextArgs,
   false, "")` call inside the dispatch loop). A non-reentrant per-session
   execution mutex deadlocks on any alias, because the inner call is the same
   goroutine on the same session.

2. **`at <room> <command>` nests a command on the same session and depends on it
   having completed** (`pkg/session/wiz_movement.go`, `cmdAt`). It sets the
   wizard's room, calls `ExecuteCommand(s, ...)`, and then immediately tests
   `s.player.GetRoom() == dest` to decide whether to restore the original room.
   Deferring a contended nested call — which is what any "queue instead of
   execute" serialization would do — breaks that restore.

3. **"stay immediate" is deliberate and C-faithful** (`pkg/session/session_login.go`:
   *"Internal ExecuteCommand callers (order/force) bypass handleCommand and stay
   immediate"*). C's `do_order` calls `command_interpreter(vict, buf)` directly
   inside the orderer's command, bypassing the victim's descriptor input queue
   and their wait state. Enqueueing the ordered command onto the victim's drain
   lane would move it up to one game pulse later, which is a player-visible
   timing divergence — and the oracle harness observes pulse boundaries
   (`DP_CLOCK`), so it is an R1 risk, not a theoretical one.

Go has no goroutine identity, so a reentrant execution lock cannot distinguish
"the same goroutine re-entering" (blockers 1 and 2) from "another goroutine"
without threading a flag through every dispatch path.

## What we do instead

Handler-level atomicity: every place that **checks a resource and then spends
it** must do both in one step, under the lock that owns the resource. That is
the same shape as the economy fixes (`Player.SpendGold`/`AddGold`, the
inventory/equipment collection locks).

Audited and made atomic:

| Resource | Site | Fix |
|---|---|---|
| cast mana | `pkg/session/cast_cmds.go` | reserve at the sufficiency check (`TrySpendMana`), released on every non-spending exit |
| `turn` mana | `pkg/game/skills2.go` | `TrySpendMana` + clamped `SpendManaUpTo` |
| wand/staff charges | `pkg/game/other_economy.go` | `World.takeMagicItemCharge` under the world lock |
| drink dose | `pkg/game/item_consumable.go` | `claimDrinkDose` at the emptiness check + atomic `depleteDrink` |
| spell reagents | `pkg/game/spell_reagents.go`, `pkg/spells/affect_spells.go` | consume-once: the atomic remove decides the winner |
| practice sessions | `pkg/game/spec_procs.go` | reserve at the check, refund on the intervening refusals |
| clan treasury | `pkg/game/clans.go`, `clan_bank.go`, `clan_info.go` | `Clan.mu` + `SpendTreasure`/`AddTreasure`/`TreasureBalance` |
| donate/junk gold | `pkg/game/item_donate.go` | `SpendGold` at the check, refunded if the room move fails |
| coin steal | `pkg/game/skill_stealth.go` | atomic debit on the victim (`spendTargetGold`) + `AddGold` on the thief |

Rules that follow from this decision:

- **Never serialize the dispatch.** A new command that spends a resource must
  make its own check-and-spend atomic; it must not assume the lane is exclusive.
- **The world lock is the object-state lock.** `ObjectInstance` cannot carry a
  mutex (`house_save.go` copies instances by value, which `go vet` rejects for
  copied locks), so object-field read-modify-writes serialize on `w.mu`.
- **Reserve at the check, refund on the exits.** Where C checks early and spends
  late, the atomic form reserves early and releases on every path that does not
  complete the spend, so the byte flow is unchanged.

## Remaining class (audited, not yet converted)

NPC special costs read the balance, then do side-effecting work (spawn, `Act`,
target lookup), then debit: `spec_procs2.go` (stable/assassin/healer/…),
`spec_procs3.go`, `spec_procs4.go`, `clan_settings.go`, and `shop_npc.go`'s mob
customer path. Each needs a per-site reserve-and-refund, not a mechanical swap —
the debit cannot simply move to the check without paying for a failed spawn.
They are a known, listed class rather than a silently accepted one.
