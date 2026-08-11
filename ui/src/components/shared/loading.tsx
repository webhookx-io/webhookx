import { Skeleton } from '@/components/ui/skeleton'

export function LoadingRows({ rows = 5 }: { rows?: number }) {
  return (
    <div className="divide-y divide-border" aria-label="Loading">
      {Array.from({ length: rows }).map((_, index) => (
        <Skeleton key={index} className="h-14 rounded-none bg-muted/40" />
      ))}
    </div>
  )
}
