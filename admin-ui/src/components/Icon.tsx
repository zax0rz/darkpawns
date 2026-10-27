import { LayoutDashboard, Map, Skull, Package, Network, SlidersHorizontal, Terminal, Search, RefreshCw, X, Check, Info, Users, Bell, Store, BookOpen, type LucideIcon } from 'lucide-react';

const icons = { dashboard: LayoutDashboard, zones: Map, mobs: Skull, objects: Package,
  agents: Network, operations: SlidersHorizontal, terminal: Terminal, search: Search,
  refresh: RefreshCw, close: X, check: Check, cross: X, info: Info,
  players: Users, notifications: Bell, shops: Store, guide: BookOpen } satisfies Record<string, LucideIcon>;
export type IconName = keyof typeof icons | 'pawn';
export function Icon({ name, className = 'h-4 w-4' }: { name: IconName; className?: string }) {
  if (name === 'pawn') return <svg viewBox="24 8 52 88" className={className} fill="currentColor" aria-hidden="true" focusable="false"><circle cx="50" cy="23" r="12"/><rect x="37" y="38" width="26" height="5"/><polygon points="43,46 57,46 62,75 38,75"/><rect x="31" y="77" width="38" height="6"/><rect x="26" y="85" width="48" height="7"/></svg>;
  const Glyph = icons[name];
  return <Glyph className={className} strokeWidth={1.5} aria-hidden="true" focusable="false"/>;
}
