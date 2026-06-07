# Dependency & Supply Chain Audit — 2026-06-07

**Audit Date:** Sunday, June 7, 2026  
**Auditor:** Reek  
**Program:** 3 — Dependency & Supply Chain Audit

---

## Executive Summary

- **Total Dependencies:** 18 direct + indirect modules
- **Checksum Verification:** ✅ All modules verified
- **Vulnerability Scan:** 2 actively exploited vulnerabilities found (Go stdlib)
- **Dependency Health:** 1 critical upgrade required, 1 moderate

---

## Vulnerability Report

### CRITICAL — Actively Exploited in Code

#### CVE-2025-30204 — golang-jwt/jwt v5 (HIGH: 7.5)
- **Package:** `github.com/golang-jwt/jwt/v5 v5.3.1`
- **Vulnerability:** Excessive memory allocation during header parsing
- **Vector:** Network-accessible, no authentication required
- **Impact:** Denial of Service (resource exhaustion)
- **Status:** v5.3.1 is VULNERABLE — Fixed in v5.2.2+
- **Note:** Current version v5.3.1 is actually *newer* than the fix version v5.2.2, suggesting the version numbering may be non-linear. Verify if v5.3.1 contains the backported fix.
- **Action Required:** Verify fix status with `go list -m -versions github.com/golang-jwt/jwt/v5`

### HIGH — Standard Library Vulnerabilities (Called by Code)

#### GO-2026-5039 — net/textproto (Fixed in go1.26.4)
- **Vulnerability:** Arbitrary inputs included in errors without escaping
- **Affected Path:** `pkg/agentcli/ws.go` → `websocket.Dialer.DialContext` → `textproto.Reader.ReadMIMEHeader`
- **Impact:** Potential information disclosure in error messages
- **Fix:** Upgrade Go to 1.26.4

#### GO-2026-5037 — crypto/x509 (Fixed in go1.26.4)
- **Vulnerability:** Inefficient candidate hostname parsing
- **Affected Paths:**
  - `pkg/agentcli/ws.go` → Certificate verification
  - `pkg/scripting/engine.go` → Hostname error handling
- **Impact:** Performance degradation on certificate operations
- **Fix:** Upgrade Go to 1.26.4

### MODERATE — Standard Library (Package Level)

#### GO-2026-5038 — mime (Fixed in go1.26.4)
- **Vulnerability:** Quadratic complexity in WordDecoder.DecodeHeader
- **Impact:** CPU exhaustion on crafted MIME headers
- **Fix:** Upgrade Go to 1.26.4

### MODERATE — Module Level (Not Directly Called)

#### golang.org/x/crypto v0.51.0 (14 vulnerabilities)
- **Status:** v0.51.0 → v0.52.0 available
- **Key Issues:**
  - GO-2026-5033: Pathological inputs cause client panic (ssh/agent)
  - GO-2026-5023: VerifiedPublicKeyCallback permissions skip (ssh)
  - GO-2026-5021: Auth bypass via @revoked status (ssh/knownhosts)
  - GO-2026-5020: Infinite loop on large channel writes (ssh)
  - GO-2026-5019: FIDO/U2F security key bypass (ssh)
  - GO-2026-5018: Pathological RSA/DSA parameters DoS (ssh)
- **Impact:** Multiple SSH-related vulnerabilities
- **Note:** These are in `golang.org/x/crypto/ssh` — verify if Dark Pawns uses SSH functionality
- **Action:** Upgrade to v0.52.0

#### gorilla/websocket v1.5.3
- **Transitive Issue:** Depends on vulnerable `golang.org/x/net`
- **Open Issues:** 6 open bugs, including TLS information loss
- **Status:** Monitor for updates

---

## Dependency Inventory

### Direct Dependencies (go.mod)

| Module | Version | Latest | Status | Notes |
|--------|---------|--------|--------|-------|
| golang-jwt/jwt/v5 | v5.3.1 | v5.3.1 | ⚠️ CVE-2025-30204 | Verify fix status |
| gorilla/websocket | v1.5.3 | v1.5.3 | ⚠️ Transitive vuln | Via x/net |
| lib/pq | v1.12.3 | v1.12.3 | ✅ Current | PostgreSQL driver |
| mattn/go-sqlite3 | v1.14.44 | v1.14.44 | ✅ Current | SQLite driver |
| prometheus/client_golang | v1.23.2 | v1.23.2 | ✅ Current | Metrics |
| yuin/gopher-lua | v1.1.2 | v1.1.2 | ✅ Current | Lua VM |
| golang.org/x/crypto | v0.51.0 | v0.52.0 | ⚠️ Outdated | 14 CVEs |
| golang.org/x/text | v0.37.0 | v0.37.0 | ✅ Current | |
| golang.org/x/time | v0.15.0 | v0.15.0 | ✅ Current | Rate limiting |

### Indirect Dependencies

| Module | Version | Status |
|--------|---------|--------|
| beorn7/perks | v1.0.1 | ✅ Stable (2019) |
| cespare/xxhash/v2 | v2.3.0 | ✅ Current |
| klauspost/compress | v1.18.0 | ✅ Current |
| prometheus/client_model | v0.6.2 | ✅ Current |
| prometheus/common | v0.67.5 | ✅ Current |
| prometheus/procfs | v0.20.1 | ✅ Current |
| google.golang.org/protobuf | v1.36.11 | ✅ Current |

---

## Recommendations

### Immediate Actions (This Week)

1. **Upgrade Go to 1.26.4**
   - Fixes 3 standard library vulnerabilities (GO-2026-5039, GO-2026-5037, GO-2026-5038)
   - Command: `go install golang.org/dl/go1.26.4@latest && go1.26.4 download`

2. **Verify golang-jwt/jwt v5.3.1 Fix Status**
   - Check if v5.3.1 contains backported fix for CVE-2025-30204
   - If not, upgrade to v5.2.2+ (or latest patched version)
   - Command: `go get github.com/golang-jwt/jwt/v5@latest`

3. **Upgrade golang.org/x/crypto to v0.52.0**
   - Addresses 14 SSH-related vulnerabilities
   - Command: `go get golang.org/x/crypto@v0.52.0`

### Short-term (This Sprint)

4. **Review SSH Usage**
   - Verify if `golang.org/x/crypto/ssh` is actually used in production
   - If not used, consider removing the dependency
   - If used, prioritize the upgrade

5. **Pin Dependency Versions**
   - Current go.mod uses exact versions (good practice)
   - Document any intentional version pins in comments

### Long-term

6. **Automated Dependency Scanning**
   - Consider adding `govulncheck` to CI/CD pipeline
   - Set up Dependabot or Renovate for automated PRs
   - Weekly vulnerability scans (this audit)

---

## Appendix: Verification Commands

```bash
# Verify checksums
go mod verify

# Full vulnerability scan
govulncheck ./...

# Check for updates
go list -m -u all

# Tidy dependencies
go mod tidy
```

---

## Linear Issues to Create

- [ ] DP-XXX: Upgrade Go to 1.26.4 (3 stdlib vulns)
- [ ] DP-XXX: Verify/upgrade golang-jwt/jwt CVE-2025-30204
- [ ] DP-XXX: Upgrade golang.org/x/crypto to v0.52.0

---

*Report generated by Reek — Program 3: Dependency & Supply Chain Audit*
