// web/src/views/__tests__/TemplateNewView.test.tsx
import { describe, it, expect, vi, afterEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { renderWithRouter } from '../../test-utils'

vi.mock('sonner', () => ({
  toast: { error: vi.fn(), success: vi.fn(), loading: vi.fn() },
  Toaster: () => null,
}))

import TemplateNewView from '../TemplateNewView'

describe('TemplateNewView', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('form renders with name input', async () => {
    renderWithRouter(<TemplateNewView />)

    await waitFor(() => {
      expect(screen.getByPlaceholderText('my-template')).toBeInTheDocument()
    })
  })

  it('project selector is present', async () => {
    renderWithRouter(<TemplateNewView />)

    await waitFor(() => {
      // CustomSelect renders "— none —" in both the summary and the list item;
      // use getAllByText and assert at least one appears
      const items = screen.getAllByText('— none —')
      expect(items.length).toBeGreaterThan(0)
    })
  })
})
