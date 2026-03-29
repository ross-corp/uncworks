// web/src/views/__tests__/RunListView.test.tsx
// Tests for RunListView — run rows, loading state, empty state
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { renderWithRouter } from '../../test-utils'
import type { AgentRun } from '../../types/agent-run'

// --- module mocks (must be at top level before any imports of the mocked modules) ---

vi.mock('../../hooks/apiFetch', () => ({
  apiFetch: vi.fn(),
  apiWsUrl: vi.fn(),
  apiSseUrl: vi.fn(),
}))

vi.mock('../../hooks/useClient', () => ({
  useClient: vi.fn(),
  mapRun: vi.fn((r: AgentRun) => r),
  ClientContext: { Provider: ({ children }: { children: React.ReactNode }) => children },
}))

vi.mock('sonner', () => ({
  toast: { error: vi.fn(), success: vi.fn(), loading: vi.fn() },
  Toaster: () => null,
}))

vi.mock('../../lib/format', () => ({
  formatAge: vi.fn(() => '1m'),
  aggregatePhase: vi.fn(() => 'succeeded'),
}))

vi.mock('../../components/RunStatusBadge', () => ({
  default: ({ phase }: { phase: string }) => <span data-testid="run-status">{phase}</span>,
}))

// Import after mocks are registered
import { apiFetch } from '../../hooks/apiFetch'
import { useClient } from '../../hooks/useClient'
import RunListView from '../RunListView'

function makeRun(id: string, displayName: string): AgentRun {
  return {
    id,
    name: id,
    spec: {
      backend: 'pod',
      repos: [],
      prompt: 'test',
      devboxConfig: '',
      ttlSeconds: 3600,
      envVars: {},
      modelTier: 'default',
      displayName,
    },
    status: {
      phase: 'succeeded',
      message: '',
      podName: '',
      traceID: '',
      startedAt: '',
      completedAt: '',
    },
    createdAt: new Date().toISOString(),
  }
}

describe('RunListView', () => {
  beforeEach(() => {
    // Mock apiFetch for chain runs endpoint — return empty array
    vi.mocked(apiFetch).mockResolvedValue(
      new Response(JSON.stringify([]), { status: 200, headers: { 'Content-Type': 'application/json' } })
    )

    // Mock useClient to return a client stub with listAgentRuns
    vi.mocked(useClient).mockReturnValue({
      listAgentRuns: vi.fn().mockResolvedValue([]),
    } as never)
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it('shows loading state initially', () => {
    // listAgentRuns never resolves during this test
    vi.mocked(useClient).mockReturnValue({
      listAgentRuns: vi.fn().mockReturnValue(new Promise(() => {})),
    } as never)

    renderWithRouter(<RunListView />)

    expect(screen.getByText('Loading...')).toBeInTheDocument()
  })

  it('shows empty state when no runs returned', async () => {
    renderWithRouter(<RunListView />)

    await waitFor(() => {
      expect(screen.getByText(/No runs yet/i)).toBeInTheDocument()
    })
  })

  it('renders run rows when API returns runs', async () => {
    const runs = [makeRun('run-1', 'My First Run'), makeRun('run-2', 'My Second Run')]

    vi.mocked(useClient).mockReturnValue({
      listAgentRuns: vi.fn().mockResolvedValue(runs),
    } as never)

    renderWithRouter(<RunListView />)

    await waitFor(() => {
      expect(screen.getByText('My First Run')).toBeInTheDocument()
      expect(screen.getByText('My Second Run')).toBeInTheDocument()
    })
  })
})
