import { createListQueryParams, listQueryString } from '@/data/list-query'
import type {
  AdminInfo,
  ApiKey,
  Attempt,
  AttemptListParams,
  AttemptPage,
  CursorPage,
  DashboardConfig,
  Delivery,
  Endpoint,
  EndpointInput,
  EndpointPage,
  EventListParams,
  EventPage,
  License,
  ListQueryParams,
  Plugin,
  PluginCatalogItem,
  PluginInput,
  Source,
  SourceInput,
  SourcePage,
  WebhookEvent,
  Workspace,
  WorkspaceInput,
  WorkspacePage,
} from '@/types'

const apiRoot = (import.meta.env.VITE_API_BASE_URL || '/api').replace(/\/$/, '')

interface Page<T> {
  data: T[]
  next?: string | null
  prev?: string | null
  total?: number
}

interface PluginCatalog {
  plugins: PluginCatalogItem[]
}

function normalizePage<T>(page: Page<T> | T[]): CursorPage<T> {
  if (Array.isArray(page)) return { data: page, next: null, prev: null }
  return {
    data: page.data,
    next: page.next ?? null,
    prev: page.prev ?? null,
    total: page.total,
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${apiRoot}${path}`, {
    ...init,
    headers: {
      Accept: 'application/json',
      ...(init?.body ? { 'Content-Type': 'application/json' } : {}),
      ...init?.headers,
    },
  })

  if (!response.ok) {
    let message = `Request failed with status ${response.status}`
    try {
      const error = (await response.json()) as { message?: string }
      if (error.message) message = error.message
    } catch {
      // The response did not contain a JSON error body.
    }
    throw new Error(message)
  }

  if (response.status === 204) return undefined as T
  return response.json() as Promise<T>
}

async function textRequest(path: string, init?: RequestInit): Promise<string> {
  const response = await fetch(`${apiRoot}${path}`, {
    ...init,
    headers: {
      Accept: 'text/plain, application/yaml',
      ...init?.headers,
    },
  })

  if (!response.ok) {
    throw new Error(`Request failed with status ${response.status}`)
  }
  return response.text()
}

async function dashboardConfig(): Promise<DashboardConfig> {
  const response = await fetch('/config', { headers: { Accept: 'application/json' } })
  if (!response.ok) {
    throw new Error(`Request failed with status ${response.status}`)
  }
  return response.json() as Promise<DashboardConfig>
}

const now = Date.now()
const ago = (seconds: number) => new Date(now - seconds * 1000).toISOString()

const dashboardDeliveries: Delivery[] = [
  {
    id: 'dlv_01K1Z0001',
    eventId: 'evt_01K1ZCWX8H8B4FX7Q6',
    endpoint: 'Production orders',
    status: 'delivered',
    attempt: 1,
    latency: 184,
    createdAt: ago(18),
    responseCode: 200,
  },
  {
    id: 'dlv_01K1Z0002',
    eventId: 'evt_01K1ZCVW3JK9CQM8N2',
    endpoint: 'Stripe mirror',
    status: 'failed',
    attempt: 4,
    latency: 2032,
    createdAt: ago(44),
    responseCode: 503,
  },
  {
    id: 'dlv_01K1Z0003',
    eventId: 'evt_01K1ZCTD4CQ5C8EAMR',
    endpoint: 'Customer notifications',
    status: 'delivered',
    attempt: 1,
    latency: 91,
    createdAt: ago(79),
    responseCode: 204,
  },
  {
    id: 'dlv_01K1Z0004',
    eventId: 'evt_01K1ZCS70GHF2QMRMV',
    endpoint: 'Legacy fulfillment',
    status: 'pending',
    attempt: 1,
    latency: 0,
    createdAt: ago(122),
    responseCode: null,
  },
  {
    id: 'dlv_01K1Z0005',
    eventId: 'evt_01K1ZCQGJ8N7V1XRZ4',
    endpoint: 'Production orders',
    status: 'retrying',
    attempt: 2,
    latency: 1184,
    createdAt: ago(188),
    responseCode: 429,
  },
  {
    id: 'dlv_01K1Z0006',
    eventId: 'evt_01K1ZCMWNX3PB2TK9E',
    endpoint: 'Stripe mirror',
    status: 'delivered',
    attempt: 1,
    latency: 241,
    createdAt: ago(260),
    responseCode: 200,
  },
  {
    id: 'dlv_01K1Z0007',
    eventId: 'evt_01K1ZCHAE9TPJ81J2V',
    endpoint: 'Data warehouse',
    status: 'failed',
    attempt: 5,
    latency: 5042,
    createdAt: ago(371),
    responseCode: 504,
  },
]

let apiKeys: ApiKey[] = [
  {
    id: 'key_01J8D4JH',
    name: 'Production deploys',
    prefix: 'whx_live_v3f8••••••••',
    createdAt: ago(86400 * 42),
    lastUsedAt: ago(480),
    scope: 'Admin',
  },
  {
    id: 'key_01J72A9M',
    name: 'Grafana metrics',
    prefix: 'whx_live_q1p2••••••••',
    createdAt: ago(86400 * 91),
    lastUsedAt: ago(41),
    scope: 'Read only',
  },
  {
    id: 'key_01J5TX7R',
    name: 'Staging CI',
    prefix: 'whx_test_m7k4••••••••',
    createdAt: ago(86400 * 128),
    lastUsedAt: ago(86400 * 12),
    scope: 'Read & write',
  },
]

const pause = <T>(value: T, delay = 240) =>
  new Promise<T>((resolve) => setTimeout(() => resolve(value), delay))

export const api = {
  dashboardConfig,
  adminInfo: () => request<AdminInfo>('/'),
  license: () => request<License>('/license'),
  workspaces: async () => {
    const params = createListQueryParams({ limit: 1000, sort: 'id.asc' })
    const page = await request<Page<Workspace> | Workspace[]>(
      `/workspaces?${listQueryString(params)}`,
    )
    return normalizePage(page).data
  },
  workspacePage: async (params: ListQueryParams): Promise<WorkspacePage> =>
    normalizePage(
      await request<Page<Workspace> | Workspace[]>(`/workspaces?${listQueryString(params)}`),
    ),
  createWorkspace: (input: WorkspaceInput) =>
    request<Workspace>('/workspaces', {
      method: 'POST',
      body: JSON.stringify({ ...input, metadata: input.metadata ?? {} }),
    }),
  updateWorkspace: (id: string, input: Partial<WorkspaceInput>) =>
    request<Workspace>(`/workspaces/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: JSON.stringify(input),
    }),
  deleteWorkspace: (id: string) =>
    request<void>(`/workspaces/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    }),
  syncWorkspaceConfig: (workspaceId: string, yaml: string) =>
    request<void>(`/workspaces/${encodeURIComponent(workspaceId)}/config/sync`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/yaml' },
      body: yaml,
    }),
  dumpWorkspaceConfig: (workspaceId: string) =>
    textRequest(`/workspaces/${encodeURIComponent(workspaceId)}/config/dump`, {
      method: 'POST',
    }),
  sources: async (workspaceId: string, params: ListQueryParams): Promise<SourcePage> =>
    normalizePage(
      await request<Page<Source> | Source[]>(
        `/workspaces/${encodeURIComponent(workspaceId)}/sources?${listQueryString(params)}`,
      ),
    ),
  source: (workspaceId: string, sourceId: string) =>
    request<Source>(
      `/workspaces/${encodeURIComponent(workspaceId)}/sources/${encodeURIComponent(sourceId)}`,
    ),
  createSource: (workspaceId: string, input: SourceInput) =>
    request<Source>(`/workspaces/${encodeURIComponent(workspaceId)}/sources`, {
      method: 'POST',
      body: JSON.stringify(input),
    }),
  updateSource: (workspaceId: string, sourceId: string, input: SourceInput) =>
    request<Source>(
      `/workspaces/${encodeURIComponent(workspaceId)}/sources/${encodeURIComponent(sourceId)}`,
      {
        method: 'PUT',
        body: JSON.stringify(input),
      },
    ),
  updateSourceEnabled: (workspaceId: string, id: string, enabled: boolean) =>
    request<Source>(
      `/workspaces/${encodeURIComponent(workspaceId)}/sources/${encodeURIComponent(id)}`,
      {
        method: 'PUT',
        body: JSON.stringify({ id, enabled }),
      },
    ),
  deleteSource: (workspaceId: string, sourceId: string) =>
    request<void>(
      `/workspaces/${encodeURIComponent(workspaceId)}/sources/${encodeURIComponent(sourceId)}`,
      { method: 'DELETE' },
    ),
  endpoints: async (workspaceId: string, params: ListQueryParams): Promise<EndpointPage> =>
    normalizePage(
      await request<Page<Endpoint> | Endpoint[]>(
        `/workspaces/${encodeURIComponent(workspaceId)}/endpoints?${listQueryString(params)}`,
      ),
    ),
  endpoint: (workspaceId: string, endpointId: string) =>
    request<Endpoint>(
      `/workspaces/${encodeURIComponent(workspaceId)}/endpoints/${encodeURIComponent(endpointId)}`,
    ),
  createEndpoint: (workspaceId: string, input: EndpointInput) =>
    request<Endpoint>(`/workspaces/${encodeURIComponent(workspaceId)}/endpoints`, {
      method: 'POST',
      body: JSON.stringify(input),
    }),
  updateEndpoint: (workspaceId: string, endpointId: string, input: EndpointInput) =>
    request<Endpoint>(
      `/workspaces/${encodeURIComponent(workspaceId)}/endpoints/${encodeURIComponent(endpointId)}`,
      {
        method: 'PUT',
        body: JSON.stringify(input),
      },
    ),
  updateEndpointEnabled: (workspaceId: string, id: string, enabled: boolean) =>
    request<Endpoint>(
      `/workspaces/${encodeURIComponent(workspaceId)}/endpoints/${encodeURIComponent(id)}`,
      {
        method: 'PUT',
        body: JSON.stringify({ id, enabled }),
      },
    ),
  deleteEndpoint: (workspaceId: string, endpointId: string) =>
    request<void>(
      `/workspaces/${encodeURIComponent(workspaceId)}/endpoints/${encodeURIComponent(endpointId)}`,
      { method: 'DELETE' },
    ),
  events: async (workspaceId: string, params: EventListParams): Promise<EventPage> =>
    normalizePage(
      await request<Page<WebhookEvent> | WebhookEvent[]>(
        `/workspaces/${encodeURIComponent(workspaceId)}/events?${listQueryString(params)}`,
      ),
    ),
  event: (workspaceId: string, eventId: string) =>
    request<WebhookEvent>(
      `/workspaces/${encodeURIComponent(workspaceId)}/events/${encodeURIComponent(eventId)}`,
    ),
  retryEvent: (workspaceId: string, eventId: string, endpointId: string) =>
    request<void>(
      `/workspaces/${encodeURIComponent(workspaceId)}/events/${encodeURIComponent(eventId)}/retry?endpoint_id=${encodeURIComponent(endpointId)}`,
      { method: 'POST' },
    ),
  attempts: async (workspaceId: string, params: AttemptListParams): Promise<AttemptPage> =>
    normalizePage(
      await request<Page<Attempt> | Attempt[]>(
        `/workspaces/${encodeURIComponent(workspaceId)}/attempts?${listQueryString(params)}`,
      ),
    ),
  attempt: (workspaceId: string, attemptId: string) =>
    request<Attempt>(
      `/workspaces/${encodeURIComponent(workspaceId)}/attempts/${encodeURIComponent(attemptId)}`,
    ),
  deliveries: () => pause([...dashboardDeliveries]),
  pluginCatalog: async () => (await request<PluginCatalog>('/catalog')).plugins,
  createPlugin: (workspaceId: string, input: PluginInput) =>
    request<Plugin>(`/workspaces/${encodeURIComponent(workspaceId)}/plugins`, {
      method: 'POST',
      body: JSON.stringify(input),
    }),
  apiKeys: () => pause([...apiKeys]),
  createApiKey: async (name: string, scope: ApiKey['scope']) => {
    const raw = `whx_live_${crypto.randomUUID().replaceAll('-', '')}`
    const key: ApiKey = {
      id: `key_${crypto.randomUUID().slice(0, 8)}`,
      name,
      prefix: `${raw.slice(0, 13)}••••••••`,
      createdAt: new Date().toISOString(),
      lastUsedAt: null,
      scope,
    }
    apiKeys = [key, ...apiKeys]
    return pause({ key, raw }, 420)
  },
  revokeApiKey: async (id: string) => {
    apiKeys = apiKeys.filter((key) => key.id !== id)
    return pause(id, 300)
  },
}

export const deliveryVolume = [
  { time: '00:00', delivered: 3800, failed: 64 },
  { time: '04:00', delivered: 2900, failed: 41 },
  { time: '08:00', delivered: 6300, failed: 93 },
  { time: '12:00', delivered: 8200, failed: 118 },
  { time: '16:00', delivered: 7100, failed: 74 },
  { time: '20:00', delivered: 9200, failed: 106 },
  { time: 'Now', delivered: 8600, failed: 59 },
]
