// web/vitest.config.ts — Vitest configuration for React component and hook unit tests.
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'jsdom',
    globals: true,
    // The Dagger CI container is far slower to cold-start jsdom and the
    // component import graph than a local run: one file alone took 170s in CI
    // (setup 81s, import 43s) against a couple of seconds locally. The 5000ms
    // default timeout flakes there on a test that does nothing slow, so this
    // gives real headroom without masking a genuine hang.
    testTimeout: 15000,
    setupFiles: ['./src/test-setup.ts'],
    include: ['src/**/__tests__/**/*.test.{ts,tsx}', 'src/**/*.test.{ts,tsx}'],
    exclude: ['node_modules', 'e2e/**'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'lcov', 'html'],
      reportsDirectory: './coverage',
      include: ['src/hooks/**', 'src/views/**', 'src/lib/**', 'src/components/**'],
      exclude: [
        'src/**/__tests__/**',
        'src/test-setup.ts',
        'src/test-utils.tsx',
        'src/main.tsx',
      ],
      // Conservative initial thresholds — ratchet up as tests are added.
      thresholds: {
        lines: 30,
        functions: 30,
        branches: 25,
        statements: 30,
      },
    },
  },
})
