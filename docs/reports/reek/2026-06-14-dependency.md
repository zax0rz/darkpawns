# Dependency & Supply Chain Audit — 2026-06-14

**Program 3** | Sunday 6:00 AM ET

---

## Summary

- **Modules verified:** ✅ all modules verified (`go mod verify`)
- **Vulnerability scan:** ✅ no known vulnerabilities (`govulncheck ./...`)
- **Tidy check:** ✅ no unused modules (`go mod tidy`)
- **Replace directives:** None
- **Direct dependencies:** 9 | **Indirect dependencies:** 10
- **Total dependencies:** 19

---

## Direct Dependencies

| Module | Current | Latest | Status | Notes |
|--------|---------|--------|--------|-------|
| `github.com/golang-jwt/jwt/v5` | v5.3.1 | v5.3.1 | ✅ Current | Active, last push 2026-06-02 |
| `github.com/gorilla/websocket` | v1.5.3 | v1.5.3 | ✅ Current | Active, last push 2025-03-19 |
| `github.com/lib/pq` | v1.12.3 | v1.12.3 | ✅ Current | Active, last push 2026-05-12 |
| `github.com/mattn/go-sqlite3` | v1.14.44 | v1.14.45 | ⚠️ Update available | Active, last push 2026-06-09 |
| `github.com/prometheus/client_golang` | v1.23.2 | v1.23.2 | ✅ Current | Active, last push 2026-06-02 |
| `github.com/yuin/gopher-lua` | v1.1.2 | v1.1.2 | ✅ Current | Active, last push 2026-04-01 |
| `golang.org/x/crypto` | v0.53.0 | v0.53.0 | ✅ Current | Active, last push 2026-06-09 |
| `golang.org/x/text` | v0.38.0 | v0.38.0 | ✅ Current | Active, last push 2026-06-08 |
| `golang.org/x/time` | v0.15.0 | v0.15.0 | ✅ Current | Active, last push 2026-03-08 |

## Indirect Dependencies with Available Updates

| Module | Current | Latest | Notes |
|--------|---------|--------|-------|
| `github.com/chzyer/logex` | v1.1.10 | v1.2.1 | Transitive (via readline) |
| `github.com/chzyer/readline` | v0.0.0-20180603 | v1.5.1 | Transitive (via kingpin) |
| `github.com/chzyer/test` | v0.0.0-20180213 | v1.0.0 | Transitive (via readline) |
| `github.com/creack/pty` | v1.1.9 | v1.1.24 | Transitive (test dep) |
| `github.com/golang/protobuf` | v1.5.0 | v1.5.4 | Transitive (via prometheus) |
| `github.com/klauspost/compress` | v1.18.0 | v1.18.6 | Transitive (via prometheus) |
| `github.com/prometheus/common` | v0.67.5 | v0.68.1 | Transitive (via prometheus) |
| `github.com/rogpeppe/go-internal` | v1.10.0 | v1.15.0 | Transitive (test dep) |
| `go.yaml.in/yaml/v2` | v2.4.3 | v2.4.4 | Transitive (via prometheus) |
| `golang.org/x/mod` | v0.36.0 | v0.37.0 | Transitive (via tools) |
| `golang.org/x/net` | v0.55.0 | v0.56.0 | Transitive (via crypto) |
| `golang.org/x/oauth2` | v0.34.0 | v0.36.0 | Transitive (via prometheus) |
| `golang.org/x/tools` | v0.45.0 | v0.46.0 | Transitive (test dep) |

## Vulnerabilities

✅ **None found.** `govulncheck ./...` returned clean.

## Recommendations

### LOW — Update `go-sqlite3` (1 available)

`github.com/mattn/go-sqlite3` v1.14.44 → v1.14.45. Patch release. SQLite is a critical data layer component — recommend keeping current for bugfixes.

```bash
go get github.com/mattn/go-sqlite3@v1.14.45
go mod tidy
```

### LOW — Indirect dependency updates available

13 indirect dependencies have updates available. Most are transitive via `prometheus/client_golang` and test tooling. No security advisories on current versions. Upgrading is optional but recommended for general hygiene — `go get -u ./... && go mod tidy` would pull them all.

### INFO — `gorilla/websocket` last push 2025-03-19

Not archived, but the last commit was over a year ago. This is a mature/stable library (not abandoned), but worth noting. No action needed.

### INFO — No `replace` directives

Clean dependency graph. No fork patches or local overrides.

---

**Audit result:** PASS. No critical or high findings. One direct dependency update available (LOW). All checksums verified. Zero known vulnerabilities.
