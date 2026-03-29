// web/src/hooks/__tests__/usePoll.test.ts
// Tests for usePoll — immediate invocation, interval repeat, cleanup on unmount
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { usePoll } from '../usePoll'

describe('usePoll', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('calls callback immediately on mount', async () => {
    const fn = vi.fn().mockResolvedValue(undefined)

    renderHook(() => usePoll(fn, 1000))

    // Flush the initial async invocation
    await act(async () => {
      await Promise.resolve()
    })

    expect(fn).toHaveBeenCalledTimes(1)
  })

  it('calls callback again after interval elapses', async () => {
    const fn = vi.fn().mockResolvedValue(undefined)

    renderHook(() => usePoll(fn, 1000))

    // Flush initial call
    await act(async () => {
      await Promise.resolve()
    })

    expect(fn).toHaveBeenCalledTimes(1)

    // Advance past the interval
    await act(async () => {
      vi.advanceTimersByTime(1000)
      await Promise.resolve()
    })

    expect(fn).toHaveBeenCalledTimes(2)
  })

  it('stops calling after unmount (cleanup)', async () => {
    const fn = vi.fn().mockResolvedValue(undefined)

    const { unmount } = renderHook(() => usePoll(fn, 1000))

    await act(async () => {
      await Promise.resolve()
    })

    expect(fn).toHaveBeenCalledTimes(1)

    unmount()

    // Advance well past the interval — callback must not fire again
    await act(async () => {
      vi.advanceTimersByTime(3000)
      await Promise.resolve()
    })

    expect(fn).toHaveBeenCalledTimes(1)
  })
})
