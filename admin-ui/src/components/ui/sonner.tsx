import { CircleCheck, CircleX, Info, LoaderCircle } from 'lucide-react';
import { Toaster as Sonner, type ToasterProps } from 'sonner';

export function Toaster(props: ToasterProps) {
  return <Sonner theme="light" position="bottom-right" duration={4000}
    icons={{ success: <CircleCheck size={16}/>, error: <CircleX size={16}/>, info: <Info size={16}/>, loading: <LoaderCircle size={16} className="animate-spin"/> }}
    toastOptions={{ unstyled: true, classNames: {
      toast: 'flex w-full items-center gap-3 border border-rule bg-paper-deep p-4 font-serif text-sm text-ink',
      error: '!border-accent !text-accent', title: 'font-semibold', description: 'text-ink-muted',
      closeButton: 'border border-rule bg-paper p-1 text-ink',
    } }} closeButton {...props}/>;
}
