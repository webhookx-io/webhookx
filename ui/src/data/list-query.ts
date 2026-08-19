import type { CreatedAtOperator, ListQueryParams, ListSort } from '@/types'

export const defaultListQueryParams: ListQueryParams = {
  limit: 20,
  sort: 'id.desc',
  metadata: {},
}

const createdAtOperators: CreatedAtOperator[] = ['eq', 'gt', 'gte', 'lt', 'lte']

export function createListQueryParams(overrides: Partial<ListQueryParams> = {}): ListQueryParams {
  const limit =
    typeof overrides.limit === 'number' && Number.isFinite(overrides.limit)
      ? Math.min(1000, Math.max(1, Math.round(overrides.limit)))
      : defaultListQueryParams.limit
  const sort: ListSort = overrides.sort === 'id.asc' ? 'id.asc' : 'id.desc'
  const params: ListQueryParams = { limit, sort, metadata: {} }

  if (typeof overrides.after === 'string' && overrides.after) params.after = overrides.after
  else if (typeof overrides.before === 'string' && overrides.before)
    params.before = overrides.before
  if (typeof overrides.name === 'string' && overrides.name.trim())
    params.name = overrides.name.trim()
  if (typeof overrides.event_type === 'string' && overrides.event_type.trim())
    params.event_type = overrides.event_type.trim()
  if (typeof overrides.unique_id === 'string' && overrides.unique_id.trim())
    params.unique_id = overrides.unique_id.trim()
  if (typeof overrides.event_id === 'string' && overrides.event_id.trim())
    params.event_id = overrides.event_id.trim()
  if (typeof overrides.endpoint_id === 'string' && overrides.endpoint_id.trim())
    params.endpoint_id = overrides.endpoint_id.trim()
  if (typeof overrides.status === 'string' && overrides.status.trim())
    params.status = overrides.status.trim()
  if (typeof overrides.enabled === 'boolean') params.enabled = overrides.enabled

  if (typeof overrides.created_at === 'number' && Number.isFinite(overrides.created_at)) {
    params.created_at = overrides.created_at
  } else if (overrides.created_at && typeof overrides.created_at === 'object') {
    const createdAt = Object.fromEntries(
      createdAtOperators.flatMap((operator) => {
        const value =
          overrides.created_at && typeof overrides.created_at === 'object'
            ? overrides.created_at[operator]
            : undefined
        return typeof value === 'number' && Number.isFinite(value) ? [[operator, value]] : []
      }),
    )
    const keys = Object.keys(createdAt)
    if (keys.length === 1 && typeof createdAt.eq === 'number') params.created_at = createdAt.eq
    else if (keys.length) params.created_at = createdAt
  }

  if (typeof overrides.ingested_at === 'number' && Number.isFinite(overrides.ingested_at)) {
    params.ingested_at = overrides.ingested_at
  } else if (overrides.ingested_at && typeof overrides.ingested_at === 'object') {
    const ingestedAt = Object.fromEntries(
      createdAtOperators.flatMap((operator) => {
        const value =
          overrides.ingested_at && typeof overrides.ingested_at === 'object'
            ? overrides.ingested_at[operator]
            : undefined
        return typeof value === 'number' && Number.isFinite(value) ? [[operator, value]] : []
      }),
    )
    const keys = Object.keys(ingestedAt)
    if (keys.length === 1 && typeof ingestedAt.eq === 'number') params.ingested_at = ingestedAt.eq
    else if (keys.length) params.ingested_at = ingestedAt
  }

  if (typeof overrides.attempted_at === 'number' && Number.isFinite(overrides.attempted_at)) {
    params.attempted_at = overrides.attempted_at
  } else if (overrides.attempted_at && typeof overrides.attempted_at === 'object') {
    const attemptedAt = Object.fromEntries(
      createdAtOperators.flatMap((operator) => {
        const value =
          overrides.attempted_at && typeof overrides.attempted_at === 'object'
            ? overrides.attempted_at[operator]
            : undefined
        return typeof value === 'number' && Number.isFinite(value) ? [[operator, value]] : []
      }),
    )
    const keys = Object.keys(attemptedAt)
    if (keys.length === 1 && typeof attemptedAt.eq === 'number') {
      params.attempted_at = attemptedAt.eq
    } else if (keys.length) params.attempted_at = attemptedAt
  }

  if (overrides.metadata && typeof overrides.metadata === 'object') {
    params.metadata = Object.fromEntries(
      Object.entries(overrides.metadata).flatMap(([key, value]) =>
        key.trim() && typeof value === 'string' ? [[key.trim(), value]] : [],
      ),
    )
  }
  return params
}

export function mergeListQueryParams(
  base: Partial<ListQueryParams>,
  overrides: Partial<ListQueryParams> = {},
) {
  return createListQueryParams({
    ...base,
    ...overrides,
    metadata: { ...(base.metadata ?? {}), ...(overrides.metadata ?? {}) },
    created_at: overrides.created_at ?? base.created_at,
    ingested_at: overrides.ingested_at ?? base.ingested_at,
    attempted_at: overrides.attempted_at ?? base.attempted_at,
  })
}

export function cloneListQueryParams(params: ListQueryParams): ListQueryParams {
  return {
    ...params,
    created_at:
      typeof params.created_at === 'object' ? { ...params.created_at } : params.created_at,
    ingested_at:
      typeof params.ingested_at === 'object' ? { ...params.ingested_at } : params.ingested_at,
    attempted_at:
      typeof params.attempted_at === 'object' ? { ...params.attempted_at } : params.attempted_at,
    metadata: { ...params.metadata },
  }
}

export function workspacePlaceholderData<T>(
  workspaceId: string | undefined,
  previousData: T | undefined,
  previousQuery: { queryKey: readonly unknown[] } | undefined,
) {
  return previousQuery?.queryKey[1] === workspaceId ? previousData : undefined
}

export function listQueryString(params: ListQueryParams) {
  const query = new URLSearchParams()
  query.set('limit', String(Math.min(1000, Math.max(1, params.limit))))
  query.set('sort', params.sort)
  if (params.after) query.set('after', params.after)
  if (params.before) query.set('before', params.before)
  if (params.name) query.set('name', params.name)
  if (params.event_type) query.set('event_type', params.event_type)
  if (params.unique_id) query.set('unique_id', params.unique_id)
  if (params.event_id) query.set('event_id', params.event_id)
  if (params.endpoint_id) query.set('endpoint_id', params.endpoint_id)
  if (params.status) query.set('status', params.status)
  if (typeof params.enabled === 'boolean') query.set('enabled', String(params.enabled))

  if (typeof params.created_at === 'number') {
    query.set('created_at', String(params.created_at))
  } else if (params.created_at) {
    Object.entries(params.created_at).forEach(([operator, value]) => {
      if (typeof value !== 'number') return
      if (operator === 'eq') query.set('created_at', String(value))
      else query.set(`created_at[${operator}]`, String(value))
    })
  }

  if (typeof params.ingested_at === 'number') {
    query.set('ingested_at', String(params.ingested_at))
  } else if (params.ingested_at) {
    Object.entries(params.ingested_at).forEach(([operator, value]) => {
      if (typeof value !== 'number') return
      if (operator === 'eq') query.set('ingested_at', String(value))
      else query.set(`ingested_at[${operator}]`, String(value))
    })
  }

  if (typeof params.attempted_at === 'number') {
    query.set('attempted_at', String(params.attempted_at))
  } else if (params.attempted_at) {
    Object.entries(params.attempted_at).forEach(([operator, value]) => {
      if (typeof value !== 'number') return
      if (operator === 'eq') query.set('attempted_at', String(value))
      else query.set(`attempted_at[${operator}]`, String(value))
    })
  }

  Object.entries(params.metadata).forEach(([key, value]) => {
    if (key) query.set(`metadata[${key}]`, value)
  })
  return query.toString()
}

export function listQueryParamsFromSearchParams(
  searchParams: URLSearchParams,
  defaults: Partial<ListQueryParams> = {},
) {
  const numberParam = (name: string) => {
    const value = searchParams.get(name)
    if (value === null || !value.trim()) return undefined
    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : undefined
  }

  const enabledValue = searchParams.get('enabled')
  const enabled =
    enabledValue === 'true' ? true : enabledValue === 'false' ? false : defaults.enabled

  const createdAtValue = numberParam('created_at')
  const createdAt = Object.fromEntries(
    createdAtOperators.flatMap((operator) => {
      const value = numberParam(`created_at[${operator}]`)
      return value === undefined ? [] : [[operator, value]]
    }),
  )
  const ingestedAtValue = numberParam('ingested_at')
  const ingestedAt = Object.fromEntries(
    createdAtOperators.flatMap((operator) => {
      const value = numberParam(`ingested_at[${operator}]`)
      return value === undefined ? [] : [[operator, value]]
    }),
  )
  const attemptedAtValue = numberParam('attempted_at')
  const attemptedAt = Object.fromEntries(
    createdAtOperators.flatMap((operator) => {
      const value = numberParam(`attempted_at[${operator}]`)
      return value === undefined ? [] : [[operator, value]]
    }),
  )
  const metadata: Record<string, string> = {}
  searchParams.forEach((value, key) => {
    const match = /^metadata\[(.+)]$/.exec(key)
    if (match?.[1]) metadata[match[1]] = value
  })

  const sortValue = searchParams.get('sort')
  return createListQueryParams({
    ...defaults,
    limit: numberParam('limit') ?? defaults.limit,
    sort: sortValue === 'id.asc' || sortValue === 'id.desc' ? sortValue : defaults.sort,
    after: searchParams.get('after') || defaults.after,
    before: searchParams.get('before') || defaults.before,
    name: searchParams.get('name') ?? defaults.name,
    event_type: searchParams.get('event_type') ?? defaults.event_type,
    unique_id: searchParams.get('unique_id') ?? defaults.unique_id,
    event_id: searchParams.get('event_id') ?? defaults.event_id,
    endpoint_id: searchParams.get('endpoint_id') ?? defaults.endpoint_id,
    status: searchParams.get('status') ?? defaults.status,
    enabled,
    created_at:
      createdAtValue !== undefined
        ? Object.keys(createdAt).length
          ? { ...createdAt, eq: createdAtValue }
          : createdAtValue
        : Object.keys(createdAt).length
          ? createdAt
          : defaults.created_at,
    ingested_at:
      ingestedAtValue !== undefined
        ? Object.keys(ingestedAt).length
          ? { ...ingestedAt, eq: ingestedAtValue }
          : ingestedAtValue
        : Object.keys(ingestedAt).length
          ? ingestedAt
          : defaults.ingested_at,
    attempted_at:
      attemptedAtValue !== undefined
        ? Object.keys(attemptedAt).length
          ? { ...attemptedAt, eq: attemptedAtValue }
          : attemptedAtValue
        : Object.keys(attemptedAt).length
          ? attemptedAt
          : defaults.attempted_at,
    metadata: Object.keys(metadata).length ? metadata : defaults.metadata,
  })
}

export function listQueryFingerprint(params: ListQueryParams) {
  const normalized = createListQueryParams(params)
  return JSON.stringify({
    ...normalized,
    created_at:
      typeof normalized.created_at === 'object'
        ? Object.fromEntries(
            Object.entries(normalized.created_at).sort(([left], [right]) =>
              left.localeCompare(right),
            ),
          )
        : normalized.created_at,
    ingested_at:
      typeof normalized.ingested_at === 'object'
        ? Object.fromEntries(
            Object.entries(normalized.ingested_at).sort(([left], [right]) =>
              left.localeCompare(right),
            ),
          )
        : normalized.ingested_at,
    attempted_at:
      typeof normalized.attempted_at === 'object'
        ? Object.fromEntries(
            Object.entries(normalized.attempted_at).sort(([left], [right]) =>
              left.localeCompare(right),
            ),
          )
        : normalized.attempted_at,
    metadata: Object.fromEntries(
      Object.entries(normalized.metadata).sort(([left], [right]) => left.localeCompare(right)),
    ),
  })
}
