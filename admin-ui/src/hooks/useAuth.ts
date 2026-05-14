import { useState, useCallback } from 'react';

interface AuthState {
  token: string | null;
  role: string | null;
  playerName: string | null;
}

export function useAuth() {
  const [auth, setAuth] = useState<AuthState>(() => {
    const token = localStorage.getItem('admin_token');
    if (token) {
      try {
        const payload = JSON.parse(atob(token.split('.')[1]));
        return {
          token,
          role: payload.role || 'player',
          playerName: payload.player_name,
        };
      } catch {
        /* fall through */
      }
    }
    return { token: null, role: null, playerName: null };
  });

  const login = useCallback(async (playerName: string, password: string) => {
    const res = await fetch('/admin/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ player_name: playerName, password }),
    });

    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: 'Login failed' }));
      throw new Error(body.error || `Login failed (${res.status})`);
    }

    const data = await res.json();
    localStorage.setItem('admin_token', data.token);
    setAuth({
      token: data.token,
      role: data.role,
      playerName: data.player_name,
    });
    return data;
  }, []);

  const logout = useCallback(() => {
    localStorage.removeItem('admin_token');
    setAuth({ token: null, role: null, playerName: null });
  }, []);

  const hasRole = useCallback(
    (required: string) => {
      const hierarchy: Record<string, number> = {
        player: 0,
        research: 1,
        builder: 2,
        admin: 3,
      };
      return (
        (hierarchy[auth.role || 'player'] || 0) >= (hierarchy[required] || 0)
      );
    },
    [auth.role]
  );

  return {
    ...auth,
    login,
    logout,
    hasRole,
    isAuthenticated: !!auth.token,
  };
}
