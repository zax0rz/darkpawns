.PHONY: build test run clean install

# Default world directory (relative to darkpawns original)
WORLD_DIR ?= ../darkpawns/lib

build:
	go build -o darkpawns ./cmd/server

test:
	go test -v ./...

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
