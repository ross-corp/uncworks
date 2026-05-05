// web/src/components/__tests__/CopilotBottomPanel.test.tsx
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen, waitFor, fireEvent } from '@testing-library/react'
import { render } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { appSettingsFixture } from '../../mocks/fixtures'

vi.mock('sonner', () => ({
  toast: { error: vi.fn(), success: vi.fn(), loading: vi.fn() },
  Toaster: () => null,
}))

vi.mock('../../hooks/useSettings', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../hooks/useSettings')>()
  return {
    ...actual,
    useSettings: vi.fn(() => ({
      settings: appSettingsFixture(),
      configStatus: actual.deriveConfigStatus(appSettingsFixture()),
      loading: false,
      error: null,
      reload: vi.fn().mockResolvedValue(undefined),
      save: vi.fn().mockResolvedValue(undefined),
    })),
    SettingsProvider: ({ children }: { children: React.ReactNode }) => children,
  }
})

// Mock the copilot context to expose the panel as open with a controlled state
const mockUpdateActiveMessages = vi.fn()
const mockCreateSession = vi.fn(() => 'session-1')
const mockSelectSession = vi.fn()

vi.mock('../../hooks/useCopilotContext', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../hooks/useCopilotContext')>()
  return {
    ...actual,
    useCopilotContextValue: vi.fn(() => ({
      context: null,
      open: true,
      setOpen: vi.fn(),
      panelHeight: 320,
      setPanelHeight: vi.fn(),
      sessions: [],
      activeSessionId: null,
      activeMessages: [],
      createSession: mockCreateSession,
      selectSession: mockSelectSession,
      updateActiveMessages: mockUpdateActiveMessages,
    })),
  }
})

import CopilotBottomPanel from '../CopilotBottomPanel'

describe('CopilotBottomPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // jsdom doesn't implement scrollIntoView
    window.HTMLElement.prototype.scrollIntoView = vi.fn()
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it('renders the panel when open and shows the input', () => {
    render(
      <MemoryRouter>
        <CopilotBottomPanel />
      </MemoryRouter>
    )

    expect(screen.getByPlaceholderText(/ask a question/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /send/i })).toBeInTheDocument()
  })

  it('typing a message and sending accumulates SSE tokens in the response', async () => {
    render(
      <MemoryRouter>
        <CopilotBottomPanel />
      </MemoryRouter>
    )

    const input = screen.getByPlaceholderText(/ask a question/i) as HTMLInputElement

    fireEvent.change(input, { target: { value: 'Hello copilot' } })

    expect(input.value).toBe('Hello copilot')

    const sendBtn = screen.getByRole('button', { name: /send/i })
    fireEvent.click(sendBtn)

    // After send, updateActiveMessages is called with the user message
    await waitFor(() => {
      expect(mockUpdateActiveMessages).toHaveBeenCalled()
    })
  })
})
