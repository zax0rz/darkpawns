(function () {
  'use strict';

  const params = new URLSearchParams(location.search);
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  const wsUrl = params.get('host') || `${proto}//${location.host}/ws`;

  const term = new Terminal({
    cursorBlink: true,
    // Game text (MOTD, room/help files) uses bare "\n" line endings. Without
    // convertEol, xterm treats a lone LF as line-feed-only — the cursor drops a
    // row but keeps its column — producing runaway "staircase" indentation.
    // convertEol makes every "\n" behave as "\r\n" so lines return to column 0.
    convertEol: true,
    fontSize: 15,
    fontFamily: '"IM Fell English", "Courier New", monospace',
    theme: {
      background: '#0a0908',
      foreground: '#c8b896',
      cursor: '#8b0000',
      selectionBackground: '#3a2a1a',
    },
  });
  const fitAddon = new FitAddon.FitAddon();
  term.loadAddon(fitAddon);
  term.open(document.getElementById('terminal'));
  fitAddon.fit();

  window.addEventListener('resize', () => fitAddon.fit());

  const statusEl = document.querySelector('.conn-status');
  const reconnectBtn = document.getElementById('reconnect-btn');
  const statusBar = document.getElementById('status-bar');
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

  function pct(cur, max) {
    return max > 0 ? Math.round((cur / max) * 100) : 0;
  }

  function hpColor(p) {
    if (p > 75) return '#4a8a4a';
    if (p > 25) return '#b8960a';
    return '#8b0000';
  }

  function manaColor(p) {
    if (p > 75) return '#3a6a9a';
    if (p > 25) return '#2a5a7a';
    return '#1a3a5a';
  }

  function moveColor(p) {
    if (p > 75) return '#6a8a3a';
    if (p > 25) return '#8a7a2a';
    return '#5a4a1a';
  }

  function updateBar(id, cur, max, colorFn) {
    const bar = document.getElementById(id);
    if (!bar) return;
    const p = pct(cur, max);
    bar.style.width = (max > 0 ? p : 0) + '%';
    bar.style.backgroundColor = colorFn(p);
  }

  function updateStatusBar() {
    if (!statusBar) return;
    updateBar('hp-bar', playerState.health, playerState.maxHealth, hpColor);
    const hpText = document.getElementById('hp-text');
    if (hpText) hpText.textContent = playerState.maxHealth > 0 ? `${playerState.health}/${playerState.maxHealth}` : '—';

    updateBar('mana-bar', playerState.mana, playerState.maxMana, manaColor);
    const manaText = document.getElementById('mana-text');
    if (manaText) manaText.textContent = playerState.maxMana > 0 ? `${playerState.mana}/${playerState.maxMana}` : '—';

    updateBar('move-bar', playerState.move, playerState.maxMove, moveColor);
    const moveText = document.getElementById('move-text');
    if (moveText) moveText.textContent = playerState.maxMove > 0 ? `${playerState.move}/${playerState.maxMove}` : '—';

    const lvlInfo = document.getElementById('level-info');
    if (lvlInfo) lvlInfo.textContent = playerState.level > 0 ? `Lv ${playerState.level}` : 'Lv —';

    const goldInfo = document.getElementById('gold-info');
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
    const connectPanel = document.getElementById('sidebar-connect-panel');
    if (connectPanel) connectPanel.classList.add('hidden');

    const panels = document.querySelectorAll('.sidebar-panel');
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
    const container = document.getElementById('minimap-container');
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
    const container = document.getElementById('room-contents-container');
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
    const container = document.getElementById('inventory-equipment-container');
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

    document.getElementById('tab-btn-inv')?.addEventListener('click', () => {
      activeTab = 'inventory';
      updateInventoryEquipment();
    });
    document.getElementById('tab-btn-eq')?.addEventListener('click', () => {
      activeTab = 'equipment';
      updateInventoryEquipment();
    });
  }

  function updateTargetDisplay(fightingData) {
    const container = document.getElementById('target-container');
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
  let loginStage = 'name'; // 'name', 'password', 'new_char', 'confirm_password'
  let username = '';
  let password = '';

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
    const connectPanel = document.getElementById('sidebar-connect-panel');
    if (connectPanel) connectPanel.classList.add('hidden');
    document.querySelectorAll('.sidebar-panel').forEach(p => p.classList.remove('hidden'));
    if (r.vnum) {
      playerState.roomVnum = r.vnum;
      updateMinimap(r.vnum);
    }
    if (r.mobs !== undefined || r.items !== undefined) {
      updateRoomContents(r.mobs, r.items);
    }
  }

  function setStatus(state) {
    statusEl.className = 'conn-status ' + state;
    const label = state === 'connected' ? 'Connected' : 'Disconnected';
    statusEl.querySelector('span:last-child').textContent = label;
    reconnectBtn.classList.toggle('visible', state === 'disconnected');
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
          term.write('\r\n' + msg.data.prompt + '\r\n');
          const promptContainsOptions = msg.data.prompt.includes('0) Exit from Dark Pawns') || msg.data.prompt.includes('[');
          if (msg.data.options && !promptContainsOptions) {
            const options = Array.isArray(msg.data.options)
              ? msg.data.options
              : Object.entries(msg.data.options).map(([key, label]) => ({ key, label }));
            for (const option of options) {
              term.write(`  [${option.key}] - ${option.label}\r\n`);
            }
          }
          term.write('> ');
        } else if (msg.type === 'error') {
          term.write('\r\n\x1b[31m' + msg.data.message + '\x1b[0m\r\n');
          // Reset login flow on failure so they can try again
          if (!loggedIn && !inCharCreation) {
            term.write('\r\nBy what name do you wish to be known? ');
            loginStage = 'name';
            username = '';
            password = '';
          }
        } else if (msg.type === 'state') {
          if (!loggedIn && msg.data && msg.data.player && msg.data.player.name) {
            loggedIn = true;
            inCharCreation = false;
            charInputSecret = false;
          }
          if (loggedIn && msg.data) {
            handleStateMsg(msg.data);
            handleStateRoom(msg.data);
          }
        } else {
          term.write(evt.data);
        }
      } catch {
        term.write(evt.data);
      }
    };

    ws.onclose = function () {
      setStatus('disconnected');
      term.writeln('\x1b[31m\r\n--- Connection lost ---\x1b[0m');
      loggedIn = false;
      inCharCreation = false;
      charInputSecret = false;
      loginStage = 'name';
      username = '';
      password = '';
      if (statusBar) statusBar.classList.add('hidden');
    };

    ws.onerror = function () {
      term.writeln('\x1b[31m\r\nConnection error.\x1b[0m');
    };
  }

  term.onData(function (data) {
    if (!ws || ws.readyState !== WebSocket.OPEN) return;

    if (data === '\r' || data === '\n') {
      term.writeln('');
      const input = inputBuffer;
      inputBuffer = '';

      if (!loggedIn && !inCharCreation) {
        // Stateful terminal authentication flow
        if (loginStage === 'name') {
          username = input.trim();
          if (username) {
            if (username.toLowerCase().startsWith('guest')) {
              // Bypasses password prompt and goes directly to guest login!
              ws.send(JSON.stringify({
                type: 'login',
                data: { player_name: username, password: '', new_char: false }
              }));
              term.write('Connecting as Guest...\r\n');
              loginStage = 'name';
            } else {
              term.write('Password: ');
              loginStage = 'password';
            }
          } else {
            term.write('By what name do you wish to be known? ');
          }
        } else if (loginStage === 'password') {
          password = input;
          term.write('\r\nIs this a new character? (y/n): ');
          loginStage = 'new_char';
        } else if (loginStage === 'new_char') {
          const choice = input.trim().toLowerCase();
          if (choice === 'y' || choice === 'yes') {
            term.write('Confirm password: ');
            loginStage = 'confirm_password';
          } else {
            // Returning character login
            ws.send(JSON.stringify({
              type: 'login',
              data: { player_name: username, password: password, new_char: false }
            }));
            term.write('Connecting...\r\n');
            loginStage = 'name'; // reset in case login fails
          }
        } else if (loginStage === 'confirm_password') {
          const confirm = input;
          if (confirm !== password) {
            term.write('\x1b[31mPasswords do not match.\x1b[0m\r\nBy what name do you wish to be known? ');
            loginStage = 'name';
            username = '';
            password = '';
          } else {
            // New character login
            ws.send(JSON.stringify({
              type: 'login',
              data: { player_name: username, password: password, new_char: true }
            }));
            term.write('Creating character...\r\n');
            loginStage = 'name'; // reset in case login fails
          }
        }
        return;
      }

      if (inCharCreation) {
        ws.send(JSON.stringify({ type: 'char_input', data: { choice: input } }));
        return;
      }

      // Normal command execution
      ws.send(JSON.stringify({ type: 'command', data: { command: input } }));
    } else if (data === '\x7f' || data === '\b') {
      if (inputBuffer.length > 0) {
        inputBuffer = inputBuffer.slice(0, -1);
        if (loginStage !== 'password' && loginStage !== 'confirm_password' && !charInputSecret) {
          term.write('\b \b');
        }
      }
    } else if (data.charCodeAt(0) >= 32) {
      inputBuffer += data;
      if (loginStage !== 'password' && loginStage !== 'confirm_password' && !charInputSecret) {
        term.write(data);
      }
    }
  });

  reconnectBtn.addEventListener('click', function () {
    loggedIn = false;
    inCharCreation = false;
    charInputSecret = false;
    loginStage = 'name';
    username = '';
    password = '';
    inputBuffer = '';
    connect();
  });

  connect();
})();
