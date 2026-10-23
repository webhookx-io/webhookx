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
  Inbox,
  Loader2,
  MoreHorizontal,
  Plus,
  Radio,
  Settings2,
  SlidersHorizontal,
  Tag,
  Trash2,
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
import type { Source, SourceInput, SourceListParams, SourceMethod, SourcePage } from '@/types'
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
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

const methods: SourceMethod[] = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH']

const sourceDefaultParams = createListQueryParams({
  limit: 20,
  sort: 'id.desc',
})

const sourceFilterConfig: QueryFilterConfig = {
  showSort: true,
  fields: [
    {
      key: 'name',
      type: 'string',
      label: 'Name',
      placeholder: 'Filter by exact source name…',
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

const sourcePresetViews: QueryPresetView[] = [
  { id: 'enabled', name: 'Enabled', params: { enabled: true } },
]

const sourceSchema = z
  .object({
    name: z.string().max(80, 'Use 80 characters or fewer.'),
    path: z
      .string()
      .max(240, 'Use 240 characters or fewer.')
      .refine((value) => !value || value.startsWith('/'), 'Paths must start with /.'),
    methods: z
      .array(z.enum(['GET', 'POST', 'PUT', 'DELETE', 'PATCH']))
      .min(1, 'Select at least one HTTP method.'),
    enabled: z.boolean(),
    asynchronous: z.boolean(),
    responseEnabled: z.boolean(),
    responseCode: z.number().int().min(200).max(599),
    responseContentType: z.string().max(120),
    responseBody: z.string().max(10_000, 'Use 10,000 characters or fewer.'),
    rateLimitEnabled: z.boolean(),
    rateLimitQuota: z.number().int().min(0),
    rateLimitPeriod: z.number().int().min(1),
    metadata: metadataEntriesSchema,
  })
  .superRefine((value, context) => {
    if (value.responseEnabled && !value.responseContentType.trim()) {
      context.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['responseContentType'],
        message: 'Enter a response content type.',
      })
    }
  })

type SourceForm = z.infer<typeof sourceSchema>

const emptySource: SourceForm = {
  name: '',
  path: '',
  methods: ['POST'],
  enabled: true,
  asynchronous: false,
  responseEnabled: false,
  responseCode: 200,
  responseContentType: 'application/json',
  responseBody: '{"message":"OK"}',
  rateLimitEnabled: false,
  rateLimitQuota: 100,
  rateLimitPeriod: 60,
  metadata: [],
}

function sourceToForm(source: Source): SourceForm {
  const http = source.config?.http ?? {}
  return {
    name: source.name ?? '',
    path: http.path ?? '',
    methods: http.methods?.length ? http.methods : ['POST'],
    enabled: source.enabled ?? true,
    asynchronous: source.async ?? false,
    responseEnabled: Boolean(http.response),
    responseCode: http.response?.code ?? 200,
    responseContentType: http.response?.content_type ?? 'application/json',
    responseBody: http.response?.body ?? '',
    rateLimitEnabled: Boolean(source.rate_limit),
    rateLimitQuota: source.rate_limit?.quota ?? 100,
    rateLimitPeriod: source.rate_limit?.period ?? 60,
    metadata: metadataToEntries(source.metadata),
  }
}

function formToInput(values: SourceForm): SourceInput {
  return {
    name: values.name.trim() || null,
    enabled: values.enabled,
    type: 'http',
    config: {
      http: {
        ...(values.path.trim() ? { path: values.path.trim() } : {}),
        methods: values.methods,
        response: values.responseEnabled
          ? {
              code: values.responseCode,
              content_type: values.responseContentType.trim(),
              ...(values.responseBody ? { body: values.responseBody } : {}),
            }
          : null,
      },
    },
    async: values.asynchronous,
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
  icon: typeof Inbox
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

function SourceFields({ form }: { form: UseFormReturn<SourceForm> }) {
  const [advancedOpen, setAdvancedOpen] = useState(false)
  const selectedMethods = form.watch('methods')
  const responseEnabled = form.watch('responseEnabled')
  const rateLimitEnabled = form.watch('rateLimitEnabled')
  const metadata = form.watch('metadata')
  const advancedHasErrors = Boolean(
    form.formState.errors.rateLimitQuota ||
    form.formState.errors.rateLimitPeriod ||
    form.formState.errors.metadata,
  )

  useEffect(() => {
    if (form.formState.submitCount > 0 && advancedHasErrors) setAdvancedOpen(true)
  }, [advancedHasErrors, form.formState.submitCount])

  const toggleMethod = (method: SourceMethod) => {
    const next = selectedMethods.includes(method)
      ? selectedMethods.filter((item) => item !== method)
      : [...selectedMethods, method]
    form.setValue('methods', next, { shouldDirty: true, shouldValidate: true })
  }

  return (
    <div>
      <FormSection
        icon={Radio}
        title="Entry point"
        description="Name the source and define where HTTP events enter the gateway."
      >
        <div className="space-y-5">
          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <label className="label" htmlFor="source-name">
                Name
              </label>
              <Input
                id="source-name"
                autoFocus
                placeholder="Storefront ingress"
                {...form.register('name')}
              />
              <p className="mt-1.5 text-[11px] leading-4 text-muted-foreground">
                A human-friendly label for this entry point.
              </p>
              {form.formState.errors.name && (
                <p className="field-error">{form.formState.errors.name.message}</p>
              )}
            </div>
            <div>
              <label className="label" htmlFor="source-path">
                HTTP path
              </label>
              <Input
                id="source-path"
                className="font-mono text-xs"
                placeholder="/events/storefront"
                {...form.register('path')}
              />
              <p className="mt-1.5 text-[11px] leading-4 text-muted-foreground">
                Leave blank to use the gateway default path.
              </p>
              {form.formState.errors.path && (
                <p className="field-error">{form.formState.errors.path.message}</p>
              )}
            </div>
          </div>

          <div>
            <span className="label" id="source-methods-label">
              Accepted methods
            </span>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  type="button"
                  variant="outline"
                  className="h-auto min-h-9 w-full justify-between px-3 py-1.5 font-normal"
                  aria-labelledby="source-methods-label source-methods-value"
                >
                  <span id="source-methods-value" className="flex min-w-0 flex-wrap gap-1.5">
                    {selectedMethods.length ? (
                      selectedMethods.map((method) => (
                        <Badge
                          key={method}
                          variant="secondary"
                          className="rounded font-mono text-[10px]"
                        >
                          {method}
                        </Badge>
                      ))
                    ) : (
                      <span className="text-sm text-muted-foreground">Select HTTP methods</span>
                    )}
                  </span>
                  <ChevronDown className="ml-3 size-4 text-muted-foreground" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start" className="min-w-56">
                <DropdownMenuLabel>HTTP methods</DropdownMenuLabel>
                {methods.map((method) => (
                  <DropdownMenuCheckboxItem
                    key={method}
                    checked={selectedMethods.includes(method)}
                    onCheckedChange={() => toggleMethod(method)}
                    onSelect={(event) => event.preventDefault()}
                    className="font-mono"
                  >
                    {method}
                  </DropdownMenuCheckboxItem>
                ))}
              </DropdownMenuContent>
            </DropdownMenu>
            <p className="mt-1.5 text-[11px] leading-4 text-muted-foreground">
              Choose one or more methods. POST is selected for new sources.
            </p>
            {form.formState.errors.methods && (
              <p className="field-error">{form.formState.errors.methods.message}</p>
            )}
          </div>
        </div>
      </FormSection>

      <FormSection
        icon={Settings2}
        title="Processing"
        description="Control availability and how requests move through the pipeline."
      >
        <div className="grid gap-3 sm:grid-cols-2">
          <Toggle
            checked={form.watch('enabled')}
            onChange={(checked) => form.setValue('enabled', checked, { shouldDirty: true })}
            label="Source enabled"
            description="Accept incoming requests immediately."
          />
          <Toggle
            checked={form.watch('asynchronous')}
            onChange={(checked) => form.setValue('asynchronous', checked, { shouldDirty: true })}
            label="Async ingestion"
            description="Queue events before processing them."
          />
        </div>
      </FormSection>

      <FormSection
        icon={Braces}
        title="HTTP response"
        description="Optionally return a fixed response to every accepted request."
      >
        <div className="overflow-hidden rounded-lg border border-border bg-muted/20">
          <div className="p-3">
            <Toggle
              checked={responseEnabled}
              onChange={(checked) =>
                form.setValue('responseEnabled', checked, { shouldDirty: true })
              }
              label="Custom response"
              description="Override the gateway's default acknowledgement."
            />
          </div>
          {responseEnabled && (
            <div className="grid gap-4 border-t border-border bg-background p-4 sm:grid-cols-[120px_1fr]">
              <div>
                <label className="label" htmlFor="response-code">
                  Status code
                </label>
                <Input
                  id="response-code"
                  type="number"
                  min={200}
                  max={599}
                  {...form.register('responseCode', { valueAsNumber: true })}
                />
                {form.formState.errors.responseCode && (
                  <p className="field-error">{form.formState.errors.responseCode.message}</p>
                )}
              </div>
              <div>
                <label className="label" htmlFor="response-content-type">
                  Content type
                </label>
                <Input
                  id="response-content-type"
                  className="font-mono text-xs"
                  placeholder="application/json"
                  {...form.register('responseContentType')}
                />
                {form.formState.errors.responseContentType && (
                  <p className="field-error">{form.formState.errors.responseContentType.message}</p>
                )}
              </div>
              <div className="sm:col-span-2">
                <label className="label" htmlFor="response-body">
                  Response body
                </label>
                <Textarea
                  id="response-body"
                  rows={4}
                  className="min-h-28 resize-y font-mono text-xs leading-5"
                  spellCheck={false}
                  {...form.register('responseBody')}
                />
                {form.formState.errors.responseBody && (
                  <p className="field-error">{form.formState.errors.responseBody.message}</p>
                )}
              </div>
            </div>
          )}
        </div>
      </FormSection>

      <section className="border-b border-border">
        <button
          type="button"
          className="flex w-full items-center gap-3 px-5 py-5 text-left transition-colors hover:bg-muted/35 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary sm:px-6"
          aria-expanded={advancedOpen}
          aria-controls="source-advanced-configuration"
          onClick={() => setAdvancedOpen((open) => !open)}
        >
          <span className="grid size-9 shrink-0 place-items-center rounded-lg border border-border bg-muted text-muted-foreground">
            <SlidersHorizontal className="size-4" />
          </span>
          <span className="min-w-0 flex-1">
            <span className="block text-sm font-semibold">Advanced Configuration</span>
            <span className="mt-1 block text-xs leading-5 text-muted-foreground">
              Configure rate limits and metadata only when you need them.
            </span>
          </span>
          <span className="flex shrink-0 items-center gap-2">
            {rateLimitEnabled && (
              <Badge variant="secondary" className="hidden text-[10px] sm:inline-flex">
                Rate limit on
              </Badge>
            )}
            {metadata.length > 0 && (
              <Badge variant="outline" className="hidden text-[10px] sm:inline-flex">
                {metadata.length} metadata
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
          <div id="source-advanced-configuration" className="border-t border-border bg-muted/10">
            <FormSection
              icon={Gauge}
              title="Traffic controls"
              description="Protect the source with a request quota over a rolling period."
            >
              <div className="overflow-hidden rounded-lg border border-border bg-muted/20">
                <div className="p-3">
                  <Toggle
                    checked={rateLimitEnabled}
                    onChange={(checked) =>
                      form.setValue('rateLimitEnabled', checked, { shouldDirty: true })
                    }
                    label="Rate limiting"
                    description="Reject requests after the configured quota is reached."
                  />
                </div>
                {rateLimitEnabled && (
                  <div className="grid gap-4 border-t border-border bg-background p-4 sm:grid-cols-2">
                    <div>
                      <label className="label" htmlFor="rate-quota">
                        Request quota
                      </label>
                      <Input
                        id="rate-quota"
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
                      <label className="label" htmlFor="rate-period">
                        Period (seconds)
                      </label>
                      <Input
                        id="rate-period"
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
                idPrefix="source"
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

function SourceDrawer({
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
  form?: UseFormReturn<SourceForm>
  busy?: boolean
  children: ReactNode
}) {
  const name = form?.watch('name').trim()
  const path = form?.watch('path').trim()
  const selectedMethods = form?.watch('methods') ?? []
  const enabled = form?.watch('enabled')
  const asynchronous = form?.watch('asynchronous')

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
            <Inbox className="size-4" />
          </span>
          <div className="min-w-0 flex-1">
            <SheetTitle className="text-base font-semibold">{title}</SheetTitle>
            <SheetDescription className="mt-1 text-xs leading-5">{description}</SheetDescription>
          </div>
          <SheetClose asChild>
            <Button variant="ghost" size="icon" aria-label="Close source drawer" disabled={busy}>
              <X className="size-4" />
            </Button>
          </SheetClose>
        </SheetHeader>

        {form && (
          <div className="border-b border-border bg-muted/35 px-5 py-3 sm:px-6">
            <div className="flex flex-wrap items-center gap-2">
              <span className="max-w-52 truncate text-xs font-medium">
                {name || 'Unnamed source'}
              </span>
              <span className="text-muted-foreground/60">/</span>
              <code className="min-w-0 max-w-72 truncate rounded-md border border-border bg-background px-2 py-1 text-[11px] text-muted-foreground">
                {path || 'Default HTTP path'}
              </code>
              <div className="ml-auto flex flex-wrap items-center gap-1.5">
                {selectedMethods.slice(0, 2).map((method) => (
                  <Badge key={method} variant="outline" className="rounded font-mono text-[9px]">
                    {method}
                  </Badge>
                ))}
                {selectedMethods.length > 2 && (
                  <Badge variant="outline" className="rounded text-[9px]">
                    +{selectedMethods.length - 2}
                  </Badge>
                )}
                <Badge variant={enabled ? 'secondary' : 'outline'} className="text-[10px]">
                  {enabled ? 'Enabled' : 'Disabled'}
                </Badge>
                <Badge variant="outline" className="text-[10px]">
                  {asynchronous ? 'Async' : 'Sync'}
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

function SourceFormContent({
  form,
  onSubmit,
  onCancel,
  pending,
  error,
  submitLabel,
  pendingLabel,
}: {
  form: UseFormReturn<SourceForm>
  onSubmit: (values: SourceForm) => void
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
        <SourceFields form={form} />
        {Boolean(error) && (
          <div className="mx-5 my-4 flex gap-3 rounded-lg border border-red-500/20 bg-red-500/5 p-3 sm:mx-6">
            <CircleAlert className="mt-0.5 size-4 shrink-0 text-red-500" />
            <div>
              <p className="text-xs font-medium">Could not save source</p>
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

export function SourcesPage() {
  const workspaceName = useWorkspaceName()
  const navigate = useNavigate()
  const location = useLocation()
  const [searchParams, setSearchParams] = useSearchParams()
  const createRoute = useMatch('/workspaces/:workspaceName/sources/create')
  const editRoute = useMatch('/workspaces/:workspaceName/sources/:sourceId/edit')
  const createOpen = Boolean(createRoute)
  const editSourceId = editRoute?.params.sourceId
  const sourcesPath = workspacePath(workspaceName, 'sources')
  const queryClient = useQueryClient()
  const searchParamsKey = searchParams.toString()
  const listParams = useMemo(
    () =>
      listQueryParamsFromSearchParams(new URLSearchParams(searchParamsKey), sourceDefaultParams),
    [searchParamsKey],
  )
  const setListParams = useCallback(
    (next: SourceListParams | ((current: SourceListParams) => SourceListParams)) => {
      const resolved = typeof next === 'function' ? next(listParams) : next
      setSearchParams(listQueryString(resolved), { replace: true })
    },
    [listParams, setSearchParams],
  )
  const [deleting, setDeleting] = useState<Source | null>(null)
  const [enabledConfirmation, setEnabledConfirmation] = useState<{
    source: Source
    enabled: boolean
  } | null>(null)
  const workspaceQuery = useQuery({
    queryKey: ['workspaces'],
    queryFn: api.workspaces,
  })
  const workspace = workspaceQuery.data?.find((item) => item.name === workspaceName)
  const workspaceId = workspace?.id ?? (workspaceName === 'default' ? 'default' : undefined)
  const sourcesQueryKey = ['workspaces', workspaceId, 'sources'] as const
  const queryKey = [...sourcesQueryKey, listParams] as const
  const sourcesQuery = useQuery({
    queryKey,
    queryFn: () => api.sources(workspaceId!, listParams),
    enabled: Boolean(workspaceId),
    placeholderData: (previous, previousQuery) =>
      workspacePlaceholderData(workspaceId, previous, previousQuery),
  })
  const page = sourcesQuery.data
  const sources = page?.data ?? []
  const sourceDetailQueryKey = [
    'workspaces',
    workspaceId,
    'sources',
    'detail',
    editSourceId,
  ] as const
  const editSourceQuery = useQuery({
    queryKey: sourceDetailQueryKey,
    queryFn: () => api.source(workspaceId!, editSourceId!),
    enabled: Boolean(workspaceId && editSourceId),
    refetchOnMount: 'always',
    refetchOnWindowFocus: false,
  })

  const createForm = useForm<SourceForm>({
    resolver: zodResolver(sourceSchema),
    defaultValues: emptySource,
  })
  const editForm = useForm<SourceForm>({
    resolver: zodResolver(sourceSchema),
    defaultValues: emptySource,
  })

  const closeSourceForm = () => {
    const openedFromList = Boolean(
      (location.state as { sourceModalFromList?: boolean } | null)?.sourceModalFromList,
    )
    if (openedFromList) {
      void navigate(-1)
      return
    }
    void navigate({ pathname: sourcesPath, search: location.search }, { replace: true })
  }

  const createSource = useMutation({
    mutationFn: (values: SourceForm) => api.createSource(workspaceId!, formToInput(values)),
    onSuccess: (source) => {
      void queryClient.invalidateQueries({ queryKey: sourcesQueryKey })
      createForm.reset(emptySource)
      closeSourceForm()
      toast.success(`${source.name || 'Source'} created.`)
    },
  })

  const updateSource = useMutation({
    mutationFn: ({ sourceId, values }: { sourceId: string; values: SourceForm }) =>
      api.updateSource(workspaceId!, sourceId, formToInput(values)),
    onSuccess: (updated) => {
      queryClient.setQueryData(sourceDetailQueryKey, updated)
      void queryClient.invalidateQueries({ queryKey: sourcesQueryKey })
      closeSourceForm()
      toast.success(`${updated.name || 'Source'} updated.`)
    },
  })

  const toggleSourceEnabled = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      api.updateSourceEnabled(workspaceId!, id, enabled),
    onMutate: async ({ id, enabled }) => {
      await queryClient.cancelQueries({ queryKey })
      const previousPage = queryClient.getQueryData<SourcePage>(queryKey)
      queryClient.setQueryData<SourcePage>(queryKey, (current) =>
        current
          ? {
              ...current,
              data: current.data.map((source) =>
                source.id === id ? { ...source, enabled } : source,
              ),
            }
          : current,
      )
      return { previousPage }
    },
    onError: (error, _variables, context) => {
      if (context?.previousPage) queryClient.setQueryData(queryKey, context.previousPage)
      toast.error(`Could not update source: ${errorMessage(error)}`)
    },
    onSuccess: (updated) => {
      queryClient.setQueryData<SourcePage>(queryKey, (current) =>
        current
          ? {
              ...current,
              data: current.data.map((source) => (source.id === updated.id ? updated : source)),
            }
          : current,
      )
      queryClient.setQueryData(
        ['workspaces', workspaceId, 'sources', 'detail', updated.id],
        updated,
      )
      void queryClient.invalidateQueries({ queryKey: sourcesQueryKey })
      setEnabledConfirmation(null)
      toast.success(`${updated.name || 'Source'} ${updated.enabled ? 'enabled' : 'disabled'}.`)
    },
  })

  const deleteSource = useMutation({
    mutationFn: (source: Source) => api.deleteSource(workspaceId!, source.id),
    onSuccess: (_, source) => {
      void queryClient.invalidateQueries({ queryKey: sourcesQueryKey })
      setDeleting(null)
      toast.success(`${source.name || 'Source'} deleted.`)
    },
  })

  const loading = (!workspaceId && workspaceQuery.isLoading) || sourcesQuery.isLoading
  const resolutionError = !workspaceId && !workspaceQuery.isLoading
  const loadError = resolutionError
    ? (workspaceQuery.error ?? new Error(`Workspace “${workspaceName}” was not found.`))
    : sourcesQuery.error

  const editLoadError = resolutionError
    ? (workspaceQuery.error ?? new Error(`Workspace “${workspaceName}” was not found.`))
    : editSourceQuery.error
  const editDetailLoading =
    Boolean(editSourceId) &&
    (workspaceQuery.isLoading ||
      Boolean(workspaceId && (editSourceQuery.isPending || editSourceQuery.isFetching)))
  const editingSource = !editDetailLoading && !editLoadError ? editSourceQuery.data : undefined

  const openCreate = () => {
    createSource.reset()
    createForm.reset(emptySource)
    void navigate(
      { pathname: `${sourcesPath}/create`, search: location.search },
      { state: { sourceModalFromList: true } },
    )
  }

  const openEdit = (source: Source) => {
    updateSource.reset()
    void navigate(
      { pathname: `${sourcesPath}/${encodeURIComponent(source.id)}/edit` },
      { state: { sourceModalFromList: true } },
    )
  }

  useEffect(() => {
    if (!editSourceId) {
      editForm.reset(emptySource)
      return
    }
    if (editSourceQuery.isFetching || !editSourceQuery.data || editSourceQuery.isError) return
    editForm.reset(sourceToForm(editSourceQuery.data))
  }, [
    editForm,
    editSourceId,
    editSourceQuery.data,
    editSourceQuery.dataUpdatedAt,
    editSourceQuery.isError,
    editSourceQuery.isFetching,
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
        title="Sources"
        description={`Manage event ingress for the ${workspaceName} workspace.`}
        actions={
          <Button onClick={openCreate} disabled={!workspaceId}>
            <Plus className="size-4" />
            Create source
          </Button>
        }
      />

      <Card className="overflow-hidden">
        <QueryFilter
          value={listParams}
          onChange={setListParams}
          config={sourceFilterConfig}
          storageKey={`webhookx:source-views:${workspaceId ?? workspaceName}`}
          defaultParams={sourceDefaultParams}
          presetViews={sourcePresetViews}
          resultCount={sources.length}
          total={page?.total}
          busy={sourcesQuery.isFetching && !sourcesQuery.isLoading}
        />

        {loading && <LoadingRows />}

        {loadError && !loading && (
          <div className="m-5 rounded-lg border border-red-500/20 bg-red-500/5 p-4">
            <div className="flex gap-3">
              <CircleAlert className="mt-0.5 size-4 shrink-0 text-red-500" />
              <div>
                <p className="text-xs font-medium">Could not load sources</p>
                <p className="mt-1 text-xs text-muted-foreground">{errorMessage(loadError)}</p>
              </div>
            </div>
            {workspaceId && (
              <Button
                variant="outline"
                size="sm"
                className="mt-3"
                onClick={() => sourcesQuery.refetch()}
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
                  <TableHead>Path</TableHead>
                  <TableHead>Methods</TableHead>
                  <TableHead>Enabled</TableHead>
                  <TableHead>Ingestion</TableHead>
                  <TableHead>Rate limit</TableHead>
                  <TableHead>Updated</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {sources.map((source) => {
                  const http = source.config?.http ?? {}
                  return (
                    <TableRow
                      key={source.id}
                      tabIndex={0}
                      aria-label={`Edit ${source.name || 'source'}`}
                      className="cursor-pointer focus-visible:bg-muted/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary [&>td]:h-12 [&>td]:py-0"
                      onClick={() => openEdit(source)}
                      onKeyDown={(event) => {
                        if (event.target !== event.currentTarget) return
                        if (event.key !== 'Enter' && event.key !== ' ') return
                        event.preventDefault()
                        openEdit(source)
                      }}
                    >
                      <TableCell className="max-w-sm">
                        <div className="flex items-center gap-3">
                          <span className="grid size-8 shrink-0 place-items-center rounded-md border border-border bg-muted">
                            <Inbox className="size-4 text-muted-foreground" />
                          </span>
                          <div className="min-w-0">
                            <p className="truncate font-medium">
                              {source.name ? (
                                source.name
                              ) : (
                                <span className="text-muted-foreground">—</span>
                              )}
                            </p>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div>{http.path || 'Default HTTP path'}</div>
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-wrap gap-1">
                          {(http.methods?.length ? http.methods : ['POST']).map((method) => (
                            <Badge key={method} className="rounded font-mono text-[9px]">
                              {method}
                            </Badge>
                          ))}
                        </div>
                      </TableCell>
                      <TableCell>
                        <Switch
                          checked={source.enabled}
                          aria-label={`${source.enabled ? 'Disable' : 'Enable'} ${source.name || 'source'}`}
                          onClick={(event) => event.stopPropagation()}
                          onCheckedChange={(enabled) => {
                            toggleSourceEnabled.reset()
                            setEnabledConfirmation({
                              source,
                              enabled,
                            })
                          }}
                          disabled={toggleSourceEnabled.isPending}
                        />
                      </TableCell>
                      <TableCell>
                        <span className="text-xs text-muted-foreground">
                          {source.async ? 'Asynchronous' : 'Synchronous'}
                        </span>
                      </TableCell>
                      <TableCell>
                        {source.rate_limit ? (
                          <span className="font-mono text-[11px]">
                            {source.rate_limit.quota} / {source.rate_limit.period}s
                          </span>
                        ) : (
                          <span className="text-muted-foreground">—</span>
                        )}
                      </TableCell>
                      <TableCell className="whitespace-nowrap text-muted-foreground">
                        <Timestamp value={source.updated_at} />
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
                                aria-label={`Actions for ${source.name || 'source'}`}
                              >
                                <MoreHorizontal className="size-4" />
                              </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end" className="w-36">
                              <DropdownMenuItem
                                variant="destructive"
                                onSelect={() => {
                                  deleteSource.reset()
                                  setDeleting(source)
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
                {!sources.length && (
                  <TableRow>
                    <TableCell colSpan={8} className="py-14 text-center text-muted-foreground">
                      No sources match the current query.
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
                aria-label="Sources per page"
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
                disabled={!page?.prev || sourcesQuery.isFetching}
              >
                <ChevronLeft className="size-3.5" />
                Previous
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => moveToCursor(page?.next)}
                disabled={!page?.next || sourcesQuery.isFetching}
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
          if (!toggleSourceEnabled.isPending) setEnabledConfirmation(null)
        }}
        title={`${enabledConfirmation?.enabled ? 'Enable' : 'Disable'} source`}
        description="This change takes effect immediately."
      >
        {enabledConfirmation && (
          <div className="p-5">
            <div className="rounded-lg border border-border bg-muted/40 p-4">
              <p className="text-sm font-medium">
                {enabledConfirmation.enabled ? 'Enable' : 'Disable'}{' '}
                {enabledConfirmation.source.name || 'this source'}?
              </p>
              <p className="mt-1 text-xs leading-5 text-muted-foreground">
                Incoming requests to{' '}
                <span className="font-mono">
                  {enabledConfirmation.source.config?.http?.path || 'its HTTP path'}
                </span>{' '}
                will {enabledConfirmation.enabled ? 'be accepted' : 'no longer be accepted'} by this
                source.
              </p>
            </div>
            {toggleSourceEnabled.isError && (
              <p className="field-error mt-3">{errorMessage(toggleSourceEnabled.error)}</p>
            )}
            <div className="mt-5 flex justify-end gap-2">
              <Button
                type="button"
                variant="outline"
                onClick={() => setEnabledConfirmation(null)}
                disabled={toggleSourceEnabled.isPending}
              >
                Cancel
              </Button>
              <Button
                type="button"
                variant={enabledConfirmation.enabled ? 'default' : 'destructive'}
                onClick={() =>
                  toggleSourceEnabled.mutate({
                    id: enabledConfirmation.source.id,
                    enabled: enabledConfirmation.enabled,
                  })
                }
                disabled={toggleSourceEnabled.isPending}
              >
                {toggleSourceEnabled.isPending
                  ? 'Updating…'
                  : `${enabledConfirmation.enabled ? 'Enable' : 'Disable'} source`}
              </Button>
            </div>
          </div>
        )}
      </AppDialog>

      <SourceDrawer
        open={createOpen}
        onClose={closeSourceForm}
        title="Create source"
        description="Configure an HTTP entry point without leaving the sources list."
        form={workspaceId ? createForm : undefined}
        busy={createSource.isPending}
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
              <Button variant="outline" onClick={closeSourceForm}>
                Back to sources
              </Button>
            </div>
          </div>
        )}
        {workspaceId && (
          <SourceFormContent
            form={createForm}
            onSubmit={(values) => createSource.mutate(values)}
            onCancel={closeSourceForm}
            pending={createSource.isPending}
            error={createSource.error}
            submitLabel="Create source"
            pendingLabel="Creating…"
          />
        )}
      </SourceDrawer>

      <SourceDrawer
        open={Boolean(editSourceId)}
        onClose={closeSourceForm}
        title="Edit source"
        description="Update how this source accepts, processes, and responds to events."
        form={editingSource ? editForm : undefined}
        busy={updateSource.isPending}
      >
        {editDetailLoading && (
          <div className="flex-1 space-y-4 p-5 sm:p-6" aria-label="Loading source">
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
                  <p className="text-sm font-medium">Could not load source</p>
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
                  if (workspaceId) void editSourceQuery.refetch()
                  else void workspaceQuery.refetch()
                }}
              >
                Try again
              </Button>
            </div>
            <div className="mt-5 flex justify-end">
              <Button variant="outline" onClick={closeSourceForm}>
                Back to sources
              </Button>
            </div>
          </div>
        )}
        {!editDetailLoading && editingSource && (
          <SourceFormContent
            form={editForm}
            onSubmit={(values) => updateSource.mutate({ sourceId: editingSource.id, values })}
            onCancel={closeSourceForm}
            pending={updateSource.isPending}
            error={updateSource.error}
            submitLabel="Save changes"
            pendingLabel="Saving…"
          />
        )}
      </SourceDrawer>

      <AppDialog
        open={Boolean(deleting)}
        onClose={() => setDeleting(null)}
        title="Delete source"
        description="This operation cannot be undone."
      >
        {deleting && (
          <div className="p-5">
            <div className="rounded-lg border border-red-500/20 bg-red-500/5 p-4">
              <p className="text-sm font-medium">Delete {deleting.name || 'this source'}?</p>
              <p className="mt-1 text-xs leading-5 text-muted-foreground">
                Requests sent to{' '}
                <span className="font-mono">{deleting.config?.http?.path || 'its HTTP path'}</span>{' '}
                will no longer be accepted by this source.
              </p>
            </div>
            {deleteSource.isError && (
              <p className="field-error mt-3">{errorMessage(deleteSource.error)}</p>
            )}
            <div className="mt-5 flex justify-end gap-2">
              <Button variant="outline" onClick={() => setDeleting(null)}>
                Cancel
              </Button>
              <Button
                variant="destructive"
                onClick={() => deleteSource.mutate(deleting)}
                disabled={deleteSource.isPending}
              >
                {deleteSource.isPending ? 'Deleting…' : 'Delete source'}
              </Button>
            </div>
          </div>
        )}
      </AppDialog>
    </>
  )
}
