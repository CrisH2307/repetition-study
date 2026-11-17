import { useEffect, useState } from 'react'
import { Math } from './Math'
import { Claim, Label, Report, SelfRating, dispute, finish, listClaims, saveSelfGrades, sendTurn } from './api'

export const WORDS: Record<Label, string> = {
  supported: 'Correct',
  partial: 'Partly right',
  contradicted: 'Not right',
  absent: 'Not mentioned',
}
const RATINGS: [SelfRating, string][] = [['right', 'Right'], ['not_sure', 'Not sure'], ['wrong', 'Wrong']]
const RATING_WORD = Object.fromEntries(RATINGS) as Record<SelfRating, string>

// Files import a component named Math, so avoid the global Math here
const fmtTime = (n: number) => `${(n - (n % 60)) / 60}:${String(n % 60).padStart(2, '0')}`
const day = (iso: string) => new Date(iso).toLocaleDateString(undefined, { weekday: 'long', month: 'long', day: 'numeric' })

export interface Session { id: string; concept_id: string; name: string }
type Stage = { s: 'teach' } | { s: 'grade' } | { s: 'report'; r: Report; mine: Record<string, SelfRating> }

/** Teach, then Self-grade, then Report. No main menu while this runs. */
export function TeachFlow({ session, onExit }: { session: Session; onExit: () => void }) {
  const [stage, setStage] = useState<Stage>({ s: 'teach' })
  const [points, setPoints] = useState<string[]>([])
  if (stage.s === 'teach') return <Teach {...session} points={points} onSent={(t) => setPoints((p) => [...p, t])} onFinish={() => setStage({ s: 'grade' })} />
  if (stage.s === 'grade') return <Grade {...session} points={points} onDone={(r, mine) => setStage({ s: 'report', r, mine })} />
  return <ReportView r={stage.r} id={session.id} mine={stage.mine} onAgain={onExit} />
}

function Teach({ id, name, points, onSent, onFinish }: Session & { points: string[]; onSent: (t: string) => void; onFinish: () => void }) {
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
      <h1>{name}</h1>
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

/** Self-grade per key idea of the topic. Saved, but changes no review date. */
function Grade({ id, concept_id, points, onDone }: Session & { points: string[]; onDone: (r: Report, mine: Record<string, SelfRating>) => void }) {
  const [claims, setClaims] = useState<Claim[] | null>(null)
  const [on, setOn] = useState<Record<string, SelfRating>>({})
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')
  useEffect(() => { listClaims(concept_id).then(setClaims, () => setClaims([])) }, [concept_id])
  // With no key ideas to grade, fall back to the points the student sent; those are not saved.
  const rows = claims?.length ? claims.map((c) => ({ key: c.id, text: c.text })) : points.map((p, i) => ({ key: `p${i}`, text: p }))
  const go = async () => {
    setBusy(true); setErr('')
    const grades = (claims ?? []).filter((c) => on[c.id]).map((c) => ({ claim_id: c.id, rating: on[c.id] }))
    if (grades.length) await saveSelfGrades(id, grades).catch(() => {}) // a lost guess must not block the check
    try { onDone(await finish(id), on) }
    catch { setErr('We could not check your explanation. Nothing was lost. Please try again.'); setBusy(false) }
  }
  if (claims === null) return <p className="loading">Loading...</p>
  return (
    <>
      <h1>How do you feel about this concept?</h1>
      <p>For each point, say whether you think you got it right, are not sure, or got it wrong.</p>
      <p className="note">Only for you. Your review dates do not change.</p>
      <ul className="plain">
        {rows.map((row, i) => (
          <li key={row.key} className="card">
            <fieldset className="grade">
              <legend>Point {i + 1}: <Math text={row.text} /></legend>
              {RATINGS.map(([v, w]) => (
                <label key={v}>
                  <input type="radio" name={row.key} checked={on[row.key] === v} onChange={() => setOn({ ...on, [row.key]: v })} />
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

/** The short reason under the date: the server's words if it sends any, else counted from the labels. */
export function reasonLine(r: Report): string {
  const said = r.schedule?.payload?.reason
  if (typeof said === 'string' && said) return said
  const n = r.results.length
  const gaps = r.results.filter((c) => c.label !== 'supported').length
  if (!n) return ''
  return gaps ? `${gaps} of ${n} points were not right or missing.` : n === 1 ? 'The point matched the course notes.' : `All ${n} points matched the course notes.`
}

function ReportView({ r: first, id, mine, onAgain }: { r: Report; id: string; mine: Record<string, SelfRating>; onAgain: () => void }) {
  const [r, setR] = useState(first)
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')
  const retry = async () => {
    setBusy(true); setErr('')
    try { setR(await finish(id)) } catch { setErr('The check is still not working. Your explanation is saved. Please try again later.') }
    setBusy(false)
  }
  const unchecked = r.status === 'unchecked'
  const right = Object.values(mine).filter((g) => g === 'right').length
  const found = r.results.filter((c) => c.label === 'supported').length
  const before = r.schedule && new Date(r.schedule.due_before).getFullYear() > 2000 ? r.schedule.due_before : null
  return (
    <>
      <h1>Your results</h1>
      {unchecked ? (
        <div className="banner" role="status">
          <strong>Not checked yet.</strong> Nothing changed: your review date stays the same and your explanation is saved.
          <div className="row"><button disabled={busy} onClick={retry}>{busy ? 'Trying...' : 'Try the check again'}</button>
            <button className="plain" onClick={onAgain}>Try later</button></div>
        </div>
      ) : r.schedule && (
        <div className="card" role="status">
          Next review: {before && <><s>{day(before)}</s> &rarr; </>}<strong>{day(r.schedule.due_after)}</strong>
          <div className="note">{reasonLine(r)}</div>
        </div>
      )}
      {err && <div className="error" role="alert">{err}</div>}
      <p>Your guess: you marked {right} {right === 1 ? 'point' : 'points'} as right.{!unchecked && <> The check found {found} correct.</>}</p>
      <ul className="plain">
        {r.results.map((c, i) => (
          <li key={i} className="card">
            <div className="row">
              {unchecked ? <span className="label absent">Not checked</span> : <span className={`label ${c.label}`}>{WORDS[c.label]}</span>}
              {c.claim_id && mine[c.claim_id] && <small>Your guess: {RATING_WORD[mine[c.claim_id]]}</small>}
            </div>
            <p>{c.student_text ? <>You said: <Math text={c.student_text} /></> : 'You did not mention this idea.'}</p>
            {!unchecked && c.correction && <p><strong>Correction:</strong> <Math text={c.correction} /></p>}
            {!unchecked && c.citation && <blockquote>&ldquo;{c.citation.quote}&rdquo;<br /><small>{c.citation.locator}</small></blockquote>}
            {!unchecked && <Dispute id={id} claim={c.claim_id} />}
          </li>
        ))}
      </ul>
      {!unchecked && <button onClick={onAgain}>Back to Today</button>}
    </>
  )
}

function Dispute({ id, claim }: { id: string; claim: string | null }) {
  const [open, setOpen] = useState(false)
  const [note, setNote] = useState('')
  const [state, setState] = useState<'' | 'busy' | 'sent' | 'failed'>('')
  if (state === 'sent') return <p className="note" role="status">Thanks. A person will read this. Your review date stays as shown.</p>
  if (!open) return <button className="plain" onClick={() => setOpen(true)}>I think this check is wrong</button>
  const send = async () => {
    setState('busy')
    try { await dispute(id, claim, note.trim()); setState('sent') } catch { setState('failed') }
  }
  return (
    <div>
      <label>What looks wrong? (you can leave this empty)
        <textarea className="short" value={note} onChange={(e) => setNote(e.target.value)} /></label>
      {state === 'failed' && <div className="error" role="alert">We could not send this. Please try again.</div>}
      <div className="row">
        <button disabled={state === 'busy'} onClick={send}>Send to a person</button>
        <button className="plain" onClick={() => setOpen(false)}>Cancel</button>
      </div>
    </div>
  )
}
