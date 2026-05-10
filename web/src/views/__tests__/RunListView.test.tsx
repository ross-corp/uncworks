// web/src/views/__tests__/RunListView.test.tsx
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithRouter } from '../../test-utils'
import { server } from '../../mocks/server'
import type { AgentRun } from '../../types/agent-run'

vi.mock('sonner', () => ({
  toast: { error: vi.fn(), success: vi.fn(), loading: vi.fn() },
  Toaster: () => null,
}))

vi.mock('../../hooks/useClient', () => ({
  useClient: vi.fn(),
  mapRun: vi.fn((r: AgentRun) => r),
  ClientContext: { Provider: ({ children }: { children: React.ReactNode }) => children },
}))

import { useClient } from '../../hooks/useClient'
import RunListView from '../RunListView'

describe('RunListView', () => {
  beforeEach(() => {
    vi.mocked(useClient).mockReturnValue({
      listAgentRuns: vi.fn().mockResolvedValue([]),
    } as never)
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it('renders runs from MSW and shows run names', async () => {
    vi.mocked(useClient).mockReturnValue({
      listAgentRuns: vi.fn().mockResolvedValue([
        {
          id: 'run-001',
          name: 'ar-run-001',
          spec: { backend: 'pod', repos: [], prompt: 'test', devboxConfig: '', ttlSeconds: 3600, envVars: {}, modelTier: 'default', projectRef: 'my-project', displayName: 'First Run' },
          status: { phase: 'succeeded', message: '', podName: '', traceID: '', startedAt: '', completedAt: '' },
          createdAt: '2026-01-01T00:00:00Z',
          updatedAt: '2026-01-01T00:00:00Z',
        } as AgentRun,
      ]),
    } as never)

    renderWithRouter(<RunListView />)

    await waitFor(() => {
      expect(screen.getByText('First Run')).toBeInTheDocument()
    })
  })

  it('null response from chain runs endpoint does not crash', async () => {
    server.use(
      http.get('/api/v1/chainruns', () => HttpResponse.json(null))
    )

    renderWithRouter(<RunListView />)

    await waitFor(() => {
      expect(screen.getByText(/No runs yet/i)).toBeInTheDocument()
    })
  })

  it('filter CustomSelect exists in the DOM', async () => {
    renderWithRouter(<RunListView />)

    await waitFor(() => {
      // CustomSelect renders as a <details> element; the summary acts as the trigger
      const summaries = document.querySelectorAll('summary')
      expect(summaries.length).toBeGreaterThan(0)
    })
  })
})
