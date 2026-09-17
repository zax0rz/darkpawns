// The one MUD client. Three surfaces speak this protocol — the self-host splash
// (web/public), the /play page on darkpawns.org, and the admin console's
// Terminal — and until 2026-09-17 each carried its own implementation of it.
// They had drifted apart in both directions, and each had a defect the other
// two did not:
//
//   splash   rendered `state` into the terminal as invented prose, so every
//            room appeared twice: once as the game wrote it, once paraphrased
//   admin    displayed char_create prompts but never sent char_input, so
//            character creation could not be completed there at all
//   all three dropped `prompt` and `token_refresh`, which the server sends
//
// Terminal construction stays with the host: the three run different xterm
// versions and mount differently (a DOM id here, a React ref there), and that
// difference is legitimate. What must not differ — the wire protocol, the
// login and character-creation state machine, typeahead, and secret masking —
// lives here and is written once.
//
// This file is served verbatim to the splash, which has no build step, so it
// must stay plain ESM with no bare-specifier imports. The two bundled hosts
// import it by relative path.
//
// Panels are optional. A host that provides the panel elements gets minimap,
// room contents, inventory/equipment and target; one that does not gets the
// terminal and status bar alone. Nothing here requires an element to exist.

// Vitals thresholds. Exported because the admin console renders its status bar
// as React components and so cannot use the DOM writers below — without a
// shared definition the two would drift, which is the failure this file exists
// to end. The bands are the same ones the bars have always used.
export function pct(cur, max) {
  return max > 0 ? Math.round((cur / max) * 100) : 0;
}

export function hpColor(p) {
  if (p > 75) return '#4a8a4a';
  if (p > 25) return '#b8960a';
  return '#8b0000';
}

export function manaColor(p) {
  if (p > 75) return '#3a6a9a';
  if (p > 25) return '#2a5a7a';
  return '#1a3a5a';
}

export function moveColor(p) {
  if (p > 75) return '#6a8a3a';
  if (p > 25) return '#8a7a2a';
  return '#5a4a1a';
}

/**
 * Attach a Dark Pawns client to a terminal.
 *
 * @param {object}   options
 * @param {object}   options.terminal  an opened xterm Terminal, owned by the host
 * @param {string}   [options.wsUrl]   defaults to /ws on the current origin
 * @param {Document} [options.doc]     document to query for optional panels
 * @returns {{ connect: () => void, disconnect: () => void }}
 */
export function createMudClient(options) {
  'use strict';

  const term = options.terminal;
  const doc = options.doc || document;
  const params = new URLSearchParams(location.search);
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  const wsUrl =
    options.wsUrl || params.get('host') || `${proto}//${location.host}/ws`;

  // Every one of these is optional. The splash has the connection chrome and
  // the status bar but none of the sidebar panels; the admin Terminal has
  // neither. Absent elements simply mean that feature does not render.
  const statusEl = doc.querySelector('.conn-status');
  const reconnectBtn = doc.getElementById('reconnect-btn');
  const statusBar = doc.getElementById('status-bar');
  let inputBuffer = '';
  let ws;

  // ── Status Bar State ──
  const playerState = {
    health: 0, maxHealth: 0,
    mana: 0, maxMana: 0,
    move: 0, maxMove: 0,
    level: 0, gold: 0,
    roomVnum: 0,
  };

  let worldMapData = null;
  fetch('/map/world-map.json')
    .then(r => r.json())
    .then(data => {
      worldMapData = data;
      if (playerState.roomVnum) {
        updateMinimap(playerState.roomVnum);
      }
    })
    .catch(err => console.warn('Failed to load world map for minimap:', err));

  function updateBar(id, cur, max, colorFn) {
    const bar = doc.getElementById(id);
    if (!bar) return;
    const p = pct(cur, max);
    bar.style.width = (max > 0 ? p : 0) + '%';
    bar.style.backgroundColor = colorFn(p);
  }

  function updateStatusBar() {
    // Fires before the DOM guard below. The admin console renders its status
    // bar from React state rather than from elements this module can write to,
    // so it takes the vitals through this hook and has no #status-bar at all.
    if (typeof options.onPlayerState === 'function') {
      options.onPlayerState(Object.assign({}, playerState));
    }
    if (!statusBar) return;
    updateBar('hp-bar', playerState.health, playerState.maxHealth, hpColor);
    const hpText = doc.getElementById('hp-text');
    if (hpText) hpText.textContent = playerState.maxHealth > 0 ? `${playerState.health}/${playerState.maxHealth}` : '—';

    updateBar('mana-bar', playerState.mana, playerState.maxMana, manaColor);
    const manaText = doc.getElementById('mana-text');
    if (manaText) manaText.textContent = playerState.maxMana > 0 ? `${playerState.mana}/${playerState.maxMana}` : '—';

    updateBar('move-bar', playerState.move, playerState.maxMove, moveColor);
    const moveText = doc.getElementById('move-text');
    if (moveText) moveText.textContent = playerState.maxMove > 0 ? `${playerState.move}/${playerState.maxMove}` : '—';

    const lvlInfo = doc.getElementById('level-info');
    if (lvlInfo) lvlInfo.textContent = playerState.level > 0 ? `Lv ${playerState.level}` : 'Lv —';

    const goldInfo = doc.getElementById('gold-info');
    if (goldInfo) goldInfo.textContent = playerState.gold > 0 ? `Gold ${playerState.gold}` : 'Gold —';

    // Show status bar once we have any real data
    if (playerState.maxHealth > 0) {
      statusBar.classList.remove('hidden');
    }
  }

  function handleStateMsg(data) {
    if (!data || !data.player) return;
    const p = data.player;
    playerState.health = p.health || 0;
    playerState.maxHealth = p.max_health || 0;
    playerState.level = p.level || 0;
    if (p.mana !== undefined) playerState.mana = p.mana;
    if (p.max_mana !== undefined) playerState.maxMana = p.max_mana;
    if (p.move !== undefined) playerState.move = p.move;
    if (p.max_move !== undefined) playerState.maxMove = p.max_move;
    if (p.gold !== undefined) playerState.gold = p.gold;
    updateStatusBar();
  }

  // Sector colors map for minimap
  const SECTOR_COLOR = {
    0: '#4a4540', 1: '#a8201a', 2: '#8c905c',  3: '#3e5c38',
    4: '#b08b5c', 5: '#6b5e50', 6: '#4a7a96',  7: '#2c5282',
    8: '#1a365d', 9: '#63b3ed', 10: '#d0a868', 11: '#e05a47',
    12: '#8d7a5b', 13: '#cbd5e0', 14: '#2c5282', 15: '#5d705c',
  };

  function handleVarsMsg(data) {
    if (!data) return;

    // Show panels and hide connection panel once logged in and receiving vars
    const connectPanel = doc.getElementById('sidebar-connect-panel');
    if (connectPanel) connectPanel.classList.add('hidden');

    const panels = doc.querySelectorAll('.sidebar-panel');
    panels.forEach(p => p.classList.remove('hidden'));

    // Update player status state
    if (data.HEALTH !== undefined) playerState.health = data.HEALTH;
    if (data.MAX_HEALTH !== undefined) playerState.maxHealth = data.MAX_HEALTH;
    if (data.MANA !== undefined) playerState.mana = data.MANA;
    if (data.MAX_MANA !== undefined) playerState.maxMana = data.MAX_MANA;
    if (data.MOVE !== undefined) playerState.move = data.MOVE;
    if (data.MAX_MOVE !== undefined) playerState.maxMove = data.MAX_MOVE;
    if (data.LEVEL !== undefined) playerState.level = data.LEVEL;
    if (data.GOLD !== undefined) playerState.gold = data.GOLD;
    updateStatusBar();

    // Minimap update
    if (data.ROOM_VNUM !== undefined) {
      playerState.roomVnum = data.ROOM_VNUM;
      updateMinimap(data.ROOM_VNUM);
    }

    // Room contents update (mobs/items)
    if (data.ROOM_MOBS !== undefined || data.ROOM_ITEMS !== undefined) {
      updateRoomContents(data.ROOM_MOBS, data.ROOM_ITEMS);
    }

    // Inventory & Equipment update
    if (data.INVENTORY !== undefined || data.EQUIPMENT !== undefined) {
      updateInventoryEquipment(data.INVENTORY, data.EQUIPMENT);
    }

    // Target / Enemy update
    if (data.FIGHTING !== undefined) {
      updateTargetDisplay(data.FIGHTING);
    }
  }

  function escHtml(s) {
    return String(s)
      .replace(/&/g, '&amp;').replace(/</g, '&lt;')
      .replace(/>/g, '&gt;').replace(/"/g, '&quot;');
  }

  function updateMinimap(currentVnum) {
    const container = doc.getElementById('minimap-container');
    if (!container) return;

    if (!worldMapData) {
      container.innerHTML = `
        <div class="sidebar-panel-title">Minimap</div>
        <div style="font-size:0.75rem; font-style:italic; color:var(--ink-muted);">Loading map data...</div>
      `;
      return;
    }

    const currentRoom = worldMapData.rooms.find(r => r.id === currentVnum);
    if (!currentRoom) {
      container.innerHTML = `
        <div class="sidebar-panel-title">Minimap</div>
        <div style="font-size:0.75rem; font-style:italic; color:var(--ink-muted);">Room #${currentVnum} not on map</div>
      `;
      return;
    }

    const zoneId = currentRoom.zone_id;
    const zoneRooms = worldMapData.rooms.filter(r => r.zone_id === zoneId);

    // Zoom box centered on current room
    const size = 150;
    const minX = currentRoom.x - size / 2;
    const minY = currentRoom.y - size / 2;

    const links = worldMapData.links.filter(l => {
      const s = zoneRooms.find(r => r.id === l.s);
      const t = zoneRooms.find(r => r.id === l.t);
      return s && t;
    });

    let svgContent = `<svg width="100%" height="150" viewBox="${minX} ${minY} ${size} ${size}" style="background:#050404; border:1px solid var(--rule); border-radius:4px;">`;

    // Draw links
    links.forEach(l => {
      const s = zoneRooms.find(r => r.id === l.s);
      const t = zoneRooms.find(r => r.id === l.t);
      if (s && t) {
        svgContent += `<line x1="${s.x}" y1="${s.y}" x2="${t.x}" y2="${t.y}" stroke="rgba(200, 184, 150, 0.18)" stroke-width="1.5" />`;
      }
    });

    // Draw dots
    zoneRooms.forEach(r => {
      if (r.x >= minX - 10 && r.x <= minX + size + 10 && r.y >= minY - 10 && r.y <= minY + size + 10) {
        const isCurrent = r.id === currentVnum;
        const color = isCurrent ? '#a8201a' : (SECTOR_COLOR[r.sector] || '#4a4540');
        const radius = isCurrent ? 5.5 : 3.5;
        const stroke = isCurrent ? '#c8b896' : 'rgba(10, 9, 8, 0.5)';
        const strokeWidth = isCurrent ? 1.5 : 0.5;
        svgContent += `<circle cx="${r.x}" cy="${r.y}" r="${radius}" fill="${color}" stroke="${stroke}" stroke-width="${strokeWidth}" />`;
      }
    });

    svgContent += `</svg>`;

    container.innerHTML = `
      <div class="sidebar-panel-title">Minimap</div>
      <div style="font-size: 0.75rem; font-family: var(--font-display); text-transform: uppercase; color: var(--ink); margin-bottom: var(--space-xs); display:flex; justify-content:space-between; align-items:center;">
        <span>${escHtml(currentRoom.name || 'Unknown Room')}</span>
        <span style="font-family:monospace; color:var(--oxblood); font-weight:bold;">#${currentVnum}</span>
      </div>
      ${svgContent}
    `;
  }

  let cachedMobs = [];
  let cachedItems = [];

  function updateRoomContents(mobs, items) {
    const container = doc.getElementById('room-contents-container');
    if (!container) return;

    if (mobs !== undefined) cachedMobs = mobs || [];
    if (items !== undefined) cachedItems = items || [];

    let mobHtml = '';
    if (cachedMobs.length > 0) {
      mobHtml = cachedMobs.map(m => `
        <div class="sidebar-list-item" style="border-left-color: var(--oxblood);">
          <span style="color: var(--oxblood); font-weight: bold;">${escHtml(m.name)}</span>
          ${m.fighting ? '<span style="font-size:0.65rem; color:#8b0000; font-family:var(--font-display); text-transform:uppercase;">[Fighting]</span>' : ''}
        </div>
      `).join('');
    }

    let itemHtml = '';
    if (cachedItems.length > 0) {
      itemHtml = cachedItems.map(i => `
        <div class="sidebar-list-item" style="border-left-color: #5d705c;">
          <span>${escHtml(i.name)}</span>
        </div>
      `).join('');
    }

    if (cachedMobs.length === 0 && cachedItems.length === 0) {
      container.innerHTML = `
        <div class="sidebar-panel-title">In the Room</div>
        <div class="sidebar-list-empty">The room is empty.</div>
      `;
      return;
    }

    container.innerHTML = `
      <div class="sidebar-panel-title">In the Room</div>
      <div class="sidebar-list">
        ${mobHtml}
        ${itemHtml}
      </div>
    `;
  }

  let cachedInventory = [];
  let cachedEquipment = [];
  let activeTab = 'inventory';

  function updateInventoryEquipment(inventory, equipment) {
    const container = doc.getElementById('inventory-equipment-container');
    if (!container) return;

    if (inventory !== undefined) cachedInventory = inventory || [];
    if (equipment !== undefined) cachedEquipment = equipment || [];

    const tabsHtml = `
      <div class="sidebar-tabs">
        <button class="sidebar-tab-btn ${activeTab === 'inventory' ? 'active' : ''}" id="tab-btn-inv">Inventory</button>
        <button class="sidebar-tab-btn ${activeTab === 'equipment' ? 'active' : ''}" id="tab-btn-eq">Equipment</button>
      </div>
    `;

    let invHtml = '';
    if (cachedInventory.length > 0) {
      invHtml = cachedInventory.map(item => `
        <div class="sidebar-list-item" style="border-left-color: var(--rule);">
          <span>${escHtml(item.name || item)}</span>
        </div>
      `).join('');
    } else {
      invHtml = `<div class="sidebar-list-empty">Your inventory is empty.</div>`;
    }

    let eqHtml = '';
    if (cachedEquipment.length > 0) {
      eqHtml = cachedEquipment.map(item => `
        <div class="sidebar-list-item" style="border-left-color: var(--oxblood);">
          <span style="font-weight: bold; font-family: var(--font-display); text-transform: uppercase; font-size: 0.65rem; color: var(--oxblood); margin-right: 8px;">
            ${escHtml(item.slot || 'worn')}
          </span>
          <span>${escHtml(item.name || item)}</span>
        </div>
      `).join('');
    } else {
      eqHtml = `<div class="sidebar-list-empty">You are wearing nothing.</div>`;
    }

    container.innerHTML = `
      ${tabsHtml}
      <div id="sidebar-tab-inv-content" class="sidebar-tab-content ${activeTab === 'inventory' ? 'active' : ''}">
        <div class="sidebar-list">${invHtml}</div>
      </div>
      <div id="sidebar-tab-eq-content" class="sidebar-tab-content ${activeTab === 'equipment' ? 'active' : ''}">
        <div class="sidebar-list">${eqHtml}</div>
      </div>
    `;

    doc.getElementById('tab-btn-inv')?.addEventListener('click', () => {
      activeTab = 'inventory';
      updateInventoryEquipment();
    });
    doc.getElementById('tab-btn-eq')?.addEventListener('click', () => {
      activeTab = 'equipment';
      updateInventoryEquipment();
    });
  }

  function updateTargetDisplay(fightingData) {
    const container = doc.getElementById('target-container');
    if (!container) return;

    if (!fightingData || !fightingData.fighting || !fightingData.target) {
      container.classList.add('hidden');
      return;
    }

    container.classList.remove('hidden');

    const name = fightingData.target;
    const curHp = fightingData.hp || 0;
    const maxHp = fightingData.max_hp || 0;
    const pctHp = maxHp > 0 ? Math.round((curHp / maxHp) * 100) : 0;

    let barColor = '#4a8a4a';
    if (pctHp <= 25) barColor = '#8b0000';
    else if (pctHp <= 75) barColor = '#b8960a';

    container.innerHTML = `
      <div class="sidebar-panel-title">Target <span class="badge" style="background:#8b0000; color:#fff;">COMBAT</span></div>
      <div class="target-name">${escHtml(name)}</div>
      <div class="target-hp-bar-container">
        <div class="bar-track" style="flex:1; height: 10px;">
          <div class="bar-fill" style="width: ${pctHp}%; background-color: ${barColor}; height: 100%; transition: width 0.3s ease;"></div>
        </div>
        <span style="font-size:0.75rem; font-family:monospace; color:var(--ink-muted);">${curHp}/${maxHp}</span>
      </div>
    `;
  }

  let loggedIn = false;
  let inCharCreation = false;
  let charInputSecret = false;
  let awaitingEntryReply = false;
  let queuedInput = '';

  const greetingsLogo =
    "\r\n\r\n" +
    "         (_____)           (_)    (_____)\r\n" +
    "   _     /  __ \\           | |    |  __ \\                            _\r\n" +
    "  ;*;   /| |  | | __ _ _ __| | __ | |__) |_ _(_      _)_ __ (___)   ;*;\r\n" +
    "   =    /| |  | |/ _` | '__| |/ / |  ___/ _` \\ \\ /\\ / / '_ \\/ __|    =\r\n" +
    " .***.  /| |__| | (_| | |  |   <  | |  | (_| |\\ V  V /| | | \\__ \\  .***.\r\n" +
    " ~~~~~  /|_____/ \\__,_|_|  |_|\\_\\ |||   \\__,_| \\_/\\_/ |_| |_|___/  ~~~~~\r\n" +
    "                                  |||\r\n" +
    "                                  |||\r\n" +
    "                                  `.'\r\n\r\n" +
    "             Based on CircleMUD 3.0 created by J. Elson and\r\n" +
    "            DikuMUD Gamma 0.0 created by K. Nyboe, T. Madsen,\r\n" +
    "                H. Staerfeldt, M. Seifert, and S. Hammer\r\n\r\n";

  // handleStateRoom updates the sidebar from a state push. The terminal must
  // NOT render the room here: the server already delivers room text through
  // the canonical act()/text stream, and state arrives on every look, room
  // entry, and state refresh — printing it here duplicated output 2-3x.
  function handleStateRoom(data) {
    const r = data.room;
    if (!r) return;
    const connectPanel = doc.getElementById('sidebar-connect-panel');
    if (connectPanel) connectPanel.classList.add('hidden');
    doc.querySelectorAll('.sidebar-panel').forEach(p => p.classList.remove('hidden'));
    if (r.vnum) {
      playerState.roomVnum = r.vnum;
      updateMinimap(r.vnum);
    }
    if (r.mobs !== undefined || r.items !== undefined) {
      updateRoomContents(r.mobs, r.items);
    }
  }

  function setStatus(state) {
    // The admin console's Terminal has no connection chrome at all, so every
    // element here is optional.
    if (statusEl) {
      statusEl.className = 'conn-status ' + state;
      const label = state === 'connected' ? 'Connected' : 'Disconnected';
      const slot = statusEl.querySelector('span:last-child');
      if (slot) slot.textContent = label;
    }
    if (reconnectBtn) {
      reconnectBtn.classList.toggle('visible', state === 'disconnected');
    }
    if (typeof options.onStatus === 'function') options.onStatus(state);
  }

  function connect() {
    setStatus('disconnected');
    term.writeln('\x1b[2mConnecting to ' + wsUrl + '...\x1b[0m');
    try {
      ws = new WebSocket(wsUrl);
    } catch (e) {
      term.writeln('\x1b[31mConnection failed: ' + e.message + '\x1b[0m');
      return;
    }

    ws.onopen = function () {
      setStatus('connected');
      term.writeln('\x1b[32mConnected.\x1b[0m\r\n');
      // The admin console already knows who is signed in, so it names the
      // character instead of asking. Everywhere else the server owns the
      // question and the player answers it.
      if (options.autoLogin) {
        term.writeln('\x1b[2mSigning in as ' + options.autoLogin + '.\x1b[0m');
        awaitingEntryReply = true;
        ws.send(
          JSON.stringify({ type: 'login', data: { player_name: options.autoLogin } })
        );
        return;
      }
      term.write(greetingsLogo);
      term.write('By what name do you wish to be known? ');
    };

    ws.onmessage = function (evt) {
      try {
        const msg = JSON.parse(evt.data);
        if (msg.type === 'event' || msg.type === 'text') {
          // Standard in-game text stream
          term.write(msg.data.text);
        } else if (msg.type === 'vars') {
          handleVarsMsg(msg.data);
        } else if (msg.type === 'char_create') {
          inCharCreation = true;
          loggedIn = false;
          charInputSecret = Boolean(msg.data.secret);
          awaitingEntryReply = false;
          term.write(msg.data.prompt);
          drainInput();
        } else if (msg.type === 'error') {
          term.write('\r\n\x1b[31m' + msg.data.message + '\x1b[0m\r\n');
          // Reset login flow on failure so they can try again
          if (!loggedIn && !inCharCreation) {
            term.write('\r\nBy what name do you wish to be known? ');
            awaitingEntryReply = false;
          }
        } else if (msg.type === 'state') {
          if (!loggedIn && msg.data && msg.data.player && msg.data.player.name) {
            loggedIn = true;
            inCharCreation = false;
            charInputSecret = false;
            awaitingEntryReply = false;
            drainInput();
          }
          if (loggedIn && msg.data) {
            handleStateMsg(msg.data);
            handleStateRoom(msg.data);
          }
        } else if (msg.type === 'prompt') {
          // Deliberately not rendered. pkg/session/session_send.go:246 states
          // the contract: "Telnet renders it as the '> ' command prompt;
          // WebSocket clients may ignore it." It reaches browsers for real —
          // Manager.flushAsyncPrompts iterates every session, not just telnet
          // ones, so an idle player who receives pulse output gets one — and
          // before this branch existed it fell through and wrote its own JSON
          // envelope into the terminal.
        } else if (msg.type === 'token_refresh') {
          // Only sessions issued a JWT receive these: maybeRefreshToken
          // returns early unless tokenIssuedAt is set, and a browser that
          // logs in by name never has one. Recognised so that an agent-shaped
          // session driving this client cannot corrupt the terminal with it.
        } else {
          // Never write an unrecognised frame to the terminal. evt.data is the
          // raw JSON envelope, so doing that prints protocol at the player
          // instead of game text — which is exactly how `prompt` surfaced.
          // Game text arrives as 'text' or 'event' and is handled above.
          if (typeof console !== 'undefined' && console.debug) {
            console.debug('mud-client: unhandled message type', msg.type);
          }
        }
      } catch {
        // Not JSON at all. The server frames everything it sends, so this is
        // either a proxy injecting something or a protocol change; the raw
        // text is the most useful thing to show and cannot be an envelope.
        term.write(evt.data);
      }
    };

    ws.onclose = function () {
      setStatus('disconnected');
      term.writeln('\x1b[31m\r\n--- Connection lost ---\x1b[0m');
      loggedIn = false;
      inCharCreation = false;
      charInputSecret = false;
      awaitingEntryReply = false;
      queuedInput = '';
      inputBuffer = '';
      if (statusBar) statusBar.classList.add('hidden');
    };

    ws.onerror = function () {
      term.writeln('\x1b[31m\r\nConnection error.\x1b[0m');
    };
  }

  // Typeahead waits for each server-owned entry state before echoing. A paste
  // containing a name and password must not echo the password as part of the name.
  term.onData(function (data) {
    if (!ws || ws.readyState !== WebSocket.OPEN) return;
    queuedInput += data.replace(/\r\n/g, '\r');
    drainInput();
  });

  function drainInput() {
    while (!awaitingEntryReply && queuedInput.length) {
      const character = queuedInput[0];
      queuedInput = queuedInput.slice(1);
      handleInputCharacter(character);
    }
  }

  function handleInputCharacter(data) {

    if (data === '\r' || data === '\n') {
      if (!charInputSecret) term.writeln('');
      const input = inputBuffer;
      inputBuffer = '';

      if (!loggedIn && !inCharCreation) {
        if (!awaitingEntryReply) {
          awaitingEntryReply = true;
          ws.send(JSON.stringify({ type: 'login', data: { player_name: input.trim() } }));
        }
        return;
      }

      if (inCharCreation) {
        awaitingEntryReply = true;
        ws.send(JSON.stringify({ type: 'char_input', data: { choice: input } }));
        return;
      }

      // Normal command execution
      ws.send(JSON.stringify({ type: 'command', data: { command: input } }));
    } else if (data === '\x7f' || data === '\b') {
      if (inputBuffer.length > 0) {
        inputBuffer = inputBuffer.slice(0, -1);
        if (!charInputSecret) {
          term.write('\b \b');
        }
      }
    } else if (data.charCodeAt(0) >= 32) {
      inputBuffer += data;
      if (!charInputSecret) {
        term.write(data);
      }
    }
  }

  function resetSession() {
    loggedIn = false;
    inCharCreation = false;
    charInputSecret = false;
    awaitingEntryReply = false;
    inputBuffer = '';
    queuedInput = '';
  }

  if (reconnectBtn) {
    reconnectBtn.addEventListener('click', function () {
      resetSession();
      connect();
    });
  }

  connect();

  // The host owns the lifetime: a React component unmounts, a page does not.
  // Without disconnect() the admin console's Terminal would leak a socket on
  // every navigation away from it.
  return {
    connect: function () {
      resetSession();
      connect();
    },
    disconnect: function () {
      if (ws) {
        ws.onclose = null;
        ws.close();
        ws = null;
      }
      resetSession();
    },
  };
}
