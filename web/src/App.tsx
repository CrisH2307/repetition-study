import { useEffect, useState } from 'react'
import { Math } from './Math'
import { DueItem, Label, Report, finish, listConcepts, sendTurn, startSession } from './api'

const WORDS: Record<Label, string> = {
  supported: 'Correct',
  partial: 'Partly right',
  contradicted: 'Not right',
  absent: 'Not mentioned',
}
// App imports a component named Math, so avoid the global Math here
const fmtTime = (n: number) => `${(n - (n % 60)) / 60}:${String(n % 60).padStart(2, '0')}`
const day = (iso: string) => new Date(iso).toLocaleDateString(undefined, { weekday: 'long', month: 'long', day: 'numeric' })

type Stage = { s: 'pick' } | { s: 'teach'; id: string; name: string } | { s: 'grade'; id: string; name: string } | { s: 'report'; r: Report; id: string; ticks: number; total: number }

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
        <Grade {...stage} points={points} onDone={(r, ticks) => setStage({ s: 'report', r, id: stage.id, ticks, total: points.length })} />
      )}
      {stage.s === 'report' && <ReportView r={stage.r} id={stage.id} guess={stage.ticks} total={stage.total} onAgain={() => setStage({ s: 'pick' })} />}
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
  const [secs, setSecs] = useState(0)
  useEffect(() => { const t = setInterval(() => setSecs((n) => n + 1), 1000); return () => clearInterval(t) }, [])
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
      <p className="row"><small>Course notes are hidden until you finish.</small>
        <small role="timer" aria-label="Time spent">Time: {fmtTime(secs)}</small></p>
      <div className="card question" aria-live="polite">
        <strong>The AI student asks:</strong>
        <div>{streaming && !q ? 'Thinking...' : q ? <Math text={q} /> : 'Start by explaining the topic below. Use $...$ around formulas.'}</div>
        <div className="note">The AI student only asks questions. It never corrects or teaches.</div>
      </div>
      {err && <div className="error" role="alert">{err}</div>}
      <label htmlFor="ed"><strong>Your explanation</strong></label>
      <textarea id="ed" value={text} onChange={(e) => setText(e.target.value)} placeholder="Example: The tail is $P(T>t)=e^{-at}$." />
      <div className="note">Put formulas between dollar signs, like $x^2$.</div>
      <div className="card preview" aria-label="Preview"><small>Preview</small><div><Math text={text || '(your formulas appear here)'} /></div></div>
      <div className="row">
        <button disabled={streaming || !text.trim()} onClick={send}>Send</button>
        <button className="plain" disabled={streaming || points.length === 0} onClick={onFinish}>I am done, check my explanation</button>
      </div>
      {points.length === 0 && <small>Send at least one explanation before finishing.</small>}
    </>
  )
}

type Guess = 'right' | 'unsure' | 'wrong'
const GUESS: [Guess, string][] = [['right', 'Right'], ['unsure', 'Not sure'], ['wrong', 'Wrong']]

function Grade({ id, points, onDone }: { id: string; name: string; points: string[]; onDone: (r: Report, ticks: number) => void }) {
  const [on, setOn] = useState<Guess[]>(points.map(() => 'unsure'))
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')
  const go = async () => {
    setBusy(true); setErr('')
    try { onDone(await finish(id), on.filter((g) => g === 'right').length) }
    catch { setErr('We could not check your explanation. Nothing was lost. Please try again.'); setBusy(false) }
  }
  return (
    <>
      <h2>How do you feel about this concept?</h2>
      <p>For each point, say whether you think it was right, not sure, or wrong.</p>
      <p className="note">Only for you. Your review dates do not change.</p>
      <ul className="plain">
        {points.map((p, i) => (
          <li key={i} className="card">
            <fieldset className="grade">
              <legend>Point {i + 1}: <Math text={p} /></legend>
              {GUESS.map(([v, w]) => (
                <label key={v}>
                  <input type="radio" name={`g${i}`} checked={on[i] === v} onChange={() => setOn(on.map((x, j) => (j === i ? v : x)))} />
                  {w}
                </label>
              ))}
            </fieldset>
          </li>
        ))}
      </ul>
      {err && <div className="error" role="alert">{err}</div>}
      <button disabled={busy} onClick={go}>{busy ? 'Checking...' : 'Show my results'}</button>
    </>
  )
}

function ReportView({ r: first, id, guess, total, onAgain }: { r: Report; id: string; guess: number; total: number; onAgain: () => void }) {
  const [r, setR] = useState(first)
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')
  const retry = async () => {
    setBusy(true); setErr('')
    try { setR(await finish(id)) } catch { setErr('The check is still not working. Your explanation is saved. Please try again later.') }
    setBusy(false)
  }
  const unchecked = r.status === 'unchecked'
  const found = r.results.filter((c) => c.label === 'supported').length
  const before = r.schedule && new Date(r.schedule.due_before).getFullYear() > 2000 ? r.schedule.due_before : null
  return (
    <>
      <h2>Your results</h2>
      {unchecked ? (
        <div className="banner" role="status">
          <strong>Not checked yet.</strong> Nothing changed: your review date stays the same and your explanation is saved.
          <div className="row"><button disabled={busy} onClick={retry}>{busy ? 'Trying...' : 'Try the check again'}</button>
            <button className="plain" onClick={onAgain}>Try later</button></div>
        </div>
      ) : r.schedule && (
        <div className="card" role="status">
          Next review: {before && <><s>{day(before)}</s> &rarr; </>}<strong>{day(r.schedule.due_after)}</strong>
        </div>
      )}
      {err && <div className="error" role="alert">{err}</div>}
      <p>Your guess: you marked {guess} of {total} points as right.{!unchecked && <> The check found {found} correct.</>}</p>
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
      {!unchecked && <button onClick={onAgain}>Pick another topic</button>}
    </>
  )
}
