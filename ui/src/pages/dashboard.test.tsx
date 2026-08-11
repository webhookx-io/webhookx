import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { summarizeDeclarativeYaml } from '@/components/workspaces/workspace-config-dialog'
import { DECLARATIVE_YAML_EXAMPLE } from '@/lib/declarative-yaml'
import { DashboardPage } from '@/pages/dashboard'

const workspace = {
  id: 'ws_1',
  name: 'default',
  description: null,
  metadata: {},
  created_at: 1,
  updated_at: 1,
}

const source = {
  id: 'src_1',
  name: 'Inbound',
  enabled: true,
  type: 'http',
  config: { http: { path: '/incoming', methods: ['POST'] } },
  async: false,
  metadata: {},
  rate_limit: null,
  created_at: 1,
  updated_at: 1,
}

const endpoint = {
  id: 'ep_1',
  name: 'Destination',
  description: null,
  enabled: true,
  request: { url: 'https://example.com/hooks', method: 'POST', headers: null, timeout: 1000 },
  retry: { strategy: 'fixed', config: { attempts: [0] } },
  events: ['*'],
  metadata: {},
  rate_limit: null,
  created_at: 1,
  updated_at: 1,
}

const event = {
  id: 'evt_1',
  event_type: 'test.created',
  data: { hello: 'world' },
  ingested_at: 1,
  unique_id: null,
  created_at: 1,
  updated_at: 1,
}

const attempt = {
  id: 'att_1',
  event_id: event.id,
  endpoint_id: endpoint.id,
  status: 'SUCCESSFUL',
  attempt_number: 1,
  scheduled_at: 1,
  attempted_at: 1,
  trigger_mode: 'INITIAL',
  exhausted: false,
  error_code: null,
  request: null,
  response: null,
  created_at: 1,
}

function jsonResponse(value: unknown) {
  return new Response(JSON.stringify(value), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  })
}

function requestPath(input: RequestInfo | URL) {
  if (typeof input === 'string') return input
  if (input instanceof URL) return input.toString()
  return input.url
}

function memoryStorage(): Storage {
  const values = new Map<string, string>()
  return {
    get length() {
      return values.size
    },
    clear: () => values.clear(),
    getItem: (key) => values.get(key) ?? null,
    key: (index) => [...values.keys()][index] ?? null,
    removeItem: (key) => values.delete(key),
    setItem: (key, value) => values.set(key, value),
  }
}

function renderOverview(fetchMock: typeof fetch) {
  vi.stubGlobal('fetch', fetchMock)
  vi.stubGlobal('localStorage', memoryStorage())
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={['/workspaces/default/overview']}>
        <Routes>
          <Route path="/workspaces/:workspaceName/overview" element={<DashboardPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('DashboardPage workspace setup', () => {
  it('starts with manual setup and exposes YAML sync as the secondary path', async () => {
    const fetchMock = vi.fn<typeof fetch>().mockImplementation((input) => {
      const path = requestPath(input)
      if (path.startsWith('/api/workspaces?')) {
        return Promise.resolve(jsonResponse({ data: [workspace] }))
      }
      if (path === '/api/workspaces/ws_1/config/sync') {
        return Promise.resolve(jsonResponse(null))
      }
      if (path.includes('/sources?') || path.includes('/endpoints?') || path.includes('/events?')) {
        return Promise.resolve(jsonResponse({ data: [] }))
      }
      return Promise.reject(new Error(`Unexpected request: ${path}`))
    })

    renderOverview(fetchMock)

    expect(await screen.findByRole('heading', { name: 'Guided setup' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Create source' })).toHaveAttribute(
      'href',
      '/workspaces/default/sources/create',
    )
    expect(screen.getByRole('button', { name: 'Create endpoint' })).toBeDisabled()

    fireEvent.click(screen.getByRole('button', { name: 'Sync YAML' }))
    expect(screen.getByRole('dialog', { name: 'Sync YAML configuration' })).toBeInTheDocument()
    expect(screen.getByText('This replaces the workspace configuration')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Insert example' }))
    expect(screen.getByText('Valid YAML')).toBeInTheDocument()
    expect(screen.getByText('1 source')).toBeInTheDocument()
    expect(screen.getByText('1 endpoint')).toBeInTheDocument()
    fireEvent.click(
      screen.getByRole('checkbox', {
        name: 'I understand that resources missing from the YAML will be removed.',
      }),
    )
    fireEvent.click(screen.getByRole('button', { name: 'Sync configuration' }))

    await waitFor(() =>
      expect(
        fetchMock.mock.calls.some(
          ([path]) => requestPath(path) === '/api/workspaces/ws_1/config/sync',
        ),
      ).toBe(true),
    )
    const syncRequest = fetchMock.mock.calls.find(
      ([path]) => requestPath(path) === '/api/workspaces/ws_1/config/sync',
    )
    expect(syncRequest?.[1]).toMatchObject({
      method: 'POST',
      body: DECLARATIVE_YAML_EXAMPLE,
    })
    expect(new Headers(syncRequest?.[1]?.headers).get('Content-Type')).toBe('application/yaml')
  })

  it('builds the test curl command from the default Gateway and source path', async () => {
    const fetchMock = vi.fn<typeof fetch>().mockImplementation((input) => {
      const path = requestPath(input)
      if (path.startsWith('/api/workspaces?')) {
        return Promise.resolve(jsonResponse({ data: [workspace] }))
      }
      if (path.includes('/sources?')) return Promise.resolve(jsonResponse({ data: [source] }))
      if (path.includes('/endpoints?')) return Promise.resolve(jsonResponse({ data: [endpoint] }))
      if (path.includes('/events?')) return Promise.resolve(jsonResponse({ data: [] }))
      return Promise.reject(new Error(`Unexpected request: ${path}`))
    })

    renderOverview(fetchMock)

    expect(await screen.findByText(/http:\/\/127\.0\.0\.1:9600\/incoming/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Copy curl command' })).toBeInTheDocument()
  })

  it('links completed setup actions to the event and delivery lists', async () => {
    const fetchMock = vi.fn<typeof fetch>().mockImplementation((input) => {
      const path = requestPath(input)
      if (path.startsWith('/api/workspaces?')) {
        return Promise.resolve(jsonResponse({ data: [workspace] }))
      }
      if (path.includes('/sources?')) return Promise.resolve(jsonResponse({ data: [source] }))
      if (path.includes('/endpoints?')) return Promise.resolve(jsonResponse({ data: [endpoint] }))
      if (path.includes('/events?')) return Promise.resolve(jsonResponse({ data: [event] }))
      if (path.includes('/attempts?')) return Promise.resolve(jsonResponse({ data: [attempt] }))
      return Promise.reject(new Error(`Unexpected request: ${path}`))
    })

    renderOverview(fetchMock)

    expect(await screen.findByRole('heading', { name: 'Workspace is ready' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'View event' })).toHaveAttribute(
      'href',
      '/workspaces/default/events',
    )
    expect(screen.getByRole('link', { name: 'View deliveries' })).toHaveAttribute(
      'href',
      '/workspaces/default/deliveries',
    )
  })
})

describe('summarizeDeclarativeYaml', () => {
  it('counts the resources that will be synced', () => {
    expect(
      summarizeDeclarativeYaml(`
sources:
  - name: first
  - name: second
endpoints:
  - name: destination
`),
    ).toEqual({ sources: 2, endpoints: 1 })
  })

  it('rejects documents without both resource arrays', () => {
    expect(() => summarizeDeclarativeYaml('sources: []')).toThrow(
      'both sources and endpoints arrays',
    )
  })
})
