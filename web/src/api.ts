export type Label = 'supported' | 'partial' | 'contradicted' | 'absent'
export interface Citation { source_id: string; quote: string; locator: string }
export interface ClaimResult {
  claim_id: string | null; student_text: string; label: Label
  citation: Citation | null; correction: string | null; probe_hint: string | null
}
export interface Schedule { due_before: string; due_after: string }
export interface Report {
  session_id: string; concept_id: string; results: ClaimResult[]
  schedule: Schedule | null; status: 'checked' | 'unchecked'
}
export interface DueItem { concept_id: string; concept_name: string; due_at: string; reason: string }
export interface Subject { id: string; name: string }

async function json<T>(r: Response): Promise<T> {
  if (!r.ok) throw new Error('The server said no. Please try again in a moment.')
  return r.json()
}
const post = (url: string, body?: unknown) =>
  fetch(url, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body ?? {}) })

export const listConcepts = () => fetch('/v1/reviews/due').then(json<DueItem[]>)
export const listSubjects = () => fetch('/v1/subjects').then(json<Subject[]>)
export const startSession = (concept_id: string) =>
  post('/v1/teach/sessions', { concept_id }).then(json<{ id: string }>)
export const finish = (id: string) => post(`/v1/teach/sessions/${id}/finish`).then(json<Report>)

export interface SSEEvent { event: string; data: any }

/** Split an SSE text buffer into complete events; returns the leftover partial text. */
export function parseSSE(buf: string): { events: SSEEvent[]; rest: string } {
  const parts = buf.split('\n\n')
  const rest = parts.pop() ?? ''
  const events = parts.flatMap((p) => {
    const ev = /^event: (.*)$/m.exec(p)?.[1]
    const data = /^data: (.*)$/m.exec(p)?.[1]
    if (!ev || data === undefined) return []
    try { return [{ event: ev, data: JSON.parse(data) }] } catch { return [] }
  })
  return { events, rest }
}

/** Send one turn and call onEvent for each SSE event. Throws on network failure. */
export async function sendTurn(id: string, transcript: string, onEvent: (e: SSEEvent) => void) {
  const r = await post(`/v1/teach/sessions/${id}/turns`, { transcript, latex: transcript })
  if (!r.ok || !r.body) throw new Error('bad status')
  const reader = r.body.getReader()
  const dec = new TextDecoder()
  let buf = ''
  for (;;) {
    const { done, value } = await reader.read()
    if (done) break
    const p = parseSSE((buf += dec.decode(value, { stream: true })))
    buf = p.rest
    p.events.forEach(onEvent)
  }
}
