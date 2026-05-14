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
    connected: { color: 'bg-green-500', label: 'Connected' },
    disconnected: { color: 'bg-red-500', label: 'Disconnected' },
    reconnecting: { color: 'bg-yellow-500', label: 'Reconnecting...' },
  }[connectionStatus];

  const isMobile = breakpoint === 'mobile';
  const isTablet = breakpoint === 'tablet';
  const showSidebar = breakpoint === 'desktop' || sidebarOpen;

  return (
    <div className="flex h-screen bg-slate-50 dark:bg-slate-900 text-slate-900 dark:text-slate-100">
      {/* Desktop sidebar */}
      {breakpoint === 'desktop' && (
        <aside className="w-64 bg-white dark:bg-slate-950 border-r border-slate-200 dark:border-slate-700 flex flex-col shrink-0">
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
            className="fixed inset-0 bg-black/50 z-40"
            onClick={closeSidebar}
          />
          {/* Sidebar */}
          <aside className="fixed inset-y-0 left-0 w-64 bg-white dark:bg-slate-950 border-r border-slate-200 dark:border-slate-700 flex flex-col z-50 animate-[slideInLeft_0.2s_ease-out]">
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
        <header className="h-12 border-b border-slate-200 dark:border-slate-700 flex items-center px-4 md:px-6 bg-white/80 dark:bg-slate-900/50 backdrop-blur shrink-0">
          {/* Hamburger (mobile/tablet) */}
          {(isMobile || isTablet) && (
            <button
              onClick={toggleSidebar}
              className="mr-3 p-1.5 rounded hover:bg-slate-200 dark:hover:bg-slate-800 transition-colors"
              aria-label="Toggle menu"
            >
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M4 6h16M4 12h16M4 18h16" />
              </svg>
            </button>
          )}

          <h2 className="text-sm text-slate-500 dark:text-slate-400 hidden sm:block">Dark Pawns Admin</h2>

          <div className="ml-auto flex items-center gap-3">
            {/* Command palette trigger */}
            <button
              onClick={openPalette}
              className="hidden sm:flex items-center gap-1.5 text-xs text-slate-400 bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 px-2.5 py-1 rounded border border-slate-200 dark:border-slate-700 transition-colors"
            >
              <span>⌘K</span>
            </button>

            {/* Connection status */}
            <span className="flex items-center gap-1.5 text-xs" title={connectionIndicator.label}>
              <span className={`w-2 h-2 rounded-full ${connectionIndicator.color}`} />
              <span className="hidden md:inline text-slate-500 dark:text-slate-400">
                {connectionIndicator.label}
              </span>
            </span>

            {/* Theme toggle */}
            <button
              onClick={toggleTheme}
              className="p-1.5 rounded hover:bg-slate-200 dark:hover:bg-slate-800 transition-colors text-sm"
              title={`Switch to ${theme === 'dark' ? 'light' : 'dark'} mode`}
            >
              {theme === 'dark' ? '☀️' : '🌙'}
            </button>
          </div>
        </header>

        {/* Page content */}
        <main className="flex-1 overflow-y-auto p-4 md:p-6">
          <Outlet />
        </main>

        {/* Mobile bottom tab bar */}
        {isMobile && (
          <nav className="flex border-t border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-950 shrink-0">
            {mobileTabItems.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                end={item.to === '/admin/'}
                className={({ isActive }) =>
                  `flex-1 flex flex-col items-center py-2 text-[10px] transition-colors ${
                    isActive
                      ? 'text-amber-600 dark:text-amber-400'
                      : 'text-slate-400'
                  }`
                }
              >
                <span className="text-lg">{item.icon}</span>
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
      <div className="p-4 border-b border-slate-200 dark:border-slate-700">
        <h1 className="text-lg font-bold text-amber-600 dark:text-amber-400 tracking-wide">
          ⚔️ Dark Pawns
        </h1>
        <p className="text-xs text-slate-400 mt-1">Admin Panel</p>
        {onClose && (
          <button
            onClick={onClose}
            className="mt-2 text-xs text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 sm:hidden"
          >
            ✕ Close
          </button>
        )}
      </div>

      {/* Navigation */}
      <nav className="flex-1 p-3 space-y-1 overflow-y-auto">
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.to === '/admin/'}
            className={({ isActive }) =>
              `flex items-center gap-2 px-3 py-2 rounded text-sm transition-colors ${
                isActive
                  ? 'bg-amber-50 dark:bg-slate-700 text-amber-700 dark:text-amber-400'
                  : 'text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800 hover:text-slate-900 dark:hover:text-white'
              }`
            }
          >
            <span>{item.icon}</span>
            <span>{item.label}</span>
          </NavLink>
        ))}
      </nav>

      {/* User info */}
      <div className="p-3 border-t border-slate-200 dark:border-slate-700">
        <div className="flex items-center justify-between">
          <div>
            <div className="text-sm font-medium text-slate-900 dark:text-white">
              {playerName || 'Unknown'}
            </div>
            <div className="text-xs text-slate-400">
              <span
                className={`inline-block px-1.5 py-0.5 rounded text-[10px] font-medium ${
                  role === 'admin'
                    ? 'bg-red-100 dark:bg-red-900 text-red-700 dark:text-red-300'
                    : role === 'builder'
                      ? 'bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300'
                      : role === 'research'
                        ? 'bg-green-100 dark:bg-green-900 text-green-700 dark:text-green-300'
                        : 'bg-slate-100 dark:bg-slate-700 text-slate-600 dark:text-slate-300'
                }`}
              >
                {role || 'player'}
              </span>
            </div>
          </div>
          <button
            onClick={onLogout}
            className="text-xs text-slate-400 hover:text-red-500 dark:hover:text-red-400 transition-colors px-2 py-1 rounded hover:bg-slate-100 dark:hover:bg-slate-800"
          >
            Logout
          </button>
        </div>
      </div>
    </>
  );
}
