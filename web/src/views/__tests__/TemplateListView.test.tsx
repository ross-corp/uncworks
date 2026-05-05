// web/src/views/__tests__/TemplateListView.test.tsx
import { describe, it, expect, vi, afterEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithRouter } from '../../test-utils'
import { server } from '../../mocks/server'

vi.mock('sonner', () => ({
  toast: { error: vi.fn(), success: vi.fn(), loading: vi.fn() },
  Toaster: () => null,
}))

import TemplateListView from '../TemplateListView'

// TemplateListView expects Kubernetes-style { metadata, spec } shape.
const templateKubeFixture = {
  metadata: { name: 'my-template', creationTimestamp: '2026-01-01T00:00:00Z' },
  spec: {
    displayName: 'My Template',
    projectRef: 'my-project',
    prompt: 'Run the tests',
  },
  status: { runCount: 0 },
}

describe('TemplateListView', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('renders template list from MSW', async () => {
    server.use(
      http.get('/api/v1/templates', () => HttpResponse.json([templateKubeFixture]))
    )

    renderWithRouter(<TemplateListView />, {
      routerProps: { initialEntries: ['/templates'] },
    })

    await waitFor(() => {
      expect(screen.getByText('My Template')).toBeInTheDocument()
    })
  })
})
