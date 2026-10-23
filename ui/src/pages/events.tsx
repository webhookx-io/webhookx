import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { toast } from 'sonner'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { Link, useLocation, useMatch, useNavigate, useSearchParams } from 'react-router-dom'
import {
  Check,
  ChevronLeft,
  ChevronRight,
  CircleAlert,
  Copy,
  ListChecks,
  Pause,
  Play,
  RefreshCcw,
  RotateCcw,
  X,
} from 'lucide-react'
import { useWorkspaceName, workspacePath } from '@/app/workspace'
import { api } from '@/data/api'
import {
  createListQueryParams,
  listQueryParamsFromSearchParams,
  listQueryString,
  workspacePlaceholderData,
} from '@/data/list-query'
import type { EventListParams, WebhookEvent } from '@/types'
import { PageHeader } from '@/components/shared/page-header'
import { LoadingRows } from '@/components/shared/loading'
import { QueryFilter, type QueryFilterConfig } from '@/components/shared/query-filter'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { NativeSelect } from '@/components/ui/native-select'
import { Timestamp } from '@/components/shared/timestamp'
import { copyText, errorMessage } from '@/lib/utils'
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'

const defaultEventParams = createListQueryParams({
  limit: 20,
  sort: 'id.desc',
})

const eventFilterConfig: QueryFilterConfig = {
  showSort: true,
  fields: [
    {
      key: 'event_type',
      type: 'string',
      label: 'Event type',
      placeholder: 'Filter by exact event type…',
      quickSearch: true,
    },
    {
      key: 'unique_id',
      type: 'string',
      label: 'Unique ID',
      placeholder: 'Filter by exact unique ID…',
    },
    { key: 'ingested_at', type: 'ingested_at', label: 'Ingested' },
    { key: 'created_at', type: 'created_at', label: 'Created' },
  ],
}

const endpointListParams = createListQueryParams({
  limit: 1000,
  sort: 'id.asc',
})

function EventInspector({
  event,
  workspaceId,
  onClose,
}: {
  event: WebhookEvent
  workspaceId: string
  onClose: () => void
}) {
  const workspaceName = useWorkspaceName()
  const [copied, setCopied] = useState(false)
  const [endpointId, setEndpointId] = useState('')
  const detailQuery = useQuery({
    queryKey: ['workspaces', workspaceId, 'events', 'detail', event.id],
    queryFn: () => api.event(workspaceId, event.id),
    placeholderData: event,
    refetchOnWindowFocus: false,
  })
  const endpointsQuery = useQuery({
    queryKey: ['workspaces', workspaceId, 'endpoints', endpointListParams],
    queryFn: () => api.endpoints(workspaceId, endpointListParams),
    refetchOnWindowFocus: false,
  })
  const detail = detailQuery.data ?? event
  const endpoints = endpointsQuery.data?.data ?? []
  const selectedEndpoint = endpoints.find((endpoint) => endpoint.id === endpointId)
  const retry = useMutation({
    mutationFn: (retryEndpointId: string) =>
      api.retryEvent(workspaceId, detail.id, retryEndpointId),
    onSuccess: (_, retryEndpointId) => {
      const retriedEndpoint = endpoints.find((endpoint) => endpoint.id === retryEndpointId)
      const endpointLabel =
        retriedEndpoint?.name || retriedEndpoint?.request.url || retriedEndpoint?.id
      toast.success(
        `Event queued for a manual delivery${endpointLabel ? ` to ${endpointLabel}` : ''}.`,
      )
    },
  })

  const copyId = async () => {
    try {
      await copyText(detail.id)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 1500)
    } catch (error) {
      toast.error(errorMessage(error, 'Could not copy the event ID.'))
    }
  }

  return (
    <Sheet open onOpenChange={(open) => !open && onClose()}>
      <SheetContent className="w-full gap-0 bg-background p-0 sm:max-w-xl" showCloseButton={false}>
        <SheetHeader className="min-h-16 flex-row items-center gap-3 border-b border-border px-5 py-3 text-left">
          <div className="min-w-0 flex-1">
            <SheetTitle className="truncate font-mono text-sm font-medium">
              {detail.event_type}
            </SheetTitle>
            <SheetDescription className="sr-only">
              Inspect event details and create a manual delivery.
            </SheetDescription>
            <Button
              type="button"
              variant="link"
              size="sm"
              onClick={() => void copyId()}
              className="mt-1 h-auto max-w-full justify-start gap-1 p-0 font-mono text-[10px] text-muted-foreground"
            >
              <span className="truncate">{detail.id}</span>
              {copied ? (
                <Check className="size-3 shrink-0 text-emerald-500" />
              ) : (
                <Copy className="size-3 shrink-0" />
              )}
            </Button>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={() => retry.mutate(endpointId)}
            disabled={!endpointId || retry.isPending || !selectedEndpoint?.enabled}
            title={endpointId ? 'Retry this event' : 'Select an endpoint before retrying'}
          >
            <RotateCcw className={`size-3.5 ${retry.isPending ? 'animate-spin' : ''}`} />
            {retry.isPending ? 'Retrying…' : 'Retry'}
          </Button>
          <SheetClose asChild>
            <Button variant="ghost" size="icon" aria-label="Close event details">
              <X className="size-4" />
            </Button>
          </SheetClose>
        </SheetHeader>

        <div className="flex-1 overflow-y-auto p-5">
          {detailQuery.isError && (
            <div className="mb-5 flex gap-3 rounded-lg border border-amber-500/20 bg-amber-500/5 p-3">
              <CircleAlert className="mt-0.5 size-4 shrink-0 text-amber-500" />
              <div>
                <p className="text-xs font-medium">Could not refresh event details</p>
                <p className="mt-1 text-xs text-muted-foreground">
                  Showing the data from the events list. {errorMessage(detailQuery.error)}
                </p>
              </div>
            </div>
          )}

          <div className="grid grid-cols-2 gap-px overflow-hidden rounded-lg border border-border bg-border">
            {[
              { label: 'Unique ID', value: detail.unique_id || '—' },
              { label: 'Ingested', value: <Timestamp value={detail.ingested_at} /> },
              { label: 'Created', value: <Timestamp value={detail.created_at} /> },
              { label: 'Updated', value: <Timestamp value={detail.updated_at} /> },
            ].map(({ label, value }) => (
              <div key={label} className="min-w-0 bg-card p-3">
                <p className="text-[10px] uppercase tracking-wider text-muted-foreground">
                  {label}
                </p>
                <div className="mt-1.5 truncate font-mono text-xs font-medium">{value}</div>
              </div>
            ))}
          </div>

          <Button asChild variant="outline" className="mt-5 w-full justify-between">
            <Link
              to={`${workspacePath(workspaceName, 'deliveries')}?event_id=${encodeURIComponent(detail.id)}`}
            >
              <span className="flex items-center gap-2">
                <ListChecks className="size-4" />
                View delivery attempts
              </span>
              <ChevronRight className="size-4" />
            </Link>
          </Button>

          <div className="mt-5 rounded-lg border border-border bg-card p-4">
            <div className="flex items-start justify-between gap-4">
              <div>
                <h3 className="text-xs font-semibold">Retry event</h3>
                <p className="mt-1 text-[11px] leading-4 text-muted-foreground">
                  Create a manual delivery attempt for a specific endpoint.
                </p>
              </div>
              <span className="rounded bg-muted px-1.5 py-0.5 font-mono text-[9px] text-muted-foreground">
                MANUAL
              </span>
            </div>
            <NativeSelect
              className="mt-3 w-full"
              aria-label="Retry endpoint"
              value={endpointId}
              onChange={(changeEvent) => {
                retry.reset()
                setEndpointId(changeEvent.target.value)
              }}
              disabled={endpointsQuery.isLoading || endpointsQuery.isError || retry.isPending}
            >
              <option value="">
                {endpointsQuery.isLoading ? 'Loading endpoints…' : 'Select an endpoint…'}
              </option>
              {endpoints.map((endpoint) => (
                <option key={endpoint.id} value={endpoint.id} disabled={!endpoint.enabled}>
                  {endpoint.name || endpoint.request.url || endpoint.id}
                  {!endpoint.enabled ? ' (disabled)' : ''}
                </option>
              ))}
            </NativeSelect>
            {!endpointsQuery.isLoading && !endpointsQuery.isError && !endpoints.length && (
              <p className="mt-2 text-[11px] text-muted-foreground">
                Create an endpoint before retrying this event.
              </p>
            )}
            {endpointsQuery.isError && (
              <p className="field-error">
                Could not load endpoints: {errorMessage(endpointsQuery.error)}
              </p>
            )}
            {retry.isError && (
              <p className="field-error">Could not retry event: {errorMessage(retry.error)}</p>
            )}
            {retry.isSuccess && (
              <p className="mt-2 text-xs text-emerald-600 dark:text-emerald-400">
                Manual delivery queued successfully.
              </p>
            )}
          </div>

          <div className="mt-5">
            <div className="mb-2 flex items-center justify-between">
              <h3 className="text-xs font-semibold">Event data</h3>
              <span className="text-[10px] text-muted-foreground">application/json</span>
            </div>
            <pre className="max-h-[32rem] overflow-auto rounded-lg border border-border bg-zinc-950 p-4 font-mono text-[11px] leading-5 text-zinc-300">
              <code>{JSON.stringify(detail.data, null, 2)}</code>
            </pre>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  )
}

export function EventsPage() {
  const workspaceName = useWorkspaceName()
  const navigate = useNavigate()
  const location = useLocation()
  const [searchParams, setSearchParams] = useSearchParams()
  const detailRoute = useMatch('/workspaces/:workspaceName/events/:eventId')
  const selectedEventId = detailRoute?.params.eventId
  const eventsPath = workspacePath(workspaceName, 'events')
  const searchParamsKey = searchParams.toString()
  const listParams = useMemo(
    () => listQueryParamsFromSearchParams(new URLSearchParams(searchParamsKey), defaultEventParams),
    [searchParamsKey],
  )
  const setListParams = useCallback(
    (next: EventListParams | ((current: EventListParams) => EventListParams)) => {
      const resolved = typeof next === 'function' ? next(listParams) : next
      setSearchParams(listQueryString(resolved), { replace: true })
    },
    [listParams, setSearchParams],
  )
  const [live, setLive] = useState(false)
  const [liveStarting, setLiveStarting] = useState(false)
  const [liveReady, setLiveReady] = useState(false)
  const [liveCursor, setLiveCursor] = useState<string>()
  const [liveEvents, setLiveEvents] = useState<WebhookEvent[]>([])
  const [liveSession, setLiveSession] = useState(0)
  const liveSessionRef = useRef(0)
  const [liveWorkspaceId, setLiveWorkspaceId] = useState<string>()
  const [liveStartError, setLiveStartError] = useState<unknown>(null)
  const workspaceQuery = useQuery({
    queryKey: ['workspaces'],
    queryFn: api.workspaces,
  })
  const workspace = workspaceQuery.data?.find((item) => item.name === workspaceName)
  const workspaceId = workspace?.id ?? (workspaceName === 'default' ? 'default' : undefined)
  const activeLive = live && liveWorkspaceId === workspaceId
  const eventsQuery = useQuery({
    queryKey: ['workspaces', workspaceId, 'events', listParams],
    queryFn: () => api.events(workspaceId!, listParams),
    enabled: Boolean(workspaceId && !activeLive),
    placeholderData: (previous, previousQuery) =>
      workspacePlaceholderData(workspaceId, previous, previousQuery),
  })
  const page = eventsQuery.data
  const historyEvents = page?.data ?? []
  const liveParams = createListQueryParams({
    limit: 100,
    sort: 'id.asc',
    after: activeLive ? liveCursor : undefined,
    event_type: listParams.event_type,
    unique_id: listParams.unique_id,
    ingested_at: listParams.ingested_at,
  })
  const liveEventsQuery = useQuery({
    queryKey: ['workspaces', workspaceId, 'events', 'live', liveSession, liveParams],
    queryFn: () => api.events(workspaceId!, liveParams),
    enabled: Boolean(workspaceId && activeLive && liveReady),
    refetchInterval: activeLive ? 3_000 : false,
    refetchOnWindowFocus: false,
  })
  const events = activeLive ? liveEvents : historyEvents
  const selectedEventFromList = selectedEventId
    ? events.find((event) => event.id === selectedEventId)
    : undefined
  const selectedEventQuery = useQuery({
    queryKey: ['workspaces', workspaceId, 'events', 'detail', selectedEventId],
    queryFn: () => api.event(workspaceId!, selectedEventId!),
    enabled: Boolean(workspaceId && selectedEventId && !selectedEventFromList),
    refetchOnWindowFocus: false,
  })
  const selectedEvent = selectedEventFromList ?? selectedEventQuery.data

  useEffect(() => {
    if (liveWorkspaceId === undefined) {
      setLiveWorkspaceId(workspaceId)
      return
    }
    if (liveWorkspaceId === workspaceId) return

    liveSessionRef.current += 1
    setLive(false)
    setLiveStarting(false)
    setLiveReady(false)
    setLiveCursor(undefined)
    setLiveEvents([])
    setLiveWorkspaceId(workspaceId)
    setLiveStartError(null)
  }, [liveWorkspaceId, workspaceId])

  useEffect(() => {
    const batch = liveEventsQuery.data?.data ?? []
    if (!activeLive || !liveReady || !batch.length) return

    setLiveEvents((current) => {
      const knownIds = new Set(current.map((event) => event.id))
      const incoming = batch.filter((event) => !knownIds.has(event.id))
      const next = [...incoming].reverse().concat(current)
      return next.length > 500 ? next.slice(0, 500) : next
    })
    setLiveCursor(batch[batch.length - 1].id)
  }, [activeLive, liveEventsQuery.data, liveReady])

  const loading =
    !activeLive && ((!workspaceId && workspaceQuery.isLoading) || eventsQuery.isLoading)
  const resolutionError = !workspaceId && !workspaceQuery.isLoading
  const loadError = resolutionError
    ? (workspaceQuery.error ?? new Error(`Workspace “${workspaceName}” was not found.`))
    : eventsQuery.error
  const liveError = activeLive ? (liveStartError ?? liveEventsQuery.error) : null

  const initializeLiveSession = async () => {
    if (!workspaceId) return
    const session = liveSessionRef.current + 1
    liveSessionRef.current = session
    setLiveSession(session)
    setLiveWorkspaceId(workspaceId)
    setLive(true)
    setLiveStarting(true)
    setLiveReady(false)
    setLiveCursor(undefined)
    setLiveEvents([])
    setLiveStartError(null)

    try {
      const latestPage = await api.events(
        workspaceId,
        createListQueryParams({ limit: 1, sort: 'id.desc' }),
      )
      if (liveSessionRef.current !== session) return
      setLiveCursor(latestPage.data[0]?.id)
      setLiveReady(true)
    } catch (error) {
      if (liveSessionRef.current !== session) return
      setLiveStartError(error)
    } finally {
      if (liveSessionRef.current === session) setLiveStarting(false)
    }
  }

  const toggleLive = () => {
    if (!activeLive) {
      void initializeLiveSession()
      return
    }
    liveSessionRef.current += 1
    setLive(false)
    setLiveStarting(false)
    setLiveReady(false)
    setLiveCursor(undefined)
    setLiveEvents([])
    setLiveWorkspaceId(workspaceId)
    setLiveStartError(null)
  }

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
        title="Events"
        description={`Inspect ingested event data and retry delivery in the ${workspaceName} workspace.`}
        actions={
          <>
            <Button
              variant={activeLive ? 'default' : 'outline'}
              onClick={toggleLive}
              disabled={!workspaceId || liveStarting}
              aria-pressed={activeLive}
            >
              {activeLive ? <Pause className="size-3.5" /> : <Play className="size-3.5" />}
              {liveStarting ? 'Starting live…' : activeLive ? 'Live mode' : 'Start live'}
            </Button>
            <Button
              variant="outline"
              size="icon"
              onClick={() => {
                if (!activeLive) {
                  void eventsQuery.refetch()
                } else if (liveReady) {
                  void liveEventsQuery.refetch()
                } else {
                  void initializeLiveSession()
                }
              }}
              disabled={!workspaceId || liveStarting}
              aria-label="Refresh events"
            >
              <RefreshCcw
                className={`size-4 ${eventsQuery.isFetching || liveEventsQuery.isFetching || liveStarting ? 'animate-spin' : ''}`}
              />
            </Button>
          </>
        }
      />

      <Card className="overflow-hidden">
        <QueryFilter
          value={listParams}
          onChange={setListParams}
          config={eventFilterConfig}
          storageKey={`webhookx:event-views:${workspaceId ?? workspaceName}`}
          defaultParams={defaultEventParams}
          resultCount={events.length}
          total={activeLive ? undefined : page?.total}
          busy={
            activeLive
              ? liveEventsQuery.isFetching
              : eventsQuery.isFetching && !eventsQuery.isLoading
          }
          disabled={activeLive}
        />

        {loading && <LoadingRows rows={7} />}

        {!activeLive && loadError && !loading && (
          <div className="m-5 rounded-lg border border-red-500/20 bg-red-500/5 p-4">
            <div className="flex gap-3">
              <CircleAlert className="mt-0.5 size-4 shrink-0 text-red-500" />
              <div>
                <p className="text-xs font-medium">Could not load events</p>
                <p className="mt-1 text-xs text-muted-foreground">{errorMessage(loadError)}</p>
              </div>
            </div>
            {workspaceId && (
              <Button
                variant="outline"
                size="sm"
                className="mt-3"
                onClick={() => eventsQuery.refetch()}
              >
                Try again
              </Button>
            )}
          </div>
        )}

        {activeLive && liveError && (
          <div className="m-5 rounded-lg border border-red-500/20 bg-red-500/5 p-4">
            <div className="flex gap-3">
              <CircleAlert className="mt-0.5 size-4 shrink-0 text-red-500" />
              <div>
                <p className="text-xs font-medium">Could not start the live event stream</p>
                <p className="mt-1 text-xs text-muted-foreground">{errorMessage(liveError)}</p>
              </div>
            </div>
            <Button
              variant="outline"
              size="sm"
              className="mt-3"
              onClick={() => void initializeLiveSession()}
              disabled={liveStarting}
            >
              Try again
            </Button>
          </div>
        )}

        {!loading && !(activeLive ? liveError : loadError) && (
          <div className="overflow-x-auto">
            <Table className="data-table">
              <TableHeader>
                <TableRow>
                  <TableHead>Type</TableHead>
                  <TableHead>ID</TableHead>
                  <TableHead>Unique ID</TableHead>
                  <TableHead>Ingested</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead>Updated</TableHead>
                  <TableHead />
                </TableRow>
              </TableHeader>
              <TableBody>
                {events.map((event) => (
                  <TableRow
                    key={event.id}
                    className="cursor-pointer [&>td]:h-12 [&>td]:py-0"
                    onClick={() =>
                      void navigate(
                        {
                          pathname: `${eventsPath}/${encodeURIComponent(event.id)}`,
                          search: location.search,
                        },
                        { state: { eventInspectorFromList: true } },
                      )
                    }
                  >
                    <TableCell className="max-w-sm">
                      <p className="truncate font-mono text-xs font-medium">{event.event_type}</p>
                    </TableCell>
                    <TableCell className="max-w-sm">
                      <p className="mono-id mt-1 truncate">{event.id}</p>
                    </TableCell>
                    <TableCell className="max-w-xs">
                      {event.unique_id ? (
                        <span
                          className="block truncate font-mono text-[11px]"
                          title={event.unique_id}
                        >
                          {event.unique_id}
                        </span>
                      ) : (
                        <span className="text-muted-foreground">—</span>
                      )}
                    </TableCell>
                    <TableCell className="whitespace-nowrap text-muted-foreground">
                      <Timestamp value={event.ingested_at} />
                    </TableCell>
                    <TableCell className="whitespace-nowrap text-muted-foreground">
                      <Timestamp value={event.created_at} />
                    </TableCell>
                    <TableCell className="whitespace-nowrap text-muted-foreground">
                      <Timestamp value={event.updated_at} />
                    </TableCell>
                    <TableCell>
                      <ChevronRight className="size-4 text-muted-foreground" />
                    </TableCell>
                  </TableRow>
                ))}
                {activeLive && !events.length && (
                  <TableRow>
                    <TableCell colSpan={6} className="py-14 text-center text-muted-foreground">
                      <p className="text-sm font-medium text-foreground">Waiting for events...</p>
                      <p className="mt-1.5 text-xs">Events will appear here as they arrive</p>
                    </TableCell>
                  </TableRow>
                )}
                {!activeLive && !events.length && (
                  <TableRow>
                    <TableCell colSpan={6} className="py-14 text-center text-muted-foreground">
                      No events match the current query.
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        )}

        {!activeLive && !loading && !loadError && (
          <div className="flex flex-wrap items-center justify-between gap-3 border-t border-border px-3 py-2.5">
            <label className="flex items-center gap-2 text-xs text-muted-foreground">
              Rows
              <NativeSelect
                className="h-8 w-20"
                aria-label="Events per page"
                value={String(listParams.limit)}
                onChange={(changeEvent) =>
                  setListParams((current) => ({
                    ...current,
                    limit: Number(changeEvent.target.value),
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
                disabled={!page?.prev || eventsQuery.isFetching}
              >
                <ChevronLeft className="size-3.5" />
                Previous
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => moveToCursor(page?.next)}
                disabled={!page?.next || eventsQuery.isFetching}
              >
                Next
                <ChevronRight className="size-3.5" />
              </Button>
            </div>
          </div>
        )}
      </Card>

      {selectedEvent && workspaceId && (
        <EventInspector
          key={selectedEvent.id}
          event={selectedEvent}
          workspaceId={workspaceId}
          onClose={() => {
            const openedFromList = Boolean(
              (location.state as { eventInspectorFromList?: boolean } | null)
                ?.eventInspectorFromList,
            )
            if (openedFromList) {
              void navigate(-1)
              return
            }
            void navigate({ pathname: eventsPath, search: location.search }, { replace: true })
          }}
        />
      )}
    </>
  )
}
