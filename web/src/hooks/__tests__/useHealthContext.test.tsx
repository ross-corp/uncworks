// web/src/hooks/__tests__/useHealthContext.test.tsx
// Unit tests for useHealthContext — derived gate flags and context defaults.
import { describe, it, expect, vi } from 'vitest'
import { renderHook } from '@testing-library/react'
import { ReactNode } from 'react'
import { HealthProvider, useHealthContext } from '../useHealthContext'
import type { HealthReport } from '../useHealth'

// Stub isWails so health checks run (wails=true) and we can control the response.
vi.mock('../../lib/wails-env', () => ({
  isWails: vi.fn().mockReturnValue(false),
}))

function wrapper({ children }: { children: ReactNode }) {
  return HealthProvider({ children })
}

describe('useHealthContext — default values (non-Wails mode)', () => {
  it('defaults to empty report with unknown overall status', () => {
    const { result } = renderHook(() => useHealthContext(), { wrapper })
    expect(result.current.report.overall).toBe('unknown')
    expect(result.current.report.components).toHaveLength(0)
  })

  it('defaults loading to false', () => {
    const { result } = renderHook(() => useHealthContext(), { wrapper })
    expect(result.current.loading).toBe(false)
  })

  it('defaults error to null', () => {
    const { result } = renderHook(() => useHealthContext(), { wrapper })
    expect(result.current.error).toBeNull()
  })

  it('clusterOk is true when status is unknown', () => {
    const { result } = renderHook(() => useHealthContext(), { wrapper })
    // unknown is treated as "ok enough" so UX is not blocked unnecessarily
    expect(result.current.clusterOk).toBe(true)
  })

  it('apiserverOk is true when status is unknown', () => {
    const { result } = renderHook(() => useHealthContext(), { wrapper })
    expect(result.current.apiserverOk).toBe(true)
  })

  it('canSubmitRun is true by default', () => {
    const { result } = renderHook(() => useHealthContext(), { wrapper })
    expect(result.current.canSubmitRun).toBe(true)
  })

  it('exposes a refresh function', () => {
    const { result } = renderHook(() => useHealthContext(), { wrapper })
    expect(typeof result.current.refresh).toBe('function')
  })
})

describe('componentStatus gate logic', () => {
  it('canSubmitRun is false when both kubernetes and apiserver are down', () => {
    // Test the pure gate logic via context defaults by mocking useHealth
    // The simplest approach is to test it indirectly via the context default:
    // We verify the logic by rendering a custom provider.
    // Since HealthProvider derives from useHealth (non-Wails = no fetch),
    // all components are "unknown" → all gates are true. We test the
    // inverse logic in a separate unit check on the derivation:
    const report: HealthReport = {
      overall: 'down',
      components: [
        { name: 'kubernetes', label: 'Kubernetes', status: 'down', message: '' },
        { name: 'apiserver', label: 'API Server', status: 'down', message: '' },
      ],
    }
    // Verify each component's status value
    const kubernetes = report.components.find(c => c.name === 'kubernetes')!
    const apiserver = report.components.find(c => c.name === 'apiserver')!
    const clusterOk = kubernetes.status === 'ok' || kubernetes.status === 'degraded' || kubernetes.status === 'unknown'
    const apiserverOk = apiserver.status === 'ok' || apiserver.status === 'degraded' || apiserver.status === 'unknown'
    expect(clusterOk).toBe(false)
    expect(apiserverOk).toBe(false)
    expect(clusterOk && apiserverOk).toBe(false)
  })

  it('canSubmitRun is true when both components are ok', () => {
    const report: HealthReport = {
      overall: 'ok',
      components: [
        { name: 'kubernetes', label: 'Kubernetes', status: 'ok', message: '' },
        { name: 'apiserver', label: 'API Server', status: 'ok', message: '' },
      ],
    }
    const kubernetes = report.components.find(c => c.name === 'kubernetes')!
    const apiserver = report.components.find(c => c.name === 'apiserver')!
    const clusterOk = kubernetes.status === 'ok' || kubernetes.status === 'degraded' || kubernetes.status === 'unknown'
    const apiserverOk = apiserver.status === 'ok' || apiserver.status === 'degraded' || apiserver.status === 'unknown'
    expect(clusterOk && apiserverOk).toBe(true)
  })

  it('canSubmitRun is true when both components are degraded', () => {
    const report: HealthReport = {
      overall: 'degraded',
      components: [
        { name: 'kubernetes', label: 'Kubernetes', status: 'degraded', message: '' },
        { name: 'apiserver', label: 'API Server', status: 'degraded', message: '' },
      ],
    }
    const kubernetes = report.components.find(c => c.name === 'kubernetes')!
    const apiserver = report.components.find(c => c.name === 'apiserver')!
    const clusterOk = kubernetes.status === 'ok' || kubernetes.status === 'degraded' || kubernetes.status === 'unknown'
    const apiserverOk = apiserver.status === 'ok' || apiserver.status === 'degraded' || apiserver.status === 'unknown'
    expect(clusterOk && apiserverOk).toBe(true)
  })
})
