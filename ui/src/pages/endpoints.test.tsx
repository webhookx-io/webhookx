import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { TooltipProvider } from '@/components/ui/tooltip'
import { EndpointsPage } from '@/pages/endpoints'

const workspace = {
  id: 'ws_1',
  name: 'default',
  description: null,
  metadata: {},
  created_at: 1,
  updated_at: 1,
}

const endpoint = {
  id: 'ep_1',
  name: 'Production orders',
  description: 'Delivers order events to production.',
  enabled: true,
  request: {
    url: 'https://example.com/hooks',
    method: 'POST',
    headers: { Authorization: 'Bearer test' },
    timeout: 10_000,
  },
  retry: { strategy: 'fixed', config: { attempts: [0, 60] } },
  events: ['order.created'],
  metadata: { environment: 'production' },
  rate_limit: { quota: 100, period: 60 },
  created_at: 1,
  updated_at: 1,
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

function renderEndpoints() {
  vi.stubGlobal(
    'ResizeObserver',
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    },
  )
  const fetchMock = vi.fn<typeof fetch>().mockImplementation((input) => {
    const path = requestPath(input)
    if (path.startsWith('/api/workspaces?')) {
      return Promise.resolve(jsonResponse({ data: [workspace] }))
    }
    if (path.includes('/api/workspaces/ws_1/endpoints?')) {
      return Promise.resolve(jsonResponse({ data: [endpoint] }))
    }
    if (path === '/api/workspaces/ws_1/endpoints/ep_1') {
      return Promise.resolve(jsonResponse(endpoint))
    }
    return Promise.reject(new Error(`Unexpected request: ${path}`))
  })
  vi.stubGlobal('fetch', fetchMock)

  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })

  render(
    <QueryClientProvider client={queryClient}>
      <TooltipProvider>
        <MemoryRouter initialEntries={['/workspaces/default/endpoints']}>
          <Routes>
            <Route path="/workspaces/:workspaceName/endpoints/*" element={<EndpointsPage />} />
          </Routes>
        </MemoryRouter>
      </TooltipProvider>
    </QueryClientProvider>,
  )
}

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

describe('EndpointsPage table and drawer interactions', () => {
  it('opens the edit drawer from a table row with advanced configuration collapsed', async () => {
    renderEndpoints()

    fireEvent.click(await screen.findByLabelText('Edit Production orders'))

    expect(await screen.findByRole('heading', { name: 'Edit endpoint' })).toBeInTheDocument()
    await screen.findByDisplayValue('Production orders')
    expect(screen.queryByRole('heading', { name: 'Request headers' })).not.toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Retry policy' })).not.toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Traffic controls' })).not.toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Metadata' })).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: /Advanced Configuration/ }))

    expect(screen.getByRole('heading', { name: 'Request headers' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Retry policy' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Traffic controls' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Metadata' })).toBeInTheDocument()
  })

  it('keeps delete as the only row action without opening the drawer', async () => {
    renderEndpoints()

    await screen.findByLabelText('Edit Production orders')
    fireEvent.pointerDown(screen.getByRole('button', { name: 'Actions for Production orders' }))

    expect(await screen.findByRole('menuitem', { name: 'Delete' })).toBeInTheDocument()
    expect(screen.queryByRole('menuitem', { name: 'Edit' })).not.toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Edit endpoint' })).not.toBeInTheDocument()
  })

  it('uses the drawer for create and defaults the HTTP method to POST', async () => {
    renderEndpoints()

    fireEvent.click(await screen.findByRole('button', { name: 'Create endpoint' }))

    expect(await screen.findByRole('heading', { name: 'Create endpoint' })).toBeInTheDocument()
    expect(screen.getByRole('combobox', { name: 'HTTP method' })).toHaveValue('POST')
    expect(screen.getByRole('button', { name: /Advanced Configuration/ })).toHaveAttribute(
      'aria-expanded',
      'false',
    )
  })
})
