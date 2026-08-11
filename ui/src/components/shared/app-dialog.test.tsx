import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { AppDialog } from '@/components/shared/app-dialog'

describe('AppDialog', () => {
  it('provides an accessible title and delegates closing to the shadcn dialog', async () => {
    const onClose = vi.fn()
    const user = userEvent.setup()
    render(
      <AppDialog
        open
        onClose={onClose}
        title="Create endpoint"
        description="Configure a delivery destination."
      >
        <p>Dialog body</p>
      </AppDialog>,
    )

    expect(screen.getByRole('dialog', { name: 'Create endpoint' })).toBeInTheDocument()
    expect(screen.getByText('Dialog body')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Close' }))
    expect(onClose).toHaveBeenCalledOnce()
  })
})
