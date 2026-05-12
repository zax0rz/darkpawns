# Dependency & Supply Chain Audit — 2026-05-10

**Audit time:** 2026-05-10 05:00 ET
**Runtime:** Go 1.26.2 (darwin/arm64)
**Module:** go 1.25.0

---

## Summary

**2 critical, 2 high, 4 medium, 5 low**

---

## CRITICAL

### Go stdlib vulnerability — GO-2026-4971
panic in Dial and LookupPort when handling NUL byte on Windows in net
- **Fixed in:** go1.26.3
- **Found in:** go1.26.2
- **Traces:**
  - `pkg/db/narrative_memory.go:230` — sql.Rows.Next → net.Dialer.Dial
  - `pkg/telnet/listener.go:44` — net.Listen
- **Vector:** Windows-only but affected paths exist in the codebase.

### Go stdlib vulnerability — GO-2026-4918
infinite loop in HTTP/2 transport with bad SETTINGS_MAX_FRAME_SIZE
- **Fixed in:** go1.26.3
- **Found in:** go1.26.2
- **Traces:**
  - `pkg/agent/memory_hooks.go:104` — http.Client.Do
  - `pkg/privacy/client.go:104` — http.Client.Post
- **Vector:** attacker-controlled HTTP/2 server can hang goroutines.

---

## HIGH

### github.com/prometheus/client_golang v1.19.1 → v1.23.2
- **Gap:** 4 minor versions
- **Notes:** v1.20+ native histograms, v1.22+ HTTP/2 transport, v1.23+ bugfixes. Current release from May 2024.
- **Usage:** `pkg/metrics` — prometheus metrics registration

### github.com/lib/pq v1.10.9 → v1.12.3
- **Gap:** 2 minor versions
- **Notes:** v1.12 brings SCRAM-SHA-256, DSN improvements, TLS bugfixes. Current from April 2023.
- **Usage:** `pkg/db` — PostgreSQL driver

---

## MEDIUM

### google.golang.org/protobuf v1.34.2 → v1.36.11
- **Gap:** 2 major versions
- **Status:** indirect (through prometheus dependencies)
- **Notes:** v1.35+ marshaling changes, v1.36+ proto3 optional support. Current from June 2024.

### github.com/prometheus/common v0.55.0 → v0.67.5
- **Gap:** 12 minor versions
- **Status:** indirect (through prometheus/client_golang)
- **Notes:** v0.60+ version-info handling changes. Wide gap.

### github.com/prometheus/procfs v0.15.1 → v0.20.1
- **Gap:** 5 minor versions
- **Status:** indirect (through prometheus/client_golang)
- **Notes:** v0.17+ CgroupV2 support, net-class metrics.

### golang.org/x/net v0.52.0 → v0.54.0
- **Gap:** 2 minor versions
- **Status:** indirect (through go toolchain)
- **Notes:** v0.53+ HTTP/2 security fixes.

---

## LOW

### github.com/mattn/go-sqlite3 v1.14.42 → v1.14.44
- **Gap:** 2 patches
- **Usage:** `pkg/storage` — SQLite driver

### golang.org/x/crypto v0.50.0 → v0.51.0
- **Gap:** 1 minor version
- **Usage:** `pkg/session` — bcrypt

### golang.org/x/text v0.36.0 → v0.37.0
- **Gap:** 1 minor version
- **Usage:** `pkg/game` — cases.Title

### golang.org/x/sys v0.43.0 → v0.44.0
- **Gap:** 1 minor version
- **Status:** indirect (through sqlite3 and prometheus)

### github.com/prometheus/client_model v0.6.1 → v0.6.2
- **Gap:** 1 patch
- **Status:** indirect (through prometheus client_golang)

---

## CLEAN — at latest version

| Dependency | Version | Status |
|---|---|---|
| github.com/golang-jwt/jwt/v5 | v5.3.1 | ✓ latest |
| github.com/gorilla/websocket | v1.5.3 | ✓ latest |
| github.com/yuin/gopher-lua | v1.1.2 | ✓ latest |
| golang.org/x/time | v0.15.0 | ✓ latest |
| github.com/beorn7/perks | v1.0.1 | ✓ latest |
| github.com/cespare/xxhash/v2 | v2.3.0 | ✓ latest |
| github.com/munnerz/goautoneg | v0.0.0-20191010 | ✓ latest (pre-release only) |

---

## Module Integrity

- `go mod verify`: **all modules verified** ✓
- `go mod tidy -diff`: **no changes needed** ✓
- Replace directives: **none** ✓
- Unused dependencies in go.mod: **none** ✓
- go.sum entries match go.mod: **verified** ✓

---

## Recommendations

1. **Update Go runtime to 1.26.3** — two active stdlib vulnerabilities (GO-2026-4971, GO-2026-4918) affecting production code paths.
2. **Bump prometheus/client_golang to v1.23.2** — 4 minor versions behind, breaking change in v1.20 (deprecated metric constructors removed) requires a test pass.
3. **Bump lib/pq to v1.12.3** — low risk, additive changes (SCRAM-SHA-256, DSN improvements).
4. **Update go module directive to 1.26.x** — currently pinned at 1.25.0 while toolchain compiles with 1.26.2.
5. **Bulk update x/ packages**: crypto v0.51, text v0.37, net v0.54, sys v0.44 — all backward-compatible.
