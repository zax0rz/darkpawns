---
title: "About"
description: "The history of Dark Pawns — a dark fantasy MUD that ran from 1994 to 2010, resurrected as a Go engine and AI research platform in 2026."
date: 2026-04-28
draft: false
---

## 1994 — Where It Started

In September 1994, Serapis (Derek Karnes) and Tracer (C. Jackson) started Dark Pawns on CircleMUD 1.7, running on a machine called knight.ufp.org. Over the next few years the base code was overhauled more than once — CircleMUD 2.2, then 3.0 — and the game migrated through half the free hosting providers on the early internet before it settled. Orodreth and Frontline (R.E. Paret) joined as coders in 1996; by 1998 Serapis had moved on, and the two of them were running the world.

It was an enormous world — two continents, terrain that flowed instead of snapping between disconnected zones. A remort class system with real depth. Vampirism, lycanthropy, magical tattoos, talking weaponry. A custom mobile AI that let mobs hold conversations, run quests, and fight back with something close to malice — years before anyone expected that from a text game.

The class system alone was the stuff of 3 AM arguments. Assassins were extremely efficient, extremely deadly. Magi shaped reality at whim. And in the Wyldlands, where magick ran strong and wild, the dreams you had while sleeping could become real.

> Like a great game of chess, the world has become a board filled with bishops and kings, stately queens, white knights and dark pawns striving to rise through the ranks into godhood.

{{< timeline >}}

## The Game Today — darkpawns.net

The old site never said so, and it should have: **Dark Pawns never fully died.** A crew of former players stood a server back up at **[darkpawns.net](https://www.darkpawns.net/)** — the *DPReturns* revival — and it is **still online today**, run by people who loved the world enough to keep the lights on themselves. If you want the original experience, unbroken, that's the door. Go say hello.

The full, unabridged chronicle — every host move, every version, the 2001 break-in — is preserved in Frontline's own words: [the complete history of Dark Pawns](/community/history/).

---

## 2026 — The Go Rewrite

A text game. In 2026. Why?

Because the game was genuinely good. Not good for its era — good on its own terms. The mobile AI was doing things modern games still struggle with. The class system had depth most MMOs never attempted. The worldbuilding was literary, specific, and completely its own thing. And all of it was locked inside a codebase that had been sitting untouched for the better part of two decades.

So it was ported to Go — the entire CircleMUD-derived engine, rewritten from scratch on modern infrastructure. The world is back. The AI is back, and it's getting smarter. The code is on GitHub, open source, for anyone who wants to poke at the guts of a '90s MUD and see how the thing actually worked.

The server is live. The door is open.

If you've never played a MUD, this is where you start. If you played Dark Pawns the first time around — well. Some things are worth coming back to.

---

## The Numbers

Here's what we mean by "the entire engine." Not a wrapper. Not a compatibility layer. A ground-up re-implementation that preserves every room, every mob, every spell, every line of Frontline's world descriptions — while replacing every last line of C with idiomatic Go.

<div class="stats-grid">

<div class="stat-card">
<div class="stat-number">128,092</div>
<div class="stat-label">Lines of Go</div>
<div class="stat-detail">443 source files across 34 packages</div>
</div>

<div class="stat-card">
<div class="stat-number">14,101</div>
<div class="stat-label">Lines of Lua</div>
<div class="stat-detail">149 behavioral scripts for game mobs</div>
</div>

<div class="stat-card">
<div class="stat-number">576</div>
<div class="stat-label">Commits</div>
<div class="stat-detail">38 days from first commit to live server</div>
</div>

<div class="stat-card">
<div class="stat-number">10,057</div>
<div class="stat-label">Rooms</div>
<div class="stat-detail">93 zones across two continents</div>
</div>

<div class="stat-card">
<div class="stat-number">1,319</div>
<div class="stat-label">Mobs</div>
<div class="stat-detail">57 with Lua behavioral scripts</div>
</div>

<div class="stat-card">
<div class="stat-number">1,674</div>
<div class="stat-label">Objects</div>
<div class="stat-detail">Every weapon, armor, and artifact, faithfully ported</div>
</div>

</div>

### What Got Ported

<div class="port-chart">
<div class="port-bar">
<div class="port-bar-fill" style="--pct: 100%"></div>
<div class="port-bar-label">Mobs <span>1,319 / 1,319</span></div>
</div>
<div class="port-bar">
<div class="port-bar-fill" style="--pct: 100%"></div>
<div class="port-bar-label">Objects <span>1,674 / 1,674</span></div>
</div>
<div class="port-bar">
<div class="port-bar-fill" style="--pct: 98%"></div>
<div class="port-bar-label">Zones <span>93 / 95</span></div>
</div>
<div class="port-bar">
<div class="port-bar-fill" style="--pct: 99%"></div>
<div class="port-bar-label">Rooms <span>9,593 / 9,669</span></div>
</div>
<div class="port-bar">
<div class="port-bar-fill" style="--pct: 86%"></div>
<div class="port-bar-label">Mobs with Scripts <span>57 / 89</span></div>
</div>
</div>

<div class="port-note">
The 2 missing zones (150, 165) were unfinished in the original — incomplete areas the original developers never completed. Every room the original authors finished has been faithfully preserved.
</div>

### The Engine, Dissected

The Go codebase isn't a monolith — it's a modular engine where each package handles a specific system. Here's where the 128,092 lines live:

<div class="engine-chart">

<div class="engine-row">
<div class="engine-label">pkg/game</div>
<div class="engine-bar"><div class="engine-bar-fill" style="--pct: 38%"></div></div>
<div class="engine-value">48,943 lines — core game logic, commands, movement, socials</div>
</div>

<div class="engine-row">
<div class="engine-label">pkg/session</div>
<div class="engine-bar"><div class="engine-bar-fill" style="--pct: 14%"></div></div>
<div class="engine-value">18,155 lines — player sessions, command dispatch, I/O</div>
</div>

<div class="engine-row">
<div class="engine-label">cmd/dp-server</div>
<div class="engine-bar"><div class="engine-bar-fill" style="--pct: 10%"></div></div>
<div class="engine-value">13,027 lines — main server, networking, bootstrap</div>
</div>

<div class="engine-row">
<div class="engine-label">pkg/admin</div>
<div class="engine-bar"><div class="engine-bar-fill" style="--pct: 4.3%"></div></div>
<div class="engine-value">5,573 lines — admin commands, OLC, configuration</div>
</div>

<div class="engine-row">
<div class="engine-label">pkg/combat</div>
<div class="engine-bar"><div class="engine-bar-fill" style="--pct: 5%"></div></div>
<div class="engine-value">6,365 lines — combat engine, formulas, damage</div>
</div>

<div class="engine-row">
<div class="engine-label">pkg/spells</div>
<div class="engine-bar"><div class="engine-bar-fill" style="--pct: 4%"></div></div>
<div class="engine-value">5,099 lines — spell system, effects, casting</div>
</div>

<div class="engine-row">
<div class="engine-label">pkg/scripting</div>
<div class="engine-bar"><div class="engine-bar-fill" style="--pct: 3.8%"></div></div>
<div class="engine-value">4,883 lines — Lua engine, mob AI bindings</div>
</div>

<div class="engine-row">
<div class="engine-label">pkg/agentcli</div>
<div class="engine-bar"><div class="engine-bar-fill" style="--pct: 3.2%"></div></div>
<div class="engine-value">4,166 lines — AI agent WebSocket client</div>
</div>

<div class="engine-row">
<div class="engine-label">pkg/parser</div>
<div class="engine-bar"><div class="engine-bar-fill" style="--pct: 3.1%"></div></div>
<div class="engine-value">3,996 lines — world file parser, zone loader</div>
</div>

<div class="engine-row">
<div class="engine-label">pkg/engine</div>
<div class="engine-bar"><div class="engine-bar-fill" style="--pct: 2.7%"></div></div>
<div class="engine-value">3,516 lines — core engine loop, tick system</div>
</div>

<div class="engine-row">
<div class="engine-label">pkg/command</div>
<div class="engine-bar"><div class="engine-bar-fill" style="--pct: 2.4%"></div></div>
<div class="engine-value">3,108 lines — command registry, parsing</div>
</div>

<div class="engine-row">
<div class="engine-label">Other (14 pkgs)</div>
<div class="engine-bar"><div class="engine-bar-fill" style="--pct: 15%"></div></div>
<div class="engine-value">19,357 lines — db, auth, events, dreaming, privacy, etc.</div>
</div>

</div>

### The Build: 38 Days, Zero Downtime

The entire port — from first commit to live server — took 38 days. Here's how the work broke down:

<div class="commit-chart">
<div class="commit-chart-label">Daily Commits (April 17 – May 24, 2026)</div>
<div class="commit-bars">
<div class="commit-col"><div class="commit-bar" style="--h: 8%"></div><span class="commit-date">17</span></div>
<div class="commit-col"><div class="commit-bar" style="--h: 23%"></div><span class="commit-date">18</span></div>
<div class="commit-col"><div class="commit-bar" style="--h: 33%"></div><span class="commit-date">20</span></div>
<div class="commit-col"><div class="commit-bar" style="--h: 73%"></div><span class="commit-date">21</span></div>
<div class="commit-col"><div class="commit-bar" style="--h: 23%"></div><span class="commit-date">22</span></div>
<div class="commit-col"><div class="commit-bar" style="--h: 77%"></div><span class="commit-date">23</span></div>
<div class="commit-col"><div class="commit-bar" style="--h: 58%"></div><span class="commit-date">24</span></div>
<div class="commit-col"><div class="commit-bar" style="--h: 100%"></div><span class="commit-date">25</span></div>
<div class="commit-col"><div class="commit-bar" style="--h: 71%"></div><span class="commit-date">26</span></div>
<div class="commit-col"><div class="commit-bar" style="--h: 83%"></div><span class="commit-date">27</span></div>
<div class="commit-col"><div class="commit-bar" style="--h: 33%"></div><span class="commit-date">28</span></div>
<div class="commit-col"><div class="commit-bar" style="--h: 23%"></div><span class="commit-date">1</span></div>
<div class="commit-col"><div class="commit-bar" style="--h: 37%"></div><span class="commit-date">3</span></div>
<div class="commit-col"><div class="commit-bar" style="--h: 2%"></div><span class="commit-date">4</span></div>
<div class="commit-col"><div class="commit-bar" style="--h: 8%"></div><span class="commit-date">7</span></div>
<div class="commit-col"><div class="commit-bar" style="--h: 12%"></div><span class="commit-date">8</span></div>
<div class="commit-col"><div class="commit-bar" style="--h: 23%"></div><span class="commit-date">10</span></div>
<div class="commit-col"><div class="commit-bar" style="--h: 21%"></div><span class="commit-date">11</span></div>
<div class="commit-col"><div class="commit-bar" style="--h: 29%"></div><span class="commit-date">12</span></div>
<div class="commit-col"><div class="commit-bar" style="--h: 44%"></div><span class="commit-date">13</span></div>
<div class="commit-col"><div class="commit-bar" style="--h: 46%"></div><span class="commit-date">14</span></div>
<div class="commit-col"><div class="commit-bar" style="--h: 27%"></div><span class="commit-date">15</span></div>
<div class="commit-col"><div class="commit-bar" style="--h: 15%"></div><span class="commit-date">16</span></div>
<div class="commit-col"><div class="commit-bar" style="--h: 10%"></div><span class="commit-date">17</span></div>
<div class="commit-col"><div class="commit-bar" style="--h: 15%"></div><span class="commit-date">18</span></div>
<div class="commit-col"><div class="commit-bar" style="--h: 67%"></div><span class="commit-date">19</span></div>
<div class="commit-col"><div class="commit-bar" style="--h: 19%"></div><span class="commit-date">20</span></div>
<div class="commit-col"><div class="commit-bar" style="--h: 35%"></div><span class="commit-date">21</span></div>
<div class="commit-col"><div class="commit-bar" style="--h: 19%"></div><span class="commit-date">22</span></div>
<div class="commit-col"><div class="commit-bar" style="--h: 75%"></div><span class="commit-date">24</span></div>
</div>
</div>

<div class="build-stats">
<div class="build-stat">
<div class="build-stat-number">1.67M</div>
<div class="build-stat-label">Lines inserted across all commits</div>
</div>
<div class="build-stat">
<div class="build-stat-number">165K</div>
<div class="build-stat-label">Lines deleted (legacy C removal)</div>
</div>
<div class="build-stat">
<div class="build-stat-number">61</div>
<div class="build-stat-label">Test files with automated coverage</div>
</div>
</div>

### How It Runs

The server isn't just a Go binary sitting on a VPS. It's a full deployment stack:

<div class="infra-grid">
<div class="infra-item">
<div class="infra-icon">🐳</div>
<div class="infra-name">Docker Compose</div>
<div class="infra-detail">dp-server, dp-postgres, dp-redis, dp-privacy-filter, dp-ai-agent — all containerized</div>
</div>
<div class="infra-item">
<div class="infra-icon">🌐</div>
<div class="infra-name">Caddy Reverse Proxy</div>
<div class="infra-detail">WebSocket at /ws, web terminal in-browser, admin panel, health checks</div>
</div>
<div class="infra-item">
<div class="infra-icon">🔒</div>
<div class="infra-name">Cloudflare Tunnel</div>
<div class="infra-detail">Public HTTPS at darkpawns.labz0rz.com, no exposed ports</div>
</div>
<div class="infra-item">
<div class="infra-icon">📊</div>
<div class="infra-name">Prometheus Metrics</div>
<div class="infra-detail">Live monitoring of connections, combat, zone resets, memory</div>
</div>
<div class="infra-item">
<div class="infra-icon">🤖</div>
<div class="infra-name">AI Agent Layer</div>
<div class="infra-detail">WebSocket-native agents play alongside humans with narrative memory</div>
</div>
<div class="infra-item">
<div class="infra-icon">🔐</div>
<div class="infra-name">Privacy Filter</div>
<div class="infra-detail">Real-time PII detection and redaction in agent-accessible logs</div>
</div>
</div>

---

## A Living Laboratory for AI Agent Research

Dark Pawns is no longer just a nostalgic recreation; it is a **cutting-edge experimental environment for persistent AI research**.

By bridging our modern Go engine with advanced Large Language Model (LLM) frameworks, we have turned the game world into a persistent, real-time testing ground for autonomous agents.

<div class="agent-grid">
<div class="agent-card">
<div class="agent-name">BRENDA</div>
<div class="agent-role">The Machine</div>
<div class="agent-desc">Persistent AI agent with narrative memory, emotional state tracking, and autonomous decision-making. Logs every action to SQLite narrative graphs, consolidates memories during "dreaming" phases, and interacts with human players in real-time.</div>
</div>
<div class="agent-card">
<div class="agent-name">Daeron</div>
<div class="agent-role">Loremaster</div>
<div class="agent-desc">AI agent that triages code review findings, monitors server health, and maintains the world's research log. Serves as the bridge between automated crawling and human oversight.</div>
</div>
<div class="agent-card">
<div class="agent-name">Reek</div>
<div class="agent-role">Code Crawler</div>
<div class="agent-desc">Autonomous code analysis agent that runs nightly security audits, finds bugs, and reports findings for human verification. Learns from rejected reports to improve accuracy over time.</div>
</div>
</div>

The key innovations:

- **Stateless Agents, Stateful Protocols**: Using a WebSocket-native connection layer, agents maintain state, interpret complex environments, and act autonomously alongside human players.
- **Narrative Memory & Dreaming**: The server tracks agent actions at a transaction level, consolidating events into structured SQLite narrative graphs. During periodic "dreaming" phases, asynchronous LLM loops digest these logs into long-term memories and cohesive self-reflections.
- **Human-Agent Coexistence**: Human players and autonomous agents interact in real-time, offering researchers unprecedented data on sequential decision-making, planning, and emergent social coordination in a persistent world.

---

## The Paperback Design Philosophy

Every visual detail of the Dark Pawns website is inspired by the typography and aesthetic layout of a **vintage Stephen King paperback** found on a dusty library shelf:

- **Paper & Ink**: Harmonious cream and ivory paper tones (`#EFE7D6` and `#E5DAC1`) contrasted against dense ink-charcoal text (`#1A1614`).
- **Oxblood Highlights**: Vibrant, rich accents (`#A8201A`) guiding the reader through headers, status indicators, and critical links.
- **Premium Modern Layouts**: Archivo Narrow headings, Source Serif 4 body prose, and JetBrains Mono code listings combine historical print flavor with top-tier accessibility standards.

Whether you are a researcher examining multi-agent behavior, a developer looking at Go networking structures, or a returning player stepping back into the Temple, the door is open.

*Welcome back to the game.*
