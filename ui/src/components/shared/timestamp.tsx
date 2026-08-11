import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { cn, formatTimestamp, timestampToDate, type TimestampValue } from '@/lib/utils'

interface TimestampProps {
  value: TimestampValue
  className?: string
  fallback?: string
  options?: Intl.DateTimeFormatOptions
}

function timeZoneName(date: Date, timeZone: string) {
  const part = new Intl.DateTimeFormat(undefined, {
    timeZone,
    timeZoneName: 'short',
  })
    .formatToParts(date)
    .find(({ type }) => type === 'timeZoneName')

  return part?.value || timeZone
}

function exactTimestamp(date: Date, timeZone: string) {
  return formatTimestamp(date, {
    dateStyle: 'medium',
    timeStyle: 'medium',
    timeZone,
  })
}

export function Timestamp({
  value,
  className,
  fallback = '—',
  options = { dateStyle: 'medium', timeStyle: 'short' },
}: TimestampProps) {
  const date = timestampToDate(value)
  if (!date) return <span className={className}>{fallback}</span>

  const localTimeZone = Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
  const localZoneName = timeZoneName(date, localTimeZone)
  const displayValue = formatTimestamp(date, options)

  return (
    <Tooltip delayDuration={350}>
      <TooltipTrigger asChild>
        <time
          dateTime={date.toISOString()}
          tabIndex={0}
          className={cn(
            'inline-block rounded-[2px] tabular-nums outline-none focus-visible:ring-2 focus-visible:ring-ring/50',
            className,
          )}
        >
          {displayValue}
        </time>
      </TooltipTrigger>
      <TooltipContent
        side="bottom"
        align="center"
        sideOffset={8}
        collisionPadding={12}
        showArrow={false}
        className="block w-[min(21rem,calc(100vw-1.5rem))] max-w-none overflow-hidden rounded-lg border border-border bg-popover p-0 text-left text-popover-foreground shadow-xl"
      >
        <div className="border-b border-border px-3 py-2 text-xs font-semibold">
          Time conversion
        </div>
        <div className="divide-y divide-border">
          <div className="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-3 gap-y-0.5 px-3 py-2 sm:grid-cols-[auto_minmax(0,1fr)_auto]">
            <span className="min-w-12 text-xs font-semibold">{localZoneName}</span>
            <span className="min-w-0 text-[11px] text-muted-foreground">
              <span className="block">Your computer</span>
              <span className="block truncate text-[10px] text-muted-foreground/70">
                {localTimeZone}
              </span>
            </span>
            <span className="col-span-2 whitespace-nowrap font-mono text-[11px] tabular-nums sm:col-span-1">
              {exactTimestamp(date, localTimeZone)}
            </span>
          </div>
          <div className="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-3 gap-y-0.5 px-3 py-2 sm:grid-cols-[auto_minmax(0,1fr)_auto]">
            <span className="min-w-12 text-xs font-semibold">UTC</span>
            <span />
            <span className="col-span-2 whitespace-nowrap font-mono text-[11px] tabular-nums sm:col-span-1">
              {exactTimestamp(date, 'UTC')}
            </span>
          </div>
        </div>
      </TooltipContent>
    </Tooltip>
  )
}
