import { expect, test } from '@playwright/test'

// One mock server for the whole file, so the tests run in order and share its state.
test.describe.configure({ mode: 'serial' })
// Consent, guide and the weekly question are covered in pilot.mock.ts; skip them here.
test.beforeEach(async ({ page, request }) => {
  await request.post('/v1/consent', { data: { text_version: 'draft-1' } })
  await page.addInitScript(() => localStorage.setItem('sr:guide:u1', '1'))
})

test('invite link signs in and shows Today', async ({ page }) => {
  await page.goto('/invite#demo')
  await expect(page).toHaveURL(/localhost:15175\/$/)
  await expect(page.getByRole('complementary', { name: 'Study Time' })).toContainText('1.2 / 3.0 h')
  await expect(page.getByRole('button', { name: 'Start Exponential races' })).toBeVisible()
})

test('upload, page Ready, add a key idea, teach, report, dispute', async ({ page }) => {
  await page.goto('/invite#demo')
  await page.getByRole('link', { name: 'Courses' }).click()
  await page.getByRole('link', { name: 'Fixture course' }).click()

  // Upload pasted notes and wait for the reading to finish
  await page.getByLabel('Title').fill('Week 1 notes')
  await page.getByLabel('Notes', { exact: true }).fill('The minimum has rate $a+b$.\n\nX finishes first with chance a / (a + b), which is the main fact.')
  await page.getByRole('button', { name: 'Add notes' }).click()
  await expect(page.getByText('Reading the file...')).toBeVisible()
  await page.getByRole('link', { name: 'Check pages' }).click()

  // Page check: mark Ready
  await expect(page.getByText('Page 1: Needs checking')).toBeVisible()
  await page.getByRole('button', { name: 'Mark Ready' }).click()
  await expect(page.getByText('Page 1: Ready')).toBeVisible()
  await page.getByRole('link', { name: 'Back to the course' }).click()
  await expect(page.getByText('Ready, 1 of 1 pages ready')).toBeVisible()

  // Add a key idea anchored to a quote from the Ready page
  const card = page.locator('.card', { hasText: 'Exponential races' }).first()
  await card.getByText('Key ideas (2)').click()
  const form = card.getByRole('form', { name: 'Add a key idea' })
  await form.getByLabel('The idea, in your words').fill('X wins with chance $a/(a+b)$.')
  await form.getByLabel('Where it is in your notes').selectOption({ index: 2 })
  await form.getByLabel('Exact words from that passage').fill('words that are not there')
  await form.getByRole('button', { name: 'Add key idea' }).click()
  await expect(form.getByRole('alert')).toContainText('not in that passage')
  await form.getByLabel('Exact words from that passage').fill('X finishes first with chance a / (a + b)')
  await form.getByRole('button', { name: 'Add key idea' }).click()
  await expect(card.getByText('Key ideas (3)')).toBeVisible()

  // Mistakes table links to the page
  await expect(page.getByRole('table')).toContainText('Not right')
  await expect(page.getByRole('link', { name: 'Page 1' })).toBeVisible()

  // Teach, self-grade, report
  await card.getByRole('button', { name: 'Start Exponential races' }).click()
  await expect(page.getByRole('navigation', { name: 'Main' })).toHaveCount(0)
  await page.getByLabel('Your explanation').fill('The minimum of two exponentials with rates $a$ and $b$ has rate $a-b$.')
  await expect(page.locator('.preview .katex').first()).toBeVisible()
  await page.getByRole('button', { name: 'Send', exact: true }).click()
  await expect(page.getByRole('button', { name: 'Send', exact: true })).toBeDisabled()
  await page.getByRole('button', { name: 'I am done, check my explanation' }).click()
  await page.getByLabel('Right').first().check()
  await page.getByRole('button', { name: 'Show my results' }).click()
  await expect(page.getByText('Next review:')).toBeVisible()
  await expect(page.locator('.card s')).toBeVisible()
  await expect(page.getByText('2 of 3 points were not right or missing.')).toBeVisible()
  await expect(page.getByText('Your guess: Right')).toBeVisible()
  await page.getByRole('button', { name: 'I think this check is wrong' }).nth(1).click()
  await page.getByLabel(/What looks wrong/).fill('My notes say b over a+b.')
  await page.getByRole('button', { name: 'Send to a person' }).click()
  await expect(page.getByText('Thanks. A person will read this.')).toBeVisible()
  await page.getByRole('button', { name: 'Back to Today' }).click()
  await expect(page.getByRole('navigation', { name: 'Main' })).toBeVisible()
})

test('checker down shows Not checked and no corrections', async ({ page }) => {
  await page.goto('/invite#demo')
  await page.getByRole('button', { name: 'Start Practice: pretend the checker is down' }).click()
  await page.getByLabel('Your explanation').fill('Some words.')
  await page.getByRole('button', { name: 'Send', exact: true }).click()
  await expect(page.getByRole('button', { name: 'Send', exact: true })).toBeDisabled()
  await page.getByRole('button', { name: 'I am done, check my explanation' }).click()
  await page.getByRole('button', { name: 'Show my results' }).click()
  await expect(page.getByText('Not checked yet.')).toBeVisible()
  await expect(page.getByText('Next review:')).toHaveCount(0)
  await expect(page.getByText('Correction:')).toHaveCount(0)
})

test('week shows the date range, schedule with today and notes', async ({ page }) => {
  await page.goto('/invite#demo')
  await page.getByRole('link', { name: 'Week' }).click()
  await expect(page.getByRole('heading', { name: 'Schedule:' })).toBeVisible()
  await expect(page.getByText('(today)')).toBeVisible()
  await expect(page.getByText('moved to Tuesday')).toBeVisible()
})

test('phone width has no sideways scroll', async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 800 })
  for (const path of ['/invite#demo', '/#/week', '/#/course/00000000-0000-0000-0000-0000000000s1', '/#/settings']) {
    await page.goto(path)
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
    expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(375)
  }
})

test('sign out, then a used link says it does not work', async ({ page }) => {
  await page.goto('/invite#demo')
  await page.getByRole('button', { name: 'Sign out' }).click()
  await expect(page.getByText('You are signed out.')).toBeVisible()
  await page.goto('/invite#used')
  await expect(page.getByText('This link does not work.')).toBeVisible()
})

test('share: make, copy, turn off; open a link and import', async ({ page, context }) => {
  await context.grantPermissions(['clipboard-read', 'clipboard-write'])
  await page.goto('/invite#demo')
  await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
  await page.goto('/#/course/00000000-0000-0000-0000-0000000000s1')
  await page.getByRole('button', { name: 'Make a link' }).click()
  await page.getByRole('button', { name: 'Copy link' }).click()
  await expect(page.getByText('Link copied.')).toBeVisible()
  await page.getByRole('button', { name: 'Turn off link' }).click()
  await expect(page.getByText('Link turned off.')).toBeVisible()
  await page.goto('/#/shared/instructor')
  await expect(page.getByText('Instructor approved')).toBeVisible()
  await page.getByRole('button', { name: 'Add to my course' }).click()
  await expect(page.getByRole('heading', { name: 'Needs a quote' })).toBeVisible()
  await page.goto('/#/shared/gone')
  await expect(page.getByRole('alert')).toHaveText('This link does not work.')
})
