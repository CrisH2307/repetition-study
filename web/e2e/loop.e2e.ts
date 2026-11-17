import { expect, test, Page } from '@playwright/test'
import { execSync } from 'node:child_process'
import { readFileSync } from 'node:fs'

// The Go API needs a sign-in (P1-API). Each test invites a new student with cmd/invite, opens the
// invite link in the browser, and builds their course through the API: paste the fixture notes,
// mark the page Ready, add the two topics with one claim each. Needs Postgres migrated, nothing else seeded.
const db = process.env.DATABASE_URL ?? 'postgres://repetition:repetition@localhost:5432/repetition?sslmode=disable'
const unit = JSON.parse(readFileSync('../content/fixture/unit.json', 'utf8'))

// One quote per topic, copied from the fixture notes.
const topics: [string, string][] = [
  ['Exponential races', 'Then the chance that X finishes before Y is a / (a + b).'],
  ['Poisson superposition', 'Merging two independent Poisson processes with rates a and b gives one Poisson process with rate a + b.'],
]

// A same-origin fetch from the page, so the session cookie rides along as it does for the app.
const api = <T>(page: Page, method: string, path: string, body?: unknown, form?: Record<string, string>) =>
  page.evaluate(async ([m, p, b, f]) => {
    const init: RequestInit = { method: m as string }
    if (f) {
      const fd = new FormData()
      Object.entries(f as Record<string, string>).forEach(([k, v]) => fd.append(k, v))
      init.body = fd
    } else if (b !== undefined) {
      init.headers = { 'Content-Type': 'application/json' }
      init.body = JSON.stringify(b)
    }
    const r = await fetch(p as string, init)
    if (!r.ok) throw new Error(`${m} ${p}: ${r.status} ${await r.text()}`)
    return r.status === 204 ? null : r.json()
  }, [method, path, body, form] as const) as Promise<T>

async function signInWithCourse(page: Page, base: string) {
  const out = execSync(`go run ./cmd/invite -name "e2e ${Date.now()}"`, { cwd: '..', env: { ...process.env, DATABASE_URL: db, APP_URL: base } }).toString()
  await page.goto(`${base}/invite#${out.trim().split('#')[1]}`)
  // Consent is tested in pilot.mock.ts. Mark it accepted in this browser so the loop runs on any API version.
  await expect(page.getByRole('button', { name: 'I agree' })).toBeVisible()
  const me = await api<{ id: string }>(page, 'GET', '/v1/me')
  await page.evaluate((id) => localStorage.setItem(`sr:consent:${id}`, 'draft-1'), me.id)
  const course = await api<{ id: string }>(page, 'POST', '/v1/courses', { name: 'Probability', weekly_hours: 3, level: 'intermediate' })
  const up = await api<{ id: string }>(page, 'POST', `/v1/courses/${course.id}/uploads`, undefined, { filename: 'notes.txt', text: unit.sources[0].text })
  await expect.poll(async () => (await api<{ status: string }>(page, 'GET', `/v1/uploads/${up.id}`)).status).toBe('needs_checking')
  const pages = await api<{ id: string }[]>(page, 'GET', `/v1/uploads/${up.id}/pages`)
  for (const p of pages) await api(page, 'POST', `/v1/pages/${p.id}/ready`)
  const blocks = await api<{ id: string; text_md: string }[]>(page, 'GET', `/v1/courses/${course.id}/blocks`)
  for (const [name, quote] of topics) {
    const concept = await api<{ id: string }>(page, 'POST', `/v1/courses/${course.id}/concepts`, { name })
    const block = blocks.find((b) => b.text_md.includes(quote))!
    await api(page, 'POST', `/v1/concepts/${concept.id}/claims`, { text: quote, kind: 'statement', required: true, anchor: { block_id: block.id, quote } })
  }
  await page.goto(base) // Today now lists the two topics
}

async function teach(page: Page, base: string, topic: string) {
  await signInWithCourse(page, base)
  await page.getByRole('button', { name: `Start ${topic}` }).click()
  await page.getByLabel('Your explanation').fill('The minimum of two exponentials with rates $a$ and $b$ has rate $a-b$.')
  await expect(page.locator('.preview .katex').first()).toBeVisible()
  await page.getByRole('button', { name: 'Send', exact: true }).click()
  await expect(page.locator('.question .katex, .question')).toContainText('The AI student asks:')
  await expect(page.getByRole('button', { name: 'Send', exact: true })).toBeDisabled() // textarea cleared after the turn
  await page.getByRole('button', { name: 'I am done, check my explanation' }).click()
  await page.getByLabel('Right').first().check()
  await page.getByRole('button', { name: 'Show my results' }).click()
}

test('typed loop shows before and after review dates', async ({ page }) => {
  await teach(page, 'http://localhost:15173', 'Exponential races')
  await expect(page.getByText('Next review:')).toBeVisible()
  await expect(page.locator('.card s')).toBeVisible() // crossed-out date before
  await expect(page.locator('.card strong')).toBeVisible() // date after
  await expect(page.getByText('Not checked yet.')).toHaveCount(0)
})

test('checker down shows the banner and moves no date', async ({ page }) => {
  await teach(page, 'http://localhost:15174', 'Poisson superposition')
  await expect(page.getByText('Not checked yet.')).toBeVisible()
  await expect(page.getByText('Next review:')).toHaveCount(0)
  await page.getByRole('button', { name: 'Try later' }).click()
  await expect(page.getByRole('button', { name: 'Start Poisson superposition' })).toBeVisible()
  await expect(page.locator('li.topic', { hasText: 'Poisson superposition' })).toContainText('Not taught yet') // the Today screen's wording since wave 2
})
