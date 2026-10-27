import { http, read, ApiError } from '../api/http';
import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react';
import { AuthContext, type AuthContextValue, type AuthState, type LoginError } from '../hooks/useAuth';

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
        const data = await read(http.GET('/admin/session'));
        if (!cancelled) setState({ ready: true, authenticated: true, role: data.role ?? 'player', playerName: data.player_name });
        return;
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
    let data;
    try { data = await read(http.POST('/admin/login', { body: { player_name: playerName, password } })); }
    catch (error) {
      const err = new Error(error instanceof Error ? error.message : 'Login failed') as LoginError;
      err.status = error instanceof ApiError ? error.status : 0;
      throw err;
    }
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
      await http.POST('/admin/logout');
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
