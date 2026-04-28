# Web Client Research — `/play` Page

**Date:** 2026-04-28  
**Status:** Research complete, decisions pending  
**Context:** Dark Pawns website master plan, `/play` section — "casual demo" web client for playing in-browser

---

## 1. Existing Infrastructure

### Go Server WebSocket Endpoint
The Dark Pawns Go server **already exposes a native WebSocket endpoint** at `/ws`:

- **Framing:** JSON messages (not raw telnet) — `ClientMessage` struct, `json.Unmarshal`
- **Per-IP limit:** 5 connections (`pkg/session/manager.go:265`) — enough for 3 characters + margin
- **Session model:** One WebSocket = one player character. `Register(playerName, session)` — logging in a second character creates a new WebSocket session
- **Session takeover:** If same character logs in elsewhere, old session gets forcibly closed
- **Ping/pong:** 54-second keepalive, 60-second read deadline
- **Group system:** `follow`, `group`, `gtell`, `ungroup`, `split` commands already implemented
- **GMCP/MSDP:** Supported via telnet negotiation layer

### Existing Web Client (`darkpawns/web/`)
- `index.html`, `client.js`, `style.css` — functional, connects to Go server
- `middleware.go` — content negotiation (HTML/MD/JSON-LD)
- Status: Works but needs redesign to match website brief (cream chrome, dark terminal)

### Key Implication
**No proxy needed.** mud-web-proxy exists to bridge `wss://` → telnet for MUD servers that don't speak WebSocket. Dark Pawns speaks WebSocket natively. Eliminating the proxy reduces latency, deployment complexity, and failure surface.

---

## 2. Candidate Clients Evaluated

### 2.1 mud-web-client (maldorne) — ⭐ PRIMARY CANDIDATE
- **Repo:** https://github.com/maldorne/mud-web-client
- **Stack:** Vue 3 + xterm.js + TypeScript + Vite
- **License:** GPL-3.0 (check before forking)
- **Activity:** 152 commits, last commit **11 hours ago** (2026-04-27), full rewrite 2 weeks ago
- **Features:** Full telnet protocol support, GMCP/MSDP/MXP, ANSI colors, golden-layout panels, Docker, embed mode
- **Architecture:** `useSocket()` composable (one WebSocket per instance), `MudTerminal.vue` (one xterm.js instance), `App.vue` (orchestrates one connection)
- **Config:** Query parameter driven — `?proxy=ws://...&host=...&port=...&mode=embed`
- **Modes:** `embed` (simple terminal, iframe-compatible) and `full` (golden-layout with draggable panels)

**Pros:**
- Actively maintained by someone who actually runs a MUD (maldorne.org)
- Clean Vue 3 component architecture — composable, forkable
- Already has embed mode designed for iframe embedding in Hugo pages
- Protocol support is deep (GMCP bidirectional, MSDP, MXP, MCCP2)
- Docker-ready, production-deployed
- golden-layout integration for panel management

**Cons:**
- Single-connection architecture — one WebSocket per app instance (needs fork for multi-character)
- Designed to talk to mud-web-proxy (JSON connect message with `{host, port}`), not directly to a Go WebSocket server with its own JSON protocol
- GPL-3.0 license (need to verify compatibility)

### 2.2 webmud3 (unitopia-de) — SECONDARY CANDIDATE
- **Repo:** https://github.com/unitopia-de/webmud3
- **Stack:** React (frontend) + Node.js (backend proxy) + Socket.IO + TypeScript
- **Activity:** 699 commits, last commit Feb 2026, production-tested on UNItopia MUD
- **Architecture:** Frontend SPA + backend proxy that translates Socket.IO → telnet
- **Features:** MCCP2, MXP, telnet negotiation, Docker, env-configurable

**Pros:**
- Battle-tested at production scale
- React matches admin panel stack
- Has its own backend proxy with telnet translation

**Cons:**
- Socket.IO adds overhead vs raw WebSocket
- Requires Node.js backend proxy running alongside Go server
- React overkill for what's essentially a terminal emulator
- Heavier dependency footprint

### 2.3 Neuro (RanvierMUD) — SKIP
- **Repo:** https://github.com/RanvierMUD/neuro
- **Last commit:** Oct 2020 (dead)
- **Tightly coupled** to Ranvier MUD engine's specific WebSocket protocol
- Electron + web hybrid, not designed for website embedding

### 2.4 Muddler (c-hudson) — SKIP
- **Repo:** https://github.com/c-hudson/muddler
- Python desktop client with bolted-on web interface
- Desktop-first, not designed for website embedding
- Last update Feb 2026 (active) but wrong use case

### 2.5 Mudslinger (mudslinger.net) — SKIP
- External service, not self-hostable
- Requires websockify (noVNC) as proxy
- No open source code found

### 2.6 Existing `darkpawns/web/client.js` — FALLBACK
- Already works, already speaks Go server's JSON protocol
- Zero dependencies, vanilla JS
- **But:** No multi-character support, no protocol handling, no modern terminal rendering
- Reinventing the wheel vs. forking a mature project

---

## 3. The Multi-Character Problem

### 3.1 Player Behavior
Dark Pawns encourages party play — up to 3 characters per player. Zach's established play style is a **3-pane tiled grid** (one terminal per character, click to focus and type). This is identical to how Mudlet/MUSHclient handle multi-session on desktop.

### 3.2 What Desktop Clients Do
- **Mudlet:** Multiple profiles, each opens in its own tab/window. Ctrl+1/2/3 to switch.
- **MUSHclient:** Multiple windows, Ctrl+Tab to switch, can tile side-by-side.
- **TinTin++:** `#session` command creates named sessions, `#sessionname` sends commands without switching.
- **Wintin95:** Named sessions with `#stop`, `#drop`, `#roll` — send commands to any session from any window.

### 3.3 What Web MUD Clients Do
**Nothing.** No existing web MUD client supports multi-character tiled layout. They're all single-connection, single-terminal. This is a feature gap and an opportunity.

### 3.4 Technical Constraints

**xterm.js multi-instance performance:**
- Each xterm.js instance renders to its own `<canvas>` element
- Multiple instances share the main JS thread — they compete for render time
- xterm.js issue #3368 documents this: "2 instances → ~30fps each, 4 → ~15fps"
- For 3 instances with MUD-speed output (not video), this should be fine — MUDs output text slowly compared to terminal benchmarks
- The FitAddon handles resize per-instance, and ResizeObserver can refit on container change
- DOMTerm (another web terminal) already uses golden-layout for tiles and tabs with xterm.js — proven approach

**WebSocket multi-connection:**
- Go server allows 5 per IP — 3 character sessions is well within limits
- Each connection is independent — no shared state between WebSockets on the server
- Browser WebSocket limit: Chrome allows ~30 per origin, Firefox ~200 — 3 is trivial
- Connection lifecycle is already isolated in `Session` struct

---

## 4. Layout Library Research

### 4.1 golden-layout (used by mud-web-client)
- **Downloads:** 12K/week, 6.6K GitHub stars
- **Supports:** Vue 3, React, vanilla TS
- **Features:** Draggable/resizable panels, tabs, split views, save/restore layout state
- **Already integrated** in mud-web-client — `GoldenLayoutAdapter` class, `TerminalPanel.vue`
- **License:** MIT
- **Rendering:** By default removes DOM when panel hidden (memory efficient)
- **Concern:** Last major release (v2.6.0) was a while ago; community activity has slowed

### 4.2 dockview (NEW — potentially better)
- **Repo:** https://github.com/mathuo/dockview
- **Downloads:** 60K/week, 3.1K GitHub stars
- **Supports:** React, Vue, Angular, vanilla TypeScript
- **Features:** Tabs, groups, grids, split views, drag-and-drop, floating panels, popout windows
- **License:** MIT
- **Zero dependencies** — lighter than golden-layout
- **Show HN:** Jan 2025, very active development (last commit Mar 2026)
- **Rendering modes:** `onlyWhenVisible` (removes from DOM, memory efficient) OR `always` (keeps DOM alive, preserves scrollback)
- **API:** `onDidVisibilityChange` event — perfect for pausing xterm.js render when panel hidden
- **Layout persistence:** Built-in save/restore via JSON

### 4.3 Comparison

| Feature | golden-layout | dockview |
|---------|---------------|----------|
| Weekly downloads | 12K | 60K |
| Dependencies | jQuery (v2 legacy) | Zero |
| Vue 3 support | Via adapter | Native wrapper |
| Floating panels | No | Yes |
| Popout windows | No | Yes |
| Rendering control | DOM removal only | Configurable per-panel |
| Visibility events | No (resize only) | `onDidVisibilityChange` |
| Layout persistence | Manual JSON | Built-in API |
| Active development | Slowed | Active (2026) |
| Bundle size | ~50KB | ~40KB |
| Already in mud-web-client | Yes | No |

### 4.4 Recommendation
**dockview** over golden-layout for a fork. The zero-dependency footprint, native Vue 3 support, visibility events (critical for pausing hidden xterm.js instances), and active development make it the better choice. The floating/popout panels are a nice-to-have for power users.

---

## 5. Proposed Architecture

### 5.1 Fork Strategy
Fork `maldorne/mud-web-client` → strip proxy dependency → replace golden-layout with dockview → add multi-character support → restyle for Dark Pawns brand.

**What to keep:**
- `MudTerminal.vue` — xterm.js terminal component (standalone, well-built)
- `useTelnetParser()` — telnet IAC/ANSI parsing composable
- `useConfig.ts` — query parameter configuration pattern
- Vite build pipeline, Docker setup
- Embed mode concept (iframe-compatible)

**What to replace:**
- `useSocket.ts` — rewrite to speak Dark Pawns' JSON protocol directly (no proxy connect message)
- golden-layout → dockview
- Single-connection App.vue → multi-connection PartyClient.vue

**What to add:**
- `PartyClient.vue` — manages 3 character sessions
- `useCharacterSession()` composable — one per character, holds socket + terminal + parser
- Custom xterm theme: `#0F0F0F` background, `#EFE7D6` accent text for Dark Pawns brand
- Tab bar with character names, connection status, HP indicator (via GMCP if available)

### 5.2 Connection Flow (No Proxy)

```
Browser                     Go Server (:8080)
  │                              │
  │── WebSocket /ws ────────────>│  (upgrade)
  │<────── JSON login msg ───────│  (existing flow)
  │── { "type": "command",       │
  │     "input": "look" } ──────>│
  │<──── game output ────────────│
  │                              │
  │── WebSocket /ws ────────────>│  (char 2)
  │── WebSocket /ws ────────────>│  (char 3)
```

The client opens 3 WebSockets to the same `/ws` endpoint, each authenticating as a different character. The Go server handles this naturally — each WebSocket creates an independent `Session`.

### 5.3 Layout

**Default: 3-pane tiled grid** (matching Zach's play style)
```
┌────────────────┬────────────────┐
│  Warrior       │  Magus          │
│  (xterm.js)    │  (xterm.js)    │
│                │                │
│                │                │
├────────────────┴────────────────┤
│           Thief                  │
│           (xterm.js)            │
│                                 │
└─────────────────────────────────┘
```

**Also supported via dockview:**
- Tab mode (click tabs to switch between characters)
- Custom split (drag to rearrange)
- Popout (drag a character's terminal to a separate browser window)

**Performance optimization:**
- Dockview's `onlyWhenVisible` rendering mode — hidden terminals don't render
- Active terminal gets 100% of render budget
- Scrollback preserved in DOM when hidden (`always` mode for xterm panels specifically)
- WebSocket receives data for all connections regardless of visibility (server pushes, can't pause)

### 5.4 Input Routing
- Click a terminal pane → it gains focus → keyboard input routes to that character's WebSocket
- Visual focus indicator (border highlight) shows which character is active
- Tab key could cycle focus between terminals (keyboard-only navigation)
- Optional: "input to all" mode for coordinated movement (future)

### 5.5 Character Status
If the Go server sends GMCP data (e.g., `Char.Vitals`), the tab bar can show:
- Character name + class icon
- HP/MP/MP bar (mini, inline)
- Status: connected, link-dead, combat (flash)

---

## 6. Integration with Hugo Site

### 6.1 Embedding Strategy

**Option A: iframe (recommended for v1)**
```html
<!-- Hugo /play page -->
<div class="play-wrapper">
  <iframe 
    src="/client/index.html?host=darkpawns.labz0rz.com&port=8080&layout=tiled"
    class="play-terminal"
    allow="clipboard-write"
  ></iframe>
  <aside class="play-sidebar">
    <h3>For serious play</h3>
    <p>Use <a href="/connect">Mudlet</a> or <a href="/connect">TinTin++</a></p>
  </aside>
</div>
```

- CSS isolation — terminal's dark theme doesn't bleed into cream page
- Separate build pipeline — client builds independently, Hugo just references the bundle
- `sandbox="allow-same-origin"` for clipboard access
- Cross-origin not an issue if served from same domain

**Option B: Mounted component (cleaner, more work)**
- Build client as a Vue web component (`defineCustomElement`)
- Mount into Hugo's `/play` page div
- Requires careful CSS scoping
- Better long-term but higher initial effort

### 6.2 URL Scheme
```
/play                    → Default single-character, embed mode
/play?mode=tiled         → 3-pane grid
/play?mode=full          → Full dockview with all panels
/play?char=Warrior       → Pre-named character tab
```

---

## 7. Build Phases

### Phase 1: Get `/play` Working (v1)
1. Fork mud-web-client, strip proxy dependency
2. Rewrite `useSocket()` to speak Go server's JSON protocol directly
3. Single-connection embed mode
4. Custom xterm theme (Dark Pawns brand)
5. Build as static assets, drop into Hugo `/play` via iframe
6. Connect to running Go server at `/ws`
7. Test basic play: login, move, combat, commands

### Phase 2: Multi-Character Tiled Grid (v1.5)
1. Replace golden-layout with dockview
2. Create `PartyClient.vue` — manages up to 3 `CharacterSession` instances
3. Default 3-pane tiled layout
4. Click-to-focus input routing
5. Tab labels with character names
6. Test: log in 3 characters simultaneously, switch between them

### Phase 3: Polish (v2)
1. GMCP integration for character status in tab bar
2. Connection status indicators (connected/dc/combat)
3. Layout persistence (save/restore via dockview API or localStorage)
4. "Input to all" mode for coordinated movement
5. Mobile-responsive: stack terminals vertically on narrow screens
6. Popout windows (dockview native support)

---

## 8. Open Questions

1. **Go server JSON protocol spec** — Need full `ClientMessage` schema to rewrite `useSocket()`. Where is the struct defined?
2. **GMCP support depth** — Does the Go server actually send GMCP `Char.Vitals`? If not, tab bar status indicators need a different data source.
3. **Authentication flow** — How does login work over WebSocket? Username/password in JSON message? Session token from a prior HTTP login? Need to trace the full auth flow.
4. **License compatibility** — mud-web-client is GPL-3.0. If we fork and ship, the Dark Pawns web client must also be GPL-3.0. Is this acceptable for the `/play` page specifically (the rest of the site is Hugo/static)?
5. **mud-web-proxy elimination** — mud-web-client's `useSocket()` sends a specific connect message format (`{connect: 1, utf8: 1, ...}`) that the proxy expects. We need to rewrite this to match the Go server's `ClientMessage` format. How different are they?
6. **Mobile** — 3 tiled terminals on mobile is unusable. What's the minimum viable mobile experience? Single terminal + character switcher?

---

## 9. References

- [mud-web-client](https://github.com/maldorne/mud-web-client) — Primary candidate, Vue 3 + xterm.js
- [mud-web-proxy](https://github.com/maldorne/mud-web-proxy) — WebSocket-to-telnet proxy (NOT NEEDED)
- [dockview](https://github.com/mathuo/dockview) — Layout manager for multi-pane (recommended over golden-layout)
- [golden-layout](https://golden-layout.github.io/) — Alternative layout manager (already in mud-web-client)
- [xterm.js](https://github.com/xtermjs/xterm.js) — Terminal emulator for web
- [xterm.js multi-instance perf](https://github.com/xtermjs/xterm.js/issues/3368) — Performance considerations
- [awesome-muds](https://github.com/maldorne/awesome-muds) — Curated MUD resource list
- [DomTerm](https://domterm.org/) — Web terminal with tiles/tabs (reference for approach)
- [mud-web-roundup](https://github.com/ryanberckmans/mud-web-roundup) — Historical survey of web MUD clients
