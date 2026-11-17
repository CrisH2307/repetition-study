import { FormEvent, useEffect, useState } from 'react'
import { FeedbackPrompt, User, acceptConsent, deleteAccount, getFeedback, sendFeedback } from './api'

export const CONSENT_VERSION = 'draft-1'

/** Small per-browser flags. Storage can be blocked, so every read and write is guarded. */
export const flag = {
  get: (k: string) => { try { return localStorage.getItem(`sr:${k}`) } catch { return null } },
  set: (k: string, v: string) => { try { localStorage.setItem(`sr:${k}`, v) } catch { /* blocked storage: shown again next time */ } },
}

const msg = (e: unknown) => (e instanceof Error ? e.message : 'Something went wrong. Please try again.')

// ponytail: the API has no "did this user accept" read, so the browser remembers it.
// A new browser asks again; accepting again is harmless (the server keeps the first time).
export function Consent({ user, onDone, onSignOut }: { user: User; onDone: () => void; onSignOut: () => void }) {
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')
  const agree = async () => {
    setBusy(true); setErr('')
    try { await acceptConsent(CONSENT_VERSION); flag.set(`consent:${user.id}`, CONSENT_VERSION); onDone() }
    catch (e) { setErr(msg(e)); setBusy(false) }
  }
  return (
    <main>
      <h1>Before you start</h1>
      <div className="card">
        <ul>
          <li>We store what you add here: courses, notes, explanations and review dates.</li>
          <li>Your explanations go to an AI model, which asks questions and checks them against your notes.</li>
          <li>Other students never see your data.</li>
          <li>In Settings you can download your data or delete your account. Deleting removes everything.</li>
        </ul>
        <p className="note">Version {CONSENT_VERSION}. Draft, not reviewed yet.</p>
        {err && <div className="error" role="alert">{err}</div>}
        <div className="row">
          <button disabled={busy} onClick={agree}>I agree</button>
          <button className="plain" onClick={onSignOut}>Sign out</button>
        </div>
      </div>
    </main>
  )
}

export function Guide({ user }: { user: User }) {
  const [hidden, setHidden] = useState(() => !!flag.get(`guide:${user.id}`))
  if (hidden) return null
  return (
    <section className="card" aria-label="First steps">
      <h3>First steps</h3>
      <ol>
        <li><a href="#/courses">Add a course</a></li>
        <li>Upload your notes on the course page</li>
        <li>Check each page and mark it Ready</li>
      </ol>
      <button className="plain" onClick={() => { flag.set(`guide:${user.id}`, '1'); setHidden(true) }}>Got it</button>
    </section>
  )
}

/** One optional question a week (D40). Hidden once answered or dismissed for that week. */
export function FeedbackCard() {
  const [p, setP] = useState<FeedbackPrompt | null>(null)
  const [answer, setAnswer] = useState('')
  const [state, setState] = useState<'open' | 'sent' | 'hidden'>('open')
  const [err, setErr] = useState('')
  useEffect(() => { getFeedback().then(setP, () => setP(null)) }, [])
  if (!p || typeof p.question !== 'string' || p.answer !== null || state === 'hidden' || flag.get(`feedback:${p.week_start}`)) return null
  if (state === 'sent') return <div className="card" role="status">Thanks for your answer.</div>
  const send = async (e: FormEvent) => {
    e.preventDefault(); setErr('')
    try { await sendFeedback(answer.trim()); setState('sent') } catch (e) { setErr(msg(e)) }
  }
  return (
    <form className="card" aria-label="Weekly question" onSubmit={send}>
      <label>{p.question} <small>(optional)</small><textarea className="short" value={answer} onChange={(e) => setAnswer(e.target.value)} /></label>
      {err && <div className="error" role="alert">{err}</div>}
      <div className="row">
        <button disabled={!answer.trim()}>Send</button>
        <button type="button" className="plain" onClick={() => { flag.set(`feedback:${p.week_start}`, '1'); setState('hidden') }}>Not now</button>
      </div>
    </form>
  )
}

const CONFIRM = 'delete my account'

export function Settings({ onDeleted }: { onDeleted: () => void }) {
  const [typed, setTyped] = useState('')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')
  const del = async (e: FormEvent) => {
    e.preventDefault(); setBusy(true); setErr('')
    try { await deleteAccount(); onDeleted() } catch (e) { setErr(msg(e)); setBusy(false) }
  }
  return (
    <>
      <h1>Settings</h1>
      <section className="card">
        <h3>Your data</h3>
        <p>One file with everything you added and every review date change.</p>
        <a className="btn" href="/v1/me/export" download="space-repetition-data.json">Download my data</a>
      </section>
      <form className="card" onSubmit={del} aria-label="Delete my account">
        <h3>Delete my account</h3>
        <p>This removes your courses, notes, explanations and review history. It cannot be undone.</p>
        <label>Type <strong>{CONFIRM}</strong> to confirm <input value={typed} onChange={(e) => setTyped(e.target.value)} autoComplete="off" /></label>
        {err && <div className="error" role="alert">{err}</div>}
        <button className="danger" disabled={busy || typed.trim().toLowerCase() !== CONFIRM}>Delete my account</button>
      </form>
    </>
  )
}
