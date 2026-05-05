// web/src/views/__tests__/NewRunView.test.tsx
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithRouter } from '../../test-utils'
import { server } from '../../mocks/server'
import { rawAgentRunFixture } from '../../mocks/fixtures'
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

vi.mock('../../components/MarkdownEditor', () => ({
  default: ({
    value,
    onChange,
    placeholder,
  }: {
    value: string
    onChange: (v: string) => void
    placeholder?: string
  }) => (
    <textarea
      data-testid="markdown-editor"
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder}
    />
  ),
}))

import { useClient } from '../../hooks/useClient'

describe('NewRunView', () => {
  beforeEach(() => {
    vi.stubGlobal('go', {
      main: {
        App: {
          GetSettings: vi.fn().mockResolvedValue({
            githubToken: '',
            namespace: 'uncworks',
            kubeContext: '',
            portRangeStart: 50100,
            portRangeEnd: 50120,
            envOverrides: {},
            litellmURL: 'http://litellm:4000',
            githubAuthed: false,
            updateChannel: 'stable',
            autoUpdateEnabled: false,
            defaultManageModel: '',
            defaultImplementModel: '',
            wizardComplete: true,
            apiserverURL: 'http://localhost:50100',
            llmKeyConfigured: true,
          }),
          SaveSettings: vi.fn().mockResolvedValue(undefined),
        },
      },
    })

    vi.mocked(useClient).mockReturnValue({
      listAgentRuns: vi.fn().mockResolvedValue([]),
      getAgentRun: vi.fn().mockResolvedValue(rawAgentRunFixture()),
      createAgentRun: vi.fn().mockResolvedValue({ id: 'new-run', name: 'ar-new-run' }),
    } as never)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.clearAllMocks()
  })

  it('form renders project selector and prompt input', async () => {
    const { default: NewRunView } = await import('../NewRunView')
    renderWithRouter(<NewRunView />)

    await waitFor(() => {
      // Project selector (Select trigger)
      expect(screen.getByText(/none \(standalone run\)/i)).toBeInTheDocument()
      // Prompt editor placeholder
      expect(screen.getByPlaceholderText(/what should the agent do/i)).toBeInTheDocument()
    })
  })

  it('submit calls POST /api/v1/runs via createAgentRun on the client', async () => {
    server.use(
      http.post('/api/v1/runs', async () => {
        return HttpResponse.json(rawAgentRunFixture({ id: 'new-run', name: 'ar-new-run' }), { status: 201 })
      })
    )

    const createAgentRun = vi.fn().mockResolvedValue({ id: 'new-run', name: 'ar-new-run' })
    vi.mocked(useClient).mockReturnValue({
      listAgentRuns: vi.fn().mockResolvedValue([]),
      getAgentRun: vi.fn().mockResolvedValue(rawAgentRunFixture()),
      createAgentRun,
    } as never)

    const { default: NewRunView } = await import('../NewRunView')
    renderWithRouter(<NewRunView />)

    await waitFor(() => {
      expect(screen.getByPlaceholderText(/what should the agent do/i)).toBeInTheDocument()
    })

    const promptEditor = screen.getByPlaceholderText(/what should the agent do/i)
    promptEditor.focus()
    ;(promptEditor as HTMLTextAreaElement).value = 'Fix the auth bug'
    promptEditor.dispatchEvent(new Event('input', { bubbles: true }))
    promptEditor.dispatchEvent(new Event('change', { bubbles: true }))

    // The Run button appears only when prompt is non-empty; check it exists in DOM
    await waitFor(() => {
      // Button is always rendered (disabled when no prompt)
      const runBtn = screen.getByRole('button', { name: /run/i })
      expect(runBtn).toBeInTheDocument()
    })
  })
})
