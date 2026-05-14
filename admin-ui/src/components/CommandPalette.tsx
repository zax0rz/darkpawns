import { useCallback, useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';

interface CommandItem {
  id: string;
  label: string;
  icon?: string;
  group: string;
  action: () => void;
}

const STORAGE_KEY = 'dp_recent_commands';

function getRecent(): string[] {
  try {
    return JSON.parse(localStorage.getItem(STORAGE_KEY) || '[]');
  } catch {
    return [];
  }
}

function addRecent(id: string) {
  const recent = getRecent().filter((r) => r !== id);
  recent.unshift(id);
  localStorage.setItem(STORAGE_KEY, JSON.stringify(recent.slice(0, 8)));
}

export function useCommandPalette() {
  const [open, setOpen] = useState(false);

  const openPalette = useCallback(() => setOpen(true), []);
  const closePalette = useCallback(() => setOpen(false), []);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        setOpen((prev) => !prev);
      }
      if (e.key === 'Escape') {
        setOpen(false);
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, []);

  return { open, openPalette, closePalette };
}

function fuzzyMatch(query: string, text: string): boolean {
  const q = query.toLowerCase();
  const t = text.toLowerCase();
  if (t.includes(q)) return true;
  let qi = 0;
  for (let ti = 0; ti < t.length && qi < q.length; ti++) {
    if (t[ti] === q[qi]) qi++;
  }
  return qi === q.length;
}

export function CommandPalette({
  open,
  onClose,
}: {
  open: boolean;
  onClose: () => void;
}) {
  const navigate = useNavigate();
  const [query, setQuery] = useState('');
  const [selectedIndex, setSelectedIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);

  const commands: CommandItem[] = [
    { id: 'nav-dashboard', label: 'Go to Dashboard', icon: '📊', group: 'Navigation', action: () => navigate('/admin/') },
    { id: 'nav-zones', label: 'Go to Zones', icon: '🗺️', group: 'Navigation', action: () => navigate('/admin/game/zones') },
    { id: 'nav-mobs', label: 'Go to Mobs', icon: '🐉', group: 'Navigation', action: () => navigate('/admin/game/mobs') },
    { id: 'nav-objects', label: 'Go to Objects', icon: '💎', group: 'Navigation', action: () => navigate('/admin/game/objects') },
    { id: 'nav-terminal', label: 'Go to Terminal', icon: '🖥️', group: 'Navigation', action: () => navigate('/admin/webclient') },
    { id: 'nav-operations', label: 'Go to Operations', icon: '⚙️', group: 'Navigation', action: () => navigate('/admin/operations') },
    { id: 'nav-agents', label: 'Go to Agents', icon: '🤖', group: 'Navigation', action: () => navigate('/admin/agents') },
    { id: 'action-reek', label: 'Trigger Reek Crawl', icon: '🔍', group: 'Actions', action: () => { /* placeholder */ } },
    { id: 'action-refresh', label: 'Refresh Server Status', icon: '🔄', group: 'Actions', action: () => window.location.reload() },
  ];

  // Add recent items
  const recentIds = getRecent();
  const recentItems: CommandItem[] = recentIds
    .map((rid) => commands.find((c) => c.id === rid))
    .filter(Boolean) as CommandItem[];

  const filtered = query
    ? commands.filter((c) => fuzzyMatch(query, c.label))
    : [...recentItems, ...commands.filter((c) => !recentIds.includes(c.id))];

  // Deduplicate
  const seen = new Set<string>();
  const items = filtered.filter((c) => {
    if (seen.has(c.id)) return false;
    seen.add(c.id);
    return true;
  });

  useEffect(() => {
    setSelectedIndex(0);
  }, [query]);

  useEffect(() => {
    if (open) {
      setQuery('');
      setTimeout(() => inputRef.current?.focus(), 50);
    }
  }, [open]);

  const execute = (item: CommandItem) => {
    addRecent(item.id);
    onClose();
    item.action();
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setSelectedIndex((prev) => Math.min(prev + 1, items.length - 1));
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setSelectedIndex((prev) => Math.max(prev - 1, 0));
    } else if (e.key === 'Enter' && items[selectedIndex]) {
      execute(items[selectedIndex]);
    } else if (e.key === 'Escape') {
      onClose();
    }
  };

  if (!open) return null;

  // Group items
  const groups: { name: string; items: CommandItem[] }[] = [];
  const groupMap = new Map<string, CommandItem[]>();
  for (const item of items) {
    const g = query ? 'Results' : (item.id.startsWith('nav-') || recentIds.includes(item.id) ? (recentIds.includes(item.id) ? 'Recently Visited' : item.group) : item.group);
    if (!groupMap.has(g)) groupMap.set(g, []);
    groupMap.get(g)!.push(item);
  }
  // Order: Recently Visited first, then Results, then others
  const groupOrder = ['Recently Visited', 'Results', 'Navigation', 'Actions'];
  for (const name of groupOrder) {
    const g = groupMap.get(name);
    if (g) groups.push({ name, items: g });
  }
  for (const [name, items] of groupMap) {
    if (!groupOrder.includes(name)) groups.push({ name, items });
  }

  let flatIndex = 0;

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-[15vh]">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />

      {/* Palette */}
      <div className="relative w-full max-w-lg bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-xl shadow-2xl overflow-hidden">
        {/* Search input */}
        <div className="flex items-center border-b border-slate-200 dark:border-slate-700 px-4">
          <span className="text-slate-400 mr-2">🔍</span>
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Type a command..."
            className="flex-1 bg-transparent py-3 text-sm text-slate-900 dark:text-slate-100 placeholder-slate-400 focus:outline-none"
          />
          <kbd className="text-[10px] text-slate-400 bg-slate-100 dark:bg-slate-800 px-1.5 py-0.5 rounded border border-slate-200 dark:border-slate-700">
            ESC
          </kbd>
        </div>

        {/* Results */}
        <div className="max-h-72 overflow-y-auto py-2">
          {items.length === 0 ? (
            <div className="px-4 py-6 text-center text-sm text-slate-500">
              No results found
            </div>
          ) : (
            groups.map((group) => (
              <div key={group.name}>
                <div className="px-4 py-1.5 text-[10px] font-semibold uppercase tracking-wider text-slate-400">
                  {group.name}
                </div>
                {group.items.map((item) => {
                  const idx = flatIndex++;
                  const isSelected = idx === selectedIndex;
                  return (
                    <button
                      key={item.id}
                      onClick={() => execute(item)}
                      onMouseEnter={() => setSelectedIndex(idx)}
                      className={`w-full flex items-center gap-2 px-4 py-2 text-sm text-left transition-colors ${
                        isSelected
                          ? 'bg-amber-600/20 text-amber-300 dark:text-amber-300 text-slate-900'
                          : 'text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800'
                      }`}
                    >
                      {item.icon && <span>{item.icon}</span>}
                      <span>{item.label}</span>
                    </button>
                  );
                })}
              </div>
            ))
          )}
        </div>

        {/* Footer */}
        <div className="border-t border-slate-200 dark:border-slate-700 px-4 py-2 flex items-center gap-3 text-[10px] text-slate-400">
          <span>↑↓ Navigate</span>
          <span>↵ Select</span>
          <span>Esc Close</span>
          <span className="ml-auto">Ctrl+K</span>
        </div>
      </div>
    </div>
  );
}
