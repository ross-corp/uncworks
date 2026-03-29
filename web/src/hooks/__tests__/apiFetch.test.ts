// web/src/hooks/__tests__/apiFetch.test.ts
// Tests for the apiFetch fetch wrapper (URL prepending, options pass-through)
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

// apiFetch reads import.meta.env at module evaluation time, so we must
// stub env vars before importing the module. Re-import per test group via
// vi.resetModules() + dynamic import.

describe('apiFetch', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn())
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.unstubAllEnvs()
    vi.resetModules()
  })

  it('prepends VITE_API_URL when set', async () => {
    vi.stubEnv('VITE_API_URL', 'http://localhost:8080')
    const { apiFetch } = await import('../apiFetch')

    const mockFetch = vi.mocked(fetch)
    mockFetch.mockResolvedValueOnce(new Response('{}', { status: 200 }))

    await apiFetch('/api/v1/runs')

    expect(mockFetch).toHaveBeenCalledOnce()
    const [url] = mockFetch.mock.calls[0]
    expect(url).toBe('http://localhost:8080/api/v1/runs')
  })

  it('uses relative URL when VITE_API_URL is not set', async () => {
    vi.stubEnv('VITE_API_URL', '')
    const { apiFetch } = await import('../apiFetch')

    const mockFetch = vi.mocked(fetch)
    mockFetch.mockResolvedValueOnce(new Response('{}', { status: 200 }))

    await apiFetch('/api/v1/runs')

    expect(mockFetch).toHaveBeenCalledOnce()
    const [url] = mockFetch.mock.calls[0]
    expect(url).toBe('/api/v1/runs')
  })

  it('passes through fetch options (method, headers, body)', async () => {
    vi.stubEnv('VITE_API_URL', '')
    const { apiFetch } = await import('../apiFetch')

    const mockFetch = vi.mocked(fetch)
    mockFetch.mockResolvedValueOnce(new Response('{}', { status: 200 }))

    const body = JSON.stringify({ foo: 'bar' })
    await apiFetch('/api/v1/runs', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body,
    })

    expect(mockFetch).toHaveBeenCalledOnce()
    const [, init] = mockFetch.mock.calls[0]
    expect((init as RequestInit).method).toBe('POST')
    expect((init as RequestInit).body).toBe(body)
    // Headers are merged into a Headers instance; verify Content-Type survived
    const headers = (init as RequestInit).headers as Headers
    expect(headers.get('Content-Type')).toBe('application/json')
  })
})
