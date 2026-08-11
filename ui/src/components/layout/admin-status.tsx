import { useEffect } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useLocation } from 'react-router-dom'
import { api } from '@/data/api'
import { cn } from '@/lib/utils'

const adminInfoQueryKey = ['admin-info'] as const

export function AdminStatusRefresher() {
  const queryClient = useQueryClient()
  const { pathname } = useLocation()

  useEffect(() => {
    if (!/^\/workspaces\/[^/]+\/overview\/?$/.test(pathname)) return

    void queryClient.refetchQueries({
      queryKey: adminInfoQueryKey,
      exact: true,
      type: 'active',
    })
  }, [pathname, queryClient])

  return null
}

export function AdminStatus() {
  const adminQuery = useQuery({
    queryKey: adminInfoQueryKey,
    queryFn: api.adminInfo,
    retry: false,
    staleTime: Infinity,
  })

  const state = adminQuery.isPending ? 'loading' : adminQuery.isError ? 'unreachable' : 'online'
  const detail =
    state === 'loading'
      ? 'Checking backend…'
      : state === 'unreachable'
        ? 'Backend unreachable'
        : `Version ${adminQuery.data?.version || 'unknown'}`

  return (
    <div
      className="m-3 rounded-lg border border-border bg-background/70 p-3"
      role="status"
      aria-live="polite"
    >
      <div className="flex items-center gap-2 text-xs font-medium">
        <span
          aria-hidden="true"
          className={cn(
            'size-2 shrink-0 rounded-full',
            state === 'loading' && 'animate-pulse bg-muted-foreground/50',
            state === 'online' && 'bg-emerald-500',
            state === 'unreachable' && 'bg-red-500',
          )}
        />
        WebhookX Admin
      </div>
      <p
        className={cn(
          'mt-1.5 text-[11px] text-muted-foreground',
          state === 'unreachable' && 'text-red-600 dark:text-red-400',
        )}
      >
        {detail}
      </p>
    </div>
  )
}
