import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { App, dueText, mondayOf } from './App'
import { reasonLine } from './Teach'
import type { Report } from './api'

beforeEach(() => localStorage.setItem('sr:consent:u', 'draft-1'))
afterEach(() => { cleanup(); vi.unstubAllGlobals(); location.hash = ''; localStorage.clear() })

const res = (body: unknown, status = 200) => new Response(status === 204 ? null : JSON.stringify(body), { status })
const today = {
  date: new Date().toISOString(),
  courses: [{ subject_id: 's', name: 'Course', minutes: 15, items: [{ concept_id: 'c', concept_name: 'Topic', subject_id: 's', kind: 'teach', minutes: 15, due_at: new Date().toISOString(), reason: '' }] }],
  study_time: [{ subject_id: 's', name: 'Course', hours_used: 1.2, hours_budget: 3.5, reason: 'Up 1.5 h: 3 gaps on Monday' }],
}

test('no server shows a clear message, not a blank page', async () => {
  vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('down')))
  render(<App />)
  await waitFor(() => expect(screen.getByRole('alert').textContent).toMatch(/could not reach the server/))
})

test('a 401 shows the signed-out screen', async () => {
  vi.stubGlobal('fetch', vi.fn(async () => res({ error: 'Please sign in.' }, 401)))
  render(<App />)
  expect((await screen.findByRole('status')).textContent).toMatch(/You are signed out/)
})

test('Today shows minutes, red due date and Study Time with its reason', async () => {
  vi.stubGlobal('fetch', vi.fn(async (url: string) => res(url.includes('/me') ? { id: 'u', name: 'D', time_zone: 'UTC' } : today)))
  render(<App />)
  expect((await screen.findByText('Due today')).className).toBe('soon')
  expect(screen.getByText('1.2 / 3.5 h')).toBeTruthy()
  expect(screen.getByText('Up 1.5 h: 3 gaps on Monday')).toBeTruthy()
})

test('self-grade is saved per key idea and unchecked report says nothing changed', async () => {
  const rep = { session_id: 's', concept_id: 'c', results: [{ claim_id: 'k1', student_text: 'x', label: 'supported', citation: null, correction: null, probe_hint: null }], schedule: null, status: 'unchecked' }
  const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
    if (url.includes('/me')) return res({ id: 'u', name: 'D', time_zone: 'UTC' })
    if (url.includes('/today')) return res(today)
    if (url.endsWith('/sessions')) return res({ id: 's' })
    if (url.endsWith('/claims')) return res([{ id: 'k1', concept_id: 'c', text: 'Key idea one', source_id: 'x', locator: 'page 1', required: true }])
    if (url.endsWith('/self-grades')) return res(null, 204)
    if (url.endsWith('/turns')) return new Response('event: token\ndata: {"text":"Why?"}\n\nevent: done\ndata: {}\n\n')
    void init
    return res(rep)
  })
  vi.stubGlobal('fetch', fetchMock)
  const u = userEvent.setup()
  render(<App />)
  await u.click(await screen.findByRole('button', { name: 'Start Topic' }))
  await u.type(await screen.findByLabelText(/Your explanation/), 'hello')
  expect(screen.queryByRole('navigation')).toBeNull() // no main menu while teaching
  await u.click(screen.getByText('Send'))
  await screen.findByText('Why?')
  await u.click(screen.getByText(/I am done/))
  await screen.findByText(/Key idea one/)
  await u.click(screen.getByRole('radio', { name: 'Wrong' }))
  await u.click(screen.getByText('Show my results'))
  expect((await screen.findByRole('status')).textContent).toMatch(/Nothing changed/)
  expect(screen.getByText('Not checked')).toBeTruthy()
  expect(screen.getByText('Your guess: Wrong')).toBeTruthy()
  const saved = fetchMock.mock.calls.find(([url]) => url.endsWith('/self-grades'))
  expect(JSON.parse(saved![1]!.body as string)).toEqual({ grades: [{ claim_id: 'k1', rating: 'wrong' }] })
})

test('due date is red within 2 days and says today for today', () => {
  const now = new Date('2026-11-14T12:00:00')
  expect(dueText('2026-11-14T20:00:00', now)).toEqual({ text: 'Due today', soon: true })
  expect(dueText('2026-11-16T09:00:00', now).soon).toBe(true)
  expect(dueText('2026-11-20T09:00:00', now).soon).toBe(false)
})

test('week starts on Monday', () => {
  expect(mondayOf(new Date('2026-11-16T10:00:00'))).toBe('2026-11-16') // Monday
  expect(mondayOf(new Date('2026-11-22T10:00:00'))).toBe('2026-11-16') // Sunday
})

test('reason line uses the server words, else counts the labels', () => {
  const r = (labels: string[], payload = {}) => ({ results: labels.map((label) => ({ label })), schedule: { payload } }) as unknown as Report
  expect(reasonLine(r(['supported'], { reason: 'Sooner: 1 gap' }))).toBe('Sooner: 1 gap')
  expect(reasonLine(r(['supported', 'absent', 'contradicted']))).toBe('2 of 3 points were not right or missing.')
  expect(reasonLine(r(['supported']))).toBe('The point matched the course notes.')
})

test('consent comes first and is recorded with its version', async () => {
  localStorage.clear()
  const fetchMock = vi.fn(async (url: string) => res(url.includes('/me') ? { id: 'u', name: 'D', time_zone: 'UTC' } : url.includes('/consent') ? { text_version: 'draft-1', accepted_at: '' } : today))
  vi.stubGlobal('fetch', fetchMock)
  render(<App />)
  await userEvent.click(await screen.findByRole('button', { name: 'I agree' }))
  await screen.findByText('Due today')
  const sent = fetchMock.mock.calls.find(([url]) => url.endsWith('/consent')) as unknown as [string, RequestInit]
  expect(JSON.parse(sent[1].body as string)).toEqual({ text_version: 'draft-1' })
})
