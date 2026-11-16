import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import { mockApi } from './mock/plugin'

// Config switch: `npm run dev` uses the built-in mock API (no Go server needed).
// `API_TARGET=http://localhost:8080 npm run dev` sends /v1 to the real server.
const target = process.env.API_TARGET

export default defineConfig({
  plugins: [react(), ...(target ? [] : [mockApi()])],
  server: target ? { proxy: { '/v1': target } } : {},
  test: { environment: 'jsdom' },
})
