import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { SourcesPage } from '@/pages/sources'
import { TooltipProvider } from '@/components/ui/tooltip'

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
  name: 'Storefront ingress',
  enabled: true,
  type: 'http',
  config: { http: { path: '/events/storefront', methods: ['POST'] } },
  async: false,
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

function renderSources() {
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
    if (path.includes('/api/workspaces/ws_1/sources?')) {
      return Promise.resolve(jsonResponse({ data: [source] }))
    }
    if (path === '/api/workspaces/ws_1/sources/src_1') {
      return Promise.resolve(jsonResponse(source))
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
        <MemoryRouter initialEntries={['/workspaces/default/sources']}>
          <Routes>
            <Route path="/workspaces/:workspaceName/sources/*" element={<SourcesPage />} />
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

describe('SourcesPage table and drawer interactions', () => {
  it('defaults new sources to POST in the accepted methods dropdown', async () => {
    renderSources()

    fireEvent.click(await screen.findByRole('button', { name: 'Create source' }))

    expect(await screen.findByRole('heading', { name: 'Create source' })).toBeInTheDocument()
    const methodsTrigger = screen.getByRole('button', { name: /Accepted methods.*POST/ })
    fireEvent.pointerDown(methodsTrigger)

    expect(await screen.findByRole('menuitemcheckbox', { name: 'POST' })).toHaveAttribute(
      'aria-checked',
      'true',
    )
  })

  it('opens the edit drawer from a table row with advanced configuration collapsed', async () => {
    renderSources()

    const row = await screen.findByLabelText('Edit Storefront ingress')
    fireEvent.click(row)

    expect(await screen.findByRole('heading', { name: 'Edit source' })).toBeInTheDocument()
    await screen.findByDisplayValue('Storefront ingress')
    expect(screen.queryByRole('heading', { name: 'Traffic controls' })).not.toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Metadata' })).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: /Advanced Configuration/ }))

    expect(screen.getByRole('heading', { name: 'Traffic controls' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Metadata' })).toBeInTheDocument()
  })

  it('keeps delete as the only row action', async () => {
    renderSources()

    await screen.findByLabelText('Edit Storefront ingress')
    fireEvent.pointerDown(screen.getByRole('button', { name: 'Actions for Storefront ingress' }))

    expect(await screen.findByRole('menuitem', { name: 'Delete' })).toBeInTheDocument()
    expect(screen.queryByRole('menuitem', { name: 'Edit' })).not.toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Edit source' })).not.toBeInTheDocument()
  })
})
