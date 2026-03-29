// web/src/hooks/__tests__/useThemeNew.test.ts
// Tests for useThemeNew — mode cycling and localStorage persistence.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useThemeNew } from '../useThemeNew'

describe('useThemeNew', () => {
  beforeEach(() => {
    localStorage.clear()
    document.documentElement.className = ''
    // Default system preference to light
    Object.defineProperty(window, 'matchMedia', {
      writable: true,
      value: vi.fn().mockImplementation((query: string) => ({
        matches: query.includes('dark') ? false : true,
        media: query,
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
      })),
    })
  })

  it('defaults to system mode when localStorage is empty', () => {
    const { result } = renderHook(() => useThemeNew())
    expect(result.current.mode).toBe('system')
  })

  it('reads initial mode from localStorage', () => {
    localStorage.setItem('aot-theme-mode', 'dark')
    const { result } = renderHook(() => useThemeNew())
    expect(result.current.mode).toBe('dark')
  })

  it('setMode updates mode and persists to localStorage', () => {
    const { result } = renderHook(() => useThemeNew())

    act(() => {
      result.current.setMode('dark')
    })

    expect(result.current.mode).toBe('dark')
    expect(localStorage.getItem('aot-theme-mode')).toBe('dark')
  })

  it('toggleMode flips from dark to light', () => {
    localStorage.setItem('aot-theme-mode', 'dark')
    const { result } = renderHook(() => useThemeNew())

    act(() => {
      result.current.toggleMode()
    })

    expect(result.current.mode).toBe('light')
    expect(localStorage.getItem('aot-theme-mode')).toBe('light')
  })

  it('toggleMode flips from light to dark', () => {
    localStorage.setItem('aot-theme-mode', 'light')
    const { result } = renderHook(() => useThemeNew())

    act(() => {
      result.current.toggleMode()
    })

    expect(result.current.mode).toBe('dark')
  })

  it('resolvedTheme is light or dark (never system)', () => {
    const { result } = renderHook(() => useThemeNew())
    expect(['light', 'dark']).toContain(result.current.resolvedTheme)
  })
})
