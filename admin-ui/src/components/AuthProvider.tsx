import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react';
import { AuthContext, type AuthContextValue, type AuthState, type LoginError } from '../hooks/useAuth';

const sessionHeaders = {
  'Content-Type': 'application/json',
  'X-Requested-With': 'darkpawns-admin',
} as const;

// The auth provider lives apart from the useAuth hook so react-refresh keeps
// working (components and hooks cannot share a file).

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>({
    ready: false,
    authenticated: false,
    role: null,
    playerName: null,
  });

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const res = await fetch('/admin/session', {
          credentials: 'same-origin',
          headers: sessionHeaders,
        });
        if (res.ok) {
          const data = await res.json();
          if (!cancelled) {
            setState({
              ready: true,
              authenticated: true,
              role: data.role ?? 'player',
              playerName: data.player_name,
            });
          }
          return;
        }
      } catch {
        /* Network failure or signed out — both mean the login page. */
      }
      if (!cancelled) {
        setState((s) => ({ ...s, ready: true, authenticated: false, role: null, playerName: null }));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const login = useCallback(async (playerName: string, password: string) => {
    const res = await fetch('/admin/login', {
      method: 'POST',
      credentials: 'same-origin',
      headers: sessionHeaders,
      body: JSON.stringify({ player_name: playerName, password }),
    });

    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: 'Login failed' }));
      // The caller decides the wording. Relaying the server's text verbatim is
      // what leaked "invalid password" versus "invalid credentials" to the
      // screen, which tells a stranger whether a character name exists.
      const err = new Error(body.error || `Login failed (${res.status})`) as LoginError;
      err.status = res.status;
      throw err;
    }

    // The token itself arrives only in the HttpOnly cookie; the body carries
    // the display identity.
    const data = await res.json();
    setState({
      ready: true,
      authenticated: true,
      role: data.role ?? 'player',
      playerName: data.player_name,
    });
    return data;
  }, []);

  const logout = useCallback(async () => {
    try {
      await fetch('/admin/logout', {
        method: 'POST',
        credentials: 'same-origin',
        headers: sessionHeaders,
      });
    } catch {
      /* The server cookie clear is best-effort; local state clears regardless. */
    }
    setState({ ready: true, authenticated: false, role: null, playerName: null });
  }, []);

  const value = useMemo<AuthContextValue>(() => ({
    ...state,
    login,
    logout,
    isAuthenticated: state.authenticated,
    hasRole: (required: string) => {
      const hierarchy: Record<string, number> = {
        player: 0,
        research: 1,
        builder: 2,
        admin: 3,
      };
      return (hierarchy[state.role ?? 'player'] || 0) >= (hierarchy[required] || 0);
    },
  }), [state, login, logout]);

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
