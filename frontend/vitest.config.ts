import { defineConfig } from 'vitest/config'

// Separate from vite.config.ts (which stays build-only) since test config
// has its own shape. Runs in Node — api.ts only touches fetch/URL, no DOM.
export default defineConfig({
  test: {
    environment: 'node',
    include: ['src/**/*.test.ts'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'lcov'],
      reportsDirectory: 'coverage',
      include: ['src/**/*.ts'],
    },
  },
})
