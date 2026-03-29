// web/src/test-utils.tsx
import { render, RenderOptions } from '@testing-library/react'
import { MemoryRouter, MemoryRouterProps } from 'react-router-dom'
import { ReactElement } from 'react'

interface TestRenderOptions extends Omit<RenderOptions, 'wrapper'> {
  routerProps?: MemoryRouterProps
}

export function renderWithRouter(
  ui: ReactElement,
  { routerProps, ...options }: TestRenderOptions = {}
) {
  return render(ui, {
    wrapper: ({ children }) => (
      <MemoryRouter {...routerProps}>{children}</MemoryRouter>
    ),
    ...options,
  })
}

export * from '@testing-library/react'
