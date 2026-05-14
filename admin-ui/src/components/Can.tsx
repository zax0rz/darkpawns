import type { ReactNode } from 'react';
import { useAuth } from '../hooks/useAuth';

export function Can({ role, children }: { role: string; children: ReactNode }) {
  const { hasRole } = useAuth();
  return hasRole(role) ? <>{children}</> : null;
}
