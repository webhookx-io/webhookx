import { toast } from 'sonner'
import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AlertCircle, Box, Plus, RefreshCw, Search, SlidersHorizontal } from 'lucide-react'
import { useWorkspaceName } from '@/app/workspace'
import { JsonSchemaForm } from '@/components/plugins/json-schema-form'
import {
  MetadataEditor,
  metadataEntriesToRecord,
  type MetadataEntry,
} from '@/components/shared/metadata-editor'
import { PageHeader } from '@/components/shared/page-header'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { AppDialog } from '@/components/shared/app-dialog'
import { Input } from '@/components/ui/input'
import { NativeSelect } from '@/components/ui/native-select'
import { Switch } from '@/components/ui/switch'
import { api } from '@/data/api'
import { openApiSchemaDefinitions } from '@/data/openapi'
import { createListQueryParams } from '@/data/list-query'
import type { PluginCatalogItem, PluginInput } from '@/types'
import { errorMessage } from '@/lib/utils'
import {
  createSchemaDefaults,
  validateSchemaValue,
  type SchemaDefinitions,
} from '@/lib/json-schema'

type TargetKind = 'source' | 'endpoint'

const resourceListParams = createListQueryParams({ limit: 1000, sort: 'id.asc' })

function pluginInitials(name: string) {
  return name
    .split(/[-_\s]+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase())
    .join('')
}

function schemaSummary(plugin: PluginCatalogItem) {
  const variants = plugin.schema.oneOf?.length ?? plugin.schema.anyOf?.length ?? 0
  if (variants) return `${variants} configuration variants`
  const fields = Object.keys(plugin.schema.properties ?? {}).length
  if (!fields) return 'No required configuration'
  return `${fields} configuration ${fields === 1 ? 'field' : 'fields'}`
}

export function PluginsPage() {
  const workspaceName = useWorkspaceName()
  const queryClient = useQueryClient()
  const [search, setSearch] = useState('')
  const [selectedPlugin, setSelectedPlugin] = useState<PluginCatalogItem | null>(null)
  const [targetKind, setTargetKind] = useState<TargetKind>('source')
  const [targetId, setTargetId] = useState('')
  const [enabled, setEnabled] = useState(true)
  const [config, setConfig] = useState<Record<string, unknown>>({})
  const [metadata, setMetadata] = useState<MetadataEntry[]>([])
  const [validationErrors, setValidationErrors] = useState<string[]>([])

  const catalogQuery = useQuery({
    queryKey: ['plugin-catalog'],
    queryFn: api.pluginCatalog,
    staleTime: 5 * 60_000,
  })
  const workspaceQuery = useQuery({
    queryKey: ['workspaces'],
    queryFn: api.workspaces,
  })
  const workspace = workspaceQuery.data?.find((item) => item.name === workspaceName)
  const workspaceId = workspace?.id ?? (workspaceName === 'default' ? 'default' : undefined)

  const sourcesQuery = useQuery({
    queryKey: ['workspaces', workspaceId, 'sources', 'plugin-targets'],
    queryFn: () => api.sources(workspaceId!, resourceListParams),
    enabled: Boolean(selectedPlugin && workspaceId && targetKind === 'source'),
    staleTime: 30_000,
  })
  const endpointsQuery = useQuery({
    queryKey: ['workspaces', workspaceId, 'endpoints', 'plugin-targets'],
    queryFn: () => api.endpoints(workspaceId!, resourceListParams),
    enabled: Boolean(selectedPlugin && workspaceId && targetKind === 'endpoint'),
    staleTime: 30_000,
  })

  const catalog = useMemo(() => catalogQuery.data ?? [], [catalogQuery.data])
  const filtered = useMemo(() => {
    const needle = search.trim().toLowerCase()
    if (!needle) return catalog
    return catalog.filter((plugin) =>
      [plugin.name, plugin.description].some((value) => value.toLowerCase().includes(needle)),
    )
  }, [catalog, search])
  const definitions: SchemaDefinitions = openApiSchemaDefinitions

  const closeCreate = () => {
    if (createPlugin.isPending) return
    setSelectedPlugin(null)
    setValidationErrors([])
  }

  const createPlugin = useMutation({
    mutationFn: (input: PluginInput) => api.createPlugin(workspaceId!, input),
    onSuccess: (plugin) => {
      void queryClient.invalidateQueries({ queryKey: ['workspaces', workspaceId, 'plugins'] })
      setSelectedPlugin(null)
      setValidationErrors([])
      toast.success(`${plugin.name} plugin created.`)
    },
  })

  const openCreate = (plugin: PluginCatalogItem) => {
    createPlugin.reset()
    setSelectedPlugin(plugin)
    setTargetKind(plugin.type === 'inbound' ? 'source' : 'endpoint')
    setTargetId('')
    setEnabled(true)
    const defaults = createSchemaDefaults(plugin.schema, definitions)
    setConfig(
      defaults && typeof defaults === 'object' && !Array.isArray(defaults)
        ? (defaults as Record<string, unknown>)
        : {},
    )
    setMetadata([])
    setValidationErrors([])
  }

  const submitPlugin = () => {
    if (!selectedPlugin || !workspaceId) return
    const errors: string[] = []
    if (!targetId) errors.push('Target: select the resource that will use this plugin.')
    errors.push(...validateSchemaValue(selectedPlugin.schema, config, definitions))
    if (errors.length) {
      setValidationErrors(errors)
      return
    }

    setValidationErrors([])
    createPlugin.mutate({
      name: selectedPlugin.name,
      enabled,
      source_id: targetKind === 'source' ? targetId : null,
      endpoint_id: targetKind === 'endpoint' ? targetId : null,
      config,
      metadata: metadataEntriesToRecord(metadata),
    })
  }

  const targetQuery = targetKind === 'source' ? sourcesQuery : endpointsQuery
  const targetResources =
    targetKind === 'source' ? (sourcesQuery.data?.data ?? []) : (endpointsQuery.data?.data ?? [])

  return (
    <>
      <PageHeader
        title="Plugins"
        description={`Create plugin instances for the ${workspaceName} workspace.`}
        actions={
          <span className="rounded-md border border-border bg-card px-3 py-2 text-xs text-muted-foreground">
            {catalog.length} available
          </span>
        }
      />

      <div className="mb-5 flex flex-col gap-3 sm:flex-row sm:items-center">
        <div className="relative flex-1 sm:max-w-sm">
          <Search className="absolute left-3 top-2.5 size-4 text-muted-foreground" />
          <Input
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="Search plugin catalog…"
            className="pl-9"
          />
        </div>
        <p className="text-xs text-muted-foreground sm:ml-auto">
          Loaded from the gateway plugin catalog
        </p>
      </div>

      {catalogQuery.isPending && (
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3" aria-label="Loading plugins">
          {Array.from({ length: 6 }).map((_, index) => (
            <div
              key={index}
              className="h-52 animate-pulse rounded-xl border border-border bg-muted/40"
            />
          ))}
        </div>
      )}

      {catalogQuery.isError && (
        <Card className="flex min-h-64 flex-col items-center justify-center px-6 py-12 text-center">
          <div className="grid size-11 place-items-center rounded-full bg-red-500/10 text-red-500">
            <AlertCircle className="size-5" />
          </div>
          <h2 className="mt-4 text-sm font-semibold">Could not load the plugin catalog</h2>
          <p className="mt-1 max-w-lg text-xs leading-5 text-muted-foreground">
            {errorMessage(catalogQuery.error)}
          </p>
          <Button
            className="mt-5"
            variant="outline"
            size="sm"
            onClick={() => catalogQuery.refetch()}
            disabled={catalogQuery.isFetching}
          >
            <RefreshCw className={`size-3.5 ${catalogQuery.isFetching ? 'animate-spin' : ''}`} />
            Retry
          </Button>
        </Card>
      )}

      {catalogQuery.data && (
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {filtered.map((plugin) => (
            <Card key={plugin.name} className="group relative overflow-hidden">
              <CardContent className="flex h-full min-h-52 flex-col p-5">
                <div className="flex items-start gap-3">
                  <div className="grid size-10 shrink-0 place-items-center rounded-lg border border-violet-500/20 bg-violet-500/10 font-mono text-xs font-semibold text-violet-600 dark:text-violet-300">
                    {pluginInitials(plugin.name)}
                  </div>
                  <div className="min-w-0 flex-1">
                    <h2 className="truncate font-mono text-sm font-semibold">{plugin.name}</h2>
                    <p className="mt-1 flex items-center gap-1.5 text-[10px] text-muted-foreground">
                      <SlidersHorizontal className="size-3" />
                      {schemaSummary(plugin)}
                    </p>
                  </div>
                </div>
                <p className="mt-4 text-xs leading-5 text-muted-foreground">
                  {plugin.description || 'No description is available for this plugin.'}
                </p>
                <div className="mt-auto flex items-center justify-between border-t border-border pt-4">
                  <span className="inline-flex items-center gap-1.5 text-[10px] text-muted-foreground">
                    <Box className="size-3" />
                    <span className="capitalize">{plugin.type} plugin</span>
                  </span>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => openCreate(plugin)}
                    disabled={!workspaceId}
                  >
                    <Plus className="size-3.5" />
                    Create instance
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {catalogQuery.data && !filtered.length && (
        <div className="rounded-xl border border-dashed border-border py-16 text-center text-sm text-muted-foreground">
          No plugins match your search.
        </div>
      )}

      <AppDialog
        open={Boolean(selectedPlugin)}
        onClose={closeCreate}
        title={selectedPlugin ? `Create ${selectedPlugin.name}` : 'Create plugin'}
        description="Choose where the plugin runs, then configure it from its schema."
      >
        {selectedPlugin && (
          <form
            className="space-y-5 p-5"
            onSubmit={(event) => {
              event.preventDefault()
              submitPlugin()
            }}
          >
            {!workspaceId && workspaceQuery.isLoading && (
              <div className="space-y-3" aria-label="Loading workspace">
                <div className="h-10 animate-pulse rounded-md bg-muted" />
                <div className="h-10 animate-pulse rounded-md bg-muted" />
              </div>
            )}

            {!workspaceId && !workspaceQuery.isLoading && (
              <div className="rounded-lg border border-red-500/20 bg-red-500/5 p-4">
                <p className="text-xs font-medium">Could not resolve workspace</p>
                <p className="mt-1 text-xs text-muted-foreground">
                  {errorMessage(
                    workspaceQuery.error ??
                      new Error(`Workspace “${workspaceName}” was not found.`),
                  )}
                </p>
              </div>
            )}

            {workspaceId && (
              <>
                <fieldset className="rounded-lg border border-border">
                  <legend className="sr-only">Plugin target</legend>
                  <div className="border-b border-border px-4 py-3">
                    <p className="text-xs font-medium">Plugin target</p>
                    <p className="mt-1 text-[11px] leading-4 text-muted-foreground">
                      Inbound plugins attach to a source; outbound plugins attach to an endpoint.
                    </p>
                  </div>
                  <div className="grid gap-4 p-4 sm:grid-cols-2">
                    <div>
                      <label className="label" htmlFor="plugin-target-kind">
                        Target type <span className="text-red-500">*</span>
                      </label>
                      <NativeSelect
                        id="plugin-target-kind"
                        className="w-full"
                        value={targetKind}
                        disabled
                      >
                        {targetKind === 'source' ? (
                          <option value="source">Source (inbound)</option>
                        ) : (
                          <option value="endpoint">Endpoint (outbound)</option>
                        )}
                      </NativeSelect>
                    </div>
                    <div>
                      <label className="label" htmlFor="plugin-target-id">
                        {targetKind === 'source' ? 'Source' : 'Endpoint'}{' '}
                        <span className="text-red-500">*</span>
                      </label>
                      <NativeSelect
                        id="plugin-target-id"
                        className="w-full"
                        value={targetId}
                        disabled={targetQuery.isPending}
                        onChange={(event) => setTargetId(event.target.value)}
                      >
                        <option value="">
                          {targetQuery.isPending ? 'Loading resources…' : `Select ${targetKind}…`}
                        </option>
                        {targetResources.map((resource) => (
                          <option key={resource.id} value={resource.id}>
                            {resource.name || resource.id}
                          </option>
                        ))}
                      </NativeSelect>
                    </div>
                  </div>
                  {targetQuery.isError && (
                    <div className="border-t border-border px-4 py-3">
                      <p className="field-error m-0">
                        Could not load {targetKind}s: {errorMessage(targetQuery.error)}
                      </p>
                    </div>
                  )}
                </fieldset>

                <div className="flex items-center justify-between gap-4 rounded-lg border border-border p-3">
                  <div>
                    <p className="text-xs font-medium">Plugin enabled</p>
                    <p className="mt-0.5 text-[11px] text-muted-foreground">
                      Start applying this configuration immediately after creation.
                    </p>
                  </div>
                  <Switch
                    checked={enabled}
                    onCheckedChange={setEnabled}
                    aria-label="Plugin enabled"
                  />
                </div>

                <fieldset className="space-y-4 rounded-lg border border-border p-4">
                  <legend className="px-1 text-xs font-medium">Plugin configuration</legend>
                  <JsonSchemaForm
                    schema={selectedPlugin.schema}
                    definitions={definitions}
                    value={config}
                    onChange={(next) => {
                      setConfig(next)
                      setValidationErrors([])
                    }}
                  />
                </fieldset>

                <MetadataEditor
                  idPrefix="plugin"
                  value={metadata}
                  onChange={setMetadata}
                  description="Add optional labels to this plugin instance."
                />

                {validationErrors.length > 0 && (
                  <div className="rounded-lg border border-red-500/20 bg-red-500/5 p-3">
                    <p className="text-xs font-medium text-red-500">Check the plugin form</p>
                    <ul className="mt-2 list-disc space-y-1 pl-4 text-[11px] text-red-500">
                      {validationErrors.map((error) => (
                        <li key={error}>{error}</li>
                      ))}
                    </ul>
                  </div>
                )}

                {createPlugin.isError && (
                  <p className="field-error">{errorMessage(createPlugin.error)}</p>
                )}

                <div className="flex justify-end gap-2 border-t border-border pt-4">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={closeCreate}
                    disabled={createPlugin.isPending}
                  >
                    Cancel
                  </Button>
                  <Button type="submit" disabled={createPlugin.isPending || !targetId}>
                    {createPlugin.isPending ? 'Creating…' : 'Create plugin'}
                  </Button>
                </div>
              </>
            )}
          </form>
        )}
      </AppDialog>
    </>
  )
}
