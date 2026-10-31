---
title: "Contribute to Dark Pawns"
description: "Read the fidelity rules, format your changes, and run the required build, static analysis, and test checks before committing."
type: guide
audience: developers
verified: v0.1.8
---

## Read the governing rulebook

Before changing player-observable behavior, read [`docs/fidelity/RULEBOOK.md`](https://github.com/zax0rz/darkpawns/blob/main/docs/fidelity/RULEBOOK.md). Its five rules govern the port:

1. Player-facing bytes are law.
2. The command surface is part of the game.
3. Random draws and operation order must remain deterministic.
4. Do not invent behavior the C game did not have.
5. Verify the reachable call path and audit the whole class of repeated failures.

Before editing, read the [development setup guide](https://github.com/zax0rz/darkpawns/blob/main/docs/DEV-SETUP.md) and the repository's [agent instructions](https://github.com/zax0rz/darkpawns/blob/main/AGENTS.md) for worktree, oracle, and evidence-handling rules.

## Format and verify

First format the repository:

```bash
make fmt
```

Then run the four required Go checks separately, and check each exit status:

```bash
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
```

Also run the dead-code ratchet before committing:

```bash
make deadcode-ratchet
```

If a change exports a
symbol used by `tools/db-migrate`, also run `make db-migrate-test`.

For website changes, run the site's content and build checks from the repository
root:

```bash
make site-check
```

## Work with the oracle

The `src/` tree and the separate C oracle are read-only ground truth. Never repair a mismatch by editing them. Add or extend a differential scenario, prove the Go output against the reachable C function, and cite the applicable rule number in the change record.
