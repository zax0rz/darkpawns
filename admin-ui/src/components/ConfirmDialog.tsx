import type { ReactNode } from 'react';
import { AlertDialog, AlertDialogContent, AlertDialogHeader, AlertDialogTitle, AlertDialogDescription, AlertDialogFooter, AlertDialogCancel, AlertDialogAction } from './ui/alert-dialog';

export function ConfirmDialog({ open, onOpenChange, title, description, action, onConfirm, children, disabled = false }: { open: boolean; onOpenChange: (open: boolean) => void; title: string; description: ReactNode; action: string; onConfirm: () => void; children?: ReactNode; disabled?: boolean }) {
  return <AlertDialog open={open} onOpenChange={onOpenChange}>
    <AlertDialogContent><AlertDialogHeader><AlertDialogTitle>{title}</AlertDialogTitle><AlertDialogDescription>{description}</AlertDialogDescription></AlertDialogHeader>
      {children}<AlertDialogFooter><AlertDialogCancel>Cancel</AlertDialogCancel><AlertDialogAction disabled={disabled} onClick={onConfirm}>{action}</AlertDialogAction></AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>;
}
