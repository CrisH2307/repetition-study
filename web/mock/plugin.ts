import type { Plugin } from 'vite'
import type { IncomingMessage } from 'node:http'

// In-memory fake of api/openapi.yaml v4 for `npm run dev` and the mock Playwright test.
// It mimics the server's answers, not its rules: the real quote check lives in internal/cite.
const S1 = '00000000-0000-0000-0000-0000000000s1'
const C1 = '00000000-0000-0000-0000-00000000f001'
const C2 = '00000000-0000-0000-0000-00000000f002'
const day = 86400000
const iso = (n: number) => new Date(Date.now() + n * day).toISOString()
const PNG = Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFBQIAX8jx0gAAAABJRU5ErkJggg==', 'base64')

type Rec = Record<string, any>
let n = 0
const newId = (p: string) => `${p}-${++n}`
let signedIn = true
let lastConcept = C1
const course = { id: S1, name: 'Fixture course', weekly_hours: 3, level: 'intermediate', weight: 1 }
const courses: Rec[] = [course]
const concepts: Rec[] = [
  { id: C1, subject_id: S1, name: 'Exponential races' },
  { id: C2, subject_id: S1, name: 'Practice: pretend the checker is down' },
]
const claims: Rec[] = [
  { id: 'c1', concept_id: C1, text: 'The density is $a e^{-at}$ for $t \\ge 0$.', source_id: 'src1', locator: 'page 1', required: true, kind: 'definition' },
  { id: 'c2', concept_id: C1, text: 'X finishes first with chance $a/(a+b)$.', source_id: 'src1', locator: 'page 1', required: true, kind: 'statement' },
]
const uploads: Rec[] = []
const pages: Rec[] = []
const blocks: Rec[] = []
const imports: Rec[] = []
const shares: Rec[] = []
const pack = (instructor: boolean) => ({ course_name: 'Shared course', owner_name: instructor ? 'Prof. Rivera' : 'Sam', instructor, sources: instructor ? ['Week 1 notes.pdf'] : [],
  concepts: [{ name: 'Exponential races', claims: [
    { text: 'X finishes first with chance $a/(a+b)$.', kind: 'statement', required: true },
    { text: 'The minimum has rate $a+b$.', kind: 'statement', required: true }] }] })
const consents: Rec[] = []
let feedback: Rec | null = null
const QUESTION = 'How would you feel if you could no longer use this?'

/** Split one CSV line on commas outside double quotes. Mock only; the server has its own parser. */
const csvLine = (l: string) => (l.match(/("([^"]|"")*"|[^,]*)(,|$)/g) ?? []).slice(0, -1)
  .map((f) => f.replace(/,$/, '').replace(/^"(.*)"$/s, '$1').replace(/""/g, '"').trim())

const upStatus = (u: Rec) => {
  const ps = pages.filter((p) => p.upload_id === u.id)
  u.pages_total = ps.length
  u.pages_ready = ps.filter((p) => p.status === 'ready').length
  if (u.status !== 'reading') u.status = ps.length && u.pages_ready === ps.length ? 'ready' : 'needs_checking'
  return u
}

const report = (concept: string) => ({
  session_id: 'mock-session',
  concept_id: concept,
  status: concept === C2 ? 'unchecked' : 'checked',
  schedule:
    concept === C2
      ? null
      : { id: 'e1', event_type: 'teach_back', concept_id: concept, card_id: null, grade: null,
          due_before: iso(6), due_after: iso(1), stability_before: 5, stability_after: 2,
          payload: {}, created_at: iso(0), session_id: 'mock-session', claim_id: null,
          self_grade: null, checker_model: 'mock', checker_version: '0' },
  results: [
    { claim_id: 'c1', student_text: 'The density is $a e^{-at}$ for $t \\ge 0$.', label: 'supported',
      citation: { source_id: 'src1', quote: 'has the density a e^(-a t) for t at least zero.', locator: 'page 1' },
      correction: null, probe_hint: null },
    { claim_id: 'c2', student_text: 'The chance X finishes first is $b/(a+b)$.', label: 'contradicted',
      citation: { source_id: 'src1', quote: 'the chance that X finishes before Y is a / (a + b).', locator: 'page 1' },
      correction: 'X finishes first with chance $a/(a+b)$, not $b/(a+b)$.', probe_hint: null },
    { claim_id: null, student_text: '', label: 'absent', citation: null, correction: null, probe_hint: null },
  ].map((r) => (concept === C2 ? { ...r, label: 'absent', citation: null, correction: null } : r)),
})

const item = (c: Rec, i: number) => ({
  concept_id: c.id, concept_name: c.name, subject_id: c.subject_id, kind: i % 2 ? 'recall' : 'teach',
  minutes: i % 2 ? 5 : 15, due_at: iso(i), reason: i ? 'Due soon' : 'New topic: not taught yet',
})
const hours = (c: Rec, i: number) => ({ subject_id: c.id, name: c.name, hours_used: 1.2, hours_budget: c.weekly_hours, reason: i ? '' : 'Up 0.5 h: 1 gap on Monday' })

async function body(req: IncomingMessage): Promise<Buffer> {
  const parts: Buffer[] = []
  for await (const d of req) parts.push(d as Buffer)
  return Buffer.concat(parts)
}

export function mockApi(): Plugin {
  return {
    name: 'mock-api',
    configureServer(server) {
      server.middlewares.use('/v1', async (req, res) => {
        const url = req.url ?? ''
        const m = req.method ?? 'GET'
        const json = (code: number, b?: unknown) => {
          res.statusCode = code
          if (b === undefined) return res.end()
          res.setHeader('Content-Type', 'application/json')
          res.end(JSON.stringify(b))
        }
        const say = (code: number, error: string) => json(code, { error })
        const raw = m === 'GET' ? Buffer.alloc(0) : await body(req)
        const b = () => JSON.parse(raw.toString() || '{}')
        const seg = url.split('?')[0].split('/').filter(Boolean) // "/pages/x/ready" -> ["pages","x","ready"]

        if (m === 'POST' && url === '/auth/redeem') {
          if (b().token === 'used') return say(404, 'This invite link does not work.')
          signedIn = true
          return json(200, { id: 'u1', name: 'Dino', time_zone: 'America/Toronto' })
        }
        if (!signedIn) return say(401, 'Please sign in.')
        if (url === '/auth/sign-out') { signedIn = false; return json(204) }
        if (url === '/me' && m === 'DELETE') {
          if (b().confirm !== 'delete my account') return say(400, 'Please type delete my account to confirm.')
          signedIn = false; return json(204)
        }
        if (url === '/me/export') {
          res.setHeader('Content-Disposition', 'attachment; filename="space-repetition-data.json"')
          return json(200, { exported_at: iso(0), user: { id: 'u1', name: 'Dino', time_zone: 'America/Toronto' }, consents, feedback: feedback ? [feedback] : [],
            courses, concepts, uploads, pages, sessions: [], turns: [], taught_claims: [], reports: [], self_grades: [], disputes: [], review_events: [], imports })
        }
        if (url === '/consent') {
          if (!b().text_version) return say(400, 'Missing text_version.')
          const c = consents.find((x) => x.text_version === b().text_version) ?? { text_version: b().text_version, accepted_at: iso(0) }
          if (!consents.includes(c)) consents.push(c)
          return json(201, c)
        }
        if (url === '/feedback') {
          const week_start = iso(-((new Date().getDay() + 6) % 7))
          if (m === 'POST') { feedback = { week_start, question: QUESTION, answer: b().answer, created_at: iso(0) }; return json(201, feedback) }
          return json(200, { week_start: feedback?.week_start ?? week_start, question: QUESTION, answer: feedback?.answer ?? null })
        }
        if (seg[0] === 'shared') {
          if (seg[1] === 'gone') return say(404, 'This link does not work.')
          if (seg[2] === 'import' && m === 'POST') {
            return json(200, { subject_id: b().subject_id, concepts_added: 1, claims_added: 1, uploads_copied: 0,
              needs_quote: [{ concept_id: C1, claim: pack(false).concepts[0].claims[1] }] })
          }
          const live = ['demo', 'instructor'].includes(seg[1]) || shares.some((x) => x.token === seg[1] && !x.revoked_at)
          return live ? json(200, pack(seg[1] === 'instructor')) : say(404, 'This link does not work.')
        }
        if (seg[0] === 'shares' && m === 'DELETE') {
          const x = shares.find((y) => y.id === seg[1])
          if (!x) return say(404, 'We could not find that link.')
          x.revoked_at = iso(0); return json(204)
        }
        if (seg[0] === 'imports') {
          const j = imports.find((x) => x.id === seg[1])
          return j ? json(200, j) : say(404, 'We could not find that import.')
        }
        if (url === '/me') return json(200, { id: 'u1', name: 'Dino', time_zone: 'America/Toronto', consent_version: consents.at(-1)?.text_version ?? null })

        if (url === '/today') {
          return json(200, {
            date: iso(0),
            courses: courses.map((c) => {
              const items = concepts.filter((k) => k.subject_id === c.id).map(item)
              return { subject_id: c.id, name: c.name, minutes: items.reduce((s, i) => s + i.minutes, 0), items }
            }),
            study_time: courses.map(hours),
          })
        }
        if (seg[0] === 'week') {
          const start = new Date(`${new URL(url, 'http://x').searchParams.get('week')}T00:00:00`)
          return json(200, {
            week_start: start.toISOString(),
            courses: courses.map(hours),
            days: [0, 1, 2, 3, 4, 5, 6].map((d) => ({
              date: new Date(start.getTime() + d * day).toISOString(),
              items: d % 2 ? [] : concepts.slice(0, 1).map(item),
            })),
            notes: ['Exponential races moved to Tuesday: 1 point was not right.'],
          })
        }
        if (url === '/courses') {
          if (m === 'POST') { const c = { ...b(), id: newId('course'), weight: 1 }; courses.push(c); return json(201, c) }
          return json(200, courses)
        }
        if (seg[0] === 'courses') {
          const c = courses.find((x) => x.id === seg[1])
          if (!c) return say(404, 'We could not find that course.')
          if (seg[2] === 'view') {
            const notes = uploads.filter((u) => u.subject_id === c.id).map(upStatus)
            const firstPage = pages[0]
            return json(200, {
              course: c, notes,
              cards: concepts.filter((k) => k.subject_id === c.id).map((k, i) => ({
                concept_id: k.id, name: k.name, recall: i ? null : 0.8, explanation: i ? null : 0.5,
                next_due: i ? null : iso(1), claims: claims.filter((x) => x.concept_id === k.id),
              })),
              mistakes: c.id !== S1 ? [] : [{ session_id: 'mock-session', concept_id: C1, claim_id: 'c2', student_text: 'The chance X finishes first is $b/(a+b)$.',
                label: 'contradicted', quote: 'the chance that X finishes before Y is a / (a + b).', source_id: 'src1',
                page_id: firstPage?.id ?? null, page: firstPage ? 1 : null, created_at: iso(-1) }],
            })
          }
          if (seg[2] === 'concepts' && m === 'POST') {
            const k = { id: newId('concept'), subject_id: c.id, name: b().name }
            concepts.push(k); return json(201, { ...k, claims: [] })
          }
          if (seg[2] === 'share' && m === 'POST') {
            const x = { id: newId('share'), subject_id: c.id, includes_sources: false, created_at: iso(0), revoked_at: null, token: `tok${n}` }
            shares.push(x); return json(201, x)
          }
          if (seg[2] === 'blocks') return json(200, blocks.filter((x) => x.subject_id === c.id))
          if (seg[2] === 'uploads' && m === 'POST') {
            const f = await new Request('http://x', { method: 'POST', headers: { 'content-type': req.headers['content-type'] ?? '' }, body: new Uint8Array(raw) }).formData()
            const file = f.get('file') as File | null
            const text = (f.get('text') as string | null) ?? ''
            if (!file && !text.trim()) return say(400, 'Please add a PDF or paste some text.')
            const u: Rec = { id: newId('upload'), subject_id: c.id, filename: f.get('filename'), status: 'reading', error: '', source_id: newId('src'), pages_total: 0, pages_ready: 0, created_at: iso(0) }
            uploads.push(u)
            setTimeout(() => { // pretend to read the file in the background
              const texts = file ? ['Page one text read from the PDF.', 'Page two text read from the PDF.'] : [text]
              texts.forEach((t, i) => pages.push({ id: newId('page'), upload_id: u.id, number: i + 1, raw_text: t, edited_text: t, status: 'needs_checking', has_image: !!file, updated_at: iso(0) }))
              u.status = 'needs_checking'; upStatus(u)
            }, 1000)
            return json(202, u)
          }
          if (seg[2] === 'import' && m === 'POST') {
            const f = await new Request('http://x', { method: 'POST', headers: { 'content-type': req.headers['content-type'] ?? '' }, body: new Uint8Array(raw) }).formData()
            const file = f.get('file') as File | null
            if (!file) return say(400, 'Please choose a CSV file.')
            const lines = (await file.text()).split(/\r?\n/).filter((l) => l.trim())
            const j: Rec = { id: newId('import'), subject_id: c.id, filename: file.name, status: 'running', rows_total: Math.max(lines.length - 1, 0), rows_imported: 0, row_errors: [], created_at: iso(0), finished_at: null }
            imports.push(j)
            setTimeout(() => { // pretend to check each row in the background
              lines.slice(1).forEach((l, i) => {
                const [name, text, kind, quote] = csvLine(l)
                const blk = quote && blocks.find((x) => x.subject_id === c.id && x.text_md.includes(quote))
                if (!name || !text) return j.row_errors.push({ row: i + 2, error: 'Concept and claim are required.' })
                if (!blk) return j.row_errors.push({ row: i + 2, error: 'This quote is not on a Ready page.' })
                let k = concepts.find((x) => x.subject_id === c.id && x.name === name)
                if (!k) concepts.push((k = { id: newId('concept'), subject_id: c.id, name }))
                claims.push({ id: newId('claim'), concept_id: k.id, text, source_id: blk.source_id, locator: `page ${blk.page}`, required: true, kind: kind || 'statement', anchor: { block_id: blk.id, quote } })
                j.rows_imported++
              })
              j.status = 'done'; j.finished_at = iso(0)
            }, 800)
            return json(202, j)
          }
          if (seg[2] === 'uploads') return json(200, uploads.filter((u) => u.subject_id === c.id).map(upStatus))
        }
        if (seg[0] === 'uploads') {
          const u = uploads.find((x) => x.id === seg[1])
          if (!u) return say(404, 'We could not find that file.')
          if (seg[2] === 'pages') return json(200, pages.filter((p) => p.upload_id === u.id))
          return json(200, upStatus(u))
        }
        if (seg[0] === 'pages') {
          const p = pages.find((x) => x.id === seg[1])
          if (!p) return say(404, 'We could not find that page.')
          const u = uploads.find((x) => x.id === p.upload_id)!
          if (seg[2] === 'image') { res.setHeader('Content-Type', 'image/png'); return res.end(PNG) }
          if (seg[2] === 'ready') {
            p.status = 'ready'
            p.edited_text.split(/\n\s*\n/).filter((t: string) => t.trim()).forEach((t: string, i: number) =>
              blocks.push({ id: newId('block'), subject_id: u.subject_id, page_id: p.id, source_id: u.source_id, page: p.number, ordinal: i, kind: 'text', text_md: t.trim(), text_norm: t.trim() }))
            return json(200, p)
          }
          if (seg[2] === 'unready') {
            const mine = blocks.filter((x) => x.page_id === p.id).map((x) => x.id)
            if (claims.some((c) => mine.includes(c.anchor?.block_id))) return say(400, 'A key idea quotes this page, so it has to stay Ready.')
            p.status = 'needs_checking'
            blocks.splice(0, blocks.length, ...blocks.filter((x) => x.page_id !== p.id))
            return json(200, p)
          }
          if (m === 'PATCH') { p.edited_text = b().edited_text; p.updated_at = iso(0) }
          return json(200, p)
        }
        if (seg[0] === 'concepts' && seg[2] === 'claims') {
          if (m === 'POST') {
            const i = b()
            const blk = blocks.find((x) => x.id === i.anchor?.block_id)
            if (!blk || !i.anchor.quote || !blk.text_md.includes(i.anchor.quote)) return say(400, 'These words are not in that passage. Copy them exactly from your notes.')
            const c = { id: newId('claim'), concept_id: seg[1], text: i.text, source_id: blk.source_id, locator: `page ${blk.page}`, required: i.required, kind: i.kind, anchor: i.anchor }
            claims.push(c); return json(201, c)
          }
          return json(200, claims.filter((c) => c.concept_id === seg[1]))
        }
        if (m === 'POST' && url === '/teach/sessions') {
          lastConcept = b().concept_id ?? C1
          return json(201, { id: 'mock-session', concept_id: lastConcept })
        }
        if (url.endsWith('/turns')) {
          res.setHeader('Content-Type', 'text/event-stream')
          const words = 'Why does the tail $P(T>t)$ matter for the minimum of two times?'.split(' ')
          let i = 0
          const t = setInterval(() => {
            if (i < words.length) res.write(`event: token\ndata: ${JSON.stringify({ text: (i ? ' ' : '') + words[i++] })}\n\n`)
            else { res.write('event: done\ndata: {}\n\n'); clearInterval(t); res.end() }
          }, 80)
          return
        }
        if (url.endsWith('/self-grades')) return json(204)
        if (url.endsWith('/disputes')) return json(201, { id: newId('dispute'), session_id: 'mock-session', ...b(), user_id: 'u1', status: 'open', created_at: iso(0) })
        if (url.endsWith('/finish') || url.endsWith('/report')) return json(200, report(lastConcept))
        say(404, 'not found')
      })
    },
  }
}
