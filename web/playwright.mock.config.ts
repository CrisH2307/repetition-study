import { defineConfig } from '@playwright/test'

// The new screens against the built-in mock API only (no Go server, no database).
// Run: npx playwright test -c playwright.mock.config.ts
export default defineConfig({
  testDir: 'e2e',
  testMatch: '*.mock.ts',
  workers: 1,
  use: { browserName: 'chromium', baseURL: 'http://localhost:15175' },
  webServer: { command: 'npx vite --port 15175 --strictPort', url: 'http://localhost:15175', reuseExistingServer: false },
})
