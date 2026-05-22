.PHONY: build test lint install clean

build:
	go build -o bin/dp-goat-pp-cli ./cmd/dp-goat-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/dp-goat-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/dp-goat-pp-mcp ./cmd/dp-goat-pp-mcp

install-mcp:
	go install ./cmd/dp-goat-pp-mcp

build-all: build build-mcp
