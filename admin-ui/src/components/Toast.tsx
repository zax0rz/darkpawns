import { useCallback, type ReactNode } from 'react';
import { toast } from 'sonner';
import { ToastContext, type ToastType } from '../hooks/useToast';
import { Toaster } from './ui/sonner';

export function ToastProvider({ children }: { children: ReactNode }) {
  const showToast = useCallback((message: string, type: ToastType = 'info') => { toast[type](message); }, []);
  return <ToastContext.Provider value={{ showToast }}>{children}<Toaster /></ToastContext.Provider>;
}
