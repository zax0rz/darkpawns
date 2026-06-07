# Briefs — 2026-06-06 Batch

Four briefs covering 14 Linear issues (DP-533 through DP-552). Ready for Claude Code execution.

## Briefs

| Brief | Issues | Priority | What It Does |
|-------|--------|----------|-------------|
| `SECURITY-001-admin-login-lockout.md` | DP-547 | HIGH | Add brute-force lockout to admin login |
| `SECURITY-002-cors-origin-hardening.md` | DP-548, DP-549, DP-551 | MEDIUM | Fix CORS/origin validation, remove placeholders |
| `INFRA-001-k8s-secrets.md` | DP-550 | MEDIUM | Add JWT_SECRET + ENCRYPTION_KEY to k8s |
| `CLEANUP-001-dead-code-and-duplicates.md` | DP-542, DP-544, DP-552, DP-538, DP-537 | LOW | Dead code, cache headers, duplicate interfaces |
| `CODE-001-test-gaps-and-tooling.md` | DP-536, DP-535, DP-534, DP-533, DP-540, DP-539 | Mixed | Tests, portability, protocol, design |

## Execution Order

1. **SECURITY-001** — highest impact, isolated change
2. **SECURITY-002** — critical config fix (k8s ENVIRONMENT=production)
3. **INFRA-001** — requires manual k8s access (The Architect)
4. **CLEANUP-001** — safe low-risk cleanup
5. **CODE-001** — mixed bag, some need investigation

## Notes

- DP-536 (Affectable god-interface) flagged URGENT but is a design smell, not runtime bug. Recommend deferring to dedicated refactor.
- DP-534 (hardcoded paths) — file may not exist in current repo. Verify before attempting fix.
- DP-533 (division by zero) — file may be in scripts/. Verify location.
- INFRA-001 requires k8s cluster access — not for Claude Code, hand off to The Architect.
