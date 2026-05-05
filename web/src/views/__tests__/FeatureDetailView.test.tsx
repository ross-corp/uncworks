// web/src/views/__tests__/FeatureDetailView.test.tsx
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { renderWithRouter } from '../../test-utils'
import { agentRunFixture } from '../../mocks/fixtures'

vi.mock('sonner', () => ({
  toast: { error: vi.fn(), success: vi.fn(), loading: vi.fn() },
  Toaster: () => null,
}))

vi.mock('../../hooks/useClient', () => ({
  useClient: vi.fn(),
  mapRun: vi.fn((r: unknown) => r),
  ClientContext: { Provider: ({ children }: { children: React.ReactNode }) => children },
}))

import { useClient } from '../../hooks/useClient'
import FeatureDetailView from '../FeatureDetailView'

describe('FeatureDetailView', () => {
  beforeEach(() => {
    const run = agentRunFixture({ spec: { ...agentRunFixture().spec, feature: 'my-feature' } })
    vi.mocked(useClient).mockReturnValue({
      listAgentRuns: vi.fn().mockResolvedValue([run]),
      createAgentRun: vi.fn().mockResolvedValue({ id: 'new-run' }),
    } as never)
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it('renders feature detail from MSW', async () => {
    renderWithRouter(<FeatureDetailView />, {
      routerProps: { initialEntries: ['/feature/my-feature'] },
      routePath: '/feature/:name',
    })

    await waitFor(() => {
      expect(screen.getByText('my-feature')).toBeInTheDocument()
    })
  })
})
