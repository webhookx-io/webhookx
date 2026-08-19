import type { ReactNode } from 'react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { cn } from '@/lib/utils'

interface AppDialogProps {
  open: boolean
  onClose: () => void
  title: string
  description?: string
  children: ReactNode
  className?: string
}

/**
 * Application-level dialog layout built on the shadcn/Radix dialog primitive.
 * It keeps feature dialogs visually consistent while Radix owns focus trapping,
 * escape handling, aria wiring, and scroll locking.
 */
export function AppDialog({
  open,
  onClose,
  title,
  description,
  children,
  className,
}: AppDialogProps) {
  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && onClose()}>
      <DialogContent
        className={cn('max-h-[90vh] gap-0 overflow-y-auto p-0 sm:max-w-lg', className)}
      >
        <DialogHeader className="border-b border-border p-5 pr-12 text-left">
          <DialogTitle>{title}</DialogTitle>
          {description ? <DialogDescription>{description}</DialogDescription> : null}
        </DialogHeader>
        {children}
      </DialogContent>
    </Dialog>
  )
}
