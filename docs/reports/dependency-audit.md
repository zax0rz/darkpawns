# Dependency Audit Log

Maintained by Daeron. Updated per Program 6 (monthly, 1st Monday).

## Audit History

### 2026-05-13 (ad-hoc — supply chain response)

**Trigger:** Mini Shai-Hulud supply chain attack (CVE-2026-45321). Architect-requested audit.

**Go dependencies updated (10 packages):**

| Package | Old | New | Notes |
|---|---|---|---|
| lib/pq | v1.10.9 | v1.12.3 | PostgreSQL driver |
| mattn/go-sqlite3 | v1.14.42 | v1.14.44 | SQLite driver |
| prometheus/client_golang | v1.19.1 | v1.23.2 | Metrics (major jump) |
| prometheus/client_model | v0.6.1 | v0.6.2 | Metrics model |
| prometheus/common | v0.55.0 | v0.67.5 | Metrics common |
| prometheus/procfs | v0.15.1 | v0.20.1 | Metrics procfs |
| golang.org/x/crypto | v0.50.0 | v0.51.0 | Crypto primitives |
| golang.org/x/sys | v0.43.0 | v0.44.0 | System calls |
| golang.org/x/text | v0.36.0 | v0.37.0 | Text processing |
| google.golang.org/protobuf | v1.34.2 | v1.36.11 | Protobuf (transitive) |

**Already at latest (no update needed):**

| Package | Version |
|---|---|
| golang-jwt/jwt/v5 | v5.3.1 |
| gorilla/websocket | v1.5.3 |
| yuin/gopher-lua | v1.1.2 |
| golang.org/x/time | v0.15.0 |
| beorn7/perks | v1.0.1 |
| cespare/xxhash/v2 | v2.3.0 |

**Python dependencies updated (requirements.txt floor bumps):**

| Package | Old Floor | Latest Available |
|---|---|---|
| openai | >=1.0.0 | 2.36.0 |
| anthropic | >=0.25.0 | 0.101.0 |
| litellm | >=1.0.0 | 1.83.14 |
| mem0ai | >=0.1.0 | 2.0.2 |
| requests | >=2.31.0 | 2.34.0 |
| websocket-client | >=1.6.0 | 1.9.0 |
| redis | >=5.0.0 | 7.4.0 |

**GitHub Actions updated (7 actions):**

| Action | Old | New |
|---|---|---|
| actions/checkout | v4 | v6 |
| actions/setup-go | v5 | v6 |
| actions/setup-python | v5 | v6 |
| docker/build-push-action | v5 | v7 |
| docker/login-action | v3 | v4 |
| docker/metadata-action | v5 | v6 |
| docker/setup-buildx-action | v3 | v4 |

**Dockerfile base images:**

| Image | Old | New |
|---|---|---|
| golang | 1.25-alpine | 1.26-alpine |

**Verification:** `go build ./... && go vet ./... && go test ./...` — all pass.

**Supply chain exposure:** None of the compromised packages (TanStack, Mistral AI, guardrails-ai, UiPath, OpenSearch) are in our dependency tree. `litellm` and `mem0ai` are unrelated, uncompromised packages.

**Actions taken:**
- CI workflow Go version bumped to match go.mod (1.26.3) — was the actual cause of 4 consecutive CI failures
- All dependencies updated to latest stable
- Node.js 20 deprecation warnings resolved by bumping to v6/v7 actions
- Dependency audit program (Program 6) added to AGENTS.md for monthly recurring audits
