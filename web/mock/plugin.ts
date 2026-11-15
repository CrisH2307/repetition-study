import type { Plugin } from 'vite'

const C1 = '00000000-0000-0000-0000-00000000f001'
const C2 = '00000000-0000-0000-0000-00000000f002'
const day = 86400000
const iso = (n: number) => new Date(Date.now() + n * day).toISOString()
let lastConcept = C1 // the mock remembers nothing else

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
  results: concept === C2 ? [] : [
    { claim_id: 'c1', student_text: 'The density is $a e^{-at}$ for $t \\ge 0$.', label: 'supported',
      citation: { source_id: 's1', quote: 'has the density a e^(-a t) for t at least zero.', locator: 'page 1' },
      correction: null, probe_hint: null },
    { claim_id: 'c2', student_text: 'The chance X finishes first is $b/(a+b)$.', label: 'contradicted',
      citation: { source_id: 's1', quote: 'the chance that X finishes before Y is a / (a + b).', locator: 'page 1' },
      correction: 'X finishes first with chance $a/(a+b)$, not $b/(a+b)$.', probe_hint: null },
    { claim_id: null, student_text: '', label: 'absent', citation: null, correction: null, probe_hint: null },
  ],
})

export function mockApi(): Plugin {
  return {
    name: 'mock-api',
    configureServer(server) {
      server.middlewares.use('/v1', (req, res) => {
        const url = req.url ?? ''
        const json = (code: number, body: unknown) => {
          res.statusCode = code
          res.setHeader('Content-Type', 'application/json')
          res.end(JSON.stringify(body))
        }
        if (url.startsWith('/subjects')) return json(200, [{ id: 's1', name: 'Fixture course', weekly_hours: 3, level: 'intermediate', weight: 1 }])
        if (url.startsWith('/reviews/due')) {
          return json(200, [
            { concept_id: C1, concept_name: 'Exponential races', card_id: null, card: null, due_at: iso(0), reason: 'Due today' },
            { concept_id: C2, concept_name: 'Practice: pretend the checker is down', card_id: null, card: null, due_at: iso(0), reason: 'Shows the "could not check" screen' },
          ])
        }
        if (req.method === 'POST' && url === '/teach/sessions') {
          let b = ''
          req.on('data', (d) => (b += d))
          req.on('end', () => {
            lastConcept = JSON.parse(b || '{}').concept_id ?? C1
            json(201, { id: 'mock-session', concept_id: lastConcept })
          })
          return
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
        if (url.endsWith('/finish') || url.endsWith('/report')) return json(200, report(lastConcept))
        json(404, { error: 'not found' })
      })
    },
  }
}
