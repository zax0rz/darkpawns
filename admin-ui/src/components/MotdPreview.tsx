import { useEffect, useRef } from 'react';
import { Terminal } from '@xterm/xterm';
import '@xterm/xterm/css/xterm.css';
import { motdTerminalText } from '../lib/motd';

export function MotdPreview({ text, columns, colour }: { text: string; columns: number; colour: boolean }) {
  const host = useRef<HTMLDivElement>(null);
  const terminal = useRef<Terminal | null>(null);
  useEffect(() => {
    if (!host.current) return;
    const term = new Terminal({ cols: columns, rows: 22, scrollback: 2500, disableStdin: true,
      convertEol: true, fontFamily: '"JetBrains Mono", monospace', fontSize: 13, lineHeight: 1.35,
      cursorBlink: false, screenReaderMode: true,
      theme: { background: '#0a0908', foreground: '#c8b896', cursor: '#0a0908', selectionBackground: '#3a2a1a' },
    });
    term.open(host.current);
    terminal.current = term;
    return () => { terminal.current = null; term.dispose(); };
  }, [columns]);
  useEffect(() => {
    const term = terminal.current;
    if (!term) return;
    term.write('\x1bc' + motdTerminalText(text, colour), () => { if (terminal.current === term) term.scrollToTop(); });
  }, [text, columns, colour]);
  return <div className="max-w-full overflow-x-auto border border-rule bg-[#0a0908] p-3" role="region" aria-label={`Terminal preview, ${columns} columns`} tabIndex={0}>
    <div ref={host} style={{ width: `${columns * 7.83 + 16}px` }} />
  </div>;
}
