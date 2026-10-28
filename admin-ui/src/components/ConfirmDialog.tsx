import type { ReactNode } from 'react';
import { AlertDialog, AlertDialogContent, AlertDialogHeader, AlertDialogTitle, AlertDialogDescription, AlertDialogFooter, AlertDialogCancel, AlertDialogAction } from './ui/alert-dialog';

export function ConfirmDialog({ open, onOpenChange, title, description, action, onConfirm }: { open: boolean; onOpenChange: (open: boolean) => void; title: string; description: ReactNode; action: string; onConfirm: () => void }) {
  return <AlertDialog open={open} onOpenChange={onOpenChange}>
    <AlertDialogContent><AlertDialogHeader><AlertDialogTitle>{title}</AlertDialogTitle><AlertDialogDescription>{description}</AlertDialogDescription></AlertDialogHeader>
      <AlertDialogFooter><AlertDialogCancel>Cancel</AlertDialogCancel><AlertDialogAction onClick={onConfirm}>{action}</AlertDialogAction></AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>;
}
