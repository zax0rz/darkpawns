(function () {
  'use strict';

  const params = new URLSearchParams(location.search);
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  const wsUrl = params.get('host') || `${proto}//${location.host}/ws`;

  const term = new Terminal({
    cursorBlink: true,
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
  };

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
    const p = pct(cur, max);
    bar.style.width = (max > 0 ? p : 0) + '%';
    bar.style.backgroundColor = colorFn(p);
  }

  function updateStatusBar() {
    updateBar('hp-bar', playerState.health, playerState.maxHealth, hpColor);
    document.getElementById('hp-text').textContent =
      playerState.maxHealth > 0 ? `${playerState.health}/${playerState.maxHealth}` : '—';

    updateBar('mana-bar', playerState.mana, playerState.maxMana, manaColor);
    document.getElementById('mana-text').textContent =
      playerState.maxMana > 0 ? `${playerState.mana}/${playerState.maxMana}` : '—';

    updateBar('move-bar', playerState.move, playerState.maxMove, moveColor);
    document.getElementById('move-text').textContent =
      playerState.maxMove > 0 ? `${playerState.move}/${playerState.maxMove}` : '—';

    document.getElementById('level-info').textContent =
      playerState.level > 0 ? `Lv ${playerState.level}` : 'Lv —';
    document.getElementById('gold-info').textContent =
      playerState.gold > 0 ? `Gold ${playerState.gold}` : 'Gold —';

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
    // Future-proof: grab mana/move/gold if server adds them
    if (p.mana !== undefined) playerState.mana = p.mana;
    if (p.max_mana !== undefined) playerState.maxMana = p.max_mana;
    if (p.move !== undefined) playerState.move = p.move;
    if (p.max_move !== undefined) playerState.maxMove = p.max_move;
    if (p.gold !== undefined) playerState.gold = p.gold;
    updateStatusBar();
  }

  function handleVarsMsg(data) {
    if (!data) return;
    if (data.HEALTH !== undefined) playerState.health = data.HEALTH;
    if (data.MAX_HEALTH !== undefined) playerState.maxHealth = data.MAX_HEALTH;
    if (data.MANA !== undefined) playerState.mana = data.MANA;
    if (data.MAX_MANA !== undefined) playerState.maxMana = data.MAX_MANA;
    if (data.LEVEL !== undefined) playerState.level = data.LEVEL;
    // Future: MOVE, MAX_MOVE, GOLD vars when server adds them
    if (data.MOVE !== undefined) playerState.move = data.MOVE;
    if (data.MAX_MOVE !== undefined) playerState.maxMove = data.MAX_MOVE;
    if (data.GOLD !== undefined) playerState.gold = data.GOLD;
    updateStatusBar();
  }

  // ── Connection ──

  function setStatus(state) {
    statusEl.className = 'conn-status ' + state;
    const label = state === 'connected' ? 'Connected' : 'Disconnected';
    statusEl.querySelector('span').textContent = label;
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
      term.writeln('\x1b[32mConnected.\x1b[0m');
      term.writeln('Enter your character name:');
    };

    ws.onmessage = function (evt) {
      let text;
      try {
        const msg = JSON.parse(evt.data);

        // Route structured messages
        if (msg.type === 'state') {
          handleStateMsg(msg.data);
          // Don't write raw state JSON to terminal; room desc comes via events
          return;
        }
        if (msg.type === 'vars') {
          handleVarsMsg(msg.data);
          return; // vars are for status bar only
        }
        if (msg.type === 'char_create') {
          // Show the prompt text for character creation
          if (msg.data && msg.data.prompt) {
            term.writeln(msg.data.prompt);
          }
          return;
        }
        if (msg.type === 'error') {
          text = '\x1b[31m' + (msg.data && msg.data.message || evt.data) + '\x1b[0m';
        } else if (msg.type === 'event') {
          text = (msg.data && msg.data.text) || '';
        } else if (msg.type === 'text') {
          text = (msg.data && msg.data.text) || evt.data;
        } else {
          text = msg.text || evt.data;
        }
      } catch {
        text = evt.data;
      }
      if (text) term.writeln(text);
    };

    ws.onclose = function () {
      setStatus('disconnected');
      term.writeln('\x1b[31m--- Connection lost ---\x1b[0m');
    };

    ws.onerror = function () {
      term.writeln('\x1b[31mConnection error.\x1b[0m');
    };
  }

  let loggedIn = false;

  term.onData(function (data) {
    if (!ws || ws.readyState !== WebSocket.OPEN) return;

    if (data === '\r' || data === '\n') {
      term.writeln('');
      if (!loggedIn) {
        const name = inputBuffer.trim();
        if (name) {
          ws.send(JSON.stringify({ type: 'login', data: { player_name: name } }));
          loggedIn = true;
        }
        inputBuffer = '';
        return;
      }
      if (inputBuffer.trim()) {
        ws.send(JSON.stringify({ type: 'command', data: { command: inputBuffer } }));
      }
      inputBuffer = '';
    } else if (data === '\x7f' || data === '\b') {
      if (inputBuffer.length > 0) {
        inputBuffer = inputBuffer.slice(0, -1);
        term.write('\b \b');
      }
    } else if (data >= ' ') {
      inputBuffer += data;
      term.write(data);
    }
  });

  reconnectBtn.addEventListener('click', function () {
    loggedIn = false;
    inputBuffer = '';
    connect();
  });

  connect();
})();
