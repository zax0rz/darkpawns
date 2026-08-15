---
title: "Persistent Agent Memory"
description: "The research question behind server-hosted narrative memory, separated from its implementation manual."
section: "research"
audience: "researcher"
order: 20
sourcePath: "website-astro/src/content/docs/research/agent-memory.md"
updated: 2026-08-14
draft: false
---

The implementation question is documented in [Narrative Memory](/docs/agents/memory/). The research question is narrower: can server-authored, emotionally weighted summaries help an autonomous player maintain goals and relationships across sessions without allowing the model to rewrite its own past?

## Why server-hosted

The game engine observes commands, combat, movement, and social events directly. That makes it a better recorder of what occurred than an agent reconstructing history from a lossy prompt transcript. Keeping the record server-side also makes memory portable across agent clients.

## What is implemented

- Structured session events.
- Event extraction and entity linking.
- Salience and valence calculations.
- Graph consolidation and pruning.
- Narrative summary generation and login delivery hooks.

## What remains a claim to test

The existence of those mechanisms does not prove improved planning, social consistency, survival, or context efficiency. Those outcomes require controlled comparisons, declared metrics, and reproducible data. Until those results exist, Docs describes the system as an experimental facility rather than a demonstrated breakthrough.
