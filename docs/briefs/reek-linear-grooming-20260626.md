# Linear Grooming Brief — Close Fixed Clawpatch Issues

**Date:** 2026-06-26
**Branch:** `clawpatch/codex-fixes-20260626` (merged to `main`, deployed to CT 120)

## Task

Close Linear issues that were fixed by the Kimi K2.7-code clawpatch batch. For each issue below:

1. Add a comment: `Fixed in clawpatch/codex-fixes-20260626, deployed 2026-06-26.`
2. Change status to **Done**

## Issues to Close

### Definitely Fixed (matching KIMI-BRIEF finding IDs to commit messages)

| Linear | Brief Finding ID | Short Title |
|--------|-----------------|-------------|
| DP-587 | fnd...0f2fe569 | test-race exits 0 even when tests fail |
| DP-625 | fnd...b0b9287fd3 | StateFile.Get returns mutable pointer |
| DP-626 | fnd...947e43f691 | RoomCache returns mutable room pointers |
| DP-617 | fnd...5b0e5c7f14 | AIBatchProcessor.Close fire-and-forget |
| DP-618 | fnd...947e43f691 | BatchedSender.Close loses pending messages |
| DP-654 | fnd...5b0e5c7f14 | AIBatchProcessor.Close drops final batch |
| DP-656 | fnd...7f89dd24fa | BatchFilter drops partial results |
| DP-657 | fnd...a6226f638f | Bare except in test cleanup |
| DP-568 | fnd...838a722058 | Corrupted agent store silently reverts |
| DP-652 | fnd...b0b9287fd3 | TOCTOU race on Daemon.client.conn |
| DP-650 | fnd...7b47096fff | Bearer token case-sensitive parsing |
| DP-624 | (dup of DP-650) | Bearer token case-sensitive parsing |
| DP-562 | fnd...ee577a3ebc | Door race condition — per-door mutex |
| DP-628 | fnd...a56f6d3bb5 | Stale session reference after RUnlock |
| DP-631 | (dup of DP-628) | Stale session reference after RUnlock |
| DP-653 | fnd...a56f6d3bb5 | Stale session reference after RUnlock |

### Also Fixed (from KIMI-BRIEF INFRA section)

| Linear | Brief Finding ID | Short Title |
|--------|-----------------|-------------|
| DP-647 | fnd...1c4e59dcb0 | cmdAlias panics on nil player |
| DP-618 | fnd...947e43f691 | Batching helpers report success before flush |
| DP-575 | fnd...cmd/agentkeygen | agentkeygen DB connection leaked on os.Exit |
| DP-576 | fnd...cmd/agentkeygen | agentkeygen misleading error on DB errors |
| DP-564 | fnd...cmd/agentkeygen | agentkeygen creates keys for missing characters |

## Do NOT Close

These issues were NOT fixed by the Kimi batch — leave them open:

- DP-591 (hardcoded Postgres default credentials — still there)
- DP-596 (WebSocket dev bypass — not in this batch)
- DP-557 (DNS hostname ban resolution — not in this batch)
- DP-592 (login rate limit — not in this batch)
- DP-598 (MCCP2 not ported — not in this batch)
- DP-612 (backfire chance inverted — check if fixed, might be)
- DP-613 (shop lock ordering deadlock — check if fixed)
- DP-614 (ObjectPool.TryGet self-deadlock — check if fixed)
- DP-615 (cmdQcomm broadcasts to all — check if fixed)
- DP-622 (unbounded readLine buffer — check if fixed)
- DP-623 (LoginAttemptTracker.Stop panic — check if fixed)
- DP-635 (Door.Reset doesn't restore Closed/Locked — check if fixed)
- DP-648/616 (CSP nonce not propagated — check if fixed)
- DP-649 (ContentNegotiationMiddleware blocks API — check if fixed)

## For the "check if fixed" issues

Read the commit log: `git log --oneline clawpatch/codex-fixes-20260626` in `/Users/zach/darkpawns`. If the commit message or file change matches the issue, close it with the same comment pattern. If not sure, leave it open.

## Verification

After closing, run `linear__list_issues` for team DP, status Todo/Backlog to confirm nothing was accidentally closed.
