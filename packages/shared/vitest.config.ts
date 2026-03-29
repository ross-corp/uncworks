// packages/shared/vitest.config.ts — Vitest configuration for @aot/shared.
import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: {
    globals: true,
    include: ['src/**/*.test.ts'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'lcov'],
      reportsDirectory: './coverage',
      include: ['src/**'],
      exclude: ['src/**/*.test.ts'],
      thresholds: { lines: 40, functions: 40 },
    },
  },
})
