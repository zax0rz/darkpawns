import { ChevronUp } from 'lucide-react';
import { Dialog, DialogContent, DialogTitle, DialogDescription } from './ui/dialog';
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuItem } from './ui/dropdown-menu';
import { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider } from './ui/tooltip';
import { Button } from './ui/button';
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
  { to: '/admin/game/shops', label: 'Shops', icon: 'shops', role: 'player', group: 'Build the world' },
  { to: '/admin/workshop', label: 'Workshop & saves', icon: 'operations', role: 'builder', group: 'Build the world' },
  { to: '/admin/workshop/help', label: 'Builder guide', icon: 'guide', role: 'builder', group: 'Build the world' },
  { to: '/admin/staff/players', label: 'Players', icon: 'players', role: 'admin', group: 'Staff tools' },
  { to: '/admin/staff/notifications', label: 'Notifications', icon: 'notifications', role: 'admin', group: 'Staff tools' },
  { to: '/admin/operations/logs', label: 'Logs', icon: 'terminal', role: 'builder', group: 'Staff tools' },
  { to: '/admin/operations', label: 'Operations', icon: 'operations', role: 'builder', group: 'Staff tools' },
  { to: '/admin/staff/settings', label: 'Configuration', icon: 'settings', role: 'admin', group: 'Staff tools' },
  { to: '/admin/staff/updates', label: 'Updates', icon: 'updates', role: 'admin', group: 'Staff tools' },
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

  return (
    <div className="flex h-screen bg-paper text-ink font-serif transition-colors duration-200">
      {/* Desktop sidebar */}
      {breakpoint === 'desktop' && (
        <aside className="w-64 bg-paper-deep border-r border-rule flex flex-col shrink-0">
          <SidebarContent
            navItems={visibleNavItems}
            playerName={playerName}
            role={role}
            onLogout={handleLogout}
          />
        </aside>
      )}

      {/* Mobile/Tablet sidebar overlay */}
      {(isMobile || isTablet) && (
        <Dialog open={sidebarOpen} onOpenChange={setSidebarOpen}>
          <DialogContent showCloseButton={false} className="top-0 left-0 flex h-dvh w-64 max-w-[85vw] translate-x-0 translate-y-0 flex-col gap-0 rounded-none bg-paper-deep p-0 sm:max-w-64">
            <DialogTitle className="sr-only">Admin navigation</DialogTitle>
            <DialogDescription className="sr-only">Choose a console page.</DialogDescription>
            <SidebarContent navItems={visibleNavItems} playerName={playerName} role={role} onLogout={handleLogout} onClose={closeSidebar} onNavigate={closeSidebar}/>
          </DialogContent>
        </Dialog>
      )}

      {/* Main content */}
      <div className="flex-1 flex flex-col overflow-hidden min-w-0">
        {/* Header */}
        <header className="h-12 border-b border-rule flex items-center px-4 md:px-6 bg-paper-deep/40 shrink-0">
          {/* Hamburger (mobile/tablet) */}
          {(isMobile || isTablet) && (
            <Button variant="ghost" size="plain"
              onClick={toggleSidebar}
              className="mr-3 p-1.5 rounded-none border border-rule hover:bg-paper-deep transition-colors"
              aria-label="Toggle menu"
            >
              <svg className="w-5 h-5 text-ink" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M4 6h16M4 12h16M4 18h16" />
              </svg>
            </Button>
          )}

          <h2 className="text-xs uppercase tracking-widest text-ink-muted font-mono hidden sm:block">
            Dark Pawns Admin
          </h2>

          <div className="ml-auto flex items-center gap-3">
            {/* Command palette trigger */}
            <TooltipProvider><Tooltip><TooltipTrigger asChild><Button variant="ghost" size="plain"
              onClick={openPalette}
              className="hidden sm:flex items-center gap-1.5 text-xs font-mono text-ink bg-paper hover:bg-paper-deep px-2.5 py-1 rounded-none border border-rule transition-colors"
            >
              <span className="opacity-70">SEARCH</span>
              <kbd className="bg-paper-deep px-1 border border-rule/50 rounded-none text-[10px]">⌘K</kbd>
            </Button></TooltipTrigger><TooltipContent>Search pages and world records (⌘K or Ctrl+K)</TooltipContent></Tooltip></TooltipProvider>

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
                end={item.to === '/admin/' || item.to === '/admin/operations'}
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
  return (
    <>
      {/* Logo / Title */}
      <div className="p-4 border-b border-rule bg-paper-deep/20">
        <h1>
          <Wordmark pawnClass="h-8" textClass="text-[1.15rem]" />
        </h1>
        {onClose && (
          <Button variant="ghost" size="plain"
            onClick={onClose}
            className="mt-2 text-xs font-mono uppercase text-ink-muted hover:text-accent border border-rule px-2 py-0.5"
          >
            Close
          </Button>
        )}
      </div>

      {/* Navigation */}
      <nav className="flex-1 p-3 overflow-y-auto bg-paper-deep" aria-label="Admin navigation">
        {(['Overview', 'Staff tools', 'Build the world'] as const).map((group) => {
          const items = navItems.filter((item) => item.group === group);
          if (items.length === 0) return null;
          return <div key={group} className="mb-5 last:mb-0">
            <p className="px-3.5 pb-2 font-serif text-xs font-semibold uppercase tracking-wider text-ink-muted">{group}</p>
            <div className="space-y-0.5">{items.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.to === '/admin/' || item.to === '/admin/operations'}
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
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="plain" className="flex w-full items-center justify-between px-2 py-2 text-left" aria-label="Account menu">
              <span className="min-w-0"><span className="block truncate text-sm font-semibold">{playerName || 'Unknown operator'}</span><span className="block text-xs text-ink-muted">{role || 'player'}</span></span>
              <ChevronUp className="h-4 w-4 shrink-0" strokeWidth={1.5} aria-hidden="true" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent side="top" align="start" className="min-w-52">
            <DropdownMenuLabel>Account</DropdownMenuLabel><DropdownMenuSeparator />
            <DropdownMenuItem asChild><NavLink to="/admin/workshop/help" onClick={onNavigate}>Builder guide</NavLink></DropdownMenuItem>
            <DropdownMenuItem onSelect={onLogout}>Sign out</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </>
  );
}
