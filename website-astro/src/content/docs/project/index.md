---
title: "Project"
description: "Read the source, follow the C-to-Go fidelity rules, find research notes, learn how to contribute, and browse the Dark Pawns project history."
type: reference
audience: developers
verified: v0.1.7
---

[Dark Pawns on GitHub](https://github.com/zax0rz/darkpawns) is the source repository for the Go server, website, world files, and research record.

## How the port preserves the game

Dark Pawns is a 1:1 port of the original C game; the Go server doesn't reinterpret it. The [Port Rulebook](https://github.com/zax0rz/darkpawns/blob/main/docs/fidelity/RULEBOOK.md) defines the five rules for player-facing bytes, commands, random draws, behavior, and verification. [Port fidelity workflow](/docs/project/port-fidelity/) explains how the read-only C oracle is used to check changes.

## Code and research

- [Server architecture](/docs/project/architecture/) maps startup and the main Go packages.
- [Contributing](/docs/project/contributing/) covers the governing rules and checks for code changes.
- [Research notes](https://github.com/zax0rz/darkpawns/tree/main/docs/research) link the project README, dated evidence, and current research record.

## Project records

- [About Dark Pawns](/about/) explains the project.
- [Credits](/credits/) names the people and source projects behind it.
- [Project history](/archive/history/) links to the site's timeline.
