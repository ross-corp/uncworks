// web/src/hooks/__tests__/useSettings.test.ts
// Tests for useSettings context — localStorage load/save round-trip and deriveConfigStatus.
import { describe, it, expect, beforeEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { ReactNode } from 'react'
import { SettingsProvider, useSettings, deriveConfigStatus, SETTINGS_DEFAULTS } from '../useSettings'

function wrapper({ children }: { children: ReactNode }) {
  return SettingsProvider({ children })
}

describe('deriveConfigStatus', () => {
  it('hasLLMKey is true for cluster LiteLLM URL', () => {
    const status = deriveConfigStatus({ ...SETTINGS_DEFAULTS, litellmURL: 'http://litellm:4000' })
    expect(status.hasLLMKey).toBe(true)
    expect(status.canUseAI).toBe(true)
  })

  it('hasLLMKey is true when llmApiKey is set', () => {
    const status = deriveConfigStatus({ ...SETTINGS_DEFAULTS, litellmURL: 'https://external.llm', llmApiKey: 'sk-test' })
    expect(status.hasLLMKey).toBe(true)
  })

  it('hasLLMKey is false for external URL without key', () => {
    const status = deriveConfigStatus({ ...SETTINGS_DEFAULTS, litellmURL: 'https://openrouter.ai', llmApiKey: '' })
    expect(status.hasLLMKey).toBe(false)
    expect(status.canUseAI).toBe(false)
  })

  it('hasGitHubToken is true when token set', () => {
    const status = deriveConfigStatus({ ...SETTINGS_DEFAULTS, githubToken: 'ghp_abc123' })
    expect(status.hasGitHubToken).toBe(true)
    expect(status.canAccessPrivateRepos).toBe(true)
  })

  it('hasGitHubToken is false when token empty', () => {
    const status = deriveConfigStatus({ ...SETTINGS_DEFAULTS, githubToken: '' })
    expect(status.hasGitHubToken).toBe(false)
  })
})

describe('useSettings (localStorage mode)', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('loads defaults when localStorage is empty', async () => {
    const { result } = renderHook(() => useSettings(), { wrapper })
    // Wait for the loading state to resolve
    await act(async () => {})
    expect(result.current.settings.namespace).toBe('uncworks')
    expect(result.current.loading).toBe(false)
  })

  it('saves settings to localStorage', async () => {
    const { result } = renderHook(() => useSettings(), { wrapper })
    await act(async () => {})

    await act(async () => {
      await result.current.save({ ...SETTINGS_DEFAULTS, githubToken: 'ghp_saved' })
    })

    expect(result.current.settings.githubToken).toBe('ghp_saved')
    // Verify persisted
    const raw = localStorage.getItem('uncworks-settings')
    expect(raw).toBeTruthy()
    expect(JSON.parse(raw!).githubToken).toBe('ghp_saved')
  })

  it('loads previously saved settings on mount', async () => {
    localStorage.setItem('uncworks-settings', JSON.stringify({ ...SETTINGS_DEFAULTS, githubToken: 'ghp_existing' }))

    const { result } = renderHook(() => useSettings(), { wrapper })
    await act(async () => {})

    expect(result.current.settings.githubToken).toBe('ghp_existing')
  })
})
