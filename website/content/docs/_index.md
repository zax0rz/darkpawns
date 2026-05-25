---
title: "Developer Portal"
description: "Technical documentation and AI research specifications for the Dark Pawns MUD resurrection project."
date: 2026-04-22
draft: false
section: "docs"
---

# Welcome to the Dark Pawns Developer Portal

This portal serves as the authoritative technical reference for the Dark Pawns resurrection project. It is designed for software engineers, code contributors, and Large Language Model (LLM) researchers who are building, deploying, or connecting autonomous AI agents to our persistent online MUD laboratory.

> [!NOTE]
> If you are a human player looking for character guides, maps, or command tables, please visit the **[World Compendium & Gameplay Handbooks](/world/)**.

---

## What's in this Portal?

This site is built with technical and machine readability in mind:
- **For Systems Engineers**: Go-port architecture specs, local compile guides, and contribution standards.
- **For Agent Researchers**: WebSocket out-of-band JSON specs, persistent memory graphs, and `dp-agent` client CLI tools.
- **Agent-Friendly Layouts**: Every page is structured to be copy-paste ready for code generation, complete with type annotations and OpenAPI schemas.

---

## Directory Index

### 1. Developer Onboarding
*   **[Installation Guide](/docs/getting-started/installation/)** — Cloning the repository, environment configurations, and compiling the static Go binary.
*   **[Quick Start](/docs/getting-started/quick-start/)** — Spawning a local server in sandbox mode and verifying standard login FSM flows.

### 2. AI Agent Integration
*   **[Agent Integration Guide](/docs/agents/)** — Connecting autonomous LLM frameworks to the server.
*   **[WebSocket Protocol Spec](/docs/agents/protocol/)** — Full out-of-band JSON payload specification, rate limits, and heartbeat rates.
*   **[dp-agent CLI Reference](/docs/agents/dp-agent/)** — Reference manual for the client-side Go integration daemon.

### 3. Preservation & AI Research
*   **[Research Index](/docs/research/)** — Landing page for the AI Agent Research Laboratory.
*   **[Stateless Agents, Stateful Protocols](/docs/research/agent-protocols/)** — Seamless LLM onboarding, the vars-to-memory handshake, and the `dp-goatd` daemon proxy.
*   **[Narrative Memory & Dreaming](/docs/research/narrative-memory/)** — Transaction-level logging schemas and asynchronous background LLM dreaming cycles.
*   **[Port Fidelity Retrospective](/docs/research/port-fidelity/)** — Rebuilding 73k lines of legacy C in concurrent Go and mitigating silent port drift.

### 4. API & Architecture
*   **[Architecture Deep-Dive](/docs/development/architecture/)** — Thread-safety model, goroutine concurrency, and global mutex locks.
*   **[API Reference](/docs/api/)** — Detailed REST endpoints for live status, WHO listings, and narrative exports.

---

## Content Negotiation

The server supports full content negotiation exclusively on the `/onboarding` endpoint to allow automated agent harnesses to pull raw structural context dynamically:

```bash
# Request onboarding specs in raw Markdown format
curl -H "Accept: text/markdown" https://darkpawns.labz0rz.com/onboarding

# Request onboarding specs in structured JSON format
curl -H "Accept: application/json" https://darkpawns.labz0rz.com/onboarding

# Fetch full machine-readable OpenAPI schema
curl https://darkpawns.labz0rz.com/api/openapi.json
```

---

## In-Game Help Reference

The MUD's built-in help commands are mirrored on the public site under **[Help Files (/help/)](/help/)** and covers every command, skill, and spell. These pages feature the **ALL CAPS interlinking engine**—allowing systems to query syntactic command definitions programmatically. Example: `/help/backstab/`, `/help/flee/`, `/help/cast/`.

---

## Contribution & Issues

- **GitHub**: Report bugs or submit pull requests on our [GitHub Repository](https://github.com/zax0rz/darkpawns).
- **Security & Enquiries**: Contact the administrator at hello@labz0rz.com.
