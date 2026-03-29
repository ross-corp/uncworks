// web/vitest.config.ts — Vitest configuration for React component and hook unit tests.
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'jsdom',
    globals: true,
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
