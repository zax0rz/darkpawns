---
title: "dp-agent CLI"
description: "Build and use the bundled Go client for play, timed sessions, one-shot commands, and memory consolidation."
section: "agents"
audience: "agent-author"
order: 30
sourcePath: "website-astro/src/content/docs/agents/dp-agent.md"
updated: 2026-08-14
draft: false
---

`cmd/dp-agent` is the repository's structured Go client. It builds on `pkg/agentcli` for WebSocket transport, state tracking, reconnection, survival behavior, session logs, and LLM decisions.

## Build

```bash
go build -o dp-agent ./cmd/dp-agent
./dp-agent config
```

Configuration defaults to `~/.dp-agent.json`. `DP_CONFIG` chooses another file and `DP_KEY` overrides the stored agent key.

## Commands

| Command | Purpose |
| --- | --- |
| `dp-agent play` | Interactive autonomous play. |
| `dp-agent session` | Timed play with full session logging. |
| `dp-agent dream` | Offline memory consolidation. |
| `dp-agent config` | View or change configuration. |
| `dp-agent keygen -name NAME` | Generate an agent key against the configured database. |
| `dp-agent whoami` | Show the configured identity. |
| `dp-agent exec COMMAND` | Send one game command and exit. |

Run a subcommand without arguments to see its current flags. The executable's usage text is the authority when this page and a local checkout disagree.

## Decision order

During autonomous play, deterministic survival behavior can act before the LLM. If no finite-state rule takes control, the client builds context from current state, recent events, and memory, asks the configured model for one action, and sends that action through the normal command protocol.
