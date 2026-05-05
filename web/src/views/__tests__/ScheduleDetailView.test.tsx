// web/src/views/__tests__/ScheduleDetailView.test.tsx
import { describe, it, expect, vi, afterEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithRouter } from '../../test-utils'
import { server } from '../../mocks/server'

vi.mock('sonner', () => ({
  toast: { error: vi.fn(), success: vi.fn(), loading: vi.fn() },
  Toaster: () => null,
}))

import ScheduleDetailView from '../ScheduleDetailView'

// ScheduleDetailView expects { metadata, spec, status } shape.
const scheduleDetailFixture = {
  metadata: { name: 'my-schedule', creationTimestamp: '2026-01-01T00:00:00Z' },
  spec: { displayName: 'My Schedule', cron: '0 * * * *', suspend: false, chainRef: 'my-chain' },
  status: { lastResult: undefined, lastRunId: undefined, nextScheduleTime: undefined },
}

describe('ScheduleDetailView', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('renders schedule detail from MSW', async () => {
    server.use(
      http.get('/api/v1/schedules/:name', () => HttpResponse.json(scheduleDetailFixture)),
      http.get('/api/v1/runs', () => HttpResponse.json([]))
    )

    renderWithRouter(<ScheduleDetailView />, {
      routerProps: { initialEntries: ['/schedules/my-schedule'] },
      routePath: '/schedules/:name',
    })

    await waitFor(() => {
      expect(screen.getByText('My Schedule')).toBeInTheDocument()
      // Cron expression should be visible
      expect(screen.getByText('0 * * * *')).toBeInTheDocument()
    })
  })
})
