import { createContext, useContext } from 'react';

export interface LoginError extends Error {
  status?: number;
}

// The admin credential lives in an HttpOnly cookie set by /admin/login, so
// JavaScript never sees the token (VULN-043). The hook therefore asks the
// server who is signed in instead of reading storage, and it lives in a
// context: ProtectedRoute, LoginPage and Layout each used to hold their own
// copy of the old localStorage-booted state, which cannot stay consistent
// once identity arrives asynchronously.
export interface AuthState {
  /** False until the boot-time /admin/session check has answered. */
  ready: boolean;
  authenticated: boolean;
  role: string | null;
  playerName: string | null;
}

export interface AuthContextValue extends AuthState {
  login: (playerName: string, password: string) => Promise<{ player_name: string; role: string }>;
  logout: () => Promise<void>;
  hasRole: (required: string) => boolean;
  /** Kept as the derived name the existing consumers gate on. */
  isAuthenticated: boolean;
}

export const AuthContext = createContext<AuthContextValue | null>(null);

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error('useAuth must be used inside AuthProvider');
  }
  return ctx;
}
