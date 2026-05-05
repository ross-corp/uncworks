// web/src/views/__tests__/SettingsView.test.tsx
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen, waitFor, fireEvent } from '@testing-library/react'
import { renderWithRouter } from '../../test-utils'
import { SETTINGS_DEFAULTS } from '../../hooks/useSettings'
import { appSettingsFixture } from '../../mocks/fixtures'

vi.mock('../../lib/wails-env', () => ({
  isWails: () => false,
}))

vi.mock('../../hooks/useSettings', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../hooks/useSettings')>()
  return {
    ...actual,
    useSettings: vi.fn(),
    SettingsProvider: actual.SettingsProvider,
  }
})

vi.mock('../../hooks/useThemeNew', () => ({
  useThemeNew: vi.fn(() => ({
    mode: 'system',
    setMode: vi.fn(),
    toggleMode: vi.fn(),
    resolvedTheme: 'light',
  })),
}))

vi.mock('../../components/SetupWizard', () => ({
  default: () => null,
  GitHubAuthModal: () => null,
}))

import { useSettings } from '../../hooks/useSettings'
import SettingsView from '../SettingsView'

function mockSettings(overrides = {}) {
  const settings = { ...appSettingsFixture(), ...overrides }
  vi.mocked(useSettings).mockReturnValue({
    settings,
    configStatus: {
      hasLLMKey: true,
      hasGitHubToken: Boolean(settings.githubToken),
      hasGitHubOAuth: false,
      wizardComplete: settings.wizardComplete,
      canUseAI: true,
      canAccessPrivateRepos: Boolean(settings.githubToken),
      canCreatePRs: Boolean(settings.githubToken),
    },
    loading: false,
    error: null,
    reload: vi.fn().mockResolvedValue(undefined),
    save: vi.fn().mockResolvedValue(undefined),
  })
}

describe('SettingsView', () => {
  beforeEach(() => {
    mockSettings()
    localStorage.clear()
    vi.stubGlobal('runtime', { EventsOn: vi.fn() })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.clearAllMocks()
  })

  it('namespace field is present', () => {
    renderWithRouter(<SettingsView />)
    expect(screen.getByDisplayValue('uncworks')).toBeInTheDocument()
  })

  it('save button enabled after changing namespace', async () => {
    renderWithRouter(<SettingsView />)

    const namespaceInput = screen.getByDisplayValue('uncworks')
    fireEvent.change(namespaceInput, { target: { value: 'new-namespace' } })

    await waitFor(() => {
      const saveBtn = screen.getByRole('button', { name: /^save$/i })
      expect(saveBtn).not.toBeDisabled()
    })
  })

  it('save function called with correct namespace on save', async () => {
    const save = vi.fn().mockResolvedValue(undefined)
    const reload = vi.fn().mockResolvedValue(undefined)
    vi.mocked(useSettings).mockReturnValue({
      settings: { ...SETTINGS_DEFAULTS, namespace: 'uncworks' },
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
      reload,
      save,
    })

    renderWithRouter(<SettingsView />)

    const namespaceInput = screen.getByDisplayValue('uncworks')
    fireEvent.change(namespaceInput, { target: { value: 'my-namespace' } })

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /^save$/i })).not.toBeDisabled()
    })

    fireEvent.click(screen.getByRole('button', { name: /^save$/i }))

    await waitFor(() => {
      expect(save).toHaveBeenCalledWith(expect.objectContaining({ namespace: 'my-namespace' }))
    })
  })
})
