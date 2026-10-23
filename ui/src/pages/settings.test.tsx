import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { TooltipProvider } from '@/components/ui/tooltip'
import { SettingsPage } from '@/pages/settings'

const license = {
  id: '00000000-0000-0000-0000-000000000000',
  plan: 'free',
  customer: 'anonymous',
  expired_at: '2099-12-31T23:59:59Z',
  created_at: '1996-08-24T00:00:00Z',
  version: '1',
  signature: '',
}

const workspace = {
  id: 'ws_01TEST',
  name: 'default',
  description: 'Default production workspace',
  metadata: { environment: 'production', owner: 'platform' },
  created_at: 1_700_000_000_000,
  updated_at: 1_710_000_000_000,
}

function renderSettings(path = '/settings/license') {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })

  return render(
    <QueryClientProvider client={queryClient}>
      <TooltipProvider>
        <MemoryRouter initialEntries={[path]}>
          <SettingsPage />
        </MemoryRouter>
      </TooltipProvider>
    </QueryClientProvider>,
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('SettingsPage general section', () => {
  it('displays information for the current workspace', async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(
      new Response(JSON.stringify({ data: [workspace] }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    renderSettings('/settings')

    expect(
      await screen.findByRole('heading', { name: 'Workspace information' }),
    ).toBeInTheDocument()
    expect(screen.getByText(workspace.name)).toBeInTheDocument()
    expect(screen.getByText(workspace.id)).toBeInTheDocument()
    expect(screen.getByText(workspace.description)).toBeInTheDocument()
    expect(screen.getByText('environment')).toBeInTheDocument()
    expect(screen.getByText('production')).toBeInTheDocument()
    expect(screen.queryByText('Primary region')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Save changes' })).not.toBeInTheDocument()

    expect(fetchMock).toHaveBeenCalledOnce()
    expect(fetchMock.mock.calls[0][0]).toMatch(/^\/api\/workspaces\?/)
  })
})

describe('SettingsPage license section', () => {
  it('loads and displays the instance license', async () => {
    const fetchMock = vi
      .fn<typeof fetch>()
      .mockResolvedValue(new Response(JSON.stringify(license), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    renderSettings()

    expect(await screen.findByRole('heading', { name: 'License information' })).toBeInTheDocument()
    expect(screen.getByText('free plan')).toBeInTheDocument()
    expect(screen.getByText('anonymous')).toBeInTheDocument()
    expect(screen.getByText(license.id)).toBeInTheDocument()
    expect(screen.getByText('Not provided')).toBeInTheDocument()
    expect(screen.getByText('WEBHOOKX_LICENSE')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Request a license' })).toHaveAttribute(
      'href',
      'https://form.jotform.com/webhookx/request-a-license',
    )
    expect(screen.getByLabelText('Raw license JSON').textContent).toBe(
      JSON.stringify(license, null, 2),
    )
    expect(screen.getByRole('button', { name: 'Copy JSON' })).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledOnce()
    const [path, init] = fetchMock.mock.calls[0]
    expect(path).toBe('/api/license')
    expect(init?.headers).toMatchObject({ Accept: 'application/json' })
  })
})
