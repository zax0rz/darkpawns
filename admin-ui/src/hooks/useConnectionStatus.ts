import { useState, useEffect, useCallback } from 'react';
import { api } from '../api/client';

type ConnectionStatus = 'connected' | 'disconnected' | 'reconnecting';

export function useConnectionStatus(intervalMs = 30000) {
  const [status, setStatus] = useState<ConnectionStatus>('connected');

  const check = useCallback(async () => {
    try {
      await api.health();
      setStatus('connected');
    } catch {
      setStatus((prev) => (prev === 'disconnected' ? 'disconnected' : 'disconnected'));
    }
  }, []);

  useEffect(() => {
    check();
    const id = setInterval(check, intervalMs);
    return () => clearInterval(id);
  }, [check, intervalMs]);

  return status;
}
