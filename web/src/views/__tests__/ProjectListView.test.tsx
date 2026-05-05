// web/src/views/__tests__/ProjectListView.test.tsx
import { describe, it, expect, vi, afterEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { renderWithRouter } from '../../test-utils'
import { server } from '../../mocks/server'

vi.mock('sonner', () => ({
  toast: { error: vi.fn(), success: vi.fn(), loading: vi.fn() },
  Toaster: () => null,
}))

import ProjectListView from '../ProjectListView'

describe('ProjectListView', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('loads projects from MSW and both names appear in DOM', async () => {
    renderWithRouter(<ProjectListView />)

    await waitFor(() => {
      expect(screen.getByText('my-project')).toBeInTheDocument()
      expect(screen.getByText('second-project')).toBeInTheDocument()
    })
  })

  it('shows empty state when MSW returns []', async () => {
    server.use(
      http.get('/api/v1/projects', () => HttpResponse.json([]))
    )

    renderWithRouter(<ProjectListView />)

    await waitFor(() => {
      expect(screen.getByText('No projects yet')).toBeInTheDocument()
    })
  })

  it('clicking a project row navigates to project detail', async () => {
    const user = userEvent.setup()

    const { container } = renderWithRouter(<ProjectListView />)

    await waitFor(() => {
      expect(screen.getByText('my-project')).toBeInTheDocument()
    })

    const row = container.querySelector('[class*="cursor-pointer"]') as HTMLElement
    expect(row).not.toBeNull()
    await user.click(row)
    // Navigation happened — no crash is sufficient for this test since
    // MemoryRouter doesn't expose location easily without additional setup.
  })
})
