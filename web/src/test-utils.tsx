// web/src/test-utils.tsx
import { render, RenderOptions } from '@testing-library/react'
import { MemoryRouter, MemoryRouterProps, Routes, Route } from 'react-router-dom'
import { ReactElement } from 'react'

interface TestRenderOptions extends Omit<RenderOptions, 'wrapper'> {
  routerProps?: MemoryRouterProps
  /** When set, wraps `ui` in a <Routes><Route path={routePath}> so useParams works. */
  routePath?: string
}

export function renderWithRouter(
  ui: ReactElement,
  { routerProps, routePath, ...options }: TestRenderOptions = {}
) {
  return render(ui, {
    wrapper: ({ children }) => (
      <MemoryRouter {...routerProps}>
        {routePath ? (
          <Routes>
            <Route path={routePath} element={children} />
          </Routes>
        ) : children}
      </MemoryRouter>
    ),
    ...options,
  })
}

export * from '@testing-library/react'
