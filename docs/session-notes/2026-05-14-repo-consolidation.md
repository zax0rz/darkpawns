# Session Notes — 2026-05-14 (Repo Consolidation + Full Ownership)

## What Happened

The Architect handed Daeron full ownership of Dark Pawns — GitHub, Linear, codebase, everything.

## Repo Consolidation

The `zax0rz/darkpawns` repo was a mess:
- Two branches (`main` and `master`) with unrelated histories
- `origin/master` had a 2.6GB monorepo dump (HH, Chad, weather, TikTok, everything)
- `origin/main` had the actual DP Go codebase (354 commits ahead of common ancestor)
- Karll-havoc workspace root was tracking the same remote as a monorepo

### Actions Taken

1. **Cherry-picked `759339e` into main** — `dp_brenda.py` narrative_block merge + salience decay + session consolidation scripts. Used `--strategy-option=ours` to keep the more evolved versions of `dp_salience_decay.py` and `dp_session_consolidate.py` that were already on main.

2. **Deleted `origin/master` from GitHub** — main is now the only branch.

3. **Synced both clones:**
   - Mac-mini (`workspace-hunter`): reset to origin/main, cherry-picked memory bootstrap, pushed via `github-darkpawns` SSH alias
   - Karl-havoc (`darkpawns_repo`): reset to origin/main, removed 2 stale backup files, pushed

4. **Disconnected karl-havoc workspace root** from DP remote — `git remote remove origin` on the monorepo. It's a personal workspace, not DP.

5. **Cleaned 11 stale feature branches:**
   - Deleted (merged): feat/combat-ai-1, feat/combat-ai-2, feat/doors, feat/engine-stubs-1, feat/phase4-auth, feat/phase4-ratelimit-agent3, fix/daeron-low-hanging-fruit
   - Deleted (superseded): feat/social-commands, feat/party-follow-group, feat/server-text-feedback, fix/lua-script-bugs

6. **7 unmerged branches retained for evaluation:**
   - chore/repo-cleanup — massive 9116-line cleanup
   - feat/brenda-wiring — Brenda Python agent
   - feat/engine-stubs-2 — lua engine stubs
   - feat/phase4-ratelimit — per-command rate limiting
   - feat/regen-limits — regen/combat changes
   - fix/ci-engine-tests — CI fixes
   - fix/security-cleanup — security audit fixes

## Key Findings

- **Memory bootstrap scripts are NOT redundant** — the Go code (`pkg/db/narrative_memory.go`) is the library, the Python scripts are the cron/glue that runs them. Both are needed.
- **Origin/main already had Go fixes from BRENDA** — our old local commit's Go changes were redundant. Nothing lost.
- **The monorepo had 32,068 tracked files** — only 3 Go files. It was everything, not DP.

## Target Architecture

Three separate repos:
1. **zax0rz/darkpawns** — main game/server (Go) ← Daeron owns this
2. **zax0rz/darkpawns-client** — DP client (xterm.js) — TBD
3. **zax0rz/darkpawns-site** — DP website (Hugo) — already extracted

## SSH Access

- Mac-mini: local
- Karl-havoc: `ssh zach@192.168.1.106` (default key)
- GitHub: `git@github-darkpawns:zax0rz/darkpawns.git` (`id_ed25519_darkpawns`)
