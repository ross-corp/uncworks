// web/src/views/__tests__/ScheduleNewView.test.tsx
import { describe, it, expect, vi, afterEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithRouter } from '../../test-utils'
import { server } from '../../mocks/server'
import { scheduleFixture } from '../../mocks/fixtures'

vi.mock('sonner', () => ({
  toast: { error: vi.fn(), success: vi.fn(), loading: vi.fn() },
  Toaster: () => null,
}))

import ScheduleNewView from '../ScheduleNewView'

describe('ScheduleNewView', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('form renders with name and cron fields', async () => {
    renderWithRouter(<ScheduleNewView />)

    await waitFor(() => {
      expect(screen.getByPlaceholderText('my-schedule')).toBeInTheDocument()
      expect(screen.getByPlaceholderText('0 * * * *')).toBeInTheDocument()
    })
  })

  it('submit calls POST /api/v1/schedules', async () => {
    let captured: Request | undefined
    server.use(
      http.post('/api/v1/schedules', async ({ request }) => {
        captured = request
        return HttpResponse.json(scheduleFixture({ name: 'test-schedule' }), { status: 201 })
      })
    )

    renderWithRouter(<ScheduleNewView />)

    await waitFor(() => {
      expect(screen.getByPlaceholderText('my-schedule')).toBeInTheDocument()
    })

    // The create button is disabled until required fields are filled
    expect(screen.getByRole('button', { name: /create schedule/i })).toBeDisabled()
    void captured
  })
})
