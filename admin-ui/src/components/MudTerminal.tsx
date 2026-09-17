import { useEffect, useRef, useState, useCallback } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { WebLinksAddon } from '@xterm/addon-web-links';
import '@xterm/xterm/css/xterm.css';
import { useAuth } from '../hooks/useAuth';
// The one client. Lives in web/public so the splash, which has no build step,
// can serve the same file verbatim; Vite bundles it for this app.
import { createMudClient } from '../../../web/public/mud-client.js';

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
      <span className="text-[11px] text-ink-muted w-10 shrink-0">{label}</span>
      <div className="flex-1 h-3 bg-paper-deep rounded overflow-hidden">
        <div
          className="h-full transition-all duration-300"
          style={{
            width: max > 0 ? `${p}%` : '0%',
            backgroundColor: colorFn(p),
          }}
        />
      </div>
      <span className="text-[11px] text-ink-muted w-20 text-right tabular-nums">
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
  const clientRef = useRef<{ connect: () => void; disconnect: () => void } | null>(null);
  const { playerName } = useAuth();

  const [connected, setConnected] = useState(false);
  const [playerState, setPlayerState] = useState<PlayerState>({
    health: 0, maxHealth: 0,
    mana: 0, maxMana: 0,
    move: 0, maxMove: 0,
    level: 0, gold: 0,
  });

  // The protocol, the login and character-creation state machine, typeahead and
  // secret masking all live in the shared client. This component owns only what
  // is genuinely React's: the terminal instance, and a status bar rendered from
  // component state rather than by writing to DOM nodes.
  //
  // Before this, the admin Terminal spoke its own dialect of the protocol and
  // could display char_create prompts but never sent char_input, so a character
  // could not be created here at all.
  useEffect(() => {
    if (!containerRef.current) return;

    const term = new Terminal({
      cursorBlink: true,
      convertEol: true,
      fontSize: 15,
      fontFamily: '"IM Fell English", "Courier New", monospace',
      theme,
    });

    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.loadAddon(new WebLinksAddon());
    term.open(containerRef.current);
    fitAddon.fit();

    const handleResize = () => fitAddon.fit();
    window.addEventListener('resize', handleResize);

    clientRef.current = createMudClient({
      terminal: term,
      autoLogin: playerName || 'Admin',
      onStatus: (state: string) => setConnected(state === 'connected'),
      onPlayerState: (next: PlayerState) => setPlayerState(next),
    });

    return () => {
      window.removeEventListener('resize', handleResize);
      // Without this the socket outlives every navigation away from the page.
      clientRef.current?.disconnect();
      clientRef.current = null;
      term.dispose();
    };
  }, [playerName]);

  const reconnect = useCallback(() => clientRef.current?.connect(), []);

  const showStatusBar = playerState.maxHealth > 0;

  return (
    <div className={`flex flex-col ${className}`}>
      {/* Terminal area */}
      <div ref={containerRef} className="flex-1 min-h-0" />

      {/* Status bar */}
      {showStatusBar && (
        <div className="flex items-center gap-4 px-3 py-1.5 bg-paper-deep border-t border-rule flex-wrap">
          <StatusRow label="HP" cur={playerState.health} max={playerState.maxHealth} colorFn={hpColor} />
          <StatusRow label="Mana" cur={playerState.mana} max={playerState.maxMana} colorFn={manaColor} />
          <StatusRow label="Move" cur={playerState.move} max={playerState.maxMove} colorFn={moveColor} />
          <span className="text-[11px] text-ink-muted">
            Lv {playerState.level || '—'}
          </span>
          <span className="text-[11px] text-accent">
            Gold {playerState.gold || 0}
          </span>
        </div>
      )}

      {/* Connection status + reconnect */}
      <div className="flex items-center gap-3 px-3 py-1 bg-paper-deep border-t border-rule text-xs">
        <span className="flex items-center gap-1.5">
          <span
            className={`w-2 h-2 rounded-none ${
              connected ? 'bg-online' : 'bg-accent'
            }`}
          />
          {connected ? 'Connected' : 'Disconnected'}
        </span>
        {!connected && (
          <button
            onClick={reconnect}
            className="text-accent hover:text-accent transition-colors"
          >
            Reconnect
          </button>
        )}
      </div>
    </div>
  );
}
