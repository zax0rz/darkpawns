# Dependency & Supply Chain Audit — 2026-05-17

**Auditor:** Reek
**Program:** 3 — Dependency & Supply Chain Audit (Sunday 5 AM)
**Go version:** 1.26.3
**Module:** github.com/zax0rz/darkpawns

---

## Verification Results

| Check | Status |
|---|---|
| `go mod verify` | ✅ PASS — all modules verified |
| `govulncheck ./...` | ✅ PASS — no vulnerabilities found |
| `go mod tidy -v` | ✅ PASS — no changes made, no unused deps |
| Replace directives | ✅ NONE — clean go.mod |

---

## Direct Dependencies (9)

| Dependency | Version | Date | Latest | Status | Used In |
|---|---|---|---|---|---|
| `golang-jwt/jwt/v5` | v5.3.1 | 2026-01-28 | v5.3.1 | ✅ Current | `pkg/auth/jwt.go` |
| `gorilla/websocket` | v1.5.3 | 2024-06-14 | v1.5.3 | ✅ Current | `pkg/session/`, `pkg/agentcli/` |
| `lib/pq` | v1.12.3 | 2026-04-03 | v1.12.3 | ✅ Current | `pkg/db/`, `pkg/storage/` |
| `mattn/go-sqlite3` | v1.14.44 | 2026-04-29 | v1.14.44 | ✅ Current | `pkg/storage/sqlite.go` |
| `prometheus/client_golang` | v1.23.2 | 2025-09-05 | v1.23.2 | ✅ Current | `pkg/metrics/metrics.go` |
| `yuin/gopher-lua` | v1.1.2 | 2026-04-01 | v1.1.2 | ✅ Current | `pkg/scripting/`, `pkg/events/` |
| `golang.org/x/crypto` | v0.51.0 | 2026-05-08 | v0.51.0 | ✅ Current | `pkg/auth/` |
| `golang.org/x/text` | v0.37.0 | 2026-05-08 | v0.37.0 | ✅ Current | `pkg/game/clan_admin.go` |
| `golang.org/x/time` | v0.15.0 | 2026-02-11 | v0.15.0 | ✅ Current | `pkg/auth/ratelimit.go` |

All 9 direct dependencies are at their latest stable releases.

---

## Indirect Dependencies (10)

| Dependency | Version | Role | Status |
|---|---|---|---|
| `beorn7/perks` | v1.0.1 (2019) | Prometheus: quantile buckets | ✅ Stable, no newer release |
| `cespare/xxhash/v2` | v2.3.0 (2024) | Prometheus: hash tables | ✅ Current |
| `kr/text` | v0.2.0 (2020) | yaml fork: test deps | ✅ Stable |
| `munnerz/goautoneg` | v0.0.0-20191010 (2019) | Prometheus: HTTP content negotiation | ✅ Stable, no releases |
| `prometheus/client_model` | v0.6.2 (2025) | Prometheus: data model | ✅ Current |
| `prometheus/common` | v0.67.5 (2026-01) | Prometheus: shared utils | ✅ Current |
| `prometheus/procfs` | v0.20.1 (2026-02) | Prometheus: /proc metrics | ✅ Current |
| `go.yaml.in/yaml/v2` | v2.4.3 (2025-09) | Prometheus yaml fork | ✅ Prometheus-standard |
| `golang.org/x/sys` | v0.44.0 (2026-04) | Platform syscalls | ✅ Current |
| `google.golang.org/protobuf` | v1.36.11 (2025-12) | Protobuf for metrics | ✅ Current |

---

## Findings

### MEDIUM-001: Prometheus dependency weight for minimal usage
- **File:** `go.mod` → `pkg/metrics/metrics.go` (177 lines)
- **What:** Prometheus client_golang pulls ~15 transitive deps into the build graph
- **Why it matters:** The metrics package is 177 lines with a handful of counters/gauges. The Prometheus dependency is large — it brings in protobuf, yaml fork, procfs, auto-negotiation, xxhash, and multiple Prometheus sub-libraries. If the metrics surface stays small, a lighter alternative (expvar, simple log-based metrics, or a hand-rolled /metrics endpoint) would reduce the module graph by 15+ entries and shrink build times.
- **Suggested fix:** Evaluate whether a bare `/debug/vars` (expvar) or minimal HTTP handler suffices. If not, keep Prometheus — it works, it's standard, and it's audited.
- **Linear:** MED-001

### LOW-001: Unmaintained indirect deps — beorn7/perks, munnerz/goautoneg
- **File:** `go.mod` (indirect require blocks)
- **What:** `beorn7/perks v1.0.1` and `munnerz/goautoneg` have no new releases since 2019
- **Why it matters:** These are tiny, stable libraries (~200 lines each) used by Prometheus internally. Risk is near-zero but they'd be flagged by automated supply chain scoring tools.
- **Suggested fix:** No action needed. If supply chain scanners complain, vendor the few lines directly.
- **Linear:** LOW-001

### LOW-002: Test-only transitive dependencies inflate go.sum
- **What:** `kylelemons/godebug`, `creack/pty`, `go.uber.org/goleak`, `stretchr/testify`, `davecgh/go-spew`, `pmezard/go-difflib`, `rogpeppe/go-internal`, `google/go-cmp`, `kr/pretty`, `check.v1` all appear but are only needed by `*_test.go` files in Prometheus or yaml fork tests
- **Why it matters:** These inflate `go.sum` and add ~2s to `go mod download`. They're harmless — they don't compile into production. But the sum file is larger than it needs to be.
- **Suggested fix:** No action. This is normal Go module behavior for test-heavy transitive deps.
- **Linear:** LOW-002

### LOW-003: go.yaml.in/yaml/v2 is a Prometheus-specific fork
- **File:** `go.mod` (indirect)
- **What:** `go.yaml.in/yaml/v2 v2.4.3` is the Prometheus fork of `gopkg.in/yaml.v2`, imported via prometheus/common
- **Why it matters:** This vanity import (`go.yaml.in`) is non-standard. Some Go tooling (private proxies, some linters) may not resolve it. It resolves correctly via `proxy.golang.org` but could fail in air-gapped environments or custom mirrors.
- **Suggested fix:** Ensure `GOPROXY=direct` or standard proxy is configured. If the project ever moves off Prometheus, this dep goes away.
- **Linear:** LOW-003

---

## Deep Dive — Sunday Packages

### `pkg/admin/`
No findings. Admin panel API is lean.

### `pkg/optimization/`
WebSocket pooling (`websocket.go`) and database optimization layer (`database.go`) — imports gorilla/websocket and lib/pq respectively. Clean.

### `pkg/moderation/`
Uses lib/pq for PostgreSQL. No anomalies.

### `pkg/validation/`
No external deps. Pure Go.

### `pkg/telnet/`
No external deps. Pure Go telnet protocol parser.

### `pkg/ai/`
No external deps. Pure Go.

## Summary

- **9 direct dependencies** — all at latest versions
- **10 indirect dependencies** — all current or stable
- **0 vulnerabilities** (govulncheck)
- **0 unused deps** (go mod tidy)
- **0 replace directives**
- **4 findings** (0 critical, 0 high, 1 medium, 3 low)

0 critical, 0 high, 1 medium, 3 low.
