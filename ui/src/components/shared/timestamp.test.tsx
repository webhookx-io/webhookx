import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterAll, describe, expect, it, vi } from 'vitest'
import { Timestamp } from '@/components/shared/timestamp'
import { TooltipProvider } from '@/components/ui/tooltip'

vi.stubGlobal(
  'ResizeObserver',
  class ResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  },
)

afterAll(() => vi.unstubAllGlobals())

describe('Timestamp', () => {
  it('delays the precise local and UTC values until the timestamp is hovered', async () => {
    const user = userEvent.setup()
    const value = Date.UTC(2026, 7, 7, 7, 56, 17)
    const display = new Intl.DateTimeFormat(undefined, {
      dateStyle: 'medium',
      timeStyle: 'short',
      timeZone: 'UTC',
    }).format(value)

    render(
      <TooltipProvider>
        <Timestamp
          value={value}
          options={{ dateStyle: 'medium', timeStyle: 'short', timeZone: 'UTC' }}
        />
      </TooltipProvider>,
    )

    const trigger = screen.getByText(display)
    expect(trigger).toHaveAttribute('datetime', '2026-08-07T07:56:17.000Z')

    await user.hover(trigger)
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()

    const tooltip = await screen.findByRole('tooltip')
    expect(tooltip).toHaveTextContent('Time conversion')
    expect(tooltip).toHaveTextContent('Your computer')
    expect(tooltip).toHaveTextContent('UTC')
    expect(tooltip.querySelector('svg')).not.toBeInTheDocument()
  })

  it('renders a fallback without an interactive tooltip for invalid values', () => {
    render(<Timestamp value="not-a-timestamp" />)
    expect(screen.getByText('—').tagName).toBe('SPAN')
  })
})
