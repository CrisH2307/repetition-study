import { defineConfig } from '@playwright/test'

// Two API servers share one database: one with the fake checker, one with fake-down.
// Two preview servers proxy /v1 to them. Start Postgres, migrate and seed first (see rpi/P1-E2E/plan.md).
const db = process.env.DATABASE_URL ?? 'postgres://repetition:repetition@localhost:5432/repetition?sslmode=disable'
const api = (port: number, checker: string) => ({
  command: `go run ./cmd/api`,
  cwd: '..',
  url: `http://localhost:${port}/v1/subjects`,
  env: { DATABASE_URL: db, PORT: String(port), STUDENT_MODEL: 'fake', CHECKER_MODEL: checker },
  reuseExistingServer: false,
})
const web = (port: number, apiPort: number) => ({
  command: `npx vite preview --port ${port} --strictPort`,
  url: `http://localhost:${port}`,
  env: { API_TARGET: `http://localhost:${apiPort}` },
  reuseExistingServer: false,
})

export default defineConfig({
  testDir: 'e2e',
  testMatch: '*.e2e.ts', // not .spec: vitest would pick that up
  workers: 1,
  use: { browserName: 'chromium' },
  webServer: [api(18080, 'fake'), api(18081, 'fake-down'), web(15173, 18080), web(15174, 18081)],
})
