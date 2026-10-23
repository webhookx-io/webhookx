import { Textarea } from '@/components/ui/textarea'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { toast } from 'sonner'
import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm, type UseFormReturn } from 'react-hook-form'
import { useLocation, useMatch, useNavigate, useSearchParams } from 'react-router-dom'
import { z } from 'zod'
import {
  Braces,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  CircleAlert,
  Gauge,
  Loader2,
  MoreHorizontal,
  Plus,
  RotateCcw,
  Send,
  Settings2,
  SlidersHorizontal,
  Tag,
  Trash2,
  Webhook,
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
import type {
  Endpoint,
  EndpointInput,
  EndpointListParams,
  EndpointMethod,
  EndpointPage,
} from '@/types'
import { errorMessage } from '@/lib/utils'
import { PageHeader } from '@/components/shared/page-header'
import { LoadingRows } from '@/components/shared/loading'
import { Timestamp } from '@/components/shared/timestamp'
import {
  MetadataEditor,
  metadataEntriesSchema,
  metadataEntriesToRecord,
  metadataToEntries,
} from '@/components/shared/metadata-editor'
import {
  QueryFilter,
  type QueryFilterConfig,
  type QueryPresetView,
} from '@/components/shared/query-filter'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { AppDialog } from '@/components/shared/app-dialog'
import { Input } from '@/components/ui/input'
import { NativeSelect } from '@/components/ui/native-select'
import { Switch } from '@/components/ui/switch'
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

const methods: EndpointMethod[] = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH']

const endpointDefaultParams = createListQueryParams({
  limit: 20,
  sort: 'id.desc',
})

const endpointFilterConfig: QueryFilterConfig = {
  showSort: true,
  fields: [
    {
      key: 'name',
      type: 'string',
      label: 'Name',
      placeholder: 'Filter by exact endpoint name…',
      quickSearch: true,
    },
    {
      key: 'enabled',
      type: 'boolean',
      label: 'Status',
      trueLabel: 'Enabled',
      falseLabel: 'Disabled',
    },
    { key: 'created_at', type: 'created_at', label: 'Created' },
    { key: 'metadata', type: 'metadata', label: 'Metadata' },
  ],
}

const endpointPresetViews: QueryPresetView[] = [
  { id: 'enabled', name: 'Enabled', params: { enabled: true } },
]

function parseCommaSeparated(value: string) {
  return value
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
}

function validRetryAttempts(value: string) {
  const attempts = parseCommaSeparated(value)
  return (
    attempts.length > 0 &&
    attempts.every((attempt) => /^\d+$/.test(attempt) && Number.isSafeInteger(Number(attempt)))
  )
}

const endpointSchema = z.object({
  name: z.string(),
  description: z.string(),
  enabled: z.boolean(),
  url: z.string().min(1, 'Enter a request URL.'),
  method: z.enum(['GET', 'POST', 'PUT', 'DELETE', 'PATCH']),
  headers: metadataEntriesSchema,
  timeout: z
    .number()
    .int('Use a whole number of milliseconds.')
    .min(0, 'Timeout cannot be negative.')
    .max(60_000, 'Timeout cannot exceed 60,000 milliseconds.'),
  retryAttempts: z
    .string()
    .refine(
      validRetryAttempts,
      'Enter one or more non-negative whole numbers separated by commas.',
    ),
  events: z.string(),
  rateLimitEnabled: z.boolean(),
  rateLimitQuota: z.number().int().min(0),
  rateLimitPeriod: z.number().int().min(1),
  metadata: metadataEntriesSchema,
})

type EndpointForm = z.infer<typeof endpointSchema>

const emptyEndpoint: EndpointForm = {
  name: '',
  description: '',
  enabled: true,
  url: '',
  method: 'POST',
  headers: [],
  timeout: 10_000,
  retryAttempts: '0, 60, 3600',
  events: '',
  rateLimitEnabled: false,
  rateLimitQuota: 100,
  rateLimitPeriod: 60,
  metadata: [],
}

function endpointToForm(endpoint: Endpoint): EndpointForm {
  return {
    name: endpoint.name ?? '',
    description: endpoint.description ?? '',
    enabled: endpoint.enabled ?? true,
    url: endpoint.request?.url ?? '',
    method: endpoint.request?.method ?? 'POST',
    headers: metadataToEntries(endpoint.request?.headers),
    timeout: endpoint.request?.timeout ?? 10_000,
    retryAttempts: (endpoint.retry?.config?.attempts?.length
      ? endpoint.retry.config.attempts
      : [0, 60, 3600]
    ).join(', '),
    events: (endpoint.events ?? []).join(', '),
    rateLimitEnabled: Boolean(endpoint.rate_limit),
    rateLimitQuota: endpoint.rate_limit?.quota ?? 100,
    rateLimitPeriod: endpoint.rate_limit?.period ?? 60,
    metadata: metadataToEntries(endpoint.metadata),
  }
}

function formToInput(values: EndpointForm): EndpointInput {
  const headers = metadataEntriesToRecord(values.headers)
  return {
    name: values.name.trim() || null,
    description: values.description.trim() || null,
    enabled: values.enabled,
    request: {
      url: values.url.trim(),
      method: values.method,
      headers: Object.keys(headers).length ? headers : null,
      timeout: values.timeout,
    },
    retry: {
      strategy: 'fixed',
      config: {
        attempts: parseCommaSeparated(values.retryAttempts).map(Number),
      },
    },
    events: parseCommaSeparated(values.events),
    metadata: metadataEntriesToRecord(values.metadata),
    rate_limit: values.rateLimitEnabled
      ? { quota: values.rateLimitQuota, period: values.rateLimitPeriod }
      : null,
  }
}

function Toggle({
  checked,
  onChange,
  label,
  description,
}: {
  checked: boolean
  onChange: (checked: boolean) => void
  label: string
  description: string
}) {
  return (
    <div className="flex min-h-20 items-center justify-between gap-4 rounded-lg border border-border bg-background p-3.5 transition-colors hover:border-foreground/15">
      <div className="min-w-0">
        <p className="text-sm font-medium">{label}</p>
        <p className="mt-1 text-xs leading-5 text-muted-foreground">{description}</p>
      </div>
      <Switch checked={checked} aria-label={label} onCheckedChange={onChange} />
    </div>
  )
}

function FormSection({
  icon: Icon,
  title,
  description,
  children,
}: {
  icon: typeof Webhook
  title: string
  description: string
  children: ReactNode
}) {
  return (
    <section className="grid gap-5 border-b border-border px-5 py-6 sm:px-6 md:grid-cols-[150px_minmax(0,1fr)] md:gap-7">
      <div>
        <div className="flex items-center gap-2.5">
          <span className="grid size-7 shrink-0 place-items-center rounded-md border border-primary/15 bg-primary/8 text-primary">
            <Icon className="size-3.5" />
          </span>
          <h3 className="text-sm font-semibold">{title}</h3>
        </div>
        <p className="mt-2 text-xs leading-5 text-muted-foreground md:pl-9">{description}</p>
      </div>
      <div className="min-w-0">{children}</div>
    </section>
  )
}

function EndpointFields({ form }: { form: UseFormReturn<EndpointForm> }) {
  const [advancedOpen, setAdvancedOpen] = useState(false)
  const rateLimitEnabled = form.watch('rateLimitEnabled')
  const headers = form.watch('headers')
  const metadata = form.watch('metadata')
  const retryAttempts = parseCommaSeparated(form.watch('retryAttempts')).length
  const advancedHasErrors = Boolean(
    form.formState.errors.headers ||
    form.formState.errors.retryAttempts ||
    form.formState.errors.rateLimitQuota ||
    form.formState.errors.rateLimitPeriod ||
    form.formState.errors.metadata,
  )

  useEffect(() => {
    if (form.formState.submitCount > 0 && advancedHasErrors) setAdvancedOpen(true)
  }, [advancedHasErrors, form.formState.submitCount])

  return (
    <div>
      <FormSection
        icon={Webhook}
        title="Destination"
        description="Identify the endpoint and define where matching events are delivered."
      >
        <div className="space-y-5">
          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <label className="label" htmlFor="endpoint-name">
                Name
              </label>
              <Input
                id="endpoint-name"
                autoFocus
                placeholder="Production orders"
                {...form.register('name')}
              />
              {form.formState.errors.name && (
                <p className="field-error">{form.formState.errors.name.message}</p>
              )}
            </div>
            <div>
              <label className="label" htmlFor="endpoint-url">
                Request URL
              </label>
              <Input
                id="endpoint-url"
                className="font-mono text-xs"
                placeholder="https://api.example.com/webhooks"
                {...form.register('url')}
              />
              {form.formState.errors.url && (
                <p className="field-error">{form.formState.errors.url.message}</p>
              )}
            </div>
          </div>

          <div>
            <label className="label" htmlFor="endpoint-description">
              Description
            </label>
            <Textarea
              id="endpoint-description"
              rows={2}
              className="resize-y"
              placeholder="Where this endpoint delivers matching events."
              {...form.register('description')}
            />
          </div>

          <Toggle
            checked={form.watch('enabled')}
            onChange={(checked) => form.setValue('enabled', checked, { shouldDirty: true })}
            label="Endpoint enabled"
            description="Deliver matching webhook events to this endpoint."
          />
        </div>
      </FormSection>

      <FormSection
        icon={Send}
        title="HTTP request"
        description="Configure the outbound request used for each delivery attempt."
      >
        <div className="grid gap-4 sm:grid-cols-2">
          <div>
            <label className="label" htmlFor="endpoint-method">
              HTTP method
            </label>
            <NativeSelect id="endpoint-method" className="w-full" {...form.register('method')}>
              {methods.map((method) => (
                <option key={method} value={method}>
                  {method}
                </option>
              ))}
            </NativeSelect>
          </div>
          <div>
            <label className="label" htmlFor="endpoint-timeout">
              Timeout (milliseconds)
            </label>
            <Input
              id="endpoint-timeout"
              type="number"
              min={0}
              max={60_000}
              {...form.register('timeout', { valueAsNumber: true })}
            />
            {form.formState.errors.timeout && (
              <p className="field-error">{form.formState.errors.timeout.message}</p>
            )}
          </div>
        </div>
      </FormSection>

      <FormSection
        icon={Settings2}
        title="Event routing"
        description="Choose which event types should be delivered to this destination."
      >
        <div>
          <label className="label" htmlFor="endpoint-events">
            Event types
          </label>
          <Input
            id="endpoint-events"
            className="font-mono text-xs"
            placeholder="order.created, order.updated"
            {...form.register('events')}
          />
          <p className="mt-1.5 text-[11px] leading-4 text-muted-foreground">
            Enter comma-separated event type names. Leave blank to keep the subscription list empty.
          </p>
        </div>
      </FormSection>

      <section className="border-b border-border">
        <button
          type="button"
          className="flex w-full items-center gap-3 px-5 py-5 text-left transition-colors hover:bg-muted/35 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary sm:px-6"
          aria-expanded={advancedOpen}
          aria-controls="endpoint-advanced-configuration"
          onClick={() => setAdvancedOpen((open) => !open)}
        >
          <span className="grid size-9 shrink-0 place-items-center rounded-lg border border-border bg-muted text-muted-foreground">
            <SlidersHorizontal className="size-4" />
          </span>
          <span className="min-w-0 flex-1">
            <span className="block text-sm font-semibold">Advanced Configuration</span>
            <span className="mt-1 block text-xs leading-5 text-muted-foreground">
              Configure headers, retries, rate limits, and metadata only when needed.
            </span>
          </span>
          <span className="flex shrink-0 items-center gap-2">
            {headers.length > 0 && (
              <Badge variant="outline" className="hidden text-[10px] sm:inline-flex">
                {headers.length} headers
              </Badge>
            )}
            {rateLimitEnabled && (
              <Badge variant="secondary" className="hidden text-[10px] sm:inline-flex">
                Rate limit on
              </Badge>
            )}
            {advancedHasErrors && (
              <Badge variant="destructive" className="hidden text-[10px] sm:inline-flex">
                Needs attention
              </Badge>
            )}
            <ChevronDown
              className={`size-4 text-muted-foreground transition-transform ${advancedOpen ? 'rotate-180' : ''}`}
            />
          </span>
        </button>

        {advancedOpen && (
          <div id="endpoint-advanced-configuration" className="border-t border-border bg-muted/10">
            <FormSection
              icon={Braces}
              title="Request headers"
              description="Attach optional headers to every outbound request."
            >
              <MetadataEditor
                idPrefix="endpoint-header"
                label="Request headers"
                description="Add optional headers to every outbound request."
                value={headers}
                onChange={(nextHeaders) =>
                  form.setValue('headers', nextHeaders, {
                    shouldDirty: true,
                    shouldValidate: true,
                  })
                }
              />
            </FormSection>

            <FormSection
              icon={RotateCcw}
              title="Retry policy"
              description="Schedule fixed delays for subsequent delivery attempts."
            >
              <div>
                <label className="label" htmlFor="endpoint-retry-attempts">
                  Fixed retry attempts (seconds)
                </label>
                <Input
                  id="endpoint-retry-attempts"
                  className="font-mono text-xs"
                  placeholder="0, 60, 3600"
                  {...form.register('retryAttempts')}
                />
                <p className="mt-1.5 text-[11px] leading-4 text-muted-foreground">
                  {retryAttempts} configured {retryAttempts === 1 ? 'attempt' : 'attempts'}. Use
                  comma-separated non-negative delays.
                </p>
                {form.formState.errors.retryAttempts && (
                  <p className="field-error">{form.formState.errors.retryAttempts.message}</p>
                )}
              </div>
            </FormSection>

            <FormSection
              icon={Gauge}
              title="Traffic controls"
              description="Limit outbound deliveries with a quota over a rolling period."
            >
              <div className="overflow-hidden rounded-lg border border-border bg-muted/20">
                <div className="p-3">
                  <Toggle
                    checked={rateLimitEnabled}
                    onChange={(checked) =>
                      form.setValue('rateLimitEnabled', checked, { shouldDirty: true })
                    }
                    label="Rate limiting"
                    description="Limit deliveries with a quota and period."
                  />
                </div>
                {rateLimitEnabled && (
                  <div className="grid gap-4 border-t border-border bg-background p-4 sm:grid-cols-2">
                    <div>
                      <label className="label" htmlFor="endpoint-rate-quota">
                        Request quota
                      </label>
                      <Input
                        id="endpoint-rate-quota"
                        type="number"
                        min={0}
                        {...form.register('rateLimitQuota', { valueAsNumber: true })}
                      />
                      {form.formState.errors.rateLimitQuota && (
                        <p className="field-error">
                          {form.formState.errors.rateLimitQuota.message}
                        </p>
                      )}
                    </div>
                    <div>
                      <label className="label" htmlFor="endpoint-rate-period">
                        Period (seconds)
                      </label>
                      <Input
                        id="endpoint-rate-period"
                        type="number"
                        min={1}
                        {...form.register('rateLimitPeriod', { valueAsNumber: true })}
                      />
                      {form.formState.errors.rateLimitPeriod && (
                        <p className="field-error">
                          {form.formState.errors.rateLimitPeriod.message}
                        </p>
                      )}
                    </div>
                  </div>
                )}
              </div>
            </FormSection>

            <FormSection
              icon={Tag}
              title="Metadata"
              description="Attach key-value context for filtering, grouping, and automation."
            >
              <MetadataEditor
                idPrefix="endpoint"
                value={metadata}
                onChange={(nextMetadata) =>
                  form.setValue('metadata', nextMetadata, {
                    shouldDirty: true,
                    shouldValidate: true,
                  })
                }
              />
            </FormSection>
          </div>
        )}
      </section>
    </div>
  )
}

function EndpointDrawer({
  open,
  onClose,
  title,
  description,
  form,
  busy = false,
  children,
}: {
  open: boolean
  onClose: () => void
  title: string
  description: string
  form?: UseFormReturn<EndpointForm>
  busy?: boolean
  children: ReactNode
}) {
  const name = form?.watch('name').trim()
  const url = form?.watch('url').trim()
  const method = form?.watch('method')
  const enabled = form?.watch('enabled')
  const eventCount = parseCommaSeparated(form?.watch('events') ?? '').length

  return (
    <Sheet open={open} onOpenChange={(nextOpen) => !nextOpen && !busy && onClose()}>
      <SheetContent
        className="gap-0 bg-background p-0"
        showCloseButton={false}
        onEscapeKeyDown={(event) => busy && event.preventDefault()}
        onPointerDownOutside={(event) => busy && event.preventDefault()}
      >
        <SheetHeader className="flex-row items-start gap-3 border-b border-border px-5 py-4 text-left sm:px-6">
          <span className="mt-0.5 grid size-9 shrink-0 place-items-center rounded-lg border border-primary/15 bg-primary/8 text-primary">
            <Webhook className="size-4" />
          </span>
          <div className="min-w-0 flex-1">
            <SheetTitle className="text-base font-semibold">{title}</SheetTitle>
            <SheetDescription className="mt-1 text-xs leading-5">{description}</SheetDescription>
          </div>
          <SheetClose asChild>
            <Button variant="ghost" size="icon" aria-label="Close endpoint drawer" disabled={busy}>
              <X className="size-4" />
            </Button>
          </SheetClose>
        </SheetHeader>

        {form && (
          <div className="border-b border-border bg-muted/35 px-5 py-3 sm:px-6">
            <div className="flex flex-wrap items-center gap-2">
              <span className="max-w-52 truncate text-xs font-medium">
                {name || 'Unnamed endpoint'}
              </span>
              <Badge variant="outline" className="rounded font-mono text-[9px]">
                {method}
              </Badge>
              <code className="min-w-0 max-w-80 truncate rounded-md border border-border bg-background px-2 py-1 text-[11px] text-muted-foreground">
                {url || 'Request URL not set'}
              </code>
              <div className="ml-auto flex items-center gap-1.5">
                {eventCount > 0 && (
                  <Badge variant="outline" className="text-[10px]">
                    {eventCount} {eventCount === 1 ? 'event' : 'events'}
                  </Badge>
                )}
                <Badge variant={enabled ? 'secondary' : 'outline'} className="text-[10px]">
                  {enabled ? 'Enabled' : 'Disabled'}
                </Badge>
              </div>
            </div>
          </div>
        )}

        {children}
      </SheetContent>
    </Sheet>
  )
}

function EndpointFormContent({
  form,
  onSubmit,
  onCancel,
  pending,
  error,
  submitLabel,
  pendingLabel,
}: {
  form: UseFormReturn<EndpointForm>
  onSubmit: (values: EndpointForm) => void
  onCancel: () => void
  pending: boolean
  error?: unknown
  submitLabel: string
  pendingLabel: string
}) {
  return (
    <form
      className="flex min-h-0 flex-1 flex-col"
      onSubmit={form.handleSubmit(onSubmit)}
      aria-busy={pending}
    >
      <div className="min-h-0 flex-1 overflow-y-auto">
        <EndpointFields form={form} />
        {Boolean(error) && (
          <div className="mx-5 my-4 flex gap-3 rounded-lg border border-red-500/20 bg-red-500/5 p-3 sm:mx-6">
            <CircleAlert className="mt-0.5 size-4 shrink-0 text-red-500" />
            <div>
              <p className="text-xs font-medium">Could not save endpoint</p>
              <p className="mt-1 text-xs leading-5 text-muted-foreground">{errorMessage(error)}</p>
            </div>
          </div>
        )}
      </div>

      <div className="flex flex-col-reverse gap-3 border-t border-border bg-background/95 px-5 py-4 supports-backdrop-filter:backdrop-blur sm:flex-row sm:items-center sm:justify-between sm:px-6">
        <p className="text-xs text-muted-foreground">
          {form.formState.isDirty ? 'You have unsaved changes.' : 'No unsaved changes.'}
        </p>
        <div className="flex justify-end gap-2">
          <Button type="button" variant="outline" onClick={onCancel} disabled={pending}>
            Cancel
          </Button>
          <Button type="submit" disabled={pending}>
            {pending && <Loader2 className="size-3.5 animate-spin" />}
            {pending ? pendingLabel : submitLabel}
          </Button>
        </div>
      </div>
    </form>
  )
}

export function EndpointsPage() {
  const workspaceName = useWorkspaceName()
  const navigate = useNavigate()
  const location = useLocation()
  const [searchParams, setSearchParams] = useSearchParams()
  const createRoute = useMatch('/workspaces/:workspaceName/endpoints/create')
  const editRoute = useMatch('/workspaces/:workspaceName/endpoints/:endpointId/edit')
  const createOpen = Boolean(createRoute)
  const editEndpointId = editRoute?.params.endpointId
  const endpointsPath = workspacePath(workspaceName, 'endpoints')
  const queryClient = useQueryClient()
  const searchParamsKey = searchParams.toString()
  const listParams = useMemo(
    () =>
      listQueryParamsFromSearchParams(new URLSearchParams(searchParamsKey), endpointDefaultParams),
    [searchParamsKey],
  )
  const setListParams = useCallback(
    (next: EndpointListParams | ((current: EndpointListParams) => EndpointListParams)) => {
      const resolved = typeof next === 'function' ? next(listParams) : next
      setSearchParams(listQueryString(resolved), { replace: true })
    },
    [listParams, setSearchParams],
  )
  const [deleting, setDeleting] = useState<Endpoint | null>(null)
  const [enabledConfirmation, setEnabledConfirmation] = useState<{
    endpoint: Endpoint
    enabled: boolean
  } | null>(null)
  const workspaceQuery = useQuery({
    queryKey: ['workspaces'],
    queryFn: api.workspaces,
  })
  const workspace = workspaceQuery.data?.find((item) => item.name === workspaceName)
  const workspaceId = workspace?.id ?? (workspaceName === 'default' ? 'default' : undefined)
  const endpointsQueryKey = ['workspaces', workspaceId, 'endpoints'] as const
  const queryKey = [...endpointsQueryKey, listParams] as const
  const endpointsQuery = useQuery({
    queryKey,
    queryFn: () => api.endpoints(workspaceId!, listParams),
    enabled: Boolean(workspaceId),
    placeholderData: (previous, previousQuery) =>
      workspacePlaceholderData(workspaceId, previous, previousQuery),
  })
  const page = endpointsQuery.data
  const endpoints = page?.data ?? []
  const endpointDetailQueryKey = [
    'workspaces',
    workspaceId,
    'endpoints',
    'detail',
    editEndpointId,
  ] as const
  const editEndpointQuery = useQuery({
    queryKey: endpointDetailQueryKey,
    queryFn: () => api.endpoint(workspaceId!, editEndpointId!),
    enabled: Boolean(workspaceId && editEndpointId),
    refetchOnMount: 'always',
    refetchOnWindowFocus: false,
  })

  const createForm = useForm<EndpointForm>({
    resolver: zodResolver(endpointSchema),
    defaultValues: emptyEndpoint,
  })
  const editForm = useForm<EndpointForm>({
    resolver: zodResolver(endpointSchema),
    defaultValues: emptyEndpoint,
  })

  const closeEndpointForm = () => {
    const openedFromList = Boolean(
      (location.state as { endpointModalFromList?: boolean } | null)?.endpointModalFromList,
    )
    if (openedFromList) {
      void navigate(-1)
      return
    }
    void navigate({ pathname: endpointsPath }, { replace: true })
  }

  const createEndpoint = useMutation({
    mutationFn: (values: EndpointForm) => api.createEndpoint(workspaceId!, formToInput(values)),
    onSuccess: (endpoint) => {
      void queryClient.invalidateQueries({ queryKey: endpointsQueryKey })
      createForm.reset(emptyEndpoint)
      closeEndpointForm()
      toast.success(`${endpoint.name || 'Endpoint'} created.`)
    },
  })

  const updateEndpoint = useMutation({
    mutationFn: ({ endpointId, values }: { endpointId: string; values: EndpointForm }) =>
      api.updateEndpoint(workspaceId!, endpointId, formToInput(values)),
    onSuccess: (updated) => {
      queryClient.setQueryData(endpointDetailQueryKey, updated)
      void queryClient.invalidateQueries({ queryKey: endpointsQueryKey })
      closeEndpointForm()
      toast.success(`${updated.name || 'Endpoint'} updated.`)
    },
  })

  const toggleEndpointEnabled = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      api.updateEndpointEnabled(workspaceId!, id, enabled),
    onMutate: async ({ id, enabled }) => {
      await queryClient.cancelQueries({ queryKey })
      const previousPage = queryClient.getQueryData<EndpointPage>(queryKey)
      queryClient.setQueryData<EndpointPage>(queryKey, (current) =>
        current
          ? {
              ...current,
              data: current.data.map((endpoint) =>
                endpoint.id === id ? { ...endpoint, enabled } : endpoint,
              ),
            }
          : current,
      )
      return { previousPage }
    },
    onError: (error, _variables, context) => {
      if (context?.previousPage) queryClient.setQueryData(queryKey, context.previousPage)
      toast.error(`Could not update endpoint: ${errorMessage(error)}`)
    },
    onSuccess: (updated) => {
      queryClient.setQueryData<EndpointPage>(queryKey, (current) =>
        current
          ? {
              ...current,
              data: current.data.map((endpoint) =>
                endpoint.id === updated.id ? updated : endpoint,
              ),
            }
          : current,
      )
      queryClient.setQueryData(
        ['workspaces', workspaceId, 'endpoints', 'detail', updated.id],
        updated,
      )
      void queryClient.invalidateQueries({ queryKey: endpointsQueryKey })
      setEnabledConfirmation(null)
      toast.success(`${updated.name || 'Endpoint'} ${updated.enabled ? 'enabled' : 'disabled'}.`)
    },
  })

  const deleteEndpoint = useMutation({
    mutationFn: (endpoint: Endpoint) => api.deleteEndpoint(workspaceId!, endpoint.id),
    onSuccess: (_, endpoint) => {
      void queryClient.invalidateQueries({ queryKey: endpointsQueryKey })
      setDeleting(null)
      toast.success(`${endpoint.name || 'Endpoint'} deleted.`)
    },
  })

  const loading = (!workspaceId && workspaceQuery.isLoading) || endpointsQuery.isLoading
  const resolutionError = !workspaceId && !workspaceQuery.isLoading
  const loadError = resolutionError
    ? (workspaceQuery.error ?? new Error(`Workspace “${workspaceName}” was not found.`))
    : endpointsQuery.error
  const editLoadError = resolutionError
    ? (workspaceQuery.error ?? new Error(`Workspace “${workspaceName}” was not found.`))
    : editEndpointQuery.error
  const editDetailLoading =
    Boolean(editEndpointId) &&
    (workspaceQuery.isLoading ||
      Boolean(workspaceId && (editEndpointQuery.isPending || editEndpointQuery.isFetching)))
  const editingEndpoint = !editDetailLoading && !editLoadError ? editEndpointQuery.data : undefined

  const openCreate = () => {
    createEndpoint.reset()
    createForm.reset(emptyEndpoint)
    void navigate(
      { pathname: `${endpointsPath}/create` },
      { state: { endpointModalFromList: true } },
    )
  }

  const openEdit = (endpoint: Endpoint) => {
    updateEndpoint.reset()
    void navigate(
      { pathname: `${endpointsPath}/${encodeURIComponent(endpoint.id)}/edit` },
      { state: { endpointModalFromList: true } },
    )
  }

  useEffect(() => {
    if (!editEndpointId) {
      editForm.reset(emptyEndpoint)
      return
    }
    if (editEndpointQuery.isFetching || !editEndpointQuery.data || editEndpointQuery.isError) return
    editForm.reset(endpointToForm(editEndpointQuery.data))
  }, [
    editForm,
    editEndpointId,
    editEndpointQuery.data,
    editEndpointQuery.dataUpdatedAt,
    editEndpointQuery.isError,
    editEndpointQuery.isFetching,
  ])

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
        title="Endpoints"
        description={`Configure webhook destinations for the ${workspaceName} workspace.`}
        actions={
          <Button onClick={openCreate} disabled={!workspaceId}>
            <Plus className="size-4" />
            Create endpoint
          </Button>
        }
      />

      <Card className="overflow-hidden">
        <QueryFilter
          value={listParams}
          onChange={setListParams}
          config={endpointFilterConfig}
          storageKey={`webhookx:endpoint-views:${workspaceId ?? workspaceName}`}
          defaultParams={endpointDefaultParams}
          presetViews={endpointPresetViews}
          resultCount={endpoints.length}
          total={page?.total}
          busy={endpointsQuery.isFetching && !endpointsQuery.isLoading}
        />

        {loading && <LoadingRows />}

        {loadError && !loading && (
          <div className="m-5 rounded-lg border border-red-500/20 bg-red-500/5 p-4">
            <div className="flex gap-3">
              <CircleAlert className="mt-0.5 size-4 shrink-0 text-red-500" />
              <div>
                <p className="text-xs font-medium">Could not load endpoints</p>
                <p className="mt-1 text-xs text-muted-foreground">{errorMessage(loadError)}</p>
              </div>
            </div>
            {workspaceId && (
              <Button
                variant="outline"
                size="sm"
                className="mt-3"
                onClick={() => endpointsQuery.refetch()}
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
                  <TableHead>Name</TableHead>
                  <TableHead>URL</TableHead>
                  <TableHead>Method</TableHead>
                  <TableHead>Events</TableHead>
                  <TableHead>Enabled</TableHead>
                  <TableHead>Retry</TableHead>
                  <TableHead>Rate limit</TableHead>
                  <TableHead>Updated</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {endpoints.map((endpoint) => {
                  const attempts = endpoint.retry?.config?.attempts ?? []
                  return (
                    <TableRow
                      key={endpoint.id}
                      tabIndex={0}
                      aria-label={`Edit ${endpoint.name || 'endpoint'}`}
                      className="cursor-pointer focus-visible:bg-muted/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary"
                      onClick={() => openEdit(endpoint)}
                      onKeyDown={(event) => {
                        if (event.target !== event.currentTarget) return
                        if (event.key !== 'Enter' && event.key !== ' ') return
                        event.preventDefault()
                        openEdit(endpoint)
                      }}
                    >
                      <TableCell className="max-w-xs">
                        <div className="flex items-center gap-3">
                          <span className="grid size-8 shrink-0 place-items-center rounded-md border border-border bg-muted">
                            <Webhook className="size-4 text-muted-foreground" />
                          </span>
                          <div className="min-w-0">
                            <p className="truncate font-medium">
                              {endpoint.name || <span className="text-muted-foreground">—</span>}
                            </p>
                            <p className="mt-0.5 truncate text-[10px] text-muted-foreground">
                              {endpoint.description || endpoint.id}
                            </p>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell className="max-w-xs">
                        <span
                          className="block truncate font-mono text-[11px]"
                          title={endpoint.request.url}
                        >
                          {endpoint.request.url}
                        </span>
                      </TableCell>
                      <TableCell>
                        <Badge className="rounded font-mono text-[9px]">
                          {endpoint.request.method || 'POST'}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        {endpoint.events?.length ? (
                          <div className="flex max-w-[220px] flex-wrap gap-1">
                            {endpoint.events.slice(0, 2).map((eventType) => (
                              <Badge key={eventType} className="rounded font-mono text-[9px]">
                                {eventType}
                              </Badge>
                            ))}
                            {endpoint.events.length > 2 && (
                              <span className="text-[10px] text-muted-foreground">
                                +{endpoint.events.length - 2}
                              </span>
                            )}
                          </div>
                        ) : (
                          <span className="text-muted-foreground">—</span>
                        )}
                      </TableCell>
                      <TableCell>
                        <Switch
                          checked={endpoint.enabled}
                          aria-label={`${endpoint.enabled ? 'Disable' : 'Enable'} ${endpoint.name || 'endpoint'}`}
                          onClick={(event) => event.stopPropagation()}
                          onCheckedChange={(enabled) => {
                            toggleEndpointEnabled.reset()
                            setEnabledConfirmation({
                              endpoint,
                              enabled,
                            })
                          }}
                          disabled={toggleEndpointEnabled.isPending}
                        />
                      </TableCell>
                      <TableCell>
                        <span
                          className="whitespace-nowrap font-mono text-[11px]"
                          title={attempts.join(', ')}
                        >
                          {attempts.length} {attempts.length === 1 ? 'attempt' : 'attempts'}
                        </span>
                      </TableCell>
                      <TableCell>
                        {endpoint.rate_limit ? (
                          <span className="whitespace-nowrap font-mono text-[11px]">
                            {endpoint.rate_limit.quota} / {endpoint.rate_limit.period}s
                          </span>
                        ) : (
                          <span className="text-muted-foreground">—</span>
                        )}
                      </TableCell>
                      <TableCell className="whitespace-nowrap text-muted-foreground">
                        <Timestamp value={endpoint.updated_at} />
                      </TableCell>
                      <TableCell>
                        <div
                          className="flex justify-end"
                          onClick={(event) => event.stopPropagation()}
                        >
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <Button
                                variant="ghost"
                                size="icon"
                                aria-label={`Actions for ${endpoint.name || 'endpoint'}`}
                              >
                                <MoreHorizontal className="size-4" />
                              </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end" className="w-36">
                              <DropdownMenuItem
                                variant="destructive"
                                onSelect={() => {
                                  deleteEndpoint.reset()
                                  setDeleting(endpoint)
                                }}
                              >
                                <Trash2 />
                                Delete
                              </DropdownMenuItem>
                            </DropdownMenuContent>
                          </DropdownMenu>
                        </div>
                      </TableCell>
                    </TableRow>
                  )
                })}
                {!endpoints.length && (
                  <TableRow>
                    <TableCell colSpan={9} className="py-14 text-center text-muted-foreground">
                      No endpoints match the current query.
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
                aria-label="Endpoints per page"
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
                disabled={!page?.prev || endpointsQuery.isFetching}
              >
                <ChevronLeft className="size-3.5" />
                Previous
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => moveToCursor(page?.next)}
                disabled={!page?.next || endpointsQuery.isFetching}
              >
                Next
                <ChevronRight className="size-3.5" />
              </Button>
            </div>
          </div>
        )}
      </Card>

      <AppDialog
        open={Boolean(enabledConfirmation)}
        onClose={() => {
          if (!toggleEndpointEnabled.isPending) setEnabledConfirmation(null)
        }}
        title={`${enabledConfirmation?.enabled ? 'Enable' : 'Disable'} endpoint`}
        description="This change takes effect immediately."
      >
        {enabledConfirmation && (
          <div className="p-5">
            <div className="rounded-lg border border-border bg-muted/40 p-4">
              <p className="text-sm font-medium">
                {enabledConfirmation.enabled ? 'Enable' : 'Disable'}{' '}
                {enabledConfirmation.endpoint.name || 'this endpoint'}?
              </p>
              <p className="mt-1 text-xs leading-5 text-muted-foreground">
                Webhook deliveries to{' '}
                <span className="font-mono">{enabledConfirmation.endpoint.request.url}</span> will{' '}
                {enabledConfirmation.enabled ? 'resume' : 'stop'}.
              </p>
            </div>
            {toggleEndpointEnabled.isError && (
              <p className="field-error mt-3">{errorMessage(toggleEndpointEnabled.error)}</p>
            )}
            <div className="mt-5 flex justify-end gap-2">
              <Button
                type="button"
                variant="outline"
                onClick={() => setEnabledConfirmation(null)}
                disabled={toggleEndpointEnabled.isPending}
              >
                Cancel
              </Button>
              <Button
                type="button"
                variant={enabledConfirmation.enabled ? 'default' : 'destructive'}
                onClick={() =>
                  toggleEndpointEnabled.mutate({
                    id: enabledConfirmation.endpoint.id,
                    enabled: enabledConfirmation.enabled,
                  })
                }
                disabled={toggleEndpointEnabled.isPending}
              >
                {toggleEndpointEnabled.isPending
                  ? 'Updating…'
                  : `${enabledConfirmation.enabled ? 'Enable' : 'Disable'} endpoint`}
              </Button>
            </div>
          </div>
        )}
      </AppDialog>

      <EndpointDrawer
        open={createOpen}
        onClose={closeEndpointForm}
        title="Create endpoint"
        description="Configure a webhook destination without leaving the endpoints list."
        form={workspaceId ? createForm : undefined}
        busy={createEndpoint.isPending}
      >
        {!workspaceId && workspaceQuery.isLoading && (
          <div className="flex-1 space-y-4 p-5 sm:p-6" aria-label="Loading workspace">
            <div className="h-20 animate-pulse rounded-lg bg-muted" />
            <div className="h-36 animate-pulse rounded-lg bg-muted" />
            <div className="h-28 animate-pulse rounded-lg bg-muted" />
          </div>
        )}
        {!workspaceId && !workspaceQuery.isLoading && (
          <div className="flex-1 p-5 sm:p-6">
            <div className="rounded-lg border border-red-500/20 bg-red-500/5 p-4">
              <p className="text-sm font-medium">Could not resolve workspace</p>
              <p className="mt-1 text-xs text-muted-foreground">{errorMessage(loadError)}</p>
            </div>
            <div className="mt-5 flex justify-end">
              <Button variant="outline" onClick={closeEndpointForm}>
                Back to endpoints
              </Button>
            </div>
          </div>
        )}
        {workspaceId && (
          <EndpointFormContent
            form={createForm}
            onSubmit={(values) => createEndpoint.mutate(values)}
            onCancel={closeEndpointForm}
            pending={createEndpoint.isPending}
            error={createEndpoint.error}
            submitLabel="Create endpoint"
            pendingLabel="Creating…"
          />
        )}
      </EndpointDrawer>

      <EndpointDrawer
        open={Boolean(editEndpointId)}
        onClose={closeEndpointForm}
        title="Edit endpoint"
        description="Update delivery, routing, retry, and traffic settings for this endpoint."
        form={editingEndpoint ? editForm : undefined}
        busy={updateEndpoint.isPending}
      >
        {editDetailLoading && (
          <div className="flex-1 space-y-4 p-5 sm:p-6" aria-label="Loading endpoint">
            <div className="h-20 animate-pulse rounded-lg bg-muted" />
            <div className="h-36 animate-pulse rounded-lg bg-muted" />
            <div className="h-28 animate-pulse rounded-lg bg-muted" />
          </div>
        )}
        {!editDetailLoading && editLoadError && (
          <div className="flex-1 p-5 sm:p-6">
            <div className="rounded-lg border border-red-500/20 bg-red-500/5 p-4">
              <div className="flex gap-3">
                <CircleAlert className="mt-0.5 size-4 shrink-0 text-red-500" />
                <div>
                  <p className="text-sm font-medium">Could not load endpoint</p>
                  <p className="mt-1 text-xs text-muted-foreground">
                    {errorMessage(editLoadError)}
                  </p>
                </div>
              </div>
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="mt-3"
                onClick={() => {
                  if (workspaceId) void editEndpointQuery.refetch()
                  else void workspaceQuery.refetch()
                }}
              >
                Try again
              </Button>
            </div>
            <div className="mt-5 flex justify-end">
              <Button variant="outline" onClick={closeEndpointForm}>
                Back to endpoints
              </Button>
            </div>
          </div>
        )}
        {!editDetailLoading && editingEndpoint && (
          <EndpointFormContent
            form={editForm}
            onSubmit={(values) => updateEndpoint.mutate({ endpointId: editingEndpoint.id, values })}
            onCancel={closeEndpointForm}
            pending={updateEndpoint.isPending}
            error={updateEndpoint.error}
            submitLabel="Save changes"
            pendingLabel="Saving…"
          />
        )}
      </EndpointDrawer>

      <AppDialog
        open={Boolean(deleting)}
        onClose={() => setDeleting(null)}
        title="Delete endpoint"
        description="This operation cannot be undone."
      >
        {deleting && (
          <div className="p-5">
            <div className="rounded-lg border border-red-500/20 bg-red-500/5 p-4">
              <p className="text-sm font-medium">Delete {deleting.name || 'this endpoint'}?</p>
              <p className="mt-1 text-xs leading-5 text-muted-foreground">
                Webhook deliveries to <span className="font-mono">{deleting.request.url}</span> will
                stop permanently.
              </p>
            </div>
            {deleteEndpoint.isError && (
              <p className="field-error mt-3">{errorMessage(deleteEndpoint.error)}</p>
            )}
            <div className="mt-5 flex justify-end gap-2">
              <Button variant="outline" onClick={() => setDeleting(null)}>
                Cancel
              </Button>
              <Button
                variant="destructive"
                onClick={() => deleteEndpoint.mutate(deleting)}
                disabled={deleteEndpoint.isPending}
              >
                {deleteEndpoint.isPending ? 'Deleting…' : 'Delete endpoint'}
              </Button>
            </div>
          </div>
        )}
      </AppDialog>
    </>
  )
}
