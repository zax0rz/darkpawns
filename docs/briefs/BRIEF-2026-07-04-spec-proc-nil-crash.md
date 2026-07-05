# Brief: Fix spec proc nil pointer crashes during autonomous mob activity

## Problem

The server crash-loops on startup. `mobileActivityForMob` (mobact.go:141) calls all
registered spec procs with `nil` for the `ch *Player` parameter:

```go
specFn(w, nil, ch, "", "")
```

This is correct per the C port — during autonomous activity there is no target player.
But several spec procs call `ch.GetPosition()`, `ch.GetHP()`, `ch.GetFighting()`, etc.
without nil-checking `ch`, causing nil pointer dereference panics.

## Crashing spec procs (confirmed from production logs)

1. **specTakeToJail** — `spec_procs2.go:790` — calls `ch.GetPosition()` and `ch.GetHP()`
2. **specTroll** — `spec_procs3.go:581` — calls `ch.GetPosition()`, `ch.GetHP()`, `ch.GetFighting()`, plus `npcRegen(ch)` which accesses `ch.Health` directly
3. **specRescuer** — `spec_procs2.go:417` — calls `ch.GetPosition()`
4. **specNormalChecker** — `spec_procs2.go:135` — calls `ch.GetPosition()`

There may be others. Any spec proc that accesses `ch` before checking `cmd != ""` will crash.

## Fix (two parts)

### Part 1: Safety wrapper in mobact.go (prevents cascading crashes)

In `pkg/game/mobact.go`, wrap the spec dispatch in a `recover()`:

```go
// -- MOB_SPEC: special procedure dispatch --
if hasMobFlag(ch, "spec") {
    specFn := getMobVNumSpec(ch.Prototype.VNum)
    if specFn != nil {
        func() {
            defer func() {
                if r := recover(); r != nil {
                    // spec proc panicked (likely nil ch) — skip this tick
                }
            }()
            specFn(w, nil, ch, "", "")
        }()
    }
}
```

This is a safety net. It prevents one bad spec proc from crashing the entire server.
The recover should be silent (or log at DEBUG level) — these panics happen every tick
for every mob with a spec flag, so logging would be extremely noisy.

### Part 2: Fix individual spec procs

For each crashing spec proc, add nil guards on `ch`:

**Pattern for autonomous-only procs (specTakeToJail, specTroll, specRescuer, specNormalChecker):**

These procs are called during autonomous activity (ch=nil) AND during player commands (ch=player).
The fix: check `ch != nil` before accessing ch's methods.

For `specTakeToJail`:
- The proc arrests players in the room. `ch` is not used for the arrest logic — `pl` (from `GetPlayersInRoom`) is the target.
- Remove the `ch.GetPosition()` and `ch.GetHP()` checks (they check the wrong thing — should check `me` for the mob's state, not a nil player).
- The `sendToChar(ch, ...)` call sends "You're under arrest!" to nil — remove it (the room message is sufficient).

For `specTroll`:
- The proc regenerates the troll mob's health. `ch` is not involved.
- Replace all `ch.Get*()` with `me.Get*()` — the troll IS the mob, not a player.
- Replace `npcRegen(ch)` with inline regen on `me` (npcRegen takes `*Player`, but `me` is `*MobInstance`):
  ```go
  regenRate := 2
  newHP := me.GetHP() + me.GetLevel()*regenRate
  if newHP > me.GetMaxHP() {
      newHP = me.GetMaxHP()
  }
  me.SetHealth(newHP)
  ```

For `specRescuer`:
- Check what it does and add nil guard on ch.

For `specNormalChecker`:
- Check what it does and add nil guard on ch.

**Scan all spec procs** for the same pattern: any spec proc registered via `RegisterSpec`
that accesses `ch` without first checking `cmd != ""` or `ch != nil`. Fix them all.

## Verification

1. `go build ./... && go vet ./... && go test ./pkg/game/...`
2. Cross-compile: `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o darkpawns-server ./cmd/server`
3. Deploy to CT 120 and verify server stays up past the first game tick (check `journalctl -u dark-pawns.service` for panics)
4. Verify health endpoint: `curl -s https://darkpawns.labz0rz.com/health`

## Context

- Server is currently running on the OLD binary (rolled back). New binary is on CT 120 at `/opt/darkpawns/darkpawns-server.bak`.
- JWT_SECRET is already configured in the systemd unit (that part of the deploy worked).
- The nil `ch` is by design — it's how CircleMUD worked. The fix is defensive nil checks, not changing the dispatch.
- Do NOT change `mobileActivityForMob` to pass a non-nil value — that would break the contract.
