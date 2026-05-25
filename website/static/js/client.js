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
  let inputBuffer = '';
  let ws;

  let loggedIn = false;
  let inCharCreation = false;
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

  function renderRoom(data) {
    const r = data.room;
    if (!r) return;
    term.writeln('\r\n\x1b[1;36m' + (r.name || '') + '\x1b[0m');
    if (r.description) {
      term.write(r.description.replace(/\r?\n/g, '\r\n'));
      if (!/\n$/.test(r.description)) term.writeln('');
    }
    const exits = (r.exits && r.exits.length) ? r.exits.join(' ') : 'none';
    term.writeln('\x1b[32m[ Exits: ' + exits + ' ]\x1b[0m');
    if (r.mobs) r.mobs.forEach(function(m) { term.writeln('\x1b[33m' + m + '\x1b[0m'); });
    if (r.players) r.players.forEach(function(p) { term.writeln(p + ' is here.'); });
    if (r.items) r.items.forEach(function(i) { term.writeln(i); });
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
        } else if (msg.type === 'char_create') {
          inCharCreation = true;
          loggedIn = false;
          term.write('\r\n' + msg.data.prompt + '\r\n');
          if (msg.data.options) {
            for (const [key, desc] of Object.entries(msg.data.options)) {
              term.write(`  [${key}] - ${desc}\r\n`);
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
          }
          if (loggedIn && msg.data) {
            renderRoom(msg.data);
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
            term.write('Password: ');
            loginStage = 'password';
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
        if (input.trim()) {
          ws.send(JSON.stringify({ type: 'char_input', data: { choice: input.trim() } }));
        }
        return;
      }

      // Normal command execution
      ws.send(JSON.stringify({ type: 'command', data: { command: input } }));
    } else if (data === '\x7f' || data === '\b') {
      if (inputBuffer.length > 0) {
        inputBuffer = inputBuffer.slice(0, -1);
        if (loginStage !== 'password' && loginStage !== 'confirm_password') {
          term.write('\b \b');
        }
      }
    } else if (data.charCodeAt(0) >= 32) {
      inputBuffer += data;
      if (loginStage !== 'password' && loginStage !== 'confirm_password') {
        term.write(data);
      }
    }
  });

  reconnectBtn.addEventListener('click', function () {
    loggedIn = false;
    inCharCreation = false;
    loginStage = 'name';
    username = '';
    password = '';
    inputBuffer = '';
    connect();
  });

  connect();
})();
