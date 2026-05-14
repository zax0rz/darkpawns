import { useEffect, useRef, useState, useCallback } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { WebLinksAddon } from '@xterm/addon-web-links';
import '@xterm/xterm/css/xterm.css';
import { useAuth } from '../hooks/useAuth';

const theme = {
  background: '#0a0908',
  foreground: '#c8b896',
  cursor: '#8b0000',
  selectionBackground: '#3a2a1a',
};

interface PlayerState {
  health: number;
  maxHealth: number;
  mana: number;
  maxMana: number;
  move: number;
  maxMove: number;
  level: number;
  gold: number;
}

function pct(cur: number, max: number) {
  return max > 0 ? Math.round((cur / max) * 100) : 0;
}

function hpColor(p: number) {
  if (p > 75) return '#4a8a4a';
  if (p > 25) return '#b8960a';
  return '#8b0000';
}

function manaColor(p: number) {
  if (p > 75) return '#3a6a9a';
  if (p > 25) return '#2a5a7a';
  return '#1a3a5a';
}

function moveColor(p: number) {
  if (p > 75) return '#6a8a3a';
  if (p > 25) return '#8a7a2a';
  return '#5a4a1a';
}

function StatusRow({
  label,
  cur,
  max,
  colorFn,
}: {
  label: string;
  cur: number;
  max: number;
  colorFn: (p: number) => string;
}) {
  const p = pct(cur, max);
  return (
    <div className="flex items-center gap-1.5 min-w-[140px]">
      <span className="text-[11px] text-slate-400 w-10 shrink-0">{label}</span>
      <div className="flex-1 h-3 bg-slate-800 rounded overflow-hidden">
        <div
          className="h-full transition-all duration-300"
          style={{
            width: max > 0 ? `${p}%` : '0%',
            backgroundColor: colorFn(p),
          }}
        />
      </div>
      <span className="text-[11px] text-slate-300 w-20 text-right tabular-nums">
        {max > 0 ? `${cur}/${max}` : '—'}
      </span>
    </div>
  );
}

interface MudTerminalProps {
  className?: string;
}

export function MudTerminal({ className = '' }: MudTerminalProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const termRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const inputBufferRef = useRef('');
  const loggedInRef = useRef(false);
  const { playerName } = useAuth();

  const [connected, setConnected] = useState(false);
  const [playerState, setPlayerState] = useState<PlayerState>({
    health: 0, maxHealth: 0,
    mana: 0, maxMana: 0,
    move: 0, maxMove: 0,
    level: 0, gold: 0,
  });

  const handleStateMsg = useCallback((data: any) => {
    if (!data?.player) return;
    const p = data.player;
    setPlayerState((prev) => ({
      health: p.health || 0,
      maxHealth: p.max_health || 0,
      mana: p.mana ?? prev.mana,
      maxMana: p.max_mana ?? prev.maxMana,
      move: p.move ?? prev.move,
      maxMove: p.max_move ?? prev.maxMove,
      level: p.level || 0,
      gold: p.gold ?? prev.gold,
    }));
  }, []);

  const handleVarsMsg = useCallback((data: any) => {
    if (!data) return;
    setPlayerState((prev) => {
      const next = { ...prev };
      if (data.HEALTH !== undefined) next.health = data.HEALTH;
      if (data.MAX_HEALTH !== undefined) next.maxHealth = data.MAX_HEALTH;
      if (data.MANA !== undefined) next.mana = data.MANA;
      if (data.MAX_MANA !== undefined) next.maxMana = data.MAX_MANA;
      if (data.MOVE !== undefined) next.move = data.MOVE;
      if (data.MAX_MOVE !== undefined) next.maxMove = data.MAX_MOVE;
      if (data.LEVEL !== undefined) next.level = data.LEVEL;
      if (data.GOLD !== undefined) next.gold = data.GOLD;
      return next;
    });
  }, []);

  const connect = useCallback(() => {
    const term = termRef.current;
    if (!term) return;

    // Close existing connection
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }

    setConnected(false);
    loggedInRef.current = false;
    inputBufferRef.current = '';

    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    // Dev proxy rewrites /ws, so just use origin-relative URL
    const wsUrl = `${proto}//${window.location.host}/ws`;

    term.writeln('\x1b[2mConnecting...\x1b[0m');

    try {
      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;
    } catch (e: any) {
      term.writeln(`\x1b[31mConnection failed: ${e.message}\x1b[0m`);
      return;
    }

    const ws = wsRef.current!;

    ws.onopen = () => {
      setConnected(true);
      term.writeln('\x1b[32mConnected.\x1b[0m');
    };

    ws.onmessage = (evt) => {
      let text: string | undefined;
      try {
        const msg = JSON.parse(evt.data);

        if (msg.type === 'state') {
          handleStateMsg(msg.data);
          return;
        }
        if (msg.type === 'vars') {
          handleVarsMsg(msg.data);
          return;
        }
        if (msg.type === 'char_create') {
          if (msg.data?.prompt) {
            term.writeln(msg.data.prompt);
          }
          return;
        }
        if (msg.type === 'error') {
          text = `\x1b[31m${msg.data?.message || evt.data}\x1b[0m`;
        } else if (msg.type === 'event') {
          text = msg.data?.text || '';
        } else if (msg.type === 'text') {
          text = msg.data?.text || evt.data;
        } else {
          text = msg.text || evt.data;
        }
      } catch {
        text = evt.data;
      }
      if (text) term.writeln(text);
    };

    ws.onclose = () => {
      setConnected(false);
      term.writeln('\x1b[31m--- Connection lost ---\x1b[0m');
    };

    ws.onerror = () => {
      term.writeln('\x1b[31mConnection error.\x1b[0m');
    };

    // Auto-login with admin player name
    const name = playerName || 'Admin';
    term.writeln(`Enter your character name [auto-logging as ${name}]:`);
    setTimeout(() => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'login', data: { player_name: name } }));
        loggedInRef.current = true;
        inputBufferRef.current = '';
      }
    }, 500);
  }, [playerName, handleStateMsg, handleVarsMsg]);

  // Initialize terminal
  useEffect(() => {
    if (!containerRef.current) return;

    const term = new Terminal({
      cursorBlink: true,
      fontSize: 15,
      fontFamily: '"IM Fell English", "Courier New", monospace',
      theme,
    });

    const fitAddon = new FitAddon();
    const webLinksAddon = new WebLinksAddon();
    term.loadAddon(fitAddon);
    term.loadAddon(webLinksAddon);
    term.open(containerRef.current);
    fitAddon.fit();

    termRef.current = term;
    fitAddonRef.current = fitAddon;

    const handleResize = () => fitAddon.fit();
    window.addEventListener('resize', handleResize);

    // Handle user input
    const disposable = term.onData((data: string) => {
      const ws = wsRef.current;
      if (!ws || ws.readyState !== WebSocket.OPEN) return;

      if (data === '\r' || data === '\n') {
        term.writeln('');
        if (!loggedInRef.current) {
          const name = inputBufferRef.current.trim();
          if (name) {
            ws.send(JSON.stringify({ type: 'login', data: { player_name: name } }));
            loggedInRef.current = true;
          }
          inputBufferRef.current = '';
          return;
        }
        if (inputBufferRef.current.trim()) {
          ws.send(
            JSON.stringify({
              type: 'command',
              data: { command: inputBufferRef.current },
            })
          );
        }
        inputBufferRef.current = '';
      } else if (data === '\x7f' || data === '\b') {
        if (inputBufferRef.current.length > 0) {
          inputBufferRef.current = inputBufferRef.current.slice(0, -1);
          term.write('\b \b');
        }
      } else if (data >= ' ') {
        inputBufferRef.current += data;
        term.write(data);
      }
    });

    // Connect on mount
    connect();

    return () => {
      disposable.dispose();
      window.removeEventListener('resize', handleResize);
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
      term.dispose();
      termRef.current = null;
      fitAddonRef.current = null;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const showStatusBar = playerState.maxHealth > 0;

  return (
    <div className={`flex flex-col ${className}`}>
      {/* Terminal area */}
      <div ref={containerRef} className="flex-1 min-h-0" />

      {/* Status bar */}
      {showStatusBar && (
        <div className="flex items-center gap-4 px-3 py-1.5 bg-slate-950 border-t border-slate-700 flex-wrap">
          <StatusRow label="HP" cur={playerState.health} max={playerState.maxHealth} colorFn={hpColor} />
          <StatusRow label="Mana" cur={playerState.mana} max={playerState.maxMana} colorFn={manaColor} />
          <StatusRow label="Move" cur={playerState.move} max={playerState.maxMove} colorFn={moveColor} />
          <span className="text-[11px] text-slate-400">
            Lv {playerState.level || '—'}
          </span>
          <span className="text-[11px] text-amber-400">
            Gold {playerState.gold || 0}
          </span>
        </div>
      )}

      {/* Connection status + reconnect */}
      <div className="flex items-center gap-3 px-3 py-1 bg-slate-950 border-t border-slate-800 text-xs">
        <span className="flex items-center gap-1.5">
          <span
            className={`w-2 h-2 rounded-full ${
              connected ? 'bg-green-500' : 'bg-red-500'
            }`}
          />
          {connected ? 'Connected' : 'Disconnected'}
        </span>
        {!connected && (
          <button
            onClick={connect}
            className="text-amber-400 hover:text-amber-300 transition-colors"
          >
            Reconnect
          </button>
        )}
      </div>
    </div>
  );
}
