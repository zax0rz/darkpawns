# Brief: Character Creation & Login Flow — Word-for-Word Fidelity + Bug Fixes

**Date:** 2026-05-27
**Requested by:** The Architect (Zach)
**Priority:** HIGH — Blocks new player experience
**Scope:** Character creation flow, login/ reconnect flow, MOTD display, web client bugs

---

## Summary

The character creation and login/reconnect flow has multiple fidelity issues against the C source (`src/interpreter.c` nanny() function, lines 1693–2293) and several web client bugs. This brief covers:

1. **Race names are wrong** — "Halfling" instead of "Kender" (C source says "Kenderkin")
2. **Race descriptions are generic/AI slop** — not the original C text
3. **MOTD is wrong** — shows ASCII logo + custom text instead of the original MOTD file
4. **Character creation flow order differs from C source** — missing name confirmation step
5. **Movement broken after creation** — player can't move in the starting room
6. **Reconnect flow broken** — "is this a new character y/n" prompt shouldn't exist, then hangs on "Connecting..."
7. **Web client status bar overlaps text input**
8. **Telnet connectivity broken** — `telnet darkpawns.labz0rz.com 7777` doesn't work (DNS resolution / port forwarding / Cloudflare issue)

---

## Issue 1: Race Names Are Wrong

**Current Go code** (`pkg/session/char_creation.go` line 367):
```go
"3": "Halfling",
```

**C source** (`src/constants.c` line 196, `src/interpreter.c` line 2058):
```
[K]enderkin
```

**Race constant** (`pkg/game/character.go` line 29):
```go
RaceKender   = 3
```

The race at index 3 is **Kender**, not Halfling. The C source calls them "Kenderkin" in the race menu. The Go code has the constant named correctly (`RaceKender`) but the display string in char creation says "Halfling".

**Fix:** Change `"3": "Halfling"` to `"3": "Kenderkin"` in `getRaceOptions()`.

---

## Issue 2: Race Descriptions Are AI Slop

**Current Go code** (`pkg/session/char_creation.go` `getRaceOptions()`) returns a flat map of number→name with no descriptions.

**C source** (`src/constants.c` lines 208–343) has detailed race help text accessed via `?` during race selection:

- `race_help` — general race info (terrain vision, magic resistance, attitudes)
- `help_human` — full paragraph description
- `help_dwarf` — full paragraph description
- `help_elf` — full paragraph description
- `help_kender` — full paragraph description
- `help_minotaur` — full paragraph description
- `help_rakshasa` — full paragraph description
- `help_ssaur` — full paragraph description

The Go char creation sends a bare menu with no help text and no `?` option.

**Fix:** Port the exact C race help text into the Go codebase. Add `?` support to the race selection stage:
- `?` alone → show `race_help` (general info)
- `?H` / `?E` / `?D` / `?K` / `?M` / `?R` / `?S` → show specific race help
- Re-display race menu after showing help

The race menu text itself should also match C exactly:
```
Choose a race:
  [H]uman        [E]lven       [D]warven      [K]enderkin
  [M]inotaur     [R]akshasan   [S]sauran
  [?]Help on races in general
  [?<race abbreviation>] Help on a specific race (i.e ?D for help on dwarves)
```

---

## Issue 3: MOTD Is Wrong

**Current behavior:** `sendWelcome()` in `pkg/session/session_send.go` calls `game.ShowMOTD()` which reads `lib/world/text/motd`. That file currently contains an ASCII logo + custom rules text.

**C source** (`src/interpreter.c` lines 1930–1938, 2130–2138):
- After password verification for returning players: `SEND_TO_Q(motd, d)` then `STATE(d) = CON_RMOTD`
- After stat roll acceptance for new players: `SEND_TO_Q(motd, d)` then `STATE(d) = CON_RMOTD`
- `CON_RMOTD` state: `SEND_TO_Q(MENU, d)` → `STATE(d) = CON_MENU`

The C source MOTD is a simple text file. The `MENU` is a separate constant shown after pressing RETURN at the MOTD.

**What should happen:**
1. Show MOTD text (the file at `lib/world/text/motd`)
2. Player presses RETURN
3. Show the main menu (not implemented in web client — web client skips straight to game)

**What's happening:** The MOTD file has an ASCII logo that's "mushed up" because the web client's terminal doesn't render it properly, and it's showing content that shouldn't be there.

**Fix:** Replace `lib/world/text/motd` with the original C source MOTD. The C source MOTD was a simple text file — find the original in the SVN history or write a faithful recreation. The current file appears to be a custom creation with an ASCII logo that doesn't belong.

---

## Issue 8: Telnet Connectivity Broken

**Symptom:** `telnet darkpawns.labz0rz.com 7777` doesn't work. DNS doesn't resolve to the MUD port.

**Root cause:** This is likely NOT a code issue. The MUD server runs on `frankendell` (192.168.1.15) behind a home router, and `darkpawns.labz0rz.com` is routed through Cloudflare Tunnel (which handles HTTP/HTTPS on port 443). Cloudflare Tunnel does NOT forward raw TCP ports like 7777.

**The problem:**
- Cloudflare Tunnel (`cloudflared`) tunnels HTTP/HTTPS traffic from the Cloudflare edge to the local machine
- Telnet is raw TCP, not HTTP — Cloudflare Tunnel doesn't handle it
- The home router likely doesn't have port 7777 forwarded to 192.168.1.15
- Even if it did, `darkpawns.labz0rz.com` resolves to Cloudflare IPs, not the home IP

**Possible solutions (The Architect to decide):**

1. **Port forwarding + DNS record**
   - Forward TCP port 7777 on the router to 192.168.1.15:7777
   - Add a separate DNS A record (e.g., `telnet.darkpawns.labz0rz.com`) pointing to the home public IP
   - Pros: Simple, direct, low latency
   - Cons: Exposes home IP, requires dynamic DNS or static IP, opens a port on the router

2. **Cloudflare Spectrum** (paid, $1/partner port/month)
   - Cloudflare Spectrum can proxy raw TCP traffic through Cloudflare's network
   - Would allow `darkpawns.labz0rz.com:7777` to work via Cloudflare
   - Pros: No port forwarding, home IP stays hidden, Cloudflare DDoS protection
   - Cons: Costs money, requires Spectrum plan

3. **SSL/TLS tunnel via stunnel or HAProxy**
   - Wrap the telnet connection in TLS, tunnel through Cloudflare's HTTPS endpoint
   - Pros: Uses existing Cloudflare Tunnel infrastructure
   - Cons: Requires a TLS-wrapping client (not standard telnet)

4. **Separate VPS/reverse proxy**
   - Run a small VPS that forwards TCP 7777 to the home server via WireGuard/tailscale
   - Pros: Clean separation, static IP
   - Cons: Another moving part, monthly cost

5. **Accept telnet is dead, web only**
   - The web client is the primary interface
   - Telnet is a legacy convenience
   - Pros: No infrastructure work
   - Cons: Loses the authentic MUD experience, breaks `telnet` in the landing page

**Recommendation:** Option 1 (port forwarding + DNS) is the simplest for a home server. Option 2 (Spectrum) if you want Cloudflare to handle everything. Either way, this needs The Architect to make a routing decision.

**This is NOT a code fix — it's infrastructure.** The MUD server is already listening on port 7777. The issue is getting traffic to it from the public internet.

---

## Issue 4: Character Creation Flow Order Differs from C Source

**C source flow** (nanny() in `src/interpreter.c`):

1. `CON_GET_NAME` — "Name: " prompt
2. `CON_NAME_CNFRM` — "Did I get that right, %s (Y/N)? " (for both new AND existing deleted chars)
3. `CON_PASSWORD` — "Password: " (for existing characters)
4. `CON_NEWPASSWD` — "Give me a password for %s: " (new chars)
5. `CON_CNFPASSWD` — "Please retype password: "
6. `CON_COLOR` — "Do you want ANSI color (Y/N)? "
7. `CON_QSEX` — "What is your sex (M/F)? "
8. `CON_QRACE` — race_menu + "Race: "
9. `CON_QCLASS` — class_menu + "Class: "
10. `CON_HOMETOWN` — hometown_menu + "Select: "
11. `CON_ROLLABL1` — ability scores display
12. `CON_ROLLABL2` — "Press 'Y' to keep these stats, and 'N' to reroll:"
13. `CON_RMOTD` — MOTD + "*** PRESS RETURN: "
14. `CON_MENU` — main menu

**Current Go flow** (web client):

1. Client asks for name
2. Client asks for password
3. Server sends `char_create` with color prompt
4. Sex → Race → Class → Hometown → Stats roll
5. Done → enters world

**Missing steps:**
- No name confirmation ("Did I get that right, %s (Y/N)?")
- No password confirmation ("Please retype password: ")
- The web client handles name+password locally, never sends `new_char: true` properly

**The "is this a new character y/n" problem:** The web client doesn't send `new_char` in its login message. Looking at `client.js`, it sends:
```js
{ type: 'login', data: { player_name: name, password: pw, mode: 'player' } }
```
No `new_char` field. The server's `handleLogin` then hits this logic (line 158–221):
- If player exists in DB and `!login.NewChar` → password auth → login
- If `login.NewChar` → new character flow
- If player doesn't exist and `new_char` not set → start creation

For a **returning player**, this should work — the player exists, `NewChar` is false, so it does password auth. But the Architect reports being asked "is this a new character y/n" — this suggests either:
1. The web client is showing something it shouldn't, OR
2. The server is sending a prompt that triggers the web client to ask this

**Actually:** Looking more carefully, the web client doesn't have a "new character" prompt at all. The `new_char` field is never sent. The issue is likely that the **telnet path** or a **different client** is showing this. But the Architect says it happened. Need to verify which client was used.

**For the web client:** The flow should be:
1. Enter name → if name exists, ask password → authenticate → enter world
2. Enter name → if name doesn't exist, ask password (twice for confirmation) → enter char creation

The web client currently doesn't handle the "name doesn't exist → create new character" flow properly. It just sends the login and hopes for the best.

---

## Issue 5: Movement Broken After Character Creation

**Symptom:** After creating a character, player is dropped in the temple altar (room 8004, correct), can `look` but can't move. Every direction command (n/s/e/w) doesn't enter anything.

**Possible causes:**

1. **Web client input handling:** The `client.js` sends commands as:
   ```js
   ws.send(JSON.stringify({ type: 'command', data: { command: inputBuffer } }));
   ```
   The `command` field is the raw input string (e.g., "n" or "north"). The server's `handleCommand` passes this to `ExecuteCommand(s, cmd.Command, cmd.Args)`.

2. **Command parsing:** Check if `ExecuteCommand` handles single-letter directions. The C source uses `SCMD_NORTH` (1) through `SCMD_DOWN` (6) and `IS_MOVE(cmdnum)` checks `cmdnum >= 1 && cmdnum <= 6`. The Go command registry needs to map "n", "north", "s", "south", etc. to movement commands.

3. **Room exits:** Room 8004 might have no exits defined in the world data. Check `lib/world/wld/` for the room's exit data.

4. **Starting room issue:** `completeCharCreation()` sets `s.player.RoomVNum = game.LoginStartRoom(s.player)` which returns `MortalStartRoom` (8004). But the comment says "Room 8099 (A Burning Hut) is the C source intro room... but it has no exits and no mob spawns." The C source sends new characters to room 8099 after the menu (line 2241), not 8004.

**Investigation needed:** Check if movement commands are registered in the command registry, and if room 8004 has exits.

---

## Issue 6: Reconnect Flow Broken

**Symptom:** Reconnecting as an existing character shows: name → password → "is this a new character y/n" → hangs on "Connecting..."

**Root cause analysis:**

The web client's reconnect flow (`client.js`):
1. `loginPhase = 'name'` → enter name
2. `loginPhase = 'password'` → enter password
3. Sends `{ type: 'login', data: { player_name, password, mode: 'player' } }`
4. `loginPhase = 'done'`

The server receives this, authenticates, sends `state` message. The client handles `state` messages:
```js
if (msg.type === 'state') {
  handleStateMsg(msg.data);
  charCreating = false;
  return;
}
```

**Problem:** The client never sends `new_char: false` explicitly. Looking at the `LoginData` struct:
```go
NewChar bool `json:"new_char,omitempty"` // omitempty means false is not sent
```

So when the client sends `{ player_name, password }` without `new_char`, the server sees `login.NewChar == false`. For an existing player with correct password, this should work — it hits the `rec != nil && !login.NewChar` branch and authenticates.

**But wait** — the Architect says they see "is this a new character y/n". This prompt doesn't exist in the web client code. This suggests:
1. The Architect might have been using a **different client** (telnet?), OR
2. There's a server-side prompt being sent that the web client renders

Looking at the server code, there's no "is this a new character" prompt anywhere. The `sendErrorWithState` function has a fallback that sends a login prompt, but that's not it.

**Most likely:** The Architect was testing with a **telnet client** or the **DP-Goat agent CLI**, not the web client. The web client's login flow is hardcoded and doesn't have this prompt.

**The hang on "Connecting...":** After sending the login message, the client sets `loginPhase = 'done'`. If the server response doesn't include a `state` or `event` message that the client handles, the client just sits there. The `ws.onmessage` handler only processes specific message types — anything else falls through to `term.writeln(text)`. If the server sends an error, it would show. If the server sends nothing (connection issue), the client shows nothing.

**Fix needed:**
- The web client needs to handle the full C source login flow, including:
  - Name confirmation for new characters
  - Password confirmation (type twice)
  - The MOTD → PRESS RETURN → menu flow
- For returning players: name → password → authenticate → MOTD → enter world
- For new players: name → confirm name → password → retype password → color → sex → race → class → hometown → stats → MOTD → enter world

---

## Issue 7: Web Client Status Bar Overlaps Text Input

**Symptom:** The HP/Mana/Move bar overlaps the text entry area.

**Current layout** (`web/index.html`):
```html
<section class="terminal-section" id="terminal">
  <div class="terminal-frame">
    <div class="terminal-header">...</div>
    <div id="terminal"></div>          <!-- xterm.js renders here -->
    <div id="status-bar" class="status-bar hidden">...</div>
  </div>
</section>
```

**CSS** (`web/style.css`):
```css
.status-bar {
  display: flex;
  align-items: center;
  gap: 0.8em;
  padding: 0.4em 0.8em;
  background: #141210;
  border-top: 1px solid #2a2218;
  ...
}
```

The status bar is positioned **below** the terminal div, which should be correct. But xterm.js's terminal div might be taking up more space than expected, or the status bar might be overlapping due to absolute positioning or z-index issues.

**The real issue:** xterm.js has its own textarea for input capture. The status bar is rendered after the terminal div, but the terminal's textarea might extend beyond the visible area. Check if xterm.js's textarea is overlapping the status bar.

**Fix:** Ensure the terminal div has a fixed height or max-height, and the status bar is positioned outside the terminal's overflow area. Use `position: relative` on the terminal frame and ensure the status bar isn't covered by xterm.js's internal elements.

---

## Implementation Plan

### Phase 1: Critical Fixes (blocking new players)

1. **Fix race name** — Change `"3": "Halfling"` to `"3": "Kenderkin"` in `getRaceOptions()`
2. **Fix MOTD** — Replace `lib/world/text/motd` with original C source MOTD text
3. **Fix movement** — Check command registry for direction commands, verify room 8004 exits

### Phase 2: Fidelity (word-for-word match to C source)

4. **Port race help text** — Add all 7 race descriptions from `src/constants.c` to Go code
5. **Add `?` help to race selection** — Implement help lookup during char creation
6. **Match C source menu text exactly** — race_menu, class_menu, human_class_menu, hometown_menu

### Phase 3: Web Client Fixes

7. **Fix status bar overlap** — Adjust CSS/layout so status bar doesn't overlap terminal input
8. **Fix reconnect flow** — Ensure returning players can authenticate without hanging
9. **Add new character flow to web client** — Name confirmation, password confirmation, full creation flow

### Phase 4: Login Flow Fidelity

10. **Add name confirmation step** — "Did I get that right, %s (Y/N)?"
11. **Add password confirmation** — "Please retype password: "
12. **Add MOTD → PRESS RETURN → menu flow** — Match C source exactly

### Phase 5: Telnet Connectivity (Infrastructure)

13. **Port forwarding** — Forward TCP 7777 on router to 192.168.1.15:7777
14. **DNS** — Add A record for `telnet.darkpawns.labz0rz.com` (or similar) pointing to home public IP
15. **Verify** — `telnet darkpawns.labz0rz.com 7777` connects and shows MOTD

**Note:** Phase 5 is infrastructure, not code. Requires The Architect to make a routing/DNS decision.

---

## Testing

After each phase:
1. `go build ./... && go vet ./... && go test ./...`
2. Manual test: create new character via web client
3. Manual test: reconnect as existing character
4. Manual test: movement in starting room
5. Verify MOTD displays correctly
6. Verify race names and descriptions match C source

---

## Files to Modify

| File | Changes |
|------|---------|
| `pkg/session/char_creation.go` | Fix race name, add race help text, add `?` help support |
| `lib/world/text/motd` | Replace with original C source MOTD |
| `web/client.js` | Fix login/reconnect flow, handle new character creation |
| `web/style.css` | Fix status bar overlap |
| `web/index.html` | Possibly adjust layout |
| `pkg/session/session_send.go` | May need to adjust MOTD display |
| `pkg/game/constants.go` | May need to add race help constants |

---

## C Source References

| Location | What |
|----------|------|
| `src/interpreter.c:1693-2293` | nanny() — full login/creation flow |
| `src/interpreter.c:1743-1784` | CON_GET_NAME — name input |
| `src/interpreter.c:1819-1853` | CON_NAME_CNFRM — name confirmation |
| `src/interpreter.c:1860-1938` | CON_PASSWORD — password verification |
| `src/interpreter.c:1942-1980` | CON_NEWPASSWD/CON_CNFPASSWD — password creation |
| `src/interpreter.c:1992-2010` | CON_COLOR — color selection |
| `src/interpreter.c:2012-2032` | CON_QSEX — sex selection |
| `src/interpreter.c:2035-2078` | CON_QRACE — race selection with `?` help |
| `src/interpreter.c:2081-2093` | CON_QCLASS — class selection |
| `src/interpreter.c:2096-2104` | CON_HOMETOWN — hometown selection |
| `src/interpreter.c:2106-2158` | CON_ROLLABL1/2 — stat rolling |
| `src/interpreter.c:2160-2165` | CON_RMOTD — MOTD display |
| `src/interpreter.c:2165-2268` | CON_MENU — main menu |
| `src/constants.c:196-207` | race_menu text |
| `src/constants.c:208-220` | race_help text |
| `src/constants.c:221-343` | Individual race help texts |
| `src/class.c:86-113` | class_menu, hometown_menu, human_class_menu |
