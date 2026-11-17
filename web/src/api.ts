export type Label = 'supported' | 'partial' | 'contradicted' | 'absent'
export interface Citation { source_id: string; quote: string; locator: string }
export interface ClaimResult {
  claim_id: string | null; student_text: string; label: Label
  citation: Citation | null; correction: string | null; probe_hint: string | null
}
export interface Schedule { due_before: string; due_after: string; payload?: Record<string, unknown> }
export interface Report {
  session_id: string; concept_id: string; results: ClaimResult[]
  schedule: Schedule | null; status: 'checked' | 'unchecked'
}
export interface User { id: string; name: string; time_zone: string; consent_version?: string | null }
export interface Subject { id: string; name: string; weekly_hours: number; level: string; weight: number; exam_date?: string }
export interface TodayItem { concept_id: string; concept_name: string; subject_id: string; kind: 'teach' | 'recall'; minutes: number; due_at: string; reason: string }
export interface CourseHours { subject_id: string; name: string; hours_used: number; hours_budget: number; reason: string }
export interface Today { date: string; courses: { subject_id: string; name: string; minutes: number; items: TodayItem[] }[]; study_time: CourseHours[] }
export interface WeekPlan { week_start: string; courses: CourseHours[]; days: { date: string; items: TodayItem[] }[]; notes: string[] }
export interface Upload { id: string; subject_id: string; filename: string; status: 'reading' | 'needs_checking' | 'ready'; error: string; source_id: string | null; pages_total: number; pages_ready: number; created_at: string }
export interface Page { id: string; upload_id: string; number: number; raw_text: string; edited_text: string; status: 'needs_checking' | 'ready'; has_image: boolean; updated_at: string }
export interface SourceBlock { id: string; source_id: string; page: number | null; ordinal: number; kind: string; text_md: string }
export type ClaimKind = 'definition' | 'statement' | 'condition' | 'method'
export interface ClaimInput { text: string; kind: ClaimKind; required: boolean; anchor: { block_id: string; quote: string } }
export interface Claim { id: string; concept_id: string; text: string; source_id: string; locator: string; required: boolean; kind?: ClaimKind }
export interface StudyCard { concept_id: string; name: string; recall: number | null; explanation: number | null; next_due: string | null; claims: Claim[] }
export interface Mistake { session_id: string; concept_id: string; claim_id: string | null; student_text: string; label: Label; quote: string; source_id: string; page_id: string | null; page: number | null; created_at: string }
export interface CourseView { course: Subject; notes: Upload[]; cards: StudyCard[]; mistakes: Mistake[] }
export type SelfRating = 'right' | 'not_sure' | 'wrong'
export interface FeedbackPrompt { week_start: string; question: string; answer: string | null }
export interface ImportJob { id: string; subject_id: string; filename: string; status: 'running' | 'done' | 'failed'; rows_total: number; rows_imported: number; row_errors: { row: number; error: string }[]; created_at: string; finished_at: string | null }

/** Fired on any 401 so the app can show the signed-out screen. */
export const SIGNED_OUT = 'signed-out'

/** One fetch helper: plain-words errors from the server, 401 means signed out. */
export async function call<T>(url: string, method = 'GET', body?: unknown): Promise<T> {
  const init: RequestInit = { method }
  if (body instanceof FormData) init.body = body
  else if (body !== undefined || method !== 'GET') {
    init.headers = { 'Content-Type': 'application/json' }
    init.body = JSON.stringify(body ?? {})
  }
  const r = await fetch(url, init)
  if (r.status === 401) window.dispatchEvent(new Event(SIGNED_OUT))
  if (!r.ok) {
    const msg = await r.json().then((b) => b?.error, () => '')
    throw new Error(msg || 'The server said no. Please try again in a moment.')
  }
  return r.status === 204 ? (undefined as T) : r.json()
}

export const me = () => call<User>('/v1/me')
export const redeem = (token: string) => call<User>('/v1/auth/redeem', 'POST', { token })
export const signOut = () => call<void>('/v1/auth/sign-out', 'POST')
export const deleteAccount = () => call<void>('/v1/me', 'DELETE', { confirm: 'delete my account' })
export const acceptConsent = (text_version: string) => call<unknown>('/v1/consent', 'POST', { text_version })
export const getFeedback = () => call<FeedbackPrompt>('/v1/feedback')
export const sendFeedback = (answer: string) => call<unknown>('/v1/feedback', 'POST', { answer })
export const importCsv = (course: string, f: FormData) => call<ImportJob>(`/v1/courses/${course}/import`, 'POST', f)
export const getImport = (id: string) => call<ImportJob>(`/v1/imports/${id}`)
export const getToday = () => call<Today>('/v1/today')
export const getWeek = (monday: string) => call<WeekPlan>(`/v1/week?week=${monday}`)
export const listCourses = () => call<Subject[]>('/v1/courses')
export const addCourse = (name: string, weekly_hours: number) => call<Subject>('/v1/courses', 'POST', { name, weekly_hours, level: 'intermediate' })
export const courseView = (id: string) => call<CourseView>(`/v1/courses/${id}/view`)
export const addConcept = (course: string, name: string) => call<{ id: string }>(`/v1/courses/${course}/concepts`, 'POST', { name })
export const listBlocks = (course: string) => call<SourceBlock[]>(`/v1/courses/${course}/blocks`)
export const uploadNotes = (course: string, f: FormData) => call<Upload>(`/v1/courses/${course}/uploads`, 'POST', f)
export const getUpload = (id: string) => call<Upload>(`/v1/uploads/${id}`)
export const listPages = (upload: string) => call<Page[]>(`/v1/uploads/${upload}/pages`)
export const getPage = (id: string) => call<Page>(`/v1/pages/${id}`)
export const savePage = (id: string, edited_text: string) => call<Page>(`/v1/pages/${id}`, 'PATCH', { edited_text })
export const markReady = (id: string, ready: boolean) => call<Page>(`/v1/pages/${id}/${ready ? 'ready' : 'unready'}`, 'POST')
export const listClaims = (concept: string) => call<Claim[]>(`/v1/concepts/${concept}/claims`)
export const addClaim = (concept: string, c: ClaimInput) => call<Claim>(`/v1/concepts/${concept}/claims`, 'POST', c)
export const startSession = (concept_id: string) => call<{ id: string }>('/v1/teach/sessions', 'POST', { concept_id })
export const finish = (id: string) => call<Report>(`/v1/teach/sessions/${id}/finish`, 'POST')
export const saveSelfGrades = (id: string, grades: { claim_id: string; rating: SelfRating }[]) =>
  call<void>(`/v1/teach/sessions/${id}/self-grades`, 'POST', { grades })
export const dispute = (id: string, claim_id: string | null, note: string) =>
  call<unknown>(`/v1/teach/sessions/${id}/disputes`, 'POST', { claim_id, note })

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
  const r = await fetch(`/v1/teach/sessions/${id}/turns`, {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ transcript, latex: transcript }),
  })
  if (r.status === 401) window.dispatchEvent(new Event(SIGNED_OUT))
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
