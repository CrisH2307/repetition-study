import { expect, test } from 'vitest'
import { parseSSE } from './api'
import { renderMath } from './Math'

test('parseSSE keeps a partial event for the next chunk', () => {
  const a = parseSSE('event: token\ndata: {"text":"Hi"}\n\nevent: do')
  expect(a.events).toEqual([{ event: 'token', data: { text: 'Hi' } }])
  const b = parseSSE(a.rest + 'ne\ndata: {}\n\n')
  expect(b.events).toEqual([{ event: 'done', data: {} }])
})

test('formulas render and bad ones do not crash', () => {
  expect(renderMath('a $x^2$ b')).toContain('katex')
  expect(() => renderMath('$\\badcommand{$')).not.toThrow()
  expect(renderMath('<b>')).toContain('&lt;b&gt;')
})
