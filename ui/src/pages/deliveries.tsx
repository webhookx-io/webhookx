import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { toast } from 'sonner'
import { useCallback, useMemo } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useLocation, useMatch, useNavigate, useSearchParams } from 'react-router-dom'
import { ChevronLeft, ChevronRight, CircleAlert, RefreshCcw, RotateCcw, X } from 'lucide-react'
import { useWorkspaceName, workspacePath } from '@/app/workspace'
import { api } from '@/data/api'
import {
  createListQueryParams,
  listQueryParamsFromSearchParams,
  listQueryString,
  workspacePlaceholderData,
} from '@/data/list-query'
import type { Attempt, AttemptListParams, AttemptStatus } from '@/types'
import { errorMessage, latencyTone } from '@/lib/utils'
import { LoadingRows } from '@/components/shared/loading'
import { PageHeader } from '@/components/shared/page-header'
import { Timestamp } from '@/components/shared/timestamp'
import {
  QueryFilter,
  type QueryFilterConfig,
  type QueryPresetView,
} from '@/components/shared/query-filter'
import { StatusBadge } from '@/components/shared/status-badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { NativeSelect } from '@/components/ui/native-select'
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'

const defaultAttemptParams = createListQueryParams({
  limit: 20,
  sort: 'id.desc',
})

const attemptStatuses: Array<{ value: AttemptStatus; label: string }> = [
  { value: 'INIT', label: 'Initial' },
  { value: 'QUEUED', label: 'Queued' },
  { value: 'SUCCESSFUL', label: 'Successful' },
  { value: 'FAILED', label: 'Failed' },
  { value: 'CANCELED', label: 'Canceled' },
]

const attemptFilterConfig: QueryFilterConfig = {
  showSort: true,
  fields: [
    {
      key: 'event_id',
      type: 'string',
      label: 'Event ID',
      placeholder: 'Filter by exact event ID…',
      quickSearch: true,
    },
    {
      key: 'endpoint_id',
      type: 'string',
      label: 'Endpoint ID',
      placeholder: 'Filter by exact endpoint ID…',
    },
    { key: 'status', type: 'enum', label: 'Status', options: attemptStatuses },
    { key: 'attempted_at', type: 'attempted_at', label: 'Attempted' },
    { key: 'created_at', type: 'created_at', label: 'Created' },
  ],
}

const attemptPresetViews: QueryPresetView[] = [
  { id: 'failed', name: 'Failed', params: { status: 'FAILED' } },
]

function formatHeaders(headers: Record<string, string> | null | undefined) {
  if (!headers || !Object.keys(headers).length) return 'No headers recorded.'
  return Object.entries(headers)
    .map(([name, value]) => `${name}: ${value}`)
    .join('\n')
}

function InspectorSection({
  title,
  content,
}: {
  title: string
  content: string | null | undefined
}) {
  return (
    <section>
      <div className="mb-2 flex items-center justify-between gap-3">
        <h3 className="text-xs font-semibold">{title}</h3>
        <span className="text-[10px] text-muted-foreground">raw</span>
      </div>
      <pre className="max-h-72 overflow-auto whitespace-pre-wrap break-words rounded-lg border border-border bg-zinc-950 p-4 font-mono text-[11px] leading-5 text-zinc-300">
        <code>{content || 'No content recorded.'}</code>
      </pre>
    </section>
  )
}

function AttemptInspector({
  attempt,
  workspaceId,
  onClose,
}: {
  attempt: Attempt
  workspaceId: string
  onClose: () => void
}) {
  const queryClient = useQueryClient()
  const detailQuery = useQuery({
    queryKey: ['workspaces', workspaceId, 'attempts', 'detail', attempt.id],
    queryFn: () => api.attempt(workspaceId, attempt.id),
    placeholderData: attempt,
    refetchOnWindowFocus: false,
  })
  const detail = detailQuery.data ?? attempt
  const retry = useMutation({
    mutationFn: () => api.retryEvent(workspaceId, detail.event_id, detail.endpoint_id),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ['workspaces', workspaceId, 'attempts'],
      })
      toast.success('Manual delivery queued successfully.')
    },
  })

  const canRetry = detail.status === 'FAILED' || detail.status === 'CANCELED'

  return (
    <Sheet open onOpenChange={(open) => !open && onClose()}>
      <SheetContent className="w-full gap-0 bg-background p-0 sm:max-w-2xl" showCloseButton={false}>
        <SheetHeader className="min-h-16 flex-row items-center gap-3 border-b border-border px-5 py-3 text-left">
          <div className="min-w-0 flex-1">
            <SheetTitle className="text-sm font-semibold">
              Delivery attempt #{detail.attempt_number}
            </SheetTitle>
            <SheetDescription className="sr-only">
              Inspect the request, response, and retry state for this delivery attempt.
            </SheetDescription>
            <p className="mono-id mt-1 truncate">{detail.id}</p>
          </div>
          {canRetry && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => retry.mutate()}
              disabled={retry.isPending}
            >
              <RotateCcw className={`size-3.5 ${retry.isPending ? 'animate-spin' : ''}`} />
              {retry.isPending ? 'Retrying…' : 'Retry'}
            </Button>
          )}
          <SheetClose asChild>
            <Button variant="ghost" size="icon" aria-label="Close delivery attempt details">
              <X className="size-4" />
            </Button>
          </SheetClose>
        </SheetHeader>

        <div className="flex-1 space-y-5 overflow-y-auto p-5">
          {detailQuery.isError && (
            <div className="flex gap-3 rounded-lg border border-amber-500/20 bg-amber-500/5 p-3">
              <CircleAlert className="mt-0.5 size-4 shrink-0 text-amber-500" />
              <div>
                <p className="text-xs font-medium">Could not load complete attempt details</p>
                <p className="mt-1 text-xs text-muted-foreground">
                  Showing data from the attempts list. {errorMessage(detailQuery.error)}
                </p>
              </div>
            </div>
          )}

          <div className="flex items-center justify-between rounded-lg border border-border bg-card p-4">
            <div>
              <p className="text-[10px] uppercase tracking-wider text-muted-foreground">Status</p>
              <div className="mt-2">
                <StatusBadge status={detail.status} />
              </div>
            </div>
            <div className="text-right">
              <p className="font-mono text-xl font-semibold">{detail.response?.status ?? '—'}</p>
              <p className="mt-1 text-[10px] text-muted-foreground">
                {detail.response ? `${detail.response.latency}ms` : 'No response'}
              </p>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-px overflow-hidden rounded-lg border border-border bg-border">
            {[
              { label: 'Event ID', value: detail.event_id },
              { label: 'Endpoint ID', value: detail.endpoint_id },
              { label: 'Trigger', value: detail.trigger_mode },
              { label: 'Exhausted', value: detail.exhausted ? 'Yes' : 'No' },
              { label: 'Scheduled', value: <Timestamp value={detail.scheduled_at} /> },
              { label: 'Attempted', value: <Timestamp value={detail.attempted_at} /> },
              { label: 'Created', value: <Timestamp value={detail.created_at} /> },
              { label: 'Error code', value: detail.error_code ?? '—' },
            ].map(({ label, value }) => (
              <div key={label} className="min-w-0 bg-card p-3">
                <p className="text-[10px] uppercase tracking-wider text-muted-foreground">
                  {label}
                </p>
                <div className="mt-1.5 truncate font-mono text-xs font-medium">{value}</div>
              </div>
            ))}
          </div>

          {retry.isError && (
            <p className="field-error">Could not retry delivery: {errorMessage(retry.error)}</p>
          )}
          {retry.isSuccess && (
            <p className="text-xs text-emerald-600 dark:text-emerald-400">
              Manual delivery queued successfully.
            </p>
          )}

          <div className="grid gap-5 xl:grid-cols-2">
            <InspectorSection
              title={`${detail.request?.method ?? 'HTTP'} request headers`}
              content={formatHeaders(detail.request?.headers)}
            />
            <InspectorSection
              title="Response headers"
              content={formatHeaders(detail.response?.headers)}
            />
          </div>
          <InspectorSection title="Request body" content={detail.request?.body} />
          <InspectorSection title="Response body" content={detail.response?.body} />
        </div>
      </SheetContent>
    </Sheet>
  )
}

export function DeliveriesPage() {
  const workspaceName = useWorkspaceName()
  const navigate = useNavigate()
  const location = useLocation()
  const [searchParams, setSearchParams] = useSearchParams()
  const detailRoute = useMatch('/workspaces/:workspaceName/deliveries/:attemptId')
  const selectedAttemptId = detailRoute?.params.attemptId
  const deliveriesPath = workspacePath(workspaceName, 'deliveries')
  const searchParamsKey = searchParams.toString()
  const listParams = useMemo(
    () =>
      listQueryParamsFromSearchParams(new URLSearchParams(searchParamsKey), defaultAttemptParams),
    [searchParamsKey],
  )
  const setListParams = useCallback(
    (next: AttemptListParams | ((current: AttemptListParams) => AttemptListParams)) => {
      const resolved = typeof next === 'function' ? next(listParams) : next
      setSearchParams(listQueryString(resolved), { replace: true })
    },
    [listParams, setSearchParams],
  )
  const workspaceQuery = useQuery({
    queryKey: ['workspaces'],
    queryFn: api.workspaces,
  })
  const workspace = workspaceQuery.data?.find((item) => item.name === workspaceName)
  const workspaceId = workspace?.id ?? (workspaceName === 'default' ? 'default' : undefined)
  const attemptsQuery = useQuery({
    queryKey: ['workspaces', workspaceId, 'attempts', listParams],
    queryFn: () => api.attempts(workspaceId!, listParams),
    enabled: Boolean(workspaceId),
    placeholderData: (previous, previousQuery) =>
      workspacePlaceholderData(workspaceId, previous, previousQuery),
  })
  const page = attemptsQuery.data
  const attempts = page?.data ?? []
  const selectedAttemptFromList = selectedAttemptId
    ? attempts.find((attempt) => attempt.id === selectedAttemptId)
    : undefined
  const selectedAttemptQuery = useQuery({
    queryKey: ['workspaces', workspaceId, 'attempts', 'detail', selectedAttemptId],
    queryFn: () => api.attempt(workspaceId!, selectedAttemptId!),
    enabled: Boolean(workspaceId && selectedAttemptId && !selectedAttemptFromList),
    refetchOnWindowFocus: false,
  })
  const selectedAttempt = selectedAttemptFromList ?? selectedAttemptQuery.data
  const loading = (!workspaceId && workspaceQuery.isLoading) || attemptsQuery.isLoading
  const resolutionError = !workspaceId && !workspaceQuery.isLoading
  const loadError = resolutionError
    ? (workspaceQuery.error ?? new Error(`Workspace “${workspaceName}” was not found.`))
    : attemptsQuery.error

  const moveToCursor = (link: string | null | undefined) => {
    if (!link) return
    const url = new URL(link, window.location.origin)
    setListParams((current) => ({
      ...current,
      after: url.searchParams.get('after') || undefined,
      before: url.searchParams.get('before') || undefined,
    }))
  }

  return (
    <>
      <PageHeader
        title="Deliveries"
        description={`Inspect outgoing webhook attempts for the ${workspaceName} workspace.`}
        actions={
          <Button
            variant="outline"
            onClick={() => attemptsQuery.refetch()}
            disabled={!workspaceId || attemptsQuery.isFetching}
          >
            <RefreshCcw className={`size-3.5 ${attemptsQuery.isFetching ? 'animate-spin' : ''}`} />
            Refresh
          </Button>
        }
      />

      <Card className="overflow-hidden">
        <QueryFilter
          value={listParams}
          onChange={setListParams}
          config={attemptFilterConfig}
          storageKey={`webhookx:attempt-views:${workspaceId ?? workspaceName}`}
          defaultParams={defaultAttemptParams}
          presetViews={attemptPresetViews}
          resultCount={attempts.length}
          total={page?.total}
          busy={attemptsQuery.isFetching && !attemptsQuery.isLoading}
        />

        {loading && <LoadingRows rows={7} />}

        {loadError && !loading && (
          <div className="m-5 rounded-lg border border-red-500/20 bg-red-500/5 p-4">
            <div className="flex gap-3">
              <CircleAlert className="mt-0.5 size-4 shrink-0 text-red-500" />
              <div>
                <p className="text-xs font-medium">Could not load delivery attempts</p>
                <p className="mt-1 text-xs text-muted-foreground">{errorMessage(loadError)}</p>
              </div>
            </div>
            {workspaceId && (
              <Button
                variant="outline"
                size="sm"
                className="mt-3"
                onClick={() => attemptsQuery.refetch()}
              >
                Try again
              </Button>
            )}
          </div>
        )}

        {!loading && !loadError && (
          <div className="overflow-x-auto">
            <Table className="data-table">
              <TableHeader>
                <TableRow>
                  <TableHead>Status</TableHead>
                  <TableHead>Attempt ID</TableHead>
                  <TableHead>Endpoint</TableHead>
                  <TableHead>Attempt</TableHead>
                  <TableHead>Trigger</TableHead>
                  <TableHead>Response</TableHead>
                  <TableHead>Latency</TableHead>
                  <TableHead>Attempted</TableHead>
                  <TableHead />
                </TableRow>
              </TableHeader>
              <TableBody>
                {attempts.map((attempt) => (
                  <TableRow
                    key={attempt.id}
                    className="cursor-pointer [&>td]:h-12 [&>td]:py-0"
                    onClick={() =>
                      void navigate(
                        {
                          pathname: `${deliveriesPath}/${encodeURIComponent(attempt.id)}`,
                          search: location.search,
                        },
                        { state: { attemptInspectorFromList: true } },
                      )
                    }
                  >
                    <TableCell>
                      <StatusBadge status={attempt.status} />
                    </TableCell>
                    <TableCell className="max-w-xs">
                      <p className="truncate font-mono text-xs" title={attempt.id}>
                        {attempt.id}
                      </p>
                      <p className="mono-id mt-1 truncate" title={attempt.event_id}>
                        Event {attempt.event_id}
                      </p>
                    </TableCell>
                    <TableCell className="max-w-xs">
                      <p className="truncate font-mono text-xs" title={attempt.endpoint_id}>
                        {attempt.endpoint_id}
                      </p>
                      {attempt.request?.url && (
                        <p className="mono-id mt-1 truncate" title={attempt.request.url}>
                          {attempt.request.url}
                        </p>
                      )}
                    </TableCell>
                    <TableCell>#{attempt.attempt_number}</TableCell>
                    <TableCell className="font-mono text-[11px] text-muted-foreground">
                      {attempt.trigger_mode}
                    </TableCell>
                    <TableCell>
                      <span
                        className={`font-mono text-xs ${attempt.response && attempt.response.status >= 400 ? 'text-red-600 dark:text-red-400' : ''}`}
                      >
                        {attempt.response?.status ?? '—'}
                      </span>
                    </TableCell>
                    <TableCell
                      className={
                        attempt.response ? latencyTone(attempt.response.latency) : undefined
                      }
                    >
                      {attempt.response ? `${attempt.response.latency}ms` : '—'}
                    </TableCell>
                    <TableCell className="whitespace-nowrap text-muted-foreground">
                      <Timestamp value={attempt.attempted_at} />
                    </TableCell>
                    <TableCell>
                      <ChevronRight className="size-4 text-muted-foreground" />
                    </TableCell>
                  </TableRow>
                ))}
                {!attempts.length && (
                  <TableRow>
                    <TableCell colSpan={9} className="py-14 text-center text-muted-foreground">
                      No delivery attempts match the current query.
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        )}

        {!loading && !loadError && (
          <div className="flex flex-wrap items-center justify-between gap-3 border-t border-border px-3 py-2.5">
            <label className="flex items-center gap-2 text-xs text-muted-foreground">
              Rows
              <NativeSelect
                className="h-8 w-20"
                aria-label="Delivery attempts per page"
                value={String(listParams.limit)}
                onChange={(event) =>
                  setListParams((current) => ({
                    ...current,
                    limit: Number(event.target.value),
                    after: undefined,
                    before: undefined,
                  }))
                }
              >
                {[10, 20, 50, 100].map((limit) => (
                  <option key={limit} value={limit}>
                    {limit}
                  </option>
                ))}
              </NativeSelect>
            </label>
            <div className="flex items-center gap-2">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => moveToCursor(page?.prev)}
                disabled={!page?.prev || attemptsQuery.isFetching}
              >
                <ChevronLeft className="size-3.5" />
                Previous
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => moveToCursor(page?.next)}
                disabled={!page?.next || attemptsQuery.isFetching}
              >
                Next
                <ChevronRight className="size-3.5" />
              </Button>
            </div>
          </div>
        )}
      </Card>

      {selectedAttempt && workspaceId && (
        <AttemptInspector
          attempt={selectedAttempt}
          workspaceId={workspaceId}
          onClose={() => {
            const openedFromList = Boolean(
              (location.state as { attemptInspectorFromList?: boolean } | null)
                ?.attemptInspectorFromList,
            )
            if (openedFromList) {
              void navigate(-1)
              return
            }
            void navigate({ pathname: deliveriesPath, search: location.search }, { replace: true })
          }}
        />
      )}
    </>
  )
}
