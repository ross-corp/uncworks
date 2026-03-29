// web/src/views/__tests__/SettingsView.test.tsx
// Tests for SettingsView — field changes, save button, GitHub token visibility.
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { screen, waitFor, fireEvent } from '@testing-library/react'
import { renderWithRouter } from '../../test-utils'
import { SETTINGS_DEFAULTS } from '../../hooks/useSettings'

// Mock wails-env so isWails() returns false (web mode — no Wails bindings)
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
  const settings = { ...SETTINGS_DEFAULTS, ...overrides }
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
  })

  it('renders the settings heading', () => {
    renderWithRouter(<SettingsView />)
    expect(screen.getByText('Settings')).toBeInTheDocument()
  })

  it('shows save button as disabled when no changes made', () => {
    renderWithRouter(<SettingsView />)
    const saveBtn = screen.getByRole('button', { name: /save/i })
    expect(saveBtn).toBeDisabled()
  })

  it('enables save button after field change', async () => {
    renderWithRouter(<SettingsView />)

    const namespaceInput = screen.getByDisplayValue('uncworks')
    fireEvent.change(namespaceInput, { target: { value: 'my-namespace' } })

    await waitFor(() => {
      const saveBtn = screen.getByRole('button', { name: /save/i })
      expect(saveBtn).not.toBeDisabled()
    })
  })

  it('calls save on the settings context when Save is clicked', async () => {
    const save = vi.fn().mockResolvedValue(undefined)
    const reload = vi.fn().mockResolvedValue(undefined)
    vi.mocked(useSettings).mockReturnValue({
      settings: SETTINGS_DEFAULTS,
      configStatus: {
        hasLLMKey: true, hasGitHubToken: false, hasGitHubOAuth: false,
        wizardComplete: false, canUseAI: true, canAccessPrivateRepos: false, canCreatePRs: false,
      },
      loading: false,
      error: null,
      reload,
      save,
    })

    renderWithRouter(<SettingsView />)

    const namespaceInput = screen.getByDisplayValue('uncworks')
    fireEvent.change(namespaceInput, { target: { value: 'test-ns' } })

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /save/i })).not.toBeDisabled()
    })

    fireEvent.click(screen.getByRole('button', { name: /save/i }))

    await waitFor(() => {
      expect(save).toHaveBeenCalledWith(expect.objectContaining({ namespace: 'test-ns' }))
    })
  })

  it('shows GitHub token field', () => {
    renderWithRouter(<SettingsView />)
    // The GitHub token field label should be visible
    expect(screen.getByText(/github token/i)).toBeInTheDocument()
  })
})
