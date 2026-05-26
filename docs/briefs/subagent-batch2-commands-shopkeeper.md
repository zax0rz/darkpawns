# Subagent Brief: Command Registration + Shopkeeper Protection (Batch 2)

**Objective:** Fix 3 fidelity issues by registering dead commands and adding a combat check.

**Working directory:** `/Users/zach/.openclaw/workspace-daeron/darkpawns_repo/`

**Before committing:** Run `go build ./... && go vet ./...` to verify.

---

## Fix 1: DP-359 — Register Missing Communication Channels

**File:** `pkg/session/commands.go`

Three public communication channels exist in the game layer (`pkg/game/comm_channel.go`) but are not registered in the session command registry. The game-layer functions are `doGenComm` with subcmd SCMD_AUCTION, SCMD_GRATZ, SCMD_NEWBIE.

**Step 1:** Check if wrapper functions exist for these channels. Look in `pkg/session/comm_cmds.go` for `cmdAuction`, `cmdGratz`, `cmdNewbie`, `cmdCTell`. If they don't exist, create them.

Each wrapper should follow the pattern of `cmdShout` or `cmdGossip` in `pkg/session/comm_cmds.go`. The pattern is:
```go
func cmdAuction(s *Session, args []string) error {
    // Same pattern as cmdGossip but calls the game layer with the auction subcmd
    // See cmdGossip at line ~195 for reference
}
```

**Step 2:** Register them in `pkg/session/commands.go` near the existing shout/gossip registrations (around line 67-270):

```go
cmdRegistry.Register("auction", wrapArgs(cmdAuction), "Auction an item to the channel.", 0, 0)
cmdRegistry.Register("gratz", wrapArgs(cmdGratz), "Congratulate someone on the channel.", 0, 0)
cmdRegistry.Register("newbie", wrapArgs(cmdNewbie), "Ask a question on the newbie channel.", 0, 0)
cmdRegistry.Register("ctell", wrapArgs(cmdCTell), "Send a message to your clan.", 0, 0)
```

**Note:** There is already a `cmdNewbie` registered at line 194 but it's a WIZARD command (`LVL_IMMORT`) for giving newbie equipment. That's a different function. The public channel version needs a different name, e.g. `cmdNewbieChannel`, or check if the game layer handles it differently.

**Important:** Look at how `cmdShout` and `cmdGossip` call into the game layer. The game-layer functions in `comm_channel.go` use a `subcmd` parameter to differentiate channels. You need to pass the correct subcmd constant.

Search for SCMD_AUCTION, SCMD_GRATZ, SCMD_NEWBIE in the codebase to find the constants.

---

## Fix 2: DP-364 — Register Infobar/Lines Commands

**File:** `pkg/session/commands.go`
**Functions:** `cmdLines` and `cmdInfoBar` in `pkg/session/display_cmds.go`

These are fully implemented but marked with `//lint:file-ignore U1000` and never registered.

**Step 1:** Remove the lint ignore comments from `display_cmds.go`.

**Step 2:** Register in `commands.go`:
```go
cmdRegistry.Register("lines", wrapArgs(cmdLines), "Set your screen line count (7-50).", 0, 0)
cmdRegistry.Register("infobar", wrapArgs(cmdInfoBar), "Toggle the bottom status infobar.", 0, 0)
```

---

## Fix 3: DP-345 — Shopkeeper Protection in Combat

**File:** `pkg/combat/engine.go` — `processCombatPair()`

In C (`fight.c:1360`), any attempt to damage a shopkeeper halts combat immediately.

**Fix:** Add a shopkeeper check at the start of `processCombatPair`, before damage is dealt. The check should verify if the target has the SHOP flag:

```go
// Shopkeeper protection — C: fight.c:1360
if hasShopFlag(victim) {
    stopFighting(ch)
    stopFighting(victim)
    return false
}
```

Look for how shop flags are checked elsewhere in the codebase (grep for `SHOP` or `shop` flags in mob constants). The exact flag name and check function may vary — search for existing shopkeeper protection patterns in `pkg/game/`.

If `stopFighting` doesn't exist as a function, look for the equivalent — it may be `RemoveFromCombat` or similar.
