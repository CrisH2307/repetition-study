import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, test, vi } from 'vitest'
import { App } from './App'

afterEach(() => vi.unstubAllGlobals())

test('a failed server shows a clear message, not a blank page', async () => {
  vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('down')))
  render(<App />)
  await waitFor(() => expect(screen.getByRole('alert').textContent).toMatch(/could not load/))
})

test('unchecked report says nothing changed', async () => {
  const rep = { session_id: 's', concept_id: 'c', results: [], schedule: null, status: 'unchecked' }
  const fetchMock = vi.fn(async (url: string) => {
    if (url.includes('reviews/due')) return new Response(JSON.stringify([{ concept_id: 'c', concept_name: 'Topic', reason: 'Due' }]))
    if (url.endsWith('/sessions')) return new Response(JSON.stringify({ id: 's' }))
    if (url.endsWith('/turns')) return new Response('event: token\ndata: {"text":"Why?"}\n\nevent: done\ndata: {}\n\n')
    return new Response(JSON.stringify(rep))
  })
  vi.stubGlobal('fetch', fetchMock)
  const u = userEvent.setup()
  render(<App />)
  await u.click(await screen.findByText('Start teaching'))
  await u.type(await screen.findByLabelText(/Your explanation/), 'hello')
  await u.click(screen.getByText('Send'))
  await screen.findByText('Why?')
  await u.click(screen.getByText(/I am done/))
  await u.click(await screen.findByText('Show my results'))
  expect((await screen.findByRole('status')).textContent).toMatch(/Nothing changed/)
})
