import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

const colors: Record<string, string> = {
  delivered: 'border-emerald-500/20 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400',
  successful: 'border-emerald-500/20 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400',
  active: 'border-emerald-500/20 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400',
  failed: 'border-red-500/20 bg-red-500/10 text-red-600 dark:text-red-400',
  paused: 'border-zinc-500/20 bg-zinc-500/10 text-zinc-500 dark:text-zinc-400',
  pending: 'border-amber-500/20 bg-amber-500/10 text-amber-600 dark:text-amber-400',
  init: 'border-amber-500/20 bg-amber-500/10 text-amber-600 dark:text-amber-400',
  queued: 'border-violet-500/20 bg-violet-500/10 text-violet-600 dark:text-violet-400',
  retrying: 'border-violet-500/20 bg-violet-500/10 text-violet-600 dark:text-violet-400',
  canceled: 'border-zinc-500/20 bg-zinc-500/10 text-zinc-500 dark:text-zinc-400',
}

export function StatusBadge({ status }: { status: string }) {
  const normalized = status.toLocaleLowerCase()
  return (
    <Badge className={cn('gap-1.5 capitalize', colors[normalized])}>
      <span className="size-1.5 rounded-full bg-current" />
      {normalized}
    </Badge>
  )
}
