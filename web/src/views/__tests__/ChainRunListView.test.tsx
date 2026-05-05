// web/src/views/__tests__/ChainRunListView.test.tsx
import { describe, it, expect, vi, afterEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithRouter } from '../../test-utils'
import { server } from '../../mocks/server'

vi.mock('sonner', () => ({
  toast: { error: vi.fn(), success: vi.fn(), loading: vi.fn() },
  Toaster: () => null,
}))

import ChainRunListView from '../ChainRunListView'

// ChainRunListView calls /api/v1/chainruns and accesses cr.metadata.name and cr.spec.chainRef
const chainRunKubeFixture = {
  metadata: { name: 'cr-001', creationTimestamp: '2026-01-01T00:00:00Z' },
  spec: { chainRef: 'my-chain' },
  status: { phase: 'Succeeded' },
}

describe('ChainRunListView', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('renders chain run list from MSW', async () => {
    server.use(
      http.get('/api/v1/chainruns', () => HttpResponse.json([chainRunKubeFixture]))
    )

    renderWithRouter(<ChainRunListView />)

    await waitFor(() => {
      expect(screen.getByText('cr-001')).toBeInTheDocument()
    })
  })
})
