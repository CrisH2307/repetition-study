import { useEffect, useState } from 'react'
import { Math } from './Math'
import { DueItem, Label, Report, finish, listConcepts, sendTurn, startSession } from './api'

const WORDS: Record<Label, string> = {
  supported: 'Correct',
  partial: 'Partly right',
  contradicted: 'Not right',
  absent: 'Not mentioned',
}
const day = (iso: string) => new Date(iso).toLocaleDateString(undefined, { weekday: 'long', month: 'long', day: 'numeric' })

type Stage = { s: 'pick' } | { s: 'teach'; id: string; name: string } | { s: 'grade'; id: string; name: string } | { s: 'report'; r: Report; ticks: number; total: number }

export function App() {
  const [stage, setStage] = useState<Stage>({ s: 'pick' })
  const [points, setPoints] = useState<string[]>([])
  return (
    <main>
      <h1>Teach the AI student</h1>
      {stage.s === 'pick' && <Pick onStart={(id, name) => { setPoints([]); setStage({ s: 'teach', id, name }) }} />}
      {stage.s === 'teach' && (
        <Teach {...stage} points={points} onSent={(t) => setPoints((p) => [...p, t])}
          onFinish={() => setStage({ s: 'grade', id: stage.id, name: stage.name })} />
      )}
      {stage.s === 'grade' && (
        <Grade {...stage} points={points} onDone={(r, ticks) => setStage({ s: 'report', r, ticks, total: points.length })} />
      )}
      {stage.s === 'report' && <ReportView {...stage} onAgain={() => setStage({ s: 'pick' })} />}
    </main>
  )
}

function Pick({ onStart }: { onStart: (id: string, name: string) => void }) {
  const [items, setItems] = useState<DueItem[] | null>(null)
  const [err, setErr] = useState('')
  const [busy, setBusy] = useState(false)
  useEffect(() => { listConcepts().then(setItems, () => setErr('We could not load the topics. Please check the server is running and refresh the page.')) }, [])
  const go = async (d: DueItem) => {
    setBusy(true)
    try { onStart((await startSession(d.concept_id)).id, d.concept_name) }
    catch { setErr('We could not start the session. Please try again.'); setBusy(false) }
  }
  return (
    <>
      <p>Pick a topic, then explain it in your own words. The AI student will ask you one question at a time. You will not see the course notes until you finish.</p>
      {err && <div className="error" role="alert">{err}</div>}
      {items === null && !err && <p>Loading topics...</p>}
      {items?.length === 0 && <p>There are no topics to practise right now.</p>}
      <ul className="plain">
        {items?.map((d) => (
          <li key={d.concept_id} className="card">
            <strong>{d.concept_name}</strong><div>{d.reason}</div>
            <button disabled={busy} onClick={() => go(d)}>Start teaching</button>
          </li>
        ))}
      </ul>
    </>
  )
}

function Teach({ id, name, points, onSent, onFinish }: { id: string; name: string; points: string[]; onSent: (t: string) => void; onFinish: () => void }) {
  const [text, setText] = useState('')
  const [q, setQ] = useState('')
  const [streaming, setStreaming] = useState(false)
  const [err, setErr] = useState('')
  const send = async () => {
    const t = text.trim()
    if (!t) return
    setErr(''); setQ(''); setStreaming(true)
    let failed = false
    try {
      await sendTurn(id, t, (e) => {
        if (e.event === 'token') setQ((s) => s + e.data.text)
        if (e.event === 'error') failed = true
      })
    } catch { failed = true }
    setStreaming(false)
    if (failed) setErr('The AI student could not answer just now. Your words are kept. Press Send to try again.')
    else { onSent(t); setText('') }
  }
  return (
    <>
      <h2>{name}</h2>
      <div className="card question" aria-live="polite">
        <strong>The AI student asks:</strong>
        <div>{streaming && !q ? 'Thinking...' : q ? <Math text={q} /> : 'Start by explaining the topic below. Use $...$ around formulas.'}</div>
      </div>
      {err && <div className="error" role="alert">{err}</div>}
      <label htmlFor="ed"><strong>Your explanation</strong></label>
      <textarea id="ed" value={text} onChange={(e) => setText(e.target.value)} placeholder="Example: The tail is $P(T>t)=e^{-at}$." />
      <div className="card preview" aria-label="Preview"><small>Preview</small><div><Math text={text || '(your formulas appear here)'} /></div></div>
      <div className="row">
        <button disabled={streaming || !text.trim()} onClick={send}>Send</button>
        <button className="plain" disabled={streaming || points.length === 0} onClick={onFinish}>I am done, check my explanation</button>
      </div>
      {points.length === 0 && <small>Send at least one explanation before finishing.</small>}
    </>
  )
}

function Grade({ id, points, onDone }: { id: string; name: string; points: string[]; onDone: (r: Report, ticks: number) => void }) {
  const [on, setOn] = useState<boolean[]>(points.map(() => false))
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')
  const go = async () => {
    setBusy(true); setErr('')
    try { onDone(await finish(id), on.filter(Boolean).length) }
    catch { setErr('We could not check your explanation. Nothing was lost. Please try again.'); setBusy(false) }
  }
  return (
    <>
      <h2>Before you see the results</h2>
      <p>Tick each point you think you explained correctly. This is just for you to compare with the results.</p>
      <ul className="plain">
        {points.map((p, i) => (
          <li key={i} className="card">
            <label><input type="checkbox" checked={on[i]} onChange={() => setOn(on.map((v, j) => (j === i ? !v : v)))} /> I think this was right</label>
            <div><Math text={p} /></div>
          </li>
        ))}
      </ul>
      {err && <div className="error" role="alert">{err}</div>}
      <button disabled={busy} onClick={go}>{busy ? 'Checking...' : 'Show my results'}</button>
    </>
  )
}

function ReportView({ r, ticks, total, onAgain }: { r: Report; ticks: number; total: number; onAgain: () => void }) {
  return (
    <>
      <h2>Your results</h2>
      <p>You ticked {ticks} of {total} points as right.</p>
      {r.status === 'unchecked' ? (
        <div className="banner" role="status"><strong>We could not check this one.</strong> Nothing changed: your review dates are the same as before. Please try again later.</div>
      ) : r.schedule && (
        <div className="card" role="status">
          Next review: <s>{day(r.schedule.due_before)}</s> &rarr; <strong>{day(r.schedule.due_after)}</strong>
        </div>
      )}
      <ul className="plain">
        {r.results.map((c, i) => (
          <li key={i} className="card">
            <span className={`label ${c.label}`}>{WORDS[c.label]}</span>
            <p>{c.student_text ? <>You said: <Math text={c.student_text} /></> : 'You did not mention this idea.'}</p>
            {c.correction && <p><strong>Correction:</strong> <Math text={c.correction} /></p>}
            {c.citation && <blockquote>&ldquo;{c.citation.quote}&rdquo;<br /><small>{c.citation.locator}</small></blockquote>}
          </li>
        ))}
      </ul>
      <button onClick={onAgain}>Pick another topic</button>
    </>
  )
}
