---
title: "Run a Local Server"
description: "Build Dark Pawns, provide its world and database settings, and connect on a development machine."
section: "getting-started"
audience: "operator"
order: 10
sourcePath: "website-astro/src/content/docs/getting-started/local-server.md"
updated: 2026-08-14
draft: false
---

## Requirements

- Go at the version declared in [`go.mod`](https://github.com/zax0rz/darkpawns/blob/main/go.mod).
- PostgreSQL and a database URL. The current server requires `-db` or `DATABASE_URL` at startup, although it can continue without persistence if the configured database cannot be reached.
- The repository's `lib/` directory, which contains the original world files.

## Build

```bash
git clone https://github.com/zax0rz/darkpawns.git
cd darkpawns
go build -o server ./cmd/server
```

## Configure and start

Development mode can generate a temporary JWT secret. Provide a stable secret for anything longer-lived.

```bash
export ENVIRONMENT=development
export DATABASE_URL='postgres://postgres:postgres@localhost/darkpawns?sslmode=disable'
./server -world ./lib -port 4350 -telnet-port 7777
```

The HTTP and WebSocket server listens on `-port`; raw telnet uses `-telnet-port`. Pass `0` to disable the telnet listener.

## Connect

```bash
telnet localhost 7777
```

Structured clients connect to `ws://localhost:4350/ws`. See [Agent Protocol](/docs/agents/protocol/) for the JSON message contract.

## Verify a development checkout

```bash
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
```

The full C-oracle setup is covered in [Port Fidelity Workflow](/docs/research/port-fidelity/).
