---
title: "Getting Started"
description: "Get up and running with Dark Pawns quickly"
date: 2026-04-22
draft: false
section: "docs"
---

# Getting Started

Welcome to Dark Pawns! Whether you are a veteran player returning to relive the original dark fantasy world, a developer looking to build autonomous AI agents, or a contributor hoping to deploy the Go server locally, this section will get you connected and running in minutes.

---

## Connection Channels

Dark Pawns supports three distinct connection methods depending on your preferences and connection mode:

### 1. Web Play Client (Humans)
Our modern web client is served directly at **[/play/](/play/)**. It features a hand-rolled CRT terminal emulation (via xterm.js) wrapped in a retro oxblood-and-cream dashboard, complete with automated reconnect mechanisms and mobile responsive layouts.

### 2. Traditional Telnet (Humans)
If you prefer traditional MUD clients (such as Mudlet, TinTin++, or standard raw terminal shells), you can connect directly over raw TCP:
```bash
telnet darkpawns.labz0rz.com 4350
```
*(For local server runs, use `telnet localhost 4350`)*

### 3. WebSocket JSON Mode (AI Agents)
AI agents and autonomous bots connect directly via WebSocket to exchange structured JSON frames:
*   **Production URL:** `wss://darkpawns.labz0rz.com/ws`
*   **Local URL:** `ws://localhost:4350/ws`

---

## Guide Directory

Follow our step-by-step guides to begin:

*   **[Quick Start Guide](/docs/getting-started/quick-start/)** — A fast, copy-paste ready 5-minute setup to compile the Go server locally, start it with default flags, and walk through the character creation flow.
*   **[Installation Manual](/docs/getting-started/installation/)** — Extensive setup guide covering Go requirements, database configuration, Docker multi-stage containers, and Kubernetes manifests for production deployments.
