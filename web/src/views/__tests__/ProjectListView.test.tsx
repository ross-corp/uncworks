// web/src/views/__tests__/ProjectListView.test.tsx
// Tests for ProjectListView — project rows, empty state, project names
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { renderWithRouter } from '../../test-utils'

// --- module mocks ---

vi.mock('../../hooks/apiFetch', () => ({
  apiFetch: vi.fn(),
  apiWsUrl: vi.fn(),
  apiSseUrl: vi.fn(),
}))

vi.mock('sonner', () => ({
  toast: { error: vi.fn(), success: vi.fn(), loading: vi.fn() },
  Toaster: () => null,
}))

vi.mock('../../lib/format', () => ({
  formatAge: vi.fn(() => '2d'),
}))

// Stub UI primitives that may pull in heavy deps
vi.mock('../../components/ui/spinner', () => ({
  Spinner: () => <span data-testid="spinner" />,
}))

vi.mock('../../components/ui/empty', () => ({
  Empty: ({ children, ...props }: React.HTMLAttributes<HTMLDivElement>) => <div {...props}>{children}</div>,
  EmptyHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  EmptyTitle: ({ children }: { children: React.ReactNode }) => <h2>{children}</h2>,
  EmptyDescription: ({ children }: { children: React.ReactNode }) => <p>{children}</p>,
  EmptyContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

import { apiFetch } from '../../hooks/apiFetch'
import ProjectListView from '../ProjectListView'

interface ProjectSummary {
  name: string
  displayName: string
  description: string
  repos: { url: string; branch: string }[]
  configRepoReady: boolean
  configRepoMessage?: string
  runCount: number
  lastRunId: string
  totalCost: string
  createdAt: string
}

function makeProject(name: string, runCount = 0): ProjectSummary {
  return {
    name,
    displayName: name,
    description: '',
    repos: [],
    configRepoReady: true,
    runCount,
    lastRunId: '',
    totalCost: '',
    createdAt: new Date().toISOString(),
  }
}

function mockProjectsResponse(projects: ProjectSummary[]) {
  vi.mocked(apiFetch).mockResolvedValue(
    new Response(JSON.stringify(projects), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    })
  )
}

describe('ProjectListView', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('shows empty state when no projects returned', async () => {
    mockProjectsResponse([])

    renderWithRouter(<ProjectListView />)

    await waitFor(() => {
      expect(screen.getByText('No projects yet')).toBeInTheDocument()
    })
  })

  it('renders project rows when API returns projects', async () => {
    mockProjectsResponse([makeProject('alpha', 3), makeProject('beta', 1)])

    renderWithRouter(<ProjectListView />)

    await waitFor(() => {
      expect(screen.getByText('alpha')).toBeInTheDocument()
      expect(screen.getByText('beta')).toBeInTheDocument()
    })
  })

  it('each row has correct project name', async () => {
    const projects = ['gamma', 'delta', 'epsilon'].map((n) => makeProject(n))
    mockProjectsResponse(projects)

    renderWithRouter(<ProjectListView />)

    await waitFor(() => {
      for (const p of projects) {
        expect(screen.getByText(p.name)).toBeInTheDocument()
      }
    })
  })
})
