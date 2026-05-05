// web/src/test-setup.ts
import '@testing-library/jest-dom'
import { server } from './mocks/server'
import { beforeAll, afterEach, afterAll } from 'vitest'

// Start MSW before all tests, reset handlers after each, clean up after suite.
beforeAll(() => server.listen({ onUnhandledRequest: 'warn' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())
