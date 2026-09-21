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
    const initial = setTimeout(check, 0);
    const id = setInterval(check, intervalMs);
    return () => {
      clearTimeout(initial);
      clearInterval(id);
    };
  }, [check, intervalMs]);

  return status;
}
