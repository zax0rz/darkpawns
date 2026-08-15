.PHONY: build test run clean install monitoring-up monitoring-down monitoring-logs monitoring-restart privacy-up privacy-down privacy-logs privacy-build privacy-test test-all test-unit test-integration test-e2e test-performance test-security test-report hooks fmt check-fmt vet lint lint-fix test-parse reachability reachability-weekly scenario-coverage scenario-coverage-weekly

# Regenerate the port reachability report (C command table vs Go registry).
# Deterministic; output is dated by run date. See docs/port-reachability-map.md
# for the original manual methodology this mechanizes.
reachability:
	python3 scripts/gen_reachability.py

# Weekly snapshot: TSV + delta vs last week + JSONL time-series append.
# Exits non-zero if any command regressed from reachable to unreachable.
reachability-weekly:
	python3 scripts/reachability_weekly.py --commit

# Scenario coverage: which C commands the oracle suite exercises (static —
# probed != passing). Weekly form appends the JSONL time series.
scenario-coverage:
	python3 scripts/gen_scenario_coverage.py

scenario-coverage-weekly:
	python3 scripts/scenario_coverage_weekly.py --commit

# Default world directory — resolve relative to this Makefile so it works
# regardless of the checkout directory name.
WORLD_DIR ?= $(dir $(abspath $(lastword $(MAKEFILE_LIST))))lib

build:
	go build -o darkpawns ./cmd/server

test:
	go test -v ./...

test-all:
	./test.sh all

test-unit:
	./test.sh unit

test-integration:
	./test.sh integration

test-e2e:
	./test.sh e2e

test-performance:
	./test.sh performance

test-security:
	./test.sh security

test-report:
	./test.sh report

run: build
	./darkpawns -world $(WORLD_DIR)

parse: build
	./darkpawns -world $(WORLD_DIR) -parse-only

clean:
	rm -f darkpawns
	rm -f coverage.txt

install:
	go mod tidy
	go mod download

# Monitoring stack commands
monitoring-up:
	docker compose -f docker-compose.monitoring.yml up -d

monitoring-down:
	docker compose -f docker-compose.monitoring.yml down

monitoring-logs:
	docker compose -f docker-compose.monitoring.yml logs -f

monitoring-restart:
	docker compose -f docker-compose.monitoring.yml restart

# Privacy filter commands
privacy-up:
	docker compose -f docker-compose.yml -f docker-compose.privacy.yml up -d

privacy-down:
	docker compose -f docker-compose.yml -f docker-compose.privacy.yml down

privacy-logs:
	docker compose -f docker-compose.yml -f docker-compose.privacy.yml logs -f

privacy-build:
	docker build -f Dockerfile.privacy-filter -t darkpawns-privacy-filter .

privacy-test:
	PRIVACY_FILTER_URL=http://localhost:8001 go test -v ./pkg/privacy/...

# Combined commands
up-with-privacy: privacy-up

# Development helpers

# Install git hooks (pre-push runs gofumpt so CI's format check can't surprise
# you). Also installs gofumpt if it's missing. Run once per clone.
hooks:
	@command -v gofumpt >/dev/null 2>&1 || test -x "$$(go env GOPATH)/bin/gofumpt" || go install mvdan.cc/gofumpt@latest
	git config core.hooksPath .githooks
	@echo "git hooks enabled (core.hooksPath=.githooks); pre-push enforces gofumpt."

fmt:
	gofumpt -w .

check-fmt:
	@test -z "$$(gofumpt -l .)" || (echo "Files need gofumpt. Run: gofumpt -w ." && gofumpt -l . && exit 1)

vet:
	go vet ./...

lint: check-fmt vet
	golangci-lint run ./...

lint-fix:
	golangci-lint run --fix ./...

# Testing with actual world files
test-parse:
	go test -v ./pkg/parser

# DEPLOY_USER and DEPLOY_HOST have no defaults on purpose: the ifndef guards
# in deploy-site below must fire when they are unset, otherwise a bare
# `make deploy-site` silently targets a hardcoded host as root (DP-785).
#
# DEPLOY_PATH is the Hugo docroot Caddy serves from on prod. Caddy's fallback
# handler (`root * /srv/hugo/`) serves the built site; the Go binary's
# `-web /opt/darkpawns/web` is a separate legacy client, NOT what /play uses.
# The prod host needs `rsync` installed (apt-get install rsync).
DEPLOY_PATH ?= /srv/hugo/

# Website commands
.PHONY: parse-world-json build-site deploy-site new-post voice-lint test-voice-lint content-inventory check-content-inventory site-check

voice-lint:
	python3 website-astro/scripts/voice_lint.py

test-voice-lint:
	python3 -m unittest website-astro/scripts/test_voice_lint.py

content-inventory:
	python3 website-astro/scripts/content_inventory.py

check-content-inventory:
	python3 website-astro/scripts/content_inventory.py --check

site-check: voice-lint test-voice-lint check-content-inventory
	cd website-astro && npm run build

parse-world-json:
	python3 website/scripts/parse_world.py
	python3 website/scripts/parse_db.py
	python3 website/scripts/interlink_help.py
	python3 website/scripts/precompute_sphere.py

build-site: parse-world-json
	cd website && hugo --minify

# Create a dated news post from archetypes/news.md:
#   make new-post TITLE=my-headline
# Creates website/content/news/YYYY-MM-DD-my-headline.md (draft: true).
new-post:
	cd website && hugo new news/$$(date +%Y-%m-%d)-$(TITLE).md

deploy-site: build-site
ifndef DEPLOY_USER
	$(error DEPLOY_USER is not set)
endif
ifndef DEPLOY_HOST
	$(error DEPLOY_HOST is not set)
endif
	rsync -avz --delete website/public/ $(DEPLOY_USER)@$(DEPLOY_HOST):$(DEPLOY_PATH)

.DEFAULT_GOAL := build
