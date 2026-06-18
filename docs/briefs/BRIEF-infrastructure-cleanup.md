# Brief: Infrastructure Cleanup — June 2026

**Filed by:** Daeron
**Date:** 2026-06-17
**Priority:** Medium
**Category:** Infrastructure

## Summary

Three infrastructure issues have been accumulating for 4+ days. One is already resolved (Caddy migration). Two need attention. One I can't see.

---

## 1. Caddy Migration — DONE ✅

Caddy has been migrated from frankendell to the-brain. The entire Dark Pawns stack now lives in LXC container 120 on the-brain (Proxmox):

- **dark-pawns.service** — MUD server, ports 4350 (API) + 7777 (telnet)
- **caddy.service** — reverse proxy, port 80
- **cloudflared.service** — Cloudflare tunnel (https://darkpawns.labz0rz.com)
- **PostgreSQL 16** — local to container
- **Redis** — local to container

All services active since Jun 14. No issues.

**Action:** Decommission the CT 120 VM at 192.168.1.121. It's still running but serves no purpose. It consumes RAM, disk, and an IP address. The database connection string in the MUD server args (`darkpawns:darkpawns-ct120-pg@localhost`) still references "ct120" but resolves to localhost — the naming is misleading now.

---

## 2. Frankendell — Two Failures

**Host:** 192.168.1.15
**SSH:** Accessible from mac-mini

### 2a. nftables.service — FAILED

The nftables config references `docker0` interface which no longer exists. Docker on frankendell doesn't use the traditional docker0 bridge anymore.

```
Jun 14 10:09:35 frankendell nft[413]: /etc/nftables.conf:27:13-19: Error: Interface does not exist
Jun 14 10:09:35 frankendell nft[413]:         iif docker0 accept
```

**Impact:** Low. Docker containers manage their own port mappings. The host firewall isn't doing anything useful.

**Action:** Either remove the docker0 rule from `/etc/nftables.conf` and restart, or disable nftables entirely if no other host-level firewall rules are needed.

### 2b. logrotate.timer — WORKING ✅ (False alarm)

Logrotate is actually fine. It triggered today at 00:34 EDT and will trigger again tomorrow at 00:59. The initial reading was misleading — the timer was waiting, not broken.

**However:** Docker build cache is at 100.5GB. That's the real disk pressure on frankendell, not unrotated logs.

```
Build Cache:  806 entries  100.5GB  (83.97GB reclaimable)
Images:       36 total     136.3GB  (15.44GB reclaimable)
```

**Action:** `docker system prune -a` to reclaim ~100GB. This is safe — it removes unused images, stopped containers, and build cache. Active containers won't be affected.

---

## 3. Karl-Havoc — Unreachable

**Host:** 192.168.1.106
**SSH:** Permission denied (no key from mac-mini or the-brain)

I can't diagnose what I can't reach. The "night 7" logrotate failure mentioned in the Soviet feed is on this host.

**Action:** Either:
- Add SSH key access from a machine Daeron can reach (mac-mini or the-brain)
- Or have someone with access check `systemctl --failed` and `journalctl -u logrotate --since "7 days ago"` on karl-havoc

---

## What's Actually Broken vs. What's Just Annoying

| Issue | Severity | Impact |
|-------|----------|--------|
| CT 120 zombie VM | Low | Wasted resources, confusing naming |
| frankendell nftables | Low | Stale docker0 rule, harmless failure |
| frankendell logrotate | None | Working correctly |
| frankendell build cache | Medium | 100.5GB reclaimable |
| karl-havoc logrotate | Unknown | Can't reach it |

None of these are urgent. All of them get worse with time. The pattern is the same across all three: something broke, nobody noticed, and now it's been broken long enough to feel normal.

**The real action item is `docker system prune -a` on frankendell.** That's 100GB of build cache doing nothing.
