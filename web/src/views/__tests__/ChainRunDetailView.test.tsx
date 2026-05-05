// web/src/views/__tests__/ChainRunDetailView.test.tsx
import { describe, it, expect, vi, afterEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithRouter } from '../../test-utils'
import { server } from '../../mocks/server'

vi.mock('sonner', () => ({
  toast: { error: vi.fn(), success: vi.fn(), loading: vi.fn() },
  Toaster: () => null,
}))

vi.mock('../../components/ChainDagViz', () => ({
  default: () => <div data-testid="chain-dag-viz" />,
}))

vi.mock('../../components/ErrorBoundary', () => ({
  default: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}))

import ChainRunDetailView from '../ChainRunDetailView'

const chainRunDetailFixture = {
  metadata: { name: 'cr-001', creationTimestamp: '2026-01-01T00:00:00Z' },
  spec: { chainRef: 'my-chain', triggeredBy: undefined },
  status: { phase: 'Succeeded', steps: [], startedAt: '2026-01-01T00:00:00Z', completedAt: '2026-01-01T00:05:00Z' },
}

const chainDefFixture = {
  spec: { steps: [{ name: 'step-1', templateRef: 'my-template', dependsOn: [] }] },
}

describe('ChainRunDetailView', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('renders chain run detail from MSW', async () => {
    server.use(
      http.get('/api/v1/chainruns/:name', () => HttpResponse.json(chainRunDetailFixture)),
      http.get('/api/v1/chains/:name', () => HttpResponse.json(chainDefFixture))
    )

    renderWithRouter(<ChainRunDetailView />, {
      routerProps: { initialEntries: ['/chainrun/cr-001'] },
      routePath: '/chainrun/:name',
    })

    await waitFor(() => {
      expect(screen.getByText(/my-chain/i)).toBeInTheDocument()
    })
  })
})
