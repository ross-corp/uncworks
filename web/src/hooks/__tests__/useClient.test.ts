// web/src/hooks/__tests__/useClient.test.ts
// Unit tests for useClient — context provision, mapRun, and mapEvent utilities.
import { describe, it, expect, vi, afterEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import { createElement } from 'react'
import { ClientContext, useClient, mapRun, mapEvent } from '../useClient'
import { AOTClient } from '../../../../packages/shared/src/grpc/client'
import type { AgentRun as SharedAgentRun, AgentRunEvent as SharedEvent, AgentRunPhase } from '../../../../packages/shared/src/types/agent-run'

afterEach(() => {
  vi.resetModules()
})

describe('useClient', () => {
  it('returns the default client when no provider is present', () => {
    const { result } = renderHook(() => useClient())
    expect(result.current).toBeInstanceOf(AOTClient)
  })

  it('returns the client provided via ClientContext', () => {
    const customClient = new AOTClient({ baseUrl: 'http://custom:9999' })
    const wrapper = ({ children }: { children: React.ReactNode }) =>
      createElement(ClientContext.Provider, { value: customClient }, children)

    const { result } = renderHook(() => useClient(), { wrapper })
    expect(result.current).toBe(customClient)
  })
})

describe('mapRun', () => {
  const baseSharedRun: SharedAgentRun = {
    id: 'run-123',
    name: 'test-run',
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:01Z',
    spec: {
      backend: 'pod',
      repos: [{ url: 'https://github.com/foo/bar', branch: 'main', path: '' }],
      workspaceName: 'ws',
      prompt: 'do something',
      devboxConfig: '',
      ttlSeconds: 3600,
      envVars: {},
      modelTier: 'default',
    },
    status: {
      phase: 'Running',
      message: 'in progress',
      podName: 'pod-abc',
      traceID: 'trace-xyz',
      startedAt: '2024-01-01T00:00:01Z',
      completedAt: '',
      logOutput: '',
      debugActive: false,
    },
  }

  it('maps known phase strings correctly', () => {
    const run = mapRun({ ...baseSharedRun, status: { ...baseSharedRun.status, phase: 'Succeeded' } })
    expect(run.status.phase).toBe('succeeded')
  })

  it('falls back to "pending" for unknown phase', () => {
    // Cast through unknown to simulate server sending an unrecognized phase value
    const run = mapRun({ ...baseSharedRun, status: { ...baseSharedRun.status, phase: 'Unknown' as unknown as AgentRunPhase } })
    expect(run.status.phase).toBe('pending')
  })

  it('maps all known phases', () => {
    const phases: Array<[AgentRunPhase, string]> = [
      ['Pending', 'pending'],
      ['Running', 'running'],
      ['WaitingForInput', 'waiting_for_input'],
      ['Succeeded', 'succeeded'],
      ['Failed', 'failed'],
      ['Cancelled', 'cancelled'],
    ]
    for (const [input, expected] of phases) {
      const run = mapRun({ ...baseSharedRun, status: { ...baseSharedRun.status, phase: input } })
      expect(run.status.phase).toBe(expected)
    }
  })

  it('preserves id and name', () => {
    const run = mapRun(baseSharedRun)
    expect(run.id).toBe('run-123')
    expect(run.name).toBe('test-run')
  })

  it('maps repos with defaults for missing fields', () => {
    const run = mapRun(baseSharedRun)
    expect(run.spec.repos[0].url).toBe('https://github.com/foo/bar')
    expect(run.spec.repos[0].branch).toBe('main')
  })

  it('defaults debugActive to false when not provided', () => {
    const run = mapRun(baseSharedRun)
    expect(run.status.debugActive).toBe(false)
  })

  it('defaults ttlSeconds to 3600 when missing', () => {
    const noTtl = { ...baseSharedRun, spec: { ...baseSharedRun.spec, ttlSeconds: undefined as unknown as number } }
    const run = mapRun(noTtl)
    expect(run.spec.ttlSeconds).toBe(3600)
  })
})

describe('mapEvent', () => {
  const baseSharedEvent: SharedEvent = {
    agentRunId: 'run-123',
    type: 'log',
    payload: 'some log line',
    timestamp: '2024-01-01T00:00:05Z',
  }

  it('passes through all fields unchanged', () => {
    const event = mapEvent(baseSharedEvent)
    expect(event.agentRunId).toBe('run-123')
    expect(event.type).toBe('log')
    expect(event.payload).toBe('some log line')
    expect(event.timestamp).toBe('2024-01-01T00:00:05Z')
  })
})
