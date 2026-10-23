import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter, useNavigate } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AdminStatus, AdminStatusRefresher } from '@/components/layout/admin-status'

function NavigationHarness() {
  const navigate = useNavigate()

  return (
    <>
      <button onClick={() => void navigate('/workspaces/default/sources')}>Sources</button>
      <button onClick={() => void navigate('/workspaces/default/overview')}>Overview</button>
    </>
  )
}

function renderStatus({
  initialEntry = '/workspaces/default/sources',
  refreshOnOverview = false,
}: { initialEntry?: string; refreshOnOverview?: boolean } = {}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[initialEntry]}>
        {refreshOnOverview && <AdminStatusRefresher />}
        <AdminStatus />
        {refreshOnOverview && <NavigationHarness />}
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('AdminStatus', () => {
  it('loads the Admin API version from the root endpoint', async () => {
    const fetchMock = vi
      .fn<typeof fetch>()
      .mockResolvedValue(new Response(JSON.stringify({ version: '1.2.3' }), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    renderStatus()

    expect(screen.getByText('WebhookX Admin')).toBeInTheDocument()
    expect(await screen.findByText('Version 1.2.3')).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledOnce()
    const [path] = fetchMock.mock.calls[0]
    expect(path).toBe('/api/')
  })

  it('shows a red status when the Admin API is unreachable', async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status: 503 }))
    vi.stubGlobal('fetch', fetchMock)

    const { container } = renderStatus()

    expect(await screen.findByText('Backend unreachable')).toBeInTheDocument()
    expect(container.querySelector('[aria-hidden="true"]')).toHaveClass('bg-red-500')
    expect(fetchMock).toHaveBeenCalledOnce()
  })

  it('requests the root endpoint each time the overview page is entered', async () => {
    const response = (version: string) => new Response(JSON.stringify({ version }), { status: 200 })
    const fetchMock = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(response('1.0.0'))
      .mockResolvedValueOnce(response('1.0.1'))
      .mockResolvedValueOnce(response('1.0.2'))
    vi.stubGlobal('fetch', fetchMock)

    renderStatus({ refreshOnOverview: true })

    expect(await screen.findByText('Version 1.0.0')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Overview' }))
    expect(await screen.findByText('Version 1.0.1')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Sources' }))
    expect(fetchMock).toHaveBeenCalledTimes(2)

    fireEvent.click(screen.getByRole('button', { name: 'Overview' }))
    expect(await screen.findByText('Version 1.0.2')).toBeInTheDocument()

    expect(fetchMock).toHaveBeenCalledTimes(3)
    expect(fetchMock.mock.calls.map(([path]) => path)).toEqual(['/api/', '/api/', '/api/'])
  })
})
