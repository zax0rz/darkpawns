# Brief: Deploy Dark Pawns to Production

**Date:** 2026-05-27
**Requested by:** The Architect
**Execute via:** Claude Code or Gemini (Antigravity)

## What Changed Since Last Deploy

Two commits since the server was last updated (May 25):

1. **`fb86252`** — Linting & formatting baseline (254 files, .golangci.yml, gofumpt, dead code removal)
2. **`a767d83`** — Test coverage expansion (6 new test files, 1,586 insertions)

**No behavior changes.** Both commits are code quality infrastructure — formatting, lint config, test files. The server binary's behavior is identical to what's currently running.

## Deployment Model

- **Server:** frankendell (192.168.1.15)
- **Container:** `darkpawns-server-1` (alpine:3.20)
- **Binary mount:** `/opt/darkpawns/darkpawns-server` → `/app/server` (bind mount, read-only)
- **No Docker rebuild needed** — just replace the binary and restart the container

## Execution Steps

### Step 1: Build the binary

```bash
cd /Users/zach/.openclaw/workspace-daeron/darkpawns_repo
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o darkpawns-server ./cmd/server
```

**Important:** Must cross-compile for Linux amd64 — the server runs in an Alpine container on frankendell (x86_64), not macOS arm64.

Verify it built:
```bash
ls -la darkpawns-server
```

### Step 2: Copy to server

```bash
scp darkpawns-server root@192.168.1.15:/tmp/
ssh root@192.168.1.15 "mv /tmp/darkpawns-server /opt/darkpawns/darkpawns-server && chmod +x /opt/darkpawns/darkpawns-server"
```

**Note:** SCP to `/tmp/` first, then `mv` into place. Direct SCP to `/opt/darkpawns/` can fail with permission issues due to the container bind mount.
```

### Step 3: Restart the container

```bash
ssh root@192.168.1.15 "docker restart darkpawns-server-1"
```

### Step 4: Verify it's running

```bash
ssh root@192.168.1.15 "docker ps --format '{{.Names}} {{.Status}}' | grep darkpawns"
```

Expected: `darkpawns-server-1 Up X seconds`

### Step 5: Check the logs

```bash
ssh root@192.168.1.15 "docker logs darkpawns-server-1 --tail 30 2>&1"
```

Look for:
- "World loaded" or similar startup message
- No panics or errors
- Listening on port 4350

### Step 6: Health check

```bash
curl -s https://darkpawns.labz0rz.com/health | head -5
```

Should return a health response (not a connection error).

## Rollback

If anything goes wrong:

```bash
ssh root@192.168.1.15 "cp /opt/darkpawns/darkpawns-server.bak /opt/darkpawns/darkpawns-server && docker restart darkpawns-server-1"
```

The `.bak` file is the previous known-good binary.

## Risk Assessment

**Risk: Near-zero.** These commits change no game logic. They add a linter config, reformat code (whitespace-only changes), remove dead code (confirmed unused), and add test files. The compiled binary behavior is identical.

The only risk is if the dead code removal accidentally removed something that was actually used via string lookup — but we verified `cmdSocial` was not registered, and all other removed functions were confirmed unused by the linter.

## Commit to Deploy

```
a767d83 test: expand coverage — core game logic, spells, session, command registry, db conversion
fb86252 chore: establish linting and formatting baseline
```

Deploy from `main` at `a767d83`.
