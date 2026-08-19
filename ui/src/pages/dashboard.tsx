import { useState, type ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  AlertTriangle,
  ArrowRight,
  Check,
  CheckCircle2,
  Circle,
  Copy,
  FileUp,
  Inbox,
  Loader2,
  Plus,
  RadioTower,
  Send,
  Webhook,
} from 'lucide-react'
import { Link } from 'react-router-dom'
import { toast } from 'sonner'
import { useWorkspaceName, workspacePath } from '@/app/workspace'
import { PageHeader } from '@/components/shared/page-header'
import { WorkspaceConfigDialog } from '@/components/workspaces/workspace-config-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { api } from '@/data/api'
import { createListQueryParams } from '@/data/list-query'
import { cn, copyText, errorMessage } from '@/lib/utils'
import type { Source } from '@/types'

const latestItemParams = createListQueryParams({ limit: 1, sort: 'id.desc' })

function sourceAddress(gatewayAddress: string, source: Source) {
  const base = gatewayAddress.replace(/\/+$/, '')
  const path = source.config.http.path?.trim().replace(/^\/+/, '')
  return path ? `${base}/${path}` : base
}

function curlForSource(gatewayAddress: string, source: Source) {
  const methods = source.config.http.methods ?? []
  const method = methods.includes('POST') ? 'POST' : (methods[0] ?? 'POST')
  return [
    `curl -X ${method} '${sourceAddress(gatewayAddress, source)}' \\`,
    `  -H 'Content-Type: application/json' \\`,
    `  -d '{"event_type":"test.created","data":{"hello":"world"}}'`,
  ].join('\n')
}

function StepStatus({
  complete,
  active,
  number,
}: {
  complete: boolean
  active: boolean
  number: number
}) {
  if (complete) {
    return (
      <span className="grid size-7 shrink-0 place-items-center rounded-full bg-emerald-500 text-white">
        <Check className="size-3.5" strokeWidth={2.5} />
      </span>
    )
  }

  return (
    <span
      className={cn(
        'grid size-7 shrink-0 place-items-center rounded-full border text-xs font-semibold',
        active
          ? 'border-primary bg-primary text-primary-foreground'
          : 'border-border bg-muted text-muted-foreground',
      )}
    >
      {number}
    </span>
  )
}

function SetupStep({
  number,
  title,
  description,
  complete,
  active,
  children,
}: {
  number: number
  title: string
  description: string
  complete: boolean
  active: boolean
  children?: ReactNode
}) {
  return (
    <section className="flex gap-4 border-b border-border px-5 py-5 last:border-b-0 sm:px-6">
      <StepStatus complete={complete} active={active} number={number} />
      <div className="min-w-0 flex-1">
        <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
          <div>
            <div className="flex flex-wrap items-center gap-2">
              <h3 className="text-sm font-semibold">{title}</h3>
              {complete && (
                <span className="text-[11px] font-medium text-emerald-600 dark:text-emerald-400">
                  Complete
                </span>
              )}
            </div>
            <p className="mt-1 text-xs leading-5 text-muted-foreground">{description}</p>
          </div>
          {children}
        </div>
      </div>
    </section>
  )
}

export function DashboardPage() {
  const workspaceName = useWorkspaceName()
  const [importOpen, setImportOpen] = useState(false)
  const [curlCopied, setCurlCopied] = useState(false)
  const workspaceQuery = useQuery({ queryKey: ['workspaces'], queryFn: api.workspaces })
  const workspace = workspaceQuery.data?.find((item) => item.name === workspaceName)
  const workspaceId = workspace?.id ?? (workspaceName === 'default' ? 'default' : undefined)
  const sourcesQuery = useQuery({
    queryKey: ['workspaces', workspaceId, 'sources', 'overview'],
    queryFn: () => api.sources(workspaceId!, latestItemParams),
    enabled: Boolean(workspaceId),
  })
  const endpointsQuery = useQuery({
    queryKey: ['workspaces', workspaceId, 'endpoints', 'overview'],
    queryFn: () => api.endpoints(workspaceId!, latestItemParams),
    enabled: Boolean(workspaceId),
  })
  const eventsQuery = useQuery({
    queryKey: ['workspaces', workspaceId, 'events', 'overview'],
    queryFn: () => api.events(workspaceId!, latestItemParams),
    enabled: Boolean(workspaceId),
    refetchInterval: (query) => (query.state.data?.data[0] ? false : 5_000),
  })
  const source = sourcesQuery.data?.data[0]
  const endpoint = endpointsQuery.data?.data[0]
  const event = eventsQuery.data?.data[0]
  const attemptsQuery = useQuery({
    queryKey: ['workspaces', workspaceId, 'attempts', 'overview', event?.id],
    queryFn: () =>
      api.attempts(
        workspaceId!,
        createListQueryParams({ limit: 1, sort: 'id.desc', event_id: event!.id }),
      ),
    enabled: Boolean(workspaceId && event?.id),
    refetchInterval: (query) => (query.state.data?.data[0] ? false : 5_000),
  })
  const attempt = attemptsQuery.data?.data[0]
  const gatewayAddress =
    import.meta.env.VITE_GATEWAY_BASE_URL?.trim() || 'http://127.0.0.1:9600'
  const curl = source ? curlForSource(gatewayAddress, source) : null
  const complete = Boolean(source && endpoint && event && attempt)
  const loading = Boolean(
    workspaceQuery.isLoading ||
    (workspaceId && (sourcesQuery.isLoading || endpointsQuery.isLoading || eventsQuery.isLoading)),
  )
  const resolutionError =
    !workspaceId && !workspaceQuery.isLoading
      ? new Error(`Workspace “${workspaceName}” was not found.`)
      : null
  const progressError =
    resolutionError ??
    workspaceQuery.error ??
    sourcesQuery.error ??
    endpointsQuery.error ??
    eventsQuery.error

  const copyCurl = async () => {
    if (!curl) return
    try {
      await copyText(curl)
      setCurlCopied(true)
      toast.success('curl command copied.')
      window.setTimeout(() => setCurlCopied(false), 1500)
    } catch (error) {
      toast.error(errorMessage(error, 'Could not copy the curl command.'))
    }
  }

  return (
    <>
      <PageHeader
        title="Overview"
        description={`Set up and verify the ${workspaceName} workspace.`}
        actions={
          <Button variant="outline" onClick={() => setImportOpen(true)} disabled={!workspaceId}>
            <FileUp />
            Sync YAML
          </Button>
        }
      />

      {loading && (
        <Card>
          <CardContent className="flex min-h-44 items-center justify-center gap-2 text-sm text-muted-foreground">
            <Loader2 className="size-4 animate-spin" />
            Loading workspace setup…
          </CardContent>
        </Card>
      )}

      {!loading && progressError && (
        <Card>
          <CardContent className="flex min-h-44 flex-col items-center justify-center text-center">
            <AlertTriangle className="size-5 text-amber-500" />
            <p className="mt-3 text-sm font-medium">Could not load workspace setup</p>
            <p className="mt-1 max-w-md text-xs text-muted-foreground">
              {errorMessage(progressError)}
            </p>
          </CardContent>
        </Card>
      )}

      {!loading && !progressError && complete && (
        <Card>
          <CardContent className="flex flex-col gap-5 px-6 py-8 sm:flex-row sm:items-center sm:justify-between sm:px-8">
            <div className="flex items-start gap-4">
              <div className="grid size-11 shrink-0 place-items-center rounded-full bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
                <CheckCircle2 className="size-5" />
              </div>
              <div>
                <h2 className="text-lg font-semibold tracking-tight">Workspace is ready</h2>
                <p className="mt-1.5 text-sm text-muted-foreground">
                  WebhookX received an event and created a delivery attempt.
                </p>
              </div>
            </div>
            <div className="flex flex-wrap gap-2 pl-[3.75rem] sm:pl-0">
              {curl && (
                <Button variant="outline" onClick={() => void copyCurl()}>
                  {curlCopied ? <Check className="text-emerald-500" /> : <Copy />}
                  {curlCopied ? 'Copied' : 'Copy test curl'}
                </Button>
              )}
              <Button asChild variant="outline">
                <Link to={workspacePath(workspaceName, 'events')}>View event</Link>
              </Button>
              <Button asChild>
                <Link to={workspacePath(workspaceName, 'deliveries')}>
                  View deliveries
                  <ArrowRight />
                </Link>
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {!loading && !progressError && !complete && (
        <Card className="gap-0 py-0">
          <CardHeader className="flex flex-row items-start justify-between gap-4 border-b border-border px-5 py-5 sm:px-6">
            <div>
              <div className="flex flex-wrap items-center gap-2">
                <h2 className="text-base font-semibold">Guided setup</h2>
                <Badge variant="secondary">Recommended</Badge>
              </div>
              <p className="mt-1 text-xs leading-5 text-muted-foreground">
                Configure the workspace manually, then send one event to verify the complete path.
              </p>
            </div>
            <Webhook className="mt-0.5 size-5 shrink-0 text-muted-foreground" />
          </CardHeader>

          <SetupStep
            number={1}
            title="Create a source"
            description={
              source
                ? `${source.name || 'Unnamed source'} receives events at ${source.config.http.path || '/'}.`
                : 'Define the HTTP path where WebhookX will receive events.'
            }
            complete={Boolean(source)}
            active={!source}
          >
            <Button asChild variant={source ? 'ghost' : 'default'} size="sm">
              <Link to={workspacePath(workspaceName, source ? 'sources' : 'sources/create')}>
                {source ? <Inbox /> : <Plus />}
                {source ? 'View sources' : 'Create source'}
              </Link>
            </Button>
          </SetupStep>

          <SetupStep
            number={2}
            title="Create an endpoint"
            description={
              endpoint
                ? `${endpoint.name || 'Unnamed endpoint'} delivers events to ${endpoint.request.url}.`
                : 'Add the destination that should receive matching events.'
            }
            complete={Boolean(endpoint)}
            active={Boolean(source && !endpoint)}
          >
            <Button
              asChild={Boolean(source)}
              variant={endpoint ? 'ghost' : 'outline'}
              size="sm"
              disabled={!source}
            >
              {source ? (
                <Link
                  to={workspacePath(workspaceName, endpoint ? 'endpoints' : 'endpoints/create')}
                >
                  {endpoint ? <RadioTower /> : <Plus />}
                  {endpoint ? 'View endpoints' : 'Create endpoint'}
                </Link>
              ) : (
                <span>Create endpoint</span>
              )}
            </Button>
          </SetupStep>

          <section className="flex gap-4 border-b border-border px-5 py-5 sm:px-6">
            <StepStatus
              complete={Boolean(event)}
              active={Boolean(source && endpoint && !event)}
              number={3}
            />
            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-center gap-2">
                <h3 className="text-sm font-semibold">Send a test event</h3>
                {event && (
                  <span className="text-[11px] font-medium text-emerald-600 dark:text-emerald-400">
                    Event received
                  </span>
                )}
              </div>
              <p className="mt-1 text-xs leading-5 text-muted-foreground">
                {event
                  ? `${event.event_type} was received by this workspace.`
                  : 'Run this command from a terminal to send a test payload through the Gateway.'}
              </p>

              {source && endpoint && !event && (
                <div className="mt-4">
                  {curl && (
                    <div className="relative overflow-hidden rounded-lg border border-border bg-zinc-950">
                      <pre className="overflow-x-auto p-4 pr-12 font-mono text-[11px] leading-5 text-zinc-300">
                        <code>{curl}</code>
                      </pre>
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon-sm"
                        className="absolute right-2 top-2 text-zinc-400 hover:bg-white/10 hover:text-white"
                        onClick={() => void copyCurl()}
                        aria-label="Copy curl command"
                      >
                        {curlCopied ? <Check className="text-emerald-400" /> : <Copy />}
                      </Button>
                    </div>
                  )}
                </div>
              )}

              {event && (
                <Button asChild variant="ghost" size="sm" className="mt-3">
                  <Link to={workspacePath(workspaceName, 'events')}>
                    <Send />
                    View event
                    <ArrowRight />
                  </Link>
                </Button>
              )}
            </div>
          </section>

          <SetupStep
            number={4}
            title="Verify delivery"
            description={
              attempt
                ? `Delivery attempt #${attempt.attempt_number} is ${attempt.status.toLowerCase()}.`
                : event
                  ? 'The event was received. Waiting for its delivery attempt…'
                  : 'After the event arrives, inspect the request, response, and retry state.'
            }
            complete={Boolean(attempt)}
            active={Boolean(event && !attempt)}
          >
            {event ? (
              <Button asChild variant={attempt ? 'default' : 'outline'} size="sm">
                <Link to={workspacePath(workspaceName, 'deliveries')}>
                  {attempt ? <CheckCircle2 /> : <Circle />}
                  View deliveries
                </Link>
              </Button>
            ) : undefined}
          </SetupStep>
        </Card>
      )}

      <WorkspaceConfigDialog
        open={importOpen}
        onClose={() => setImportOpen(false)}
        workspaceId={workspaceId}
        workspaceName={workspaceName}
        hasExistingConfiguration={Boolean(source || endpoint)}
      />
    </>
  )
}
