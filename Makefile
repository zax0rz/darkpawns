.PHONY: build test run clean install monitoring-up monitoring-down monitoring-logs privacy-up privacy-down privacy-test test-all test-unit test-integration test-e2e test-performance test-security test-report

# Default world directory (relative to darkpawns original)
WORLD_DIR ?= ../darkpawns/lib

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
	docker-compose -f docker-compose.monitoring.yml up -d

monitoring-down:
	docker-compose -f docker-compose.monitoring.yml down

monitoring-logs:
	docker-compose -f docker-compose.monitoring.yml logs -f

monitoring-restart:
	docker-compose -f docker-compose.monitoring.yml restart

# Privacy filter commands
privacy-up:
	docker-compose -f docker-compose.yml -f docker-compose.privacy.yml up -d

privacy-down:
	docker-compose -f docker-compose.yml -f docker-compose.privacy.yml down

privacy-logs:
	docker-compose -f docker-compose.yml -f docker-compose.privacy.yml logs -f

privacy-build:
	docker build -f Dockerfile.privacy-filter -t darkpawns-privacy-filter .

privacy-test:
	PRIVACY_FILTER_URL=http://localhost:8001 go test -v ./pkg/privacy/...

# Combined commands
up-with-privacy: privacy-up

# Development helpers
fmt:
	go fmt ./...

vet:
	go vet ./...

lint: fmt vet
	golangci-lint run

# Testing with actual world files
test-parse:
	go test -v ./pkg/parser -world $(WORLD_DIR)

.DEFAULT_GOAL := build
