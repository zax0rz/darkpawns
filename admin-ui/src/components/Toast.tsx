import { Icon, type IconName } from './Icon';
import { useState, useCallback, type ReactNode } from 'react';
import { ToastContext, type ToastType } from '../hooks/useToast';

interface Toast {
  id: number;
  message: string;
  type: ToastType;
}

let nextId = 0;

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const showToast = useCallback((message: string, type: ToastType = 'info') => {
    const id = nextId++;
    setToasts((prev) => [...prev, { id, message, type }]);
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, 4000);
  }, []);

  const dismiss = useCallback((id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  const typeStyles: Record<ToastType, string> = {
    success: 'bg-paper-deep border-rule text-ink',
    error: 'bg-accent border-accent-deep text-paper',
    info: 'bg-paper-deep border-rule text-ink',
  };

  const typeIcons: Record<ToastType, IconName> = {
    success: 'check',
    error: 'cross',
    info: 'info',
  };

  return (
    <ToastContext.Provider value={{ showToast }}>
      {children}
      {/* Toast container */}
      <div className="fixed bottom-4 right-4 z-50 flex flex-col gap-2 pointer-events-none">
        {toasts.map((toast) => (
          <div
            key={toast.id}
            className={`pointer-events-auto flex items-center gap-2 px-4 py-2.5 rounded-none border shadow-lg text-sm animate-[slideIn_0.2s_ease-out] ${typeStyles[toast.type]}`}
          >
            <Icon name={typeIcons[toast.type]} className="h-4 w-4 shrink-0" />
            <span>{toast.message}</span>
            <button
              onClick={() => dismiss(toast.id)}
              className="ml-2 opacity-50 hover:opacity-100 text-sm"
            >
              ×
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}
