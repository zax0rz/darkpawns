---
title: "Contributing & Verification"
description: "The required checks and fidelity rules for changing the Go port safely."
section: "server"
audience: "developer"
order: 20
sourcePath: "website-astro/src/content/docs/server/contributing.md"
updated: 2026-08-14
draft: false
---

## Read the governing rulebook

Before changing player-observable behavior, read [`docs/fidelity/RULEBOOK.md`](https://github.com/zax0rz/darkpawns/blob/main/docs/fidelity/RULEBOOK.md). Its five rules govern the port:

1. Player-facing bytes are law.
2. The command surface is part of the game.
3. Random draws and operation order must remain deterministic.
4. Do not invent behavior the C game did not have.
5. Verify the reachable call path and audit the whole class of repeated failures.

## Format and verify

Run the repository formatter and all four required checks before committing:

```bash
make fmt
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
```

Website-only changes still use the Astro build from `website-astro/`:

```bash
npm run build
```

## Work with the oracle

The `src/` tree and the separate C oracle are read-only ground truth. Never repair a mismatch by editing them. Add or extend a differential scenario, prove the Go output against the reachable C function, and cite the applicable rule number in the change record.
