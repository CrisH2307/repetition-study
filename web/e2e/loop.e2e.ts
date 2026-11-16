import { expect, test, Page } from '@playwright/test'

// Both tests need the fixture pack seeded. Each test uses its own topic, so run them in one DB once.
async function teach(page: Page, base: string, topic: string) {
  await page.goto(base)
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
  await expect(page.locator('li.topic', { hasText: 'Poisson superposition' })).toContainText('New topic: not taught yet')
})
