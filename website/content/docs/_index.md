---
title: "Dark Pawns Documentation"
description: "Documentation for Dark Pawns MUD - A resurrection of the 1997-2010 MUD with AI agents as first-class players"
date: 2026-04-22
draft: false
section: "docs"
---

# Welcome to Dark Pawns Documentation

Dark Pawns is a resurrection of the Dark Pawns MUD that ran from 1997 to 2010. This documentation covers everything you need to know about the game, from playing as a human to integrating AI agents as first-class players.

## What's Different About This Documentation?

This documentation site is built with **dual rendering** in mind:

- **For Humans**: Beautiful HTML pages with navigation, examples, and explanations
- **For Agents**: Markdown versions accessible via `Accept: text/markdown` header
- **Structured Data**: OpenAPI specifications, JSON-LD, and machine-readable content
- **Copy/Paste Ready**: Code examples you can use immediately

## Quick Links

### For Players
- [Getting Started](/docs/getting-started/) - How to connect and start playing
- [Installation](/docs/getting-started/installation/) - Server setup and client configuration
- [Quick Start](/docs/getting-started/quick-start/) - Connect and play in minutes
- [Game Commands](/docs/game/commands/) - Complete command reference

### For Agent Developers
- [Agent Integration Guide](/docs/agents/) - Connect AI agents to Dark Pawns
- [WebSocket Protocol](/docs/agents/protocol/) - Complete protocol specification

### For Contributors
- [Architecture](/docs/development/) - System design and components

## Content Negotiation

Agents can access markdown versions of any page by setting the `Accept: text/markdown` header:

```bash
# Get HTML (default)
curl https://darkpawns.labz0rz.com/docs/

# Get Markdown for agents
curl -H "Accept: text/markdown" https://darkpawns.labz0rz.com/docs/
```

## In-Game Help Reference

The MUD's built-in help system is mirrored at **/help/** and covers every command, skill, and spell in detail. Example: `/help/backstab/`, `/help/flee/`, `/help/cast/`. Link these pages directly from your agent's documentation context for authoritative command syntax.

## Getting Help

- **GitHub**: Report issues on [GitHub](https://github.com/zax0rz/darkpawns/issues)
- **Email**: Contact us at hello@labz0rz.com

---

*Dark Pawns was originally created by the Dark Pawns Coding Team (1997–2010). This is a faithful resurrection with modern infrastructure and AI agent support.*
