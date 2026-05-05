// web/src/views/__tests__/Layout.test.tsx
import { describe, it, expect, vi, afterEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { renderWithRouter } from '../../test-utils'

vi.mock('sonner', () => ({
  toast: { error: vi.fn(), success: vi.fn(), loading: vi.fn() },
  Toaster: () => null,
}))

vi.mock('../../lib/wails-env', () => ({
  isWails: () => false,
}))

vi.mock('../../hooks/useSettings', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../hooks/useSettings')>()
  return {
    ...actual,
    useSettings: vi.fn(() => ({
      settings: actual.SETTINGS_DEFAULTS,
      configStatus: {
        hasLLMKey: true,
        hasGitHubToken: false,
        hasGitHubOAuth: false,
        wizardComplete: true,
        canUseAI: true,
        canAccessPrivateRepos: false,
        canCreatePRs: false,
      },
      loading: false,
      error: null,
      reload: vi.fn().mockResolvedValue(undefined),
      save: vi.fn().mockResolvedValue(undefined),
    })),
    SettingsProvider: ({ children }: { children: React.ReactNode }) => children,
  }
})

vi.mock('../../hooks/useHealthContext', () => ({
  HealthProvider: ({ children }: { children: React.ReactNode }) => children,
  useHealthContext: vi.fn(() => ({ healthy: true })),
}))

vi.mock('../../components/CopilotBottomPanel', () => ({
  default: () => null,
}))

vi.mock('../../components/SetupWizard', () => ({
  default: () => null,
}))

vi.mock('../../components/ErrorBoundary', () => ({
  default: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}))

import Layout from '../Layout'

describe('Layout', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('navigation renders with Projects link visible', async () => {
    renderWithRouter(<Layout />, {
      routerProps: { initialEntries: ['/projects'] },
    })

    await waitFor(() => {
      expect(screen.getByText('Projects')).toBeInTheDocument()
    })
  })

  it('Runs nav link is visible', async () => {
    renderWithRouter(<Layout />, {
      routerProps: { initialEntries: ['/'] },
    })

    await waitFor(() => {
      expect(screen.getByText('Runs')).toBeInTheDocument()
    })
  })
})
