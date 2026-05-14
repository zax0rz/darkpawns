import { createContext, useContext, useState, useCallback, type ReactNode } from 'react';

type ToastType = 'success' | 'error' | 'info';

interface Toast {
  id: number;
  message: string;
  type: ToastType;
}

interface ToastContextValue {
  showToast: (message: string, type?: ToastType) => void;
}

const ToastContext = createContext<ToastContextValue>({ showToast: () => {} });

export function useToast() {
  return useContext(ToastContext);
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
    success: 'bg-green-800 border-green-600 text-green-100',
    error: 'bg-red-800 border-red-600 text-red-100',
    info: 'bg-slate-800 border-slate-600 text-slate-100',
  };

  const typeIcons: Record<ToastType, string> = {
    success: '✓',
    error: '✗',
    info: 'ℹ',
  };

  return (
    <ToastContext.Provider value={{ showToast }}>
      {children}
      {/* Toast container */}
      <div className="fixed bottom-4 right-4 z-50 flex flex-col gap-2 pointer-events-none">
        {toasts.map((toast) => (
          <div
            key={toast.id}
            className={`pointer-events-auto flex items-center gap-2 px-4 py-2.5 rounded-lg border shadow-lg text-sm animate-[slideIn_0.2s_ease-out] ${typeStyles[toast.type]}`}
          >
            <span className="text-base">{typeIcons[toast.type]}</span>
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
