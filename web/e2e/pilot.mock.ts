import { expect, test } from '@playwright/test'
import { readFileSync } from 'node:fs'

// Pilot screens against the mock API. Runs before screens.mock.ts (file order), and the last test deletes the account.
test.describe.configure({ mode: 'serial' })

test('consent comes before first use, then the guide and the weekly question', async ({ page }) => {
  await page.goto('/invite#demo')
  await expect(page.getByRole('heading', { name: 'Before you start' })).toBeVisible()
  await expect(page.getByText('Version draft-1')).toBeVisible()
  await expect(page.getByRole('navigation', { name: 'Main' })).toHaveCount(0)
  await page.getByRole('button', { name: 'I agree' }).click()
  await expect(page.getByRole('navigation', { name: 'Main' })).toBeVisible()

  const guide = page.getByRole('region', { name: 'First steps' })
  await expect(guide.getByRole('listitem')).toHaveCount(3)
  await guide.getByRole('button', { name: 'Got it' }).click()
  await expect(guide).toHaveCount(0)

  const q = page.getByRole('form', { name: 'Weekly question' })
  await q.getByLabel(/no longer use this/).fill('Sad.')
  await q.getByRole('button', { name: 'Send' }).click()
  await expect(page.getByText('Thanks for your answer.')).toBeVisible()

  await page.reload() // remembered in this browser: no consent, guide or question again
  await expect(page.getByRole('navigation', { name: 'Main' })).toBeVisible()
  await expect(page.getByRole('region', { name: 'First steps' })).toHaveCount(0)
  await expect(page.getByRole('form', { name: 'Weekly question' })).toHaveCount(0)
})

test('a new course shows empty states with one next action', async ({ page }) => {
  await page.goto('/invite#demo')
  await page.getByRole('button', { name: 'I agree' }).click()
  await page.getByRole('link', { name: 'Courses' }).first().click()
  await page.getByLabel('Course name').fill('Empty course')
  await page.getByRole('button', { name: 'Add course' }).click()
  await page.getByRole('link', { name: 'Empty course' }).click()
  await expect(page.getByText('No notes yet.')).toBeVisible()
  await expect(page.getByText('No topics yet.')).toBeVisible()
})

test('CSV import shows per-row results', async ({ page }) => {
  await page.goto('/invite#demo')
  await page.getByRole('button', { name: 'I agree' }).click()
  await page.getByRole('link', { name: 'Courses' }).first().click()
  await page.getByRole('link', { name: 'Empty course' }).click() // its own course, so screens.mock.ts sees the fixture course unchanged
  // A Ready page to quote from
  await page.getByLabel('Title').fill('Import notes')
  await page.getByLabel('Notes', { exact: true }).fill('Merging two Poisson processes gives rate a + b.')
  await page.getByRole('button', { name: 'Add notes' }).click()
  await page.getByRole('link', { name: 'Check pages' }).click()
  await page.getByRole('button', { name: 'Mark Ready' }).click()
  await expect(page.getByText('Page 1: Ready')).toBeVisible()
  await page.getByRole('link', { name: 'Back to the course' }).click()

  const form = page.getByRole('form', { name: 'Import from a CSV file' })
  await form.getByLabel('CSV file').setInputFiles({ name: 'cards.csv', mimeType: 'text/csv', buffer: Buffer.from(
    'concept,claim,kind,quote\n"Poisson superposition","The merged rate is $a+b$.",statement,gives rate a + b\nPoisson superposition,Wrong quote,statement,not in the notes\n') })
  await form.getByRole('button', { name: 'Import' }).click()
  await expect(form.getByRole('status')).toContainText('1 of 2 rows imported.')
  await expect(form.getByRole('status')).toContainText('Line 3: This quote is not on a Ready page.')
  await expect(page.locator('.card', { hasText: 'Poisson superposition' }).getByText('Key ideas (1)')).toBeVisible()
})

test('download my data gives one JSON file', async ({ page }) => {
  await page.goto('/invite#demo')
  await page.getByRole('button', { name: 'I agree' }).click()
  await page.getByRole('link', { name: 'Settings' }).click()
  const [file] = await Promise.all([page.waitForEvent('download'), page.getByRole('link', { name: 'Download my data' }).click()])
  const data = JSON.parse(readFileSync((await file.path())!, 'utf8'))
  expect(data.consents[0].text_version).toBe('draft-1')
  expect(data.imports.length).toBeGreaterThan(0)
})

test('delete my account needs the typed words, then signs out', async ({ page }) => {
  await page.goto('/invite#demo')
  await page.getByRole('button', { name: 'I agree' }).click()
  await page.getByRole('link', { name: 'Settings' }).click()
  const form = page.getByRole('form', { name: 'Delete my account' })
  const del = form.getByRole('button', { name: 'Delete my account' })
  await expect(del).toBeDisabled()
  await form.getByLabel(/to confirm/).fill('delete')
  await expect(del).toBeDisabled()
  await form.getByLabel(/to confirm/).fill('delete my account')
  await del.click()
  await expect(page.getByText('Your account was deleted.')).toBeVisible()
  await page.goto('/')
  await expect(page.getByText('You are signed out.')).toBeVisible()
})
