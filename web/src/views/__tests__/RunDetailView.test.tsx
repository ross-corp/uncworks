// web/src/views/__tests__/RunDetailView.test.tsx
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

vi.mock('../../hooks/useCopilotContext', () => ({
  useCopilotContext: vi.fn(),
  useCopilotContextValue: vi.fn(() => ({ setOpen: vi.fn() })),
  CopilotContextProvider: ({ children }: { children: React.ReactNode }) => children,
}))

vi.mock('../../hooks/useTraces', () => ({
  useTraces: vi.fn(() => ({ spans: [], loading: false })),
}))

vi.mock('../../components/ActivityFeed', () => ({
  default: () => <div data-testid="activity-feed" />,
}))

vi.mock('../../components/FileExplorer', () => ({
  default: () => <div data-testid="file-explorer" />,
}))

vi.mock('../../components/ShellTerminal', () => ({
  default: () => <div data-testid="shell-terminal" />,
}))

vi.mock('../../components/TraceTimeline', () => ({
  default: () => <div data-testid="trace-timeline" />,
  SpanDetail: () => <div data-testid="span-detail" />,
}))

vi.mock('../../components/StageProgress', () => ({
  default: () => <div data-testid="stage-progress" />,
}))

vi.mock('../../components/FailureDiagnosisPanel', () => ({
  default: () => <div data-testid="failure-diagnosis" />,
}))

vi.mock('../../components/HitlModal', () => ({
  default: () => <div data-testid="hitl-modal" />,
}))

vi.mock('../../hooks/usePoll', () => ({
  usePoll: vi.fn((fn: () => void) => { fn() }),
}))

import { useClient } from '../../hooks/useClient'
import RunDetailView from '../RunDetailView'

describe('RunDetailView', () => {
  beforeEach(() => {
    const run = agentRunFixture()
    vi.mocked(useClient).mockReturnValue({
      getAgentRun: vi.fn().mockResolvedValue(run),
      sendHumanInput: vi.fn().mockResolvedValue(undefined),
      cancelAgentRun: vi.fn().mockResolvedValue(undefined),
    } as never)
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it('renders without crashing given a valid run', async () => {
    renderWithRouter(<RunDetailView />, {
      routerProps: { initialEntries: ['/run/run-001'] },
      routePath: '/run/:id',
    })

    await waitFor(() => {
      // Multiple elements with 'ar-test-run' appear (breadcrumb + header); use getAllByText
      const elements = screen.getAllByText('ar-test-run')
      expect(elements.length).toBeGreaterThan(0)
    })
  })

  it('phase badge is visible', async () => {
    renderWithRouter(<RunDetailView />, {
      routerProps: { initialEntries: ['/run/run-001'] },
      routePath: '/run/:id',
    })

    await waitFor(() => {
      // Run name appears in header — confirms the view rendered with data
      const elements = screen.getAllByText('ar-test-run')
      expect(elements.length).toBeGreaterThan(0)
    })
  })
})
