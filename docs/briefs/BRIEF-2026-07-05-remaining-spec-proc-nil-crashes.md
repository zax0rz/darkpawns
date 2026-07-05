# Brief: Fix remaining spec proc nil-ch crashes

## Context

PR #54 fixed 4 crashing spec procs (specTakeToJail, specTroll, specRescuer, specNormalChecker)
and added a recover() safety wrapper in mobact.go. The server is stable but ~12 mobs still
panic every tick (caught by recover, logged as errors).

The root cause is the same: `mobileActivityForMob` calls spec procs with `nil` for `ch *Player`.
Any spec proc that accesses `ch` without nil-checking will panic.

## Crashing mobs (from production logs)

| Mob | VNum | Spec proc (likely) |
|-----|------|-------------------|
| an intellect devourer | 14420 | specMindFlayer or similar |
| an elven sentinel | 19510 | specFighter |
| a nightbreed hunter | 7910 | specThief or specFighter |
| a monk | 19640 | specFighter |
| a watcher | 19650 | specGuildGuard or specCityguard |
| the Gatekeeper | 14401 | specGuildGuard |
| friar tuck | 19641 | specGuild or specGuildGuard |
| Habib the Slayer Genie | 19626 | specSummoner |
| Gil-Glash | 14405 | specThief or specFighter |
| a Bitter Wind | 19119 | specDragonBreath |
| Anastatia | 7900 | specMagicUser |
| the remorter | 4 | specGuild |

## Fix

For each spec proc, determine if it's called during autonomous activity (mobileActivityForMob)
and accesses `ch` without nil-checking. Add `ch != nil` guards where needed.

### Pattern 1: Combat procs (specFighter, specThief, specMagicUser, specPaladin)

These check `cmd != ""` first and return false — they only fire on player commands.
During autonomous activity, `cmd=""` so they return false immediately. These should be safe.

**Verify** by reading the code. If they DO access `ch` before the `cmd` check, add a nil guard.

### Pattern 2: Interaction procs (specGuild, specGuildGuard, specCityguard, specMayor)

These check `cmd` for specific commands. During autonomous activity, `cmd=""` so they
return false. Should be safe — but verify.

### Pattern 3: Autonomous behavior procs (specDragonBreath, specSummoner, specThief autonomous)

These fire on every tick regardless of `cmd`. They access `ch` and will crash.

**Fix:** Add nil guard at the top:
```go
if ch == nil {
    // Autonomous activity — no target player
    // Handle mob-only behavior here, or return false
    return false
}
```

Or, if the proc should do something during autonomous activity (like regen), use `me` instead of `ch`.

### Pattern 4: Unknown procs

Look up the mob's vnum in the world files to find which spec proc is assigned.
Check if it accesses `ch` before `cmd != ""`. Fix accordingly.

## Verification

1. `go build ./... && go vet ./... && go test ./pkg/game/...`
2. Cross-compile and deploy
3. Check logs: `journalctl -u dark-pawns.service | grep "spec proc panicked"` should return nothing
4. Health check: `curl -s https://darkpawns.labz0rz.com/health`

## Notes

- The recover() wrapper (PR #54) means the server won't crash even if we miss some.
  These fixes eliminate the error spam, not prevent crashes.
- Some mobs might have the same spec proc (e.g., multiple mobs using specFighter).
  Fixing the proc fixes all of them.
- The "intellect devourer" (14420) appears most frequently — likely a popular mob with a broken proc.
