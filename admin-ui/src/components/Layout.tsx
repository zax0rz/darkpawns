import { useState, useEffect, useCallback } from 'react';
import { NavLink, Outlet, useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import { useTheme } from '../hooks/useTheme';
import { useConnectionStatus } from '../hooks/useConnectionStatus';
import { useCommandPalette } from '../hooks/useCommandPalette';
import { CommandPalette } from './CommandPalette';

interface NavItem {
  to: string;
  label: string;
  icon: string;
  role: string;
}

const navItems: NavItem[] = [
  { to: '/admin/', label: 'Dashboard', icon: '📊', role: 'player' },
  { to: '/admin/game/zones', label: 'Zones', icon: '🗺️', role: 'player' },
  { to: '/admin/game/mobs', label: 'Mobs', icon: '🐉', role: 'player' },
  { to: '/admin/game/objects', label: 'Objects', icon: '💎', role: 'player' },
  { to: '/admin/agents', label: 'Agents', icon: '🤖', role: 'builder' },
  { to: '/admin/decisions', label: 'Decisions', icon: '📊', role: 'builder' },
  { to: '/admin/narrative', label: 'Mind Reader', icon: '🧠', role: 'builder' },
  { to: '/admin/operations', label: 'Operations', icon: '⚙️', role: 'builder' },
  { to: '/admin/webclient', label: 'Terminal', icon: '🖥️', role: 'player' },
];

// Bottom tab items for mobile
const mobileTabItems = [
  { to: '/admin/', label: 'Dashboard', icon: '📊' },
  { to: '/admin/game/zones', label: 'Zones', icon: '🗺️' },
  { to: '/admin/webclient', label: 'Terminal', icon: '🖥️' },
  { to: '/admin/operations', label: 'Ops', icon: '⚙️' },
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
  const { theme, toggleTheme } = useTheme();
  const connectionStatus = useConnectionStatus();
  const { open, openPalette, closePalette } = useCommandPalette();
  const navigate = useNavigate();
  const location = useLocation();
  const breakpoint = useBreakpoint();
  const [sidebarOpen, setSidebarOpen] = useState(false);

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  // Close sidebar on navigation (mobile/tablet)
  useEffect(() => {
    setSidebarOpen(false);
  }, [location.pathname]);

  const toggleSidebar = useCallback(() => setSidebarOpen((prev) => !prev), []);
  const closeSidebar = useCallback(() => setSidebarOpen(false), []);

  const visibleNavItems = navItems.filter((item) => hasRole(item.role));

  const connectionIndicator = {
    connected: { color: 'bg-accent', label: 'ONLINE' },
    disconnected: { color: 'bg-ink-muted animate-pulse', label: 'OFFLINE' },
    reconnecting: { color: 'bg-accent animate-bounce', label: 'RECONNECTING' },
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
            className="fixed inset-0 bg-black/60 z-40 transition-opacity"
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

          <h2 className="text-xs uppercase tracking-widest text-ink-muted font-sans hidden sm:block">
            Dark Pawns Administrative Ledger
          </h2>

          <div className="ml-auto flex items-center gap-3">
            {/* Command palette trigger */}
            <button
              onClick={openPalette}
              className="hidden sm:flex items-center gap-1.5 text-xs font-sans text-ink bg-paper hover:bg-paper-deep px-2.5 py-1 rounded-none border border-rule transition-colors"
            >
              <span className="opacity-70">SEARCH</span>
              <kbd className="bg-paper-deep px-1 border border-rule/50 rounded-none text-[10px]">⌘K</kbd>
            </button>

            {/* Connection status */}
            <span className="flex items-center gap-1.5 text-xs font-sans font-bold" title={connectionIndicator.label}>
              <span className={`w-2.5 h-2.5 rounded-none border border-rule ${connectionIndicator.color}`} />
              <span className="hidden md:inline text-ink-muted tracking-wider">
                {connectionIndicator.label}
              </span>
            </span>

            {/* Theme toggle */}
            <button
              onClick={toggleTheme}
              className="p-1.5 rounded-none border border-rule hover:bg-paper-deep transition-colors text-xs"
              title={`Switch to ${theme === 'dark' ? 'light' : 'dark'} mode`}
            >
              {theme === 'dark' ? '🌕 PARCHMENT' : '🌑 INK'}
            </button>
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
                  `flex-1 flex flex-col items-center py-2 text-[10px] font-sans uppercase tracking-wider transition-colors ${
                    isActive
                      ? 'text-accent font-bold bg-paper/50'
                      : 'text-ink-muted'
                  }`
                }
              >
                <span className="text-base mb-0.5">{item.icon}</span>
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
}: {
  navItems: NavItem[];
  playerName: string | null;
  role: string | null;
  onLogout: () => void;
  onClose?: () => void;
}) {
  return (
    <>
      {/* Logo / Title */}
      <div className="p-4 border-b border-rule bg-paper-deep/20">
        <h1 className="text-xl font-extrabold text-accent tracking-widest font-sans flex items-center gap-1.5">
          DARK PAWNS
        </h1>
        <p className="text-[10px] uppercase tracking-widest text-ink-muted mt-1 font-sans">
          Mythic Admin Console
        </p>
        {onClose && (
          <button
            onClick={onClose}
            className="mt-2 text-xs font-sans uppercase text-ink-muted hover:text-accent sm:hidden border border-rule px-2 py-0.5"
          >
            ✕ Close
          </button>
        )}
      </div>

      {/* Navigation */}
      <nav className="flex-1 p-3 space-y-1 overflow-y-auto bg-paper">
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.to === '/admin/'}
            className={({ isActive }) =>
              `flex items-center gap-2.5 px-3.5 py-2.5 rounded-none text-xs uppercase tracking-wider font-sans border transition-all ${
                isActive
                  ? 'bg-paper-deep text-accent border-rule font-extrabold shadow-[2px_2px_0px_0px_rgba(26,22,20,0.1)]'
                  : 'text-ink border-transparent hover:bg-paper-deep hover:text-accent hover:border-rule/30'
              }`
            }
          >
            <span className="text-sm opacity-80">{item.icon}</span>
            <span>{item.label}</span>
          </NavLink>
        ))}
      </nav>

      {/* User info */}
      <div className="p-3 border-t border-rule bg-paper-deep/30">
        <div className="flex items-center justify-between gap-2">
          <div className="min-w-0">
            <div className="text-xs font-extrabold text-ink font-sans truncate uppercase tracking-wider">
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
          <button
            onClick={onLogout}
            className="text-[10px] uppercase font-sans tracking-wider text-ink hover:text-accent border border-rule hover:bg-paper-deep px-2 py-1 transition-all shrink-0"
          >
            Logout
          </button>
        </div>
      </div>
    </>
  );
}
