# Brief: Makefile Cleanup — 2026-06-12

**Workspace:** `/Users/zach/.openclaw/workspace-daeron/darkpawns_repo`
**Repo:** `git@github-darkpawns:zax0rz/darkpawns.git` (branch: `main`)
**Build gate:** `go build ./... && go vet ./... && go test ./...` — ALL THREE MUST PASS.

---

## Fix 1: DP-571 — Replace `docker-compose` with `docker compose` (MEDIUM)

**File:** `Makefile` — 7 locations (monitoring-up, monitoring-down, monitoring-logs, monitoring-restart, privacy-up, privacy-down, privacy-logs)

**Problem:**
All `docker-compose` invocations use the legacy hyphenated standalone command. Modern Docker installs only have `docker compose` (space, v2). These targets will fail with "command not found" on current systems.

**Fix:**
Replace all 7 occurrences of `docker-compose` with `docker compose`:

```makefile
monitoring-up:
	docker compose -f docker-compose.monitoring.yml up -d

monitoring-down:
	docker compose -f docker-compose.monitoring.yml down

monitoring-logs:
	docker compose -f docker-compose.monitoring.yml logs -f

monitoring-restart:
	docker compose -f docker-compose.monitoring.yml restart

privacy-up:
	docker compose -f docker-compose.yml -f docker-compose.privacy.yml up -d

privacy-down:
	docker compose -f docker-compose.yml -f docker-compose.privacy.yml down

privacy-logs:
	docker compose -f docker-compose.yml -f docker-compose.privacy.yml logs -f
```

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Fix 2: DP-572 — Replace hardcoded deploy target (MEDIUM)

**File:** `Makefile` — `deploy-site` target (line ~111)

**Problem:**
The deploy-site target hardcodes `root@192.168.1.15` in the rsync command. This embeds infrastructure topology and a privileged credential in source control.

Current code:
```makefile
deploy-site: build-site
	rsync -avz --delete website/public/ root@192.168.1.15:/opt/darkpawns/hugo-site/
```

**Fix:**
Replace with variables and add validation:

```makefile
DEPLOY_USER ?= root
DEPLOY_HOST ?= 192.168.1.15

deploy-site: build-site
ifndef DEPLOY_USER
	$(error DEPLOY_USER is not set)
endif
ifndef DEPLOY_HOST
	$(error DEPLOY_HOST is not set)
endif
	rsync -avz --delete website/public/ $(DEPLOY_USER)@$(DEPLOY_HOST):/opt/darkpawns/hugo-site/
```

Note: `?=` means the defaults still work for local development, but the values can be overridden:
```bash
make deploy-site DEPLOY_USER=deploy DEPLOY_HOST=production.example.com
```

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Execution Order

Both fixes are in the same file. Apply together:

1. **Fix 1** (DP-571) — replace `docker-compose` with `docker compose` (7 occurrences)
2. **Fix 2** (DP-572) — add deploy variables + validation

## After All Fixes

```bash
cd /Users/zach/.openclaw/workspace-daeron/darkpawns_repo
go build ./... && go vet ./... && go test ./...
gofumpt -l .
git add -A
git commit -m "fix: Makefile — docker compose v2 + parameterized deploy target (DP-571, DP-572)"
git push -u origin fix/makefile-cleanup-2026-06-12
gh pr create --title "fix: Makefile cleanup (DP-571, DP-572)" --body "Fixes DP-571, DP-572. See docs/briefs/BRIEF-2026-06-12-makefile-cleanup.md for details."
```

Then wait for Daeron to review and merge. Do NOT merge the PR yourself.

## Linear Updates (after merge)

- DP-571: Add comment "Fixed — all 7 docker-compose invocations replaced with docker compose (v2)", commit <hash>, move to Done
- DP-572: Add comment "Fixed — deploy target now uses $(DEPLOY_USER)@$(DEPLOY_HOST) variables with validation", commit <hash>, move to Done
