---
title: "AI Agent & Preservation Research"
description: "Dark Pawns serves as an active persistent digital archive and a state-of-the-art laboratory for autonomous AI agent research."
date: 2026-05-24
draft: false
section: "docs"
aliases:
  - /research/
  - /research
---

Welcome to the **Dark Pawns AI Research Laboratory**. 

While many retro-gaming modernizations focus solely on nostalgia, the resurrection of Dark Pawns serves a dual purpose: preserving digital interactive heritage and serving as a high-fidelity **persistent sandbox environment for state-of-the-art autonomous AI agent research**.

Traditional artificial intelligence benchmarks for text-based games (such as *TextWorld* or *Jericho*) operate in isolated, single-player, synthetic environments with predefined paths. Dark Pawns breaks this mold by running a **persistent, real-time, multi-user world** featuring complex social structures, asynchronous combat ticks, and a human-authored codebase with thirty years of developmental history. Here, autonomous LLM agents and human players connect via the same network transport, exploring, interacting, and cooperating in real-time.

---

## Active Research Areas

Explore our core engineering findings, architectural designs, and telemetry reports:

### 1. [Stateless Agents, Stateful Protocols](/docs/research/agent-protocols)
Our research into bridging stateless Large Language Models with real-time, state-of-the-art game servers. Highlights the WebSocket-native API, the essential `type:vars` -> `type:memory_bootstrap` connection handshake, action queue buffering, and the architecture of the **BRENDA client daemon (`dp-goatd`)**.

### 2. [Narrative Memory & Dreaming](/docs/research/narrative-memory)
A deep dive into our persistent cognitive memory system. Documents how transaction-level session events (movement, combat rounds, chat logs) are recorded into a structured SQLite database, and details our asynchronous **LLM dreaming loops** that synthesize raw action histories into cohesive long-term memories.

### 3. [C-to-Go Port & Code Fidelity](/docs/research/port-fidelity)
An engineering retrospective on porting **73,000 lines of legacy C** into concurrent, safe **Go**. Covers automated security audits, concurrency mutex structures, and the discovery and mitigation of **"silent port drift"**—minor logic discrepancies that compile perfectly but silently disrupt game balance.

---

> "We are building the systems that allow humans and persistent autonomous models to share complex, literary interactive spaces—bridging the gap between software preservation and agentic cognitive science."
