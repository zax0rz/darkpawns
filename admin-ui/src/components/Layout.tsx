import { useState, useEffect, useCallback } from 'react';
import { NavLink, Outlet, useNavigate } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import { useConnectionStatus } from '../hooks/useConnectionStatus';
import { useCommandPalette } from '../hooks/useCommandPalette';
import { CommandPalette } from './CommandPalette';
import { Icon, type IconName } from './Icon';
import { Wordmark } from './Wordmark';

interface NavItem {
  to: string;
  label: string;
  icon: IconName;
  role: string;
  group: 'Overview' | 'Build the world' | 'Staff tools';
}

const navItems: NavItem[] = [
  { to: '/admin/', label: 'Dashboard', icon: 'dashboard', role: 'player', group: 'Overview' },
  { to: '/admin/webclient', label: 'Terminal', icon: 'terminal', role: 'player', group: 'Overview' },
  { to: '/admin/game/zones', label: 'Zones & rooms', icon: 'zones', role: 'player', group: 'Build the world' },
  { to: '/admin/game/mobs', label: 'Mobs', icon: 'mobs', role: 'player', group: 'Build the world' },
  { to: '/admin/game/objects', label: 'Objects', icon: 'objects', role: 'player', group: 'Build the world' },
  { to: '/admin/game/shops', label: 'Shops', icon: 'objects', role: 'player', group: 'Build the world' },
  { to: '/admin/workshop', label: 'Workshop & saves', icon: 'operations', role: 'builder', group: 'Build the world' },
  { to: '/admin/workshop/help', label: 'Builder guide', icon: 'info', role: 'builder', group: 'Build the world' },
  { to: '/admin/operations', label: 'Operations', icon: 'operations', role: 'builder', group: 'Staff tools' },
];

// Bottom tab items for mobile
const mobileTabItems: { to: string; label: string; icon: IconName }[] = [
  { to: '/admin/', label: 'Dashboard', icon: 'dashboard' },
  { to: '/admin/game/zones', label: 'Zones', icon: 'zones' },
  { to: '/admin/webclient', label: 'Terminal', icon: 'terminal' },
  { to: '/admin/operations', label: 'Ops', icon: 'operations' },
];

function useBreakpoint() {
  const [breakpoint, setBreakpoint] = useState<'mobile' | 'tablet' | 'desktop'>(() => {
    if (typeof window === 'undefined') return 'desktop';
    if (window.innerWidth < 768) return 'mobile';
    if (window.innerWidth < 1024) return 'tablet';
    return 'desktop';
  });

  useEffect(() => {
    const handler = () => {
      if (window.innerWidth < 768) setBreakpoint('mobile');
      else if (window.innerWidth < 1024) setBreakpoint('tablet');
      else setBreakpoint('desktop');
    };
    window.addEventListener('resize', handler);
    return () => window.removeEventListener('resize', handler);
  }, []);

  return breakpoint;
}

export function Layout() {
  const { playerName, role, logout, hasRole } = useAuth();
  const connectionStatus = useConnectionStatus();
  const { open, openPalette, closePalette } = useCommandPalette();
  const navigate = useNavigate();
  const breakpoint = useBreakpoint();
  const [sidebarOpen, setSidebarOpen] = useState(false);

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  const toggleSidebar = useCallback(() => setSidebarOpen((prev) => !prev), []);
  const closeSidebar = useCallback(() => setSidebarOpen(false), []);

  const visibleNavItems = navItems.filter((item) => hasRole(item.role));

  const connectionIndicator = {
    connected: { color: 'bg-online', label: 'ONLINE' },
    disconnected: { color: 'bg-accent', label: 'OFFLINE' },
    reconnecting: { color: 'bg-ink-muted animate-pulse', label: 'RECONNECTING' },
  }[connectionStatus];

  const isMobile = breakpoint === 'mobile';
  const isTablet = breakpoint === 'tablet';
  const showSidebar = breakpoint === 'desktop' || sidebarOpen;

  return (
    <div className="flex h-screen bg-paper text-ink font-serif transition-colors duration-200">
      {/* Desktop sidebar */}
      {breakpoint === 'desktop' && (
        <aside className="w-64 bg-paper border-r border-rule flex flex-col shrink-0">
          <SidebarContent
            navItems={visibleNavItems}
            playerName={playerName}
            role={role}
            onLogout={handleLogout}
          />
        </aside>
      )}

      {/* Mobile/Tablet sidebar overlay */}
      {(isMobile || isTablet) && showSidebar && (
        <>
          {/* Backdrop */}
          <div
            className="fixed inset-0 bg-ink/60 z-40 transition-opacity"
            onClick={closeSidebar}
          />
          {/* Sidebar */}
          <aside className="fixed inset-y-0 left-0 w-64 bg-paper border-r border-rule flex flex-col z-50 animate-[slideInLeft_0.2s_ease-out]">
            <SidebarContent
              navItems={visibleNavItems}
              playerName={playerName}
              role={role}
              onLogout={handleLogout}
              onClose={closeSidebar}
              onNavigate={closeSidebar}
            />
          </aside>
        </>
      )}

      {/* Main content */}
      <div className="flex-1 flex flex-col overflow-hidden min-w-0">
        {/* Header */}
        <header className="h-12 border-b border-rule flex items-center px-4 md:px-6 bg-paper-deep/40 backdrop-blur shrink-0">
          {/* Hamburger (mobile/tablet) */}
          {(isMobile || isTablet) && (
            <button
              onClick={toggleSidebar}
              className="mr-3 p-1.5 rounded-none border border-rule hover:bg-paper-deep transition-colors"
              aria-label="Toggle menu"
            >
              <svg className="w-5 h-5 text-ink" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M4 6h16M4 12h16M4 18h16" />
              </svg>
            </button>
          )}

          <h2 className="text-xs uppercase tracking-widest text-ink-muted font-mono hidden sm:block">
            Dark Pawns Admin
          </h2>

          <div className="ml-auto flex items-center gap-3">
            {/* Command palette trigger */}
            <button
              onClick={openPalette}
              className="hidden sm:flex items-center gap-1.5 text-xs font-mono text-ink bg-paper hover:bg-paper-deep px-2.5 py-1 rounded-none border border-rule transition-colors"
            >
              <span className="opacity-70">SEARCH</span>
              <kbd className="bg-paper-deep px-1 border border-rule/50 rounded-none text-[10px]">⌘K</kbd>
            </button>

            {/* Connection status */}
            <span className="flex items-center gap-1.5 text-xs font-mono font-bold" title={connectionIndicator.label}>
              <span className={`w-2.5 h-2.5 rounded-none border border-rule ${connectionIndicator.color}`} />
              <span className="hidden md:inline text-ink-muted tracking-wider">
                {connectionIndicator.label}
              </span>
            </span>
          </div>
        </header>

        {/* Page content */}
        <main className="flex-1 overflow-y-auto p-4 md:p-6 bg-paper">
          <Outlet />
        </main>

        {/* Mobile bottom tab bar */}
        {isMobile && (
          <nav className="flex border-t border-rule bg-paper-deep shrink-0 z-30">
            {mobileTabItems.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                end={item.to === '/admin/'}
                className={({ isActive }) =>
                  `flex-1 flex flex-col items-center py-2 text-[10px] font-mono uppercase tracking-wider transition-colors ${
                    isActive
                      ? 'text-accent font-bold bg-paper/50'
                      : 'text-ink-muted'
                  }`
                }
              >
                <Icon name={item.icon} className="h-4 w-4 mb-0.5" />
                <span>{item.label}</span>
              </NavLink>
            ))}
          </nav>
        )}
      </div>

      {/* Command Palette */}
      <CommandPalette open={open} onClose={closePalette} />
    </div>
  );
}

function SidebarContent({
  navItems,
  playerName,
  role,
  onLogout,
  onClose,
  onNavigate,
}: {
  navItems: NavItem[];
  playerName: string | null;
  role: string | null;
  onLogout: () => void;
  onClose?: () => void;
  onNavigate?: () => void;
}) {
  const [accountOpen, setAccountOpen] = useState(false);
  return (
    <>
      {/* Logo / Title */}
      <div className="p-4 border-b border-rule bg-paper-deep/20">
        <h1>
          <Wordmark pawnClass="h-8" textClass="text-[1.15rem]" />
        </h1>
        {onClose && (
          <button
            onClick={onClose}
            className="mt-2 text-xs font-mono uppercase text-ink-muted hover:text-accent sm:hidden border border-rule px-2 py-0.5"
          >
            Close
          </button>
        )}
      </div>

      {/* Navigation */}
      <nav className="flex-1 p-3 overflow-y-auto bg-paper" aria-label="Admin navigation">
        {(['Overview', 'Build the world', 'Staff tools'] as const).map((group) => {
          const items = navItems.filter((item) => item.group === group);
          if (items.length === 0) return null;
          return <div key={group} className="mb-5 last:mb-0">
            <p className="px-3.5 pb-2 font-serif text-xs font-semibold uppercase tracking-wider text-ink-muted">{group}</p>
            <div className="space-y-0.5">{items.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.to === '/admin/'}
            onClick={onNavigate}
            className={({ isActive }) =>
              `flex items-center gap-2.5 px-3.5 py-2 rounded-none text-sm border transition-colors focus-visible:outline-2 focus-visible:outline-accent ${
                // Hover used to reproduce the whole active treatment, which
                // made the current page indistinguishable from whatever the
                // cursor was resting on. Hover is now the tonal shift alone;
                // the accent and the rule belong to the active item.
                isActive
                  ? 'bg-paper-deep text-accent border-rule font-semibold'
                  : 'text-ink border-transparent hover:bg-paper-deep'
              }`
            }
          >
            <Icon name={item.icon} className="h-4 w-4 shrink-0" />
            <span>{item.label}</span>
          </NavLink>
            ))}</div>
          </div>;
        })}
      </nav>

      {/* User info */}
      <div className="p-3 border-t border-rule bg-paper-deep/30">
        <button type="button" aria-expanded={accountOpen} aria-controls="admin-account-actions" onClick={() => setAccountOpen((value) => !value)} className="flex w-full items-center justify-between gap-2 p-1 text-left hover:bg-paper-deep focus-visible:outline-2 focus-visible:outline-accent">
          <div className="min-w-0">
            <div className="text-xs font-extrabold text-ink font-mono truncate uppercase tracking-wider">
              {playerName || 'Unknown Operator'}
            </div>
            <div className="text-[10px] mt-1">
              <span
                className={`inline-block px-2 py-0.5 rounded-none text-[9px] font-bold uppercase tracking-wider border ${
                  role === 'admin'
                    ? 'bg-accent text-paper border-accent-deep'
                    : role === 'builder'
                      ? 'bg-paper-deep text-ink border-rule'
                      : role === 'research'
                        ? 'bg-paper text-accent border-accent'
                        : 'bg-paper text-ink-muted border-rule/50'
                }`}
              >
                {role || 'player'}
              </span>
            </div>
          </div>
          <span className="text-xs text-ink-muted" aria-hidden="true">{accountOpen ? '−' : '+'}</span>
        </button>
        {accountOpen && <div id="admin-account-actions" className="mt-2 border-t border-rule pt-2 space-y-1">
          <NavLink to="/admin/workshop/help" onClick={onNavigate} className="block px-2 py-1 text-sm text-ink hover:text-accent">Builder guide</NavLink>
          <button type="button" onClick={onLogout} className="w-full px-2 py-1 text-left text-sm text-ink hover:text-accent focus-visible:outline-2 focus-visible:outline-accent">Sign out</button>
        </div>}
      </div>
    </>
  );
}
