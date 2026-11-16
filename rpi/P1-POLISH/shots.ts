import { test, Page } from '@playwright/test'
const tag = process.env.SHOT ?? 'before'
const dir = '../rpi/P1-POLISH/shots'
test.describe.configure({ mode: 'serial' })
const snap = (p: Page, name: string) => p.screenshot({ path: `${dir}/${tag}-${name}.png`, fullPage: true })
for (const [w, h, sz] of [[375, 800, 'phone'], [1280, 900, 'desktop']] as const) {
  test(`shots ${sz}`, async ({ page }) => {
    await page.setViewportSize({ width: w, height: h })
    await page.goto('/invite#used'); await page.getByText('This link does not work.').waitFor()
    await snap(page, `${sz}-signin`)
    await page.goto('about:blank'); await page.goto('/invite#demo'); await page.getByRole('button', { name: /Start Exponential/ }).first().waitFor()
    await snap(page, `${sz}-today`)
    await page.goto('/#/week'); await page.getByText('(today)').waitFor(); await snap(page, `${sz}-week`)
    await page.goto('/#/course/00000000-0000-0000-0000-0000000000s1'); await page.getByRole('heading', { name: 'Study cards' }).waitFor()
    await snap(page, `${sz}-course`)
    await page.getByLabel('Title').fill('Week 1 notes')
    await page.getByLabel('Notes', { exact: true }).fill('The minimum has rate $a+b$.\n\nX finishes first with chance $a/(a+b)$.')
    await page.getByRole('button', { name: 'Add notes' }).click()
    const check = page.getByRole('link', { name: 'Check pages' }).first(); await check.waitFor()
    await snap(page, `${sz}-course`)
    { await check.click(); await page.getByText(/Page \d+: /).waitFor(); await snap(page, `${sz}-pagecheck`) }
    await page.goto('/#/'); await page.getByRole('button', { name: 'Start Exponential races' }).click()
    await page.getByLabel('Your explanation').fill('The minimum of two exponentials with rates $a$ and $b$ has rate $a-b$.')
    await page.getByRole('button', { name: 'Send', exact: true }).click()
    await page.getByRole('button', { name: 'Send', exact: true }).isDisabled(); await page.waitForFunction(() => !document.querySelector('.question')?.textContent?.includes('Thinking'))
    await page.waitForTimeout(800); await snap(page, `${sz}-teach`)
    await page.getByRole('button', { name: 'I am done, check my explanation' }).click()
    await page.getByLabel('Right').first().check(); await snap(page, `${sz}-selfgrade`)
    await page.getByRole('button', { name: 'Show my results' }).click(); await page.getByText('Next review:').waitFor()
    await snap(page, `${sz}-report`)
    await page.goto('about:blank'); await page.goto('/#/'); await page.getByRole('button', { name: 'Start Practice: pretend the checker is down' }).click()
    await page.getByLabel('Your explanation').fill('Some words.'); await page.getByRole('button', { name: 'Send', exact: true }).click()
    await page.getByRole('button', { name: 'I am done, check my explanation' }).click({ timeout: 10000 })
    await page.getByRole('button', { name: 'Show my results' }).click(); await page.getByText('Not checked yet.').waitFor()
    await snap(page, `${sz}-unchecked`)
  })
}
