// web/src/views/__tests__/ScheduleListView.test.tsx
import { describe, it, expect, vi, afterEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithRouter } from '../../test-utils'
import { server } from '../../mocks/server'

vi.mock('sonner', () => ({
  toast: { error: vi.fn(), success: vi.fn(), loading: vi.fn() },
  Toaster: () => null,
}))

import ScheduleListView from '../ScheduleListView'

// The view expects Kubernetes-style { metadata, spec, status } shape.
const scheduleKubeFixture = {
  metadata: { name: 'my-schedule', creationTimestamp: '2026-01-01T00:00:00Z' },
  spec: { displayName: 'My Schedule', cron: '0 * * * *', suspend: false, chainRef: 'my-chain' },
  status: { lastResult: undefined, lastRunId: undefined, nextScheduleTime: undefined },
}

describe('ScheduleListView', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('renders schedule list from MSW', async () => {
    server.use(
      http.get('/api/v1/schedules', () => HttpResponse.json([scheduleKubeFixture]))
    )

    renderWithRouter(<ScheduleListView />, {
      routerProps: { initialEntries: ['/schedules'] },
    })

    await waitFor(() => {
      expect(screen.getByText('My Schedule')).toBeInTheDocument()
    })
  })
})
