// web/src/views/__tests__/ChainListView.test.tsx
import { describe, it, expect, vi, afterEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithRouter } from '../../test-utils'
import { server } from '../../mocks/server'

vi.mock('sonner', () => ({
  toast: { error: vi.fn(), success: vi.fn(), loading: vi.fn() },
  Toaster: () => null,
}))

import ChainListView from '../ChainListView'

// ChainListView expects Kubernetes-style { metadata, spec } shape.
const chainKubeFixture = {
  metadata: { name: 'my-chain', creationTimestamp: '2026-01-01T00:00:00Z' },
  spec: {
    displayName: 'My Chain',
    steps: [{ name: 'step-1', templateRef: 'my-template' }],
  },
}

describe('ChainListView', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('renders chain list from MSW with chain name visible', async () => {
    server.use(
      http.get('/api/v1/chains', () => HttpResponse.json([chainKubeFixture]))
    )

    renderWithRouter(<ChainListView />, {
      routerProps: { initialEntries: ['/chains'] },
    })

    await waitFor(() => {
      expect(screen.getByText('My Chain')).toBeInTheDocument()
    })
  })
})
