import { toast } from 'sonner'
import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Bookmark,
  CalendarDays,
  Check,
  Filter,
  LoaderCircle,
  Pencil,
  Plus,
  Search,
  Trash2,
  X,
} from 'lucide-react'
import {
  cloneListQueryParams,
  createListQueryParams,
  listQueryFingerprint,
  mergeListQueryParams,
} from '@/data/list-query'
import type {
  CreatedAtFilter,
  CreatedAtOperator,
  ListQueryParams,
  ListSort,
  QueryView,
} from '@/types'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { NativeSelect } from '@/components/ui/native-select'

type StringFilterDefinition = {
  key: 'name' | 'event_type' | 'unique_id' | 'event_id' | 'endpoint_id'
  type: 'string'
  label: string
  placeholder?: string
  quickSearch?: boolean
}

type EnumFilterDefinition = {
  key: 'status'
  type: 'enum'
  label: string
  options: Array<{ value: string; label: string }>
}

type BooleanFilterDefinition = {
  key: 'enabled'
  type: 'boolean'
  label: string
  trueLabel?: string
  falseLabel?: string
}

type CreatedAtFilterDefinition = {
  key: 'created_at'
  type: 'created_at'
  label: string
}

type IngestedAtFilterDefinition = {
  key: 'ingested_at'
  type: 'ingested_at'
  label: string
}

type AttemptedAtFilterDefinition = {
  key: 'attempted_at'
  type: 'attempted_at'
  label: string
}

type TimestampFilterDefinition =
  CreatedAtFilterDefinition | IngestedAtFilterDefinition | AttemptedAtFilterDefinition

type MetadataFilterDefinition = {
  key: 'metadata'
  type: 'metadata'
  label: string
  keyPlaceholder?: string
  valuePlaceholder?: string
}

export type QueryFilterDefinition =
  | StringFilterDefinition
  | EnumFilterDefinition
  | BooleanFilterDefinition
  | CreatedAtFilterDefinition
  | IngestedAtFilterDefinition
  | AttemptedAtFilterDefinition
  | MetadataFilterDefinition

export interface QueryFilterConfig {
  fields: QueryFilterDefinition[]
  showSort?: boolean
}

export interface QueryPresetView {
  id: string
  name: string
  params: Partial<ListQueryParams>
}

interface QueryFilterProps {
  value: ListQueryParams
  onChange: (params: ListQueryParams) => void
  config: QueryFilterConfig
  storageKey: string
  defaultParams?: Partial<ListQueryParams>
  presetViews?: QueryPresetView[]
  allViewLabel?: string
  resultCount: number
  total?: number
  busy?: boolean
  disabled?: boolean
}

const createdAtLabels: Record<CreatedAtOperator, string> = {
  eq: 'is exactly',
  gt: 'is after',
  gte: 'is on or after',
  lt: 'is before',
  lte: 'is on or before',
}

const createdAtOperators = Object.keys(createdAtLabels) as CreatedAtOperator[]

function readViews(storageKey: string): QueryView[] {
  try {
    const raw = window.localStorage.getItem(storageKey)
    if (!raw) return []
    const parsed = JSON.parse(raw) as unknown
    if (!Array.isArray(parsed)) return []
    return parsed.flatMap((candidate) => {
      if (!candidate || typeof candidate !== 'object') return []
      const view = candidate as Partial<QueryView>
      if (typeof view.id !== 'string' || typeof view.name !== 'string' || !view.name.trim())
        return []
      const createdAt = typeof view.createdAt === 'number' ? view.createdAt : Date.now()
      const updatedAt = typeof view.updatedAt === 'number' ? view.updatedAt : createdAt
      return [
        {
          id: view.id,
          name: view.name.trim(),
          params: createListQueryParams(view.params),
          createdAt,
          updatedAt,
        },
      ]
    })
  } catch {
    return []
  }
}

function writeViews(storageKey: string, views: QueryView[]) {
  try {
    window.localStorage.setItem(storageKey, JSON.stringify(views))
    return true
  } catch {
    return false
  }
}

function formatCreatedAt(value: number) {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value))
}

function localDateTimeValue(timestamp = Date.now()) {
  const date = new Date(timestamp - new Date(timestamp).getTimezoneOffset() * 60_000)
  return date.toISOString().slice(0, 16)
}

function viewButtonClass(active: boolean) {
  return active
    ? 'border-2 border-foreground text-foreground'
    : 'border-transparent text-muted-foreground hover:border-border hover:text-foreground'
}

function isTimestampFilter(field: QueryFilterDefinition): field is TimestampFilterDefinition {
  return (
    field.type === 'created_at' || field.type === 'ingested_at' || field.type === 'attempted_at'
  )
}

function timestampFilterEntries(filter: CreatedAtFilter | undefined) {
  if (typeof filter === 'number') return [['eq', filter] as const]
  if (!filter) return []
  return createdAtOperators.flatMap((operator) => {
    const timestamp = filter[operator]
    return typeof timestamp === 'number' ? [[operator, timestamp] as const] : []
  })
}

export function QueryFilter({
  value,
  onChange,
  config,
  storageKey,
  defaultParams,
  presetViews = [],
  allViewLabel = 'All',
  resultCount,
  total,
  busy,
  disabled,
}: QueryFilterProps) {
  const defaults = useMemo(() => mergeListQueryParams({}, defaultParams), [defaultParams])
  const quickSearch = config.fields.find(
    (field): field is StringFilterDefinition =>
      field.type === 'string' && Boolean(field.quickSearch),
  )
  const filterFields = config.fields.filter(
    (field) => !('quickSearch' in field && field.quickSearch),
  )
  const quickSearchValue = quickSearch ? (value[quickSearch.key] ?? '') : ''
  const [searchInput, setSearchInput] = useState(quickSearchValue)
  const [filterOpen, setFilterOpen] = useState(false)
  const [filterFieldKey, setFilterFieldKey] = useState<QueryFilterDefinition['key']>(
    filterFields[0]?.key ?? 'name',
  )
  const [stringValue, setStringValue] = useState('')
  const [enumValue, setEnumValue] = useState('')
  const [booleanValue, setBooleanValue] = useState('true')
  const [createdAtOperator, setCreatedAtOperator] = useState<CreatedAtOperator>('gte')
  const [createdAtValue, setCreatedAtValue] = useState(localDateTimeValue())
  const [metadataKey, setMetadataKey] = useState('')
  const [metadataValue, setMetadataValue] = useState('')
  const [filterError, setFilterError] = useState('')
  const [views, setViews] = useState<QueryView[]>(() => readViews(storageKey))
  const [selectedViewId, setSelectedViewId] = useState<string | null>(null)
  const [creatingView, setCreatingView] = useState(false)
  const [renamingViewId, setRenamingViewId] = useState<string | null>(null)
  const [viewName, setViewName] = useState('')
  const [viewError, setViewError] = useState('')

  useEffect(() => {
    setViews(readViews(storageKey))
    setSelectedViewId(null)
    setCreatingView(false)
    setRenamingViewId(null)
  }, [storageKey])

  useEffect(() => {
    if (quickSearch) setSearchInput(quickSearchValue)
  }, [quickSearch, quickSearchValue])

  useEffect(() => {
    if (!filterFields.some((field) => field.key === filterFieldKey) && filterFields[0])
      setFilterFieldKey(filterFields[0].key)
  }, [filterFieldKey, filterFields])

  const updateFilters = useCallback(
    (patch: Partial<ListQueryParams>) => {
      setSelectedViewId(null)
      onChange(
        createListQueryParams({
          ...value,
          ...patch,
          after: undefined,
          before: undefined,
        }),
      )
    },
    [onChange, value],
  )

  useEffect(() => {
    if (!quickSearch) return
    const trimmed = searchInput.trim()
    if (trimmed === quickSearchValue) return
    const timeout = window.setTimeout(
      () =>
        updateFilters({
          [quickSearch.key]: trimmed || undefined,
        }),
      350,
    )
    return () => window.clearTimeout(timeout)
  }, [quickSearch, quickSearchValue, searchInput, updateFilters])

  const selectedField = config.fields.find((field) => field.key === filterFieldKey)
  const timestampFields = config.fields.filter(isTimestampFilter)
  const timestampEntries = timestampFields.flatMap((field) =>
    timestampFilterEntries(value[field.key]).map(([operator, timestamp]) => ({
      field,
      operator,
      timestamp,
    })),
  )

  const enabledField = config.fields.find(
    (field): field is BooleanFilterDefinition => field.type === 'boolean',
  )
  const metadataField = config.fields.find(
    (field): field is MetadataFilterDefinition => field.type === 'metadata',
  )
  const menuStringEntries = config.fields.flatMap((field) => {
    if (field.type !== 'string' || field.quickSearch) return []
    const entry = value[field.key]
    return entry ? [{ field, value: entry }] : []
  })
  const enumEntries = config.fields.flatMap((field) => {
    if (field.type !== 'enum') return []
    const entry = value[field.key]
    return entry ? [{ field, value: entry }] : []
  })
  const metadataEntries = metadataField ? Object.entries(value.metadata) : []
  const advancedFilterCount =
    (enabledField && typeof value.enabled === 'boolean' ? 1 : 0) +
    timestampEntries.length +
    metadataEntries.length +
    menuStringEntries.length +
    enumEntries.length

  const valueFingerprint = listQueryFingerprint(value)
  const defaultsFingerprint = listQueryFingerprint(defaults)
  const resolvedPresets = useMemo(
    () =>
      presetViews.map((view) => ({
        ...view,
        params: mergeListQueryParams(defaults, view.params),
      })),
    [defaults, presetViews],
  )
  const activePreset = resolvedPresets.find(
    (view) => listQueryFingerprint(view.params) === valueFingerprint,
  )
  const matchingLocalView = views.find(
    (view) => listQueryFingerprint(view.params) === valueFingerprint,
  )
  const selectedPreset = selectedViewId?.startsWith('preset:')
    ? resolvedPresets.find((view) => `preset:${view.id}` === selectedViewId)
    : undefined
  const selectedLocalView =
    selectedViewId && selectedViewId !== 'all' && !selectedViewId.startsWith('preset:')
      ? views.find((view) => view.id === selectedViewId)
      : undefined
  const selectedViewFingerprint =
    selectedViewId === 'all'
      ? defaultsFingerprint
      : selectedPreset
        ? listQueryFingerprint(selectedPreset.params)
        : selectedLocalView
          ? listQueryFingerprint(selectedLocalView.params)
          : undefined
  const explicitActiveViewId = selectedViewFingerprint === valueFingerprint ? selectedViewId : null
  const activeViewId =
    explicitActiveViewId ??
    (valueFingerprint === defaultsFingerprint
      ? 'all'
      : activePreset
        ? `preset:${activePreset.id}`
        : (matchingLocalView?.id ?? null))
  const activeLocalView =
    activeViewId && activeViewId !== 'all' && !activeViewId.startsWith('preset:')
      ? (views.find((view) => view.id === activeViewId) ?? null)
      : null

  useEffect(() => {
    if (selectedViewId && selectedViewFingerprint && selectedViewFingerprint !== valueFingerprint)
      setSelectedViewId(null)
  }, [selectedViewFingerprint, selectedViewId, valueFingerprint])

  const applyPreset = (preset: QueryPresetView) => {
    setSelectedViewId(`preset:${preset.id}`)
    onChange(cloneListQueryParams(mergeListQueryParams(defaults, preset.params)))
  }
  const applyLocalView = (view: QueryView) => {
    setSelectedViewId(view.id)
    onChange(cloneListQueryParams(view.params))
  }

  const pendingQuery = () => {
    if (!quickSearch) return cloneListQueryParams(value)
    const searchValue = searchInput.trim()
    return createListQueryParams({
      ...value,
      [quickSearch.key]: searchValue || undefined,
      ...(searchValue !== quickSearchValue ? { after: undefined, before: undefined } : {}),
    })
  }

  const submitCreateView = () => {
    const name = viewName.trim()
    if (!name) {
      setViewError('Enter a view name.')
      return
    }
    if (
      views.some((view) => view.name.toLocaleLowerCase() === name.toLocaleLowerCase()) ||
      presetViews.some((view) => view.name.toLocaleLowerCase() === name.toLocaleLowerCase())
    ) {
      setViewError('A view with this name already exists.')
      return
    }
    const now = Date.now()
    const id = globalThis.crypto?.randomUUID?.() ?? `query-view-${now}`
    const params = pendingQuery()
    const next = [...views, { id, name, params, createdAt: now, updatedAt: now }]
    if (!writeViews(storageKey, next)) {
      setViewError('The view could not be saved in this browser.')
      return
    }
    setViews(next)
    setSelectedViewId(id)
    onChange(cloneListQueryParams(params))
    setCreatingView(false)
    setViewName('')
    toast.success(`${name} saved as a view.`)
  }

  const submitRenameView = (view: QueryView) => {
    const name = viewName.trim()
    if (!name) {
      setViewError('Enter a view name.')
      return
    }
    if (
      views.some(
        (candidate) =>
          candidate.id !== view.id &&
          candidate.name.toLocaleLowerCase() === name.toLocaleLowerCase(),
      ) ||
      presetViews.some((preset) => preset.name.toLocaleLowerCase() === name.toLocaleLowerCase())
    ) {
      setViewError('A view with this name already exists.')
      return
    }
    const next = views.map((candidate) =>
      candidate.id === view.id ? { ...candidate, name, updatedAt: Date.now() } : candidate,
    )
    if (!writeViews(storageKey, next)) {
      setViewError('The view could not be renamed in this browser.')
      return
    }
    setViews(next)
    setRenamingViewId(null)
    setViewName('')
    toast.success(`View renamed to ${name}.`)
  }

  const deleteView = (view: QueryView) => {
    const next = views.filter((candidate) => candidate.id !== view.id)
    if (!writeViews(storageKey, next)) {
      setViewError('The view could not be deleted in this browser.')
      return
    }
    setViews(next)
    if (selectedViewId === view.id) setSelectedViewId(null)
    setRenamingViewId(null)
    toast.success(`${view.name} deleted.`)
  }

  const applyAdvancedFilter = () => {
    setFilterError('')
    if (!selectedField) return
    if (selectedField.type === 'string') {
      const next = stringValue.trim()
      if (!next) {
        setFilterError(`Enter a value for ${selectedField.label.toLocaleLowerCase()}.`)
        return
      }
      updateFilters({ [selectedField.key]: next })
      setStringValue('')
    } else if (selectedField.type === 'enum') {
      const next = enumValue || selectedField.options[0]?.value
      if (!next) {
        setFilterError(`Choose a value for ${selectedField.label.toLocaleLowerCase()}.`)
        return
      }
      updateFilters({ [selectedField.key]: next })
      setEnumValue('')
    } else if (selectedField.type === 'boolean') {
      updateFilters({ enabled: booleanValue === 'true' })
    } else if (selectedField.type === 'metadata') {
      const key = metadataKey.trim()
      if (!key) {
        setFilterError('Enter a metadata key.')
        return
      }
      updateFilters({ metadata: { ...value.metadata, [key]: metadataValue } })
      setMetadataKey('')
      setMetadataValue('')
    } else {
      const timestamp = new Date(createdAtValue).getTime()
      if (!createdAtValue || !Number.isFinite(timestamp)) {
        setFilterError('Choose a valid date and time.')
        return
      }
      if (createdAtOperator === 'eq') {
        updateFilters({ [selectedField.key]: timestamp })
      } else {
        const currentValue = value[selectedField.key]
        const current = typeof currentValue === 'object' ? currentValue : {}
        const next = {
          ...current,
          eq: undefined,
          [createdAtOperator]: timestamp,
        }
        updateFilters({
          [selectedField.key]: Object.fromEntries(
            Object.entries(next).filter(([, entry]) => typeof entry === 'number'),
          ),
        })
      }
    }
    setFilterOpen(false)
  }

  const removeTimestamp = (field: TimestampFilterDefinition, operator: CreatedAtOperator) => {
    const currentValue = value[field.key]
    if (typeof currentValue === 'number') {
      updateFilters({ [field.key]: undefined })
      return
    }
    const next = { ...currentValue }
    delete next[operator]
    updateFilters({
      [field.key]: Object.keys(next).length ? next : undefined,
    })
  }

  const removeMetadata = (key: string) => {
    const next = { ...value.metadata }
    delete next[key]
    updateFilters({ metadata: next })
  }

  const clearFilters = () => {
    const next: Partial<ListQueryParams> = {
      ...value,
      after: undefined,
      before: undefined,
    }
    config.fields.forEach((field) => {
      if (field.key === 'metadata') next.metadata = {}
      else if (field.key === 'name') next.name = undefined
      else if (field.key === 'event_type') next.event_type = undefined
      else if (field.key === 'unique_id') next.unique_id = undefined
      else if (field.key === 'event_id') next.event_id = undefined
      else if (field.key === 'endpoint_id') next.endpoint_id = undefined
      else if (field.key === 'status') next.status = undefined
      else if (field.key === 'enabled') next.enabled = undefined
      else if (field.key === 'created_at') next.created_at = undefined
      else if (field.key === 'ingested_at') next.ingested_at = undefined
      else if (field.key === 'attempted_at') next.attempted_at = undefined
    })
    if (quickSearch) setSearchInput('')
    setSelectedViewId(null)
    onChange(createListQueryParams(next))
  }

  const startCreatingView = () => {
    setCreatingView(true)
    setRenamingViewId(null)
    setViewName('')
    setViewError('')
  }

  const startRenamingView = (view: QueryView) => {
    setRenamingViewId(view.id)
    setCreatingView(false)
    setViewName(view.name)
    setViewError('')
  }

  return (
    <div
      className={`border-b border-border bg-card ${disabled ? 'opacity-60' : ''}`}
      aria-disabled={disabled}
      inert={disabled ? true : undefined}
    >
      <div className="border-b border-border/70 px-3 py-2">
        <div className="flex min-w-0 flex-wrap items-center gap-2">
          <div className="flex min-w-0 flex-1 items-center gap-1 overflow-x-auto pb-px">
            <Bookmark className="mr-1 size-3.5 shrink-0 text-muted-foreground" aria-hidden="true" />
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className={`h-8 shrink-0 rounded-md px-2.5 ${viewButtonClass(activeViewId === 'all')}`}
              onClick={() => {
                setSelectedViewId('all')
                onChange(cloneListQueryParams(defaults))
              }}
            >
              {allViewLabel}
            </Button>
            {resolvedPresets.map((view) => (
              <Button
                type="button"
                key={view.id}
                variant="ghost"
                size="sm"
                className={`h-8 shrink-0 rounded-md px-2.5 ${viewButtonClass(activeViewId === `preset:${view.id}`)}`}
                onClick={() => applyPreset(view)}
              >
                {view.name}
              </Button>
            ))}
            {views.map((view) =>
              renamingViewId === view.id ? (
                <form
                  key={view.id}
                  className="flex shrink-0 items-center gap-1"
                  onSubmit={(event) => {
                    event.preventDefault()
                    submitRenameView(view)
                  }}
                >
                  <Input
                    autoFocus
                    aria-label={`Rename ${view.name}`}
                    className="h-7 w-36 px-2 text-xs"
                    value={viewName}
                    onChange={(event) => {
                      setViewName(event.target.value)
                      setViewError('')
                    }}
                    onKeyDown={(event) => {
                      if (event.key === 'Escape') setRenamingViewId(null)
                    }}
                  />
                  <Button
                    type="submit"
                    variant="ghost"
                    size="icon"
                    className="size-7"
                    aria-label="Save view name"
                  >
                    <Check className="size-3.5" />
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="size-7"
                    aria-label="Cancel rename"
                    onClick={() => setRenamingViewId(null)}
                  >
                    <X className="size-3.5" />
                  </Button>
                </form>
              ) : (
                <Button
                  type="button"
                  key={view.id}
                  variant="ghost"
                  size="sm"
                  className={`h-8 shrink-0 rounded-md px-2.5 ${viewButtonClass(activeViewId === view.id)}`}
                  onClick={() => applyLocalView(view)}
                >
                  {view.name}
                </Button>
              ),
            )}
            {creatingView ? (
              <form
                className="flex shrink-0 items-center gap-1"
                onSubmit={(event) => {
                  event.preventDefault()
                  submitCreateView()
                }}
              >
                <Input
                  autoFocus
                  aria-label="New view name"
                  placeholder="View name"
                  maxLength={80}
                  className="h-7 w-36 px-2 text-xs"
                  value={viewName}
                  onChange={(event) => {
                    setViewName(event.target.value)
                    setViewError('')
                  }}
                  onKeyDown={(event) => {
                    if (event.key === 'Escape') setCreatingView(false)
                  }}
                />
                <Button
                  type="submit"
                  variant="ghost"
                  size="icon"
                  className="size-7"
                  aria-label="Save new view"
                >
                  <Check className="size-3.5" />
                </Button>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="size-7"
                  aria-label="Cancel new view"
                  onClick={() => setCreatingView(false)}
                >
                  <X className="size-3.5" />
                </Button>
              </form>
            ) : (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="h-7 shrink-0 px-2 text-xs"
                onClick={startCreatingView}
              >
                <Plus className="size-3.5" />
                Save view
              </Button>
            )}
          </div>
          {activeLocalView && renamingViewId !== activeLocalView.id && (
            <div className="flex shrink-0 items-center gap-0.5">
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="size-7"
                aria-label={`Rename ${activeLocalView.name}`}
                onClick={() => startRenamingView(activeLocalView)}
              >
                <Pencil className="size-3.5" />
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="size-7 hover:text-red-500"
                aria-label={`Delete ${activeLocalView.name}`}
                onClick={() => deleteView(activeLocalView)}
              >
                <Trash2 className="size-3.5" />
              </Button>
            </div>
          )}
          <span className="ml-auto inline-flex shrink-0 items-center gap-1.5 text-xs text-muted-foreground">
            {busy && <LoaderCircle className="size-3.5 animate-spin" aria-hidden="true" />}
            {typeof total === 'number' ? `${total} results` : `${resultCount} on this page`}
          </span>
        </div>
        {viewError && <p className="mt-1.5 text-[11px] text-red-500">{viewError}</p>}
      </div>

      <div
        className={`grid gap-2 p-3 ${quickSearch ? 'sm:grid-cols-2 lg:grid-cols-[minmax(220px,1fr)_auto_auto]' : 'sm:grid-cols-2 lg:grid-cols-[1fr_auto]'}`}
      >
        {quickSearch && (
          <div className="relative min-w-0">
            <Search
              className="absolute left-3 top-2.5 size-4 text-muted-foreground"
              aria-hidden="true"
            />
            <Input
              value={searchInput}
              onChange={(event) => setSearchInput(event.target.value)}
              placeholder={
                quickSearch.placeholder ?? `Filter by ${quickSearch.label.toLocaleLowerCase()}…`
              }
              className="pl-9 pr-9"
              aria-label={`Filter by ${quickSearch.label.toLocaleLowerCase()}`}
            />
            {searchInput && (
              <Button
                type="button"
                variant="ghost"
                size="icon-xs"
                className="absolute right-2 top-2 size-5 text-muted-foreground"
                aria-label={`Clear ${quickSearch.label.toLocaleLowerCase()}`}
                onClick={() => {
                  setSearchInput('')
                  updateFilters({ [quickSearch.key]: undefined })
                }}
              >
                <X className="size-3.5" />
              </Button>
            )}
          </div>
        )}
        {filterFields.length > 0 && (
          <Button
            type="button"
            variant={advancedFilterCount ? 'secondary' : 'outline'}
            aria-expanded={filterOpen}
            onClick={() => {
              setFilterOpen((open) => !open)
              setFilterError('')
            }}
          >
            <Filter className="size-4" />
            Filter
            {advancedFilterCount > 0 && (
              <Badge className="min-w-5 justify-center border-primary/20 bg-primary/10 px-1.5 text-primary">
                {advancedFilterCount}
              </Badge>
            )}
          </Button>
        )}
        {config.showSort !== false && (
          <NativeSelect
            aria-label="Sort results"
            value={value.sort}
            onChange={(event) => updateFilters({ sort: event.target.value as ListSort })}
          >
            <option value="id.desc">Newest first</option>
            <option value="id.asc">Oldest first</option>
          </NativeSelect>
        )}
      </div>

      {advancedFilterCount > 0 && (
        <div className="flex flex-wrap items-center gap-1.5 border-t border-border/70 px-3 py-2">
          {menuStringEntries.map(({ field, value: entry }) => (
            <Button
              type="button"
              key={field.key}
              onClick={() => updateFilters({ [field.key]: undefined })}
              variant="outline"
              size="sm"
              className="h-7 gap-1.5 border-primary/25 bg-primary/5 px-2 text-[11px] text-primary hover:bg-primary/10 hover:text-primary"
            >
              <span>
                <strong className="font-medium">{field.label}</strong> = {entry}
              </span>
              <X className="size-3" aria-hidden="true" />
            </Button>
          ))}
          {enumEntries.map(({ field, value: entry }) => (
            <Button
              type="button"
              key={field.key}
              onClick={() => updateFilters({ [field.key]: undefined })}
              variant="outline"
              size="sm"
              className="h-7 gap-1.5 border-primary/25 bg-primary/5 px-2 text-[11px] text-primary hover:bg-primary/10 hover:text-primary"
            >
              <span>
                <strong className="font-medium">{field.label}</strong> ={' '}
                {field.options.find((option) => option.value === entry)?.label ?? entry}
              </span>
              <X className="size-3" aria-hidden="true" />
            </Button>
          ))}
          {enabledField && typeof value.enabled === 'boolean' && (
            <Button
              type="button"
              onClick={() => updateFilters({ enabled: undefined })}
              variant="outline"
              size="sm"
              className="h-7 gap-1.5 border-primary/25 bg-primary/5 px-2 text-[11px] text-primary hover:bg-primary/10 hover:text-primary"
            >
              <span>
                <strong className="font-medium">{enabledField.label}</strong> ={' '}
                {value.enabled
                  ? (enabledField.trueLabel ?? 'Enabled')
                  : (enabledField.falseLabel ?? 'Disabled')}
              </span>
              <X className="size-3" aria-hidden="true" />
            </Button>
          )}
          {timestampEntries.map(({ field, operator, timestamp }) => (
            <Button
              type="button"
              key={`${field.key}:${operator}`}
              onClick={() => removeTimestamp(field, operator)}
              variant="outline"
              size="sm"
              className="h-7 gap-1.5 border-primary/25 bg-primary/5 px-2 text-[11px] text-primary hover:bg-primary/10 hover:text-primary"
              aria-label={`Remove ${field.label.toLocaleLowerCase()} filter ${createdAtLabels[operator]}`}
            >
              <CalendarDays className="size-3" aria-hidden="true" />
              <span>
                <strong className="font-medium">{field.label}</strong> {createdAtLabels[operator]}{' '}
                {formatCreatedAt(timestamp)}
              </span>
              <X className="size-3" aria-hidden="true" />
            </Button>
          ))}
          {metadataEntries.map(([key, entry]) => (
            <Button
              type="button"
              key={key}
              onClick={() => removeMetadata(key)}
              variant="outline"
              size="sm"
              className="h-7 gap-1.5 border-primary/25 bg-primary/5 px-2 font-mono text-[11px] text-primary hover:bg-primary/10 hover:text-primary"
              aria-label={`Remove metadata filter ${key}`}
            >
              <span>
                <strong className="font-medium">metadata.</strong>
                {key} = {entry || 'empty'}
              </span>
              <X className="size-3" aria-hidden="true" />
            </Button>
          ))}
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={clearFilters}
            className="ml-auto h-7 px-2 text-[11px] text-muted-foreground"
          >
            Clear filters
          </Button>
        </div>
      )}

      {filterOpen && filterFields.length > 0 && (
        <div className="m-3 mt-0 rounded-lg border border-border bg-background p-3 shadow-lg">
          <div className="grid items-end gap-3 md:grid-cols-[minmax(150px,.8fr)_minmax(160px,1fr)_minmax(190px,1.2fr)_auto]">
            <div>
              <label className="label" htmlFor="query-filter-field">
                Field
              </label>
              <NativeSelect
                id="query-filter-field"
                className="w-full"
                value={filterFieldKey}
                onChange={(event) => {
                  setFilterFieldKey(event.target.value as QueryFilterDefinition['key'])
                  setStringValue('')
                  setEnumValue('')
                  setFilterError('')
                }}
              >
                {filterFields.map((field) => (
                  <option key={field.key} value={field.key}>
                    {field.label}
                  </option>
                ))}
              </NativeSelect>
            </div>
            {selectedField?.type === 'string' && (
              <>
                <div>
                  <label className="label" htmlFor="query-filter-string-condition">
                    Condition
                  </label>
                  <NativeSelect id="query-filter-string-condition" className="w-full" disabled>
                    <option>is exactly</option>
                  </NativeSelect>
                </div>
                <div>
                  <label className="label" htmlFor="query-filter-string-value">
                    Value
                  </label>
                  <Input
                    id="query-filter-string-value"
                    value={stringValue}
                    onChange={(event) => setStringValue(event.target.value)}
                  />
                </div>
              </>
            )}
            {selectedField?.type === 'boolean' && (
              <>
                <div>
                  <label className="label" htmlFor="query-filter-boolean-condition">
                    Condition
                  </label>
                  <NativeSelect id="query-filter-boolean-condition" className="w-full" disabled>
                    <option>is</option>
                  </NativeSelect>
                </div>
                <div>
                  <label className="label" htmlFor="query-filter-boolean-value">
                    Value
                  </label>
                  <NativeSelect
                    id="query-filter-boolean-value"
                    className="w-full"
                    value={booleanValue}
                    onChange={(event) => setBooleanValue(event.target.value)}
                  >
                    <option value="true">{selectedField.trueLabel ?? 'Enabled'}</option>
                    <option value="false">{selectedField.falseLabel ?? 'Disabled'}</option>
                  </NativeSelect>
                </div>
              </>
            )}
            {selectedField?.type === 'enum' && (
              <>
                <div>
                  <label className="label" htmlFor="query-filter-enum-condition">
                    Condition
                  </label>
                  <NativeSelect id="query-filter-enum-condition" className="w-full" disabled>
                    <option>is</option>
                  </NativeSelect>
                </div>
                <div>
                  <label className="label" htmlFor="query-filter-enum-value">
                    Value
                  </label>
                  <NativeSelect
                    id="query-filter-enum-value"
                    className="w-full"
                    value={enumValue || selectedField.options[0]?.value || ''}
                    onChange={(event) => setEnumValue(event.target.value)}
                  >
                    {selectedField.options.map((option) => (
                      <option key={option.value} value={option.value}>
                        {option.label}
                      </option>
                    ))}
                  </NativeSelect>
                </div>
              </>
            )}
            {(selectedField?.type === 'created_at' ||
              selectedField?.type === 'ingested_at' ||
              selectedField?.type === 'attempted_at') && (
              <>
                <div>
                  <label className="label" htmlFor="query-filter-date-condition">
                    Condition
                  </label>
                  <NativeSelect
                    id="query-filter-date-condition"
                    className="w-full"
                    value={createdAtOperator}
                    onChange={(event) =>
                      setCreatedAtOperator(event.target.value as CreatedAtOperator)
                    }
                  >
                    {createdAtOperators.map((operator) => (
                      <option key={operator} value={operator}>
                        {createdAtLabels[operator]}
                      </option>
                    ))}
                  </NativeSelect>
                </div>
                <div>
                  <label className="label" htmlFor="query-filter-date-value">
                    Date and time
                  </label>
                  <Input
                    id="query-filter-date-value"
                    type="datetime-local"
                    value={createdAtValue}
                    onChange={(event) => setCreatedAtValue(event.target.value)}
                  />
                </div>
              </>
            )}
            {selectedField?.type === 'metadata' && (
              <>
                <div>
                  <label className="label" htmlFor="query-filter-metadata-key">
                    Metadata key
                  </label>
                  <Input
                    id="query-filter-metadata-key"
                    className="font-mono text-xs"
                    placeholder={selectedField.keyPlaceholder ?? 'environment'}
                    value={metadataKey}
                    onChange={(event) => setMetadataKey(event.target.value)}
                  />
                </div>
                <div>
                  <label className="label" htmlFor="query-filter-metadata-value">
                    Metadata value
                  </label>
                  <Input
                    id="query-filter-metadata-value"
                    className="font-mono text-xs"
                    placeholder={selectedField.valuePlaceholder ?? 'production'}
                    value={metadataValue}
                    onChange={(event) => setMetadataValue(event.target.value)}
                  />
                </div>
              </>
            )}
            <Button type="button" onClick={applyAdvancedFilter}>
              <Check className="size-4" />
              Apply
            </Button>
          </div>
          {filterError && <p className="field-error">{filterError}</p>}
        </div>
      )}
    </div>
  )
}
