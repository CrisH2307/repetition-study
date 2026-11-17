import AxeBuilder from '@axe-core/playwright'
import { expect, test, type Page } from '@playwright/test'

test.describe.configure({ mode: 'serial' })
test.beforeEach(async ({ page, request }) => {
  await request.post('/v1/consent', { data: { text_version: 'draft-1' } })
  await page.addInitScript(() => localStorage.setItem('sr:guide:u1', '1'))
})

async function axe(page: Page) {
  const r = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']).analyze()
  const bad = r.violations.filter((v) => v.impact === 'serious' || v.impact === 'critical')
  expect(bad.map((v) => `${v.id}: ${v.nodes.map((n) => n.target).join(' | ')}`)).toEqual([])
}

const COURSE = '/#/course/00000000-0000-0000-0000-0000000000s1'

test('axe: sign-in states, Today, Week, Courses, Course, Settings', async ({ page }) => {
  await page.goto('/invite#used')
  await expect(page.getByText('This link does not work.')).toBeVisible()
  await axe(page)
  await page.goto('/invite#demo')
  await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
  await axe(page)
  for (const p of ['/#/week', '/#/courses', COURSE, '/#/settings']) {
    await page.goto(p)
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
    await axe(page)
  }
})

test('axe: teach, report, page check', async ({ page }) => {
  await page.goto('/invite#demo')
  await page.getByRole('button', { name: 'Start Exponential races' }).click()
  await page.getByLabel('Your explanation').fill('The minimum has rate $a+b$.')
  await expect(page.locator('.preview .katex').first()).toBeVisible()
  await axe(page)
  await page.getByRole('button', { name: 'Send', exact: true }).click()
  await expect(page.getByRole('button', { name: 'Send', exact: true })).toBeDisabled()
  await page.getByRole('button', { name: 'I am done, check my explanation' }).click()
  await axe(page)
  await page.getByRole('button', { name: 'Show my results' }).click()
  await expect(page.getByText('Next review:')).toBeVisible()
  await axe(page)
})

test('keyboard only: invite, Courses, Settings, Teach to report', async ({ page }) => {
  await page.goto('/invite#demo')
  await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
  await page.keyboard.press('Tab')
  await expect(page.getByRole('link', { name: 'Today' })).toBeFocused()
  await page.getByRole('link', { name: 'Settings' }).focus()
  await page.keyboard.press('Enter')
  await expect(page.getByRole('heading', { name: 'Settings' })).toBeVisible()
  await page.getByLabel(/Type/).focus()
  await page.keyboard.type('delete my account')
  await expect(page.getByRole('button', { name: 'Delete my account' })).toBeEnabled()
  await page.getByLabel(/Type/).fill('')
  await page.getByRole('link', { name: 'Today' }).focus()
  await page.keyboard.press('Enter')
  await page.getByRole('button', { name: 'Start Exponential races' }).focus()
  await page.keyboard.press('Enter')
  await page.getByLabel('Your explanation').focus()
  await page.keyboard.type('Some words.')
  await page.getByRole('button', { name: 'Send', exact: true }).focus()
  await page.keyboard.press('Enter')
  await expect(page.getByRole('button', { name: 'Send', exact: true })).toBeDisabled()
  await expect(page.getByRole('button', { name: 'I am done, check my explanation' })).toBeEnabled()
  await page.getByRole('button', { name: 'I am done, check my explanation' }).focus()
  await page.keyboard.press('Enter')
  await page.getByLabel('Right').first().focus()
  await page.keyboard.press('Space')
  await page.getByRole('button', { name: 'Show my results' }).focus()
  await page.keyboard.press('Enter')
  await expect(page.getByText(/Next review:|Not checked yet/).first()).toBeVisible()
  // focus is always visible on whatever Tab lands on
  await page.keyboard.press('Tab')
  const ring = await page.evaluate(() => getComputedStyle(document.activeElement!).outlineStyle)
  expect(ring).not.toBe('none')
})

test('every input has an accessible name', async ({ page }) => {
  await page.goto('/invite#demo')
  for (const p of [COURSE, '/#/settings', '/#/courses']) {
    await page.goto(p)
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
    const unnamed = await page.evaluate(() =>
      [...document.querySelectorAll('input,select,textarea')].filter((el) => !(el as HTMLInputElement).labels?.length && !el.getAttribute('aria-label')).length)
    expect(unnamed).toBe(0)
  }
})

test('formulas have a screen-reader reading', async ({ page }) => {
  await page.goto('/invite#demo')
  await page.getByRole('button', { name: 'Start Exponential races' }).click()
  await page.getByLabel('Your explanation').fill('Rate $a+b$.')
  await expect(page.locator('.preview .katex-mathml math')).toHaveCount(1)
  await expect(page.locator('.preview .katex-html').first()).toHaveAttribute('aria-hidden', 'true')
})

test('reduced motion removes transitions', async ({ page }) => {
  await page.emulateMedia({ reducedMotion: 'reduce' })
  await page.goto('/invite#demo')
  await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
  const d = await page.evaluate(() => getComputedStyle(document.querySelector('button')!).transitionDuration)
  expect(d).toMatch(/^0s/)
})

test('no sideways scroll at 320px', async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 700 })
  await page.goto('/invite#demo')
  await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
  for (const p of ['/#/', '/#/week', '/#/courses', COURSE, '/#/settings']) {
    await page.goto(p)
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
    expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(320)
  }
  await page.getByRole('link', { name: 'Today' }).click()
  await page.getByRole('button', { name: 'Start Exponential races' }).click()
  await page.getByLabel('Your explanation').fill('A long formula $\\sum_{i=1}^{n} \\frac{a_i + b_i + c_i + d_i + e_i + f_i + g_i}{h_i + j_i + k_i + l_i}$')
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(320)
})
