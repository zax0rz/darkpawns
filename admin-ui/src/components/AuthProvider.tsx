import { http, read, ApiError } from '../api/http';
import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { AuthContext, type AuthContextValue, type AuthState, type LoginError } from '../hooks/useAuth';

// The auth provider lives apart from the useAuth hook so react-refresh keeps
// working (components and hooks cannot share a file).

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const [state, setState] = useState<AuthState>({
    ready: false,
    authenticated: false,
    role: null,
    playerName: null,
  });

  // Private query results belong to one sign-in. Clear them before exposing a
  // new identity, including a fresh sign-in by the same demoted character.
  const setIdentity = useCallback((next: AuthState) => {
    queryClient.clear();
    setState(next);
  }, [queryClient]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const data = await read(http.GET('/admin/session'));
        if (!cancelled) setIdentity({ ready: true, authenticated: true, role: data.role ?? 'player', playerName: data.player_name });
        return;
      } catch {
        /* Network failure or signed out — both mean the login page. */
      }
      if (!cancelled) {
        setIdentity({ ready: true, authenticated: false, role: null, playerName: null });
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [setIdentity]);

  const login = useCallback(async (playerName: string, password: string) => {
    let data;
    try { data = await read(http.POST('/admin/login', { body: { player_name: playerName, password } })); }
    catch (error) {
      const err = new Error(error instanceof Error ? error.message : 'Login failed') as LoginError;
      err.status = error instanceof ApiError ? error.status : 0;
      throw err;
    }
    setIdentity({
      ready: true,
      authenticated: true,
      role: data.role ?? 'player',
      playerName: data.player_name,
    });
    return data;
  }, [setIdentity]);

  const logout = useCallback(async () => {
    try {
      await http.POST('/admin/logout');
    } catch {
      /* The server cookie clear is best-effort; local state clears regardless. */
    }
    setIdentity({ ready: true, authenticated: false, role: null, playerName: null });
  }, [setIdentity]);

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
