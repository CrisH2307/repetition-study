import { useEffect, useState } from 'react'
import { SIGNED_OUT, Today, TodayItem, User, WeekPlan, getToday, getWeek, me, redeem, signOut, startSession } from './api'
import { Session, TeachFlow } from './Teach'
import { Course, Courses, PageCheck } from './Course'
import { Shared } from './Share'
import { CONSENT_VERSION, Consent, FeedbackCard, Guide, Settings, flag } from './Pilot'

const dayWord = { weekday: 'short', month: 'short', day: 'numeric' } as const
const sameDay = (a: Date, b: Date) => a.toDateString() === b.toDateString()

/** "Due today", or "Due Mon, Nov 16"; soon = due within 2 days (or already late). */
export function dueText(iso: string, now = new Date()): { text: string; soon: boolean } {
  const d = new Date(iso)
  const soon = d.getTime() - now.getTime() <= 2 * 86400000
  return { soon, text: sameDay(d, now) || d < now ? 'Due today' : `Due ${d.toLocaleDateString(undefined, dayWord)}` }
}

/** Monday of the week holding `d`, as YYYY-MM-DD in local time. */
export function mondayOf(d = new Date()): string {
  const m = new Date(d.getFullYear(), d.getMonth(), d.getDate() - ((d.getDay() + 6) % 7))
  return `${m.getFullYear()}-${String(m.getMonth() + 1).padStart(2, '0')}-${String(m.getDate()).padStart(2, '0')}`
}

const hours = (h: number) => `${h.toFixed(1)} h`
const route = () => location.hash.slice(2).split('/') // "#/course/abc" -> ["course", "abc"]

export type Start = (concept_id: string, name: string) => Promise<void>

export function App() {
  const [user, setUser] = useState<User | null | 'out' | 'bad-link' | 'down' | 'deleted'>(null)
  const [path, setPath] = useState(route)
  const [session, setSession] = useState<Session | null>(null)
  useEffect(() => {
    const out = () => setUser('out')
    const nav = () => setPath(route())
    window.addEventListener(SIGNED_OUT, out)
    window.addEventListener('hashchange', nav)
    // cmd/invite prints <site>/invite#<token>; the hash keeps the token out of server logs
    const token = location.pathname === '/invite' ? location.hash.slice(1) : ''
    if (token) {
      redeem(token).then(() => me()).then((u) => { history.replaceState(null, '', '/'); setUser(u) }, () => setUser('bad-link'))
    } else me().then(setUser, () => setUser((u) => (u === 'out' ? u : 'down'))) // a 401 already said 'out'
    return () => { window.removeEventListener(SIGNED_OUT, out); window.removeEventListener('hashchange', nav) }
  }, [])

  if (user === null) return <main><p className="loading">Loading...</p></main>
  if (user === 'down') return <main><div className="error" role="alert">We could not reach the server. Please check your connection and refresh the page.</div></main>
  if (user === 'out' || user === 'bad-link' || user === 'deleted') {
    return (
      <main>
        <h1>Space Repetition</h1>
        <div className="card" role="status">
          {user === 'deleted' ? <p><strong>Your account was deleted.</strong></p>
            : user === 'bad-link'
            ? <p><strong>This link does not work.</strong> It may have been used already. Please ask the person who sent it for a new one.</p>
            : <p><strong>You are signed out.</strong> To sign in, open the invite link you were sent.</p>}
        </div>
      </main>
    )
  }

  const out = () => signOut().finally(() => setUser('out'))
  if (user.consent_version !== CONSENT_VERSION) return <Consent user={user} onDone={() => setUser({ ...user, consent_version: CONSENT_VERSION })} onSignOut={out} />

  const start: Start = async (concept_id, name) => setSession({ id: (await startSession(concept_id)).id, concept_id, name })
  if (session) {
    return <main><TeachFlow session={session} onExit={() => { setSession(null); location.hash = '#/' }} /></main>
  }
  const [page, id] = path
  return (
    <>
      <nav className="topbar" aria-label="Main">
        <a href="#/" aria-current={!page || undefined}>Today</a>
        <a href="#/courses" aria-current={['courses', 'course', 'upload', 'page'].includes(page) || undefined}>Courses</a>
        <a href="#/week" aria-current={page === 'week' || undefined}>Week</a>
        <a href="#/settings" aria-current={page === 'settings' || undefined}>Settings</a>
        <button className="plain" onClick={out}>Sign out</button>
      </nav>
      <main>
        {page === 'week' ? <Week />
          : page === 'courses' ? <Courses />
          : page === 'course' ? <Course id={id} start={start} />
          : page === 'shared' ? <Shared token={id} />
          : page === 'upload' ? <PageCheck uploadId={id} />
          : page === 'page' ? <PageCheck pageId={id} />
          : page === 'settings' ? <Settings onDeleted={() => { location.hash = ''; setUser('deleted') }} />
          : <TodayView start={start} user={user} />}
      </main>
    </>
  )
}

/** Start button with its own busy and error state. */
export function StartButton({ id, name, start }: { id: string; name: string; start: Start }) {
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')
  return (
    <span>
      <button disabled={busy} aria-label={`Start ${name}`}
        onClick={() => { setBusy(true); setErr(''); start(id, name).catch(() => { setErr('We could not start. Please try again.'); setBusy(false) }) }}>Start</button>
      {err && <span className="error" role="alert">{err}</span>}
    </span>
  )
}

function HoursBox({ rows, title }: { rows: { subject_id: string; name: string; hours_used: number; hours_budget: number; reason: string }[]; title: string }) {
  return (
    <aside className="card" aria-label={title}>
      <h3>{title}</h3>
      {rows.map((c) => (
        <div key={c.subject_id} className="hours">
          <div className="row"><span>{c.name}</span><strong>{c.hours_used.toFixed(1)} / {hours(c.hours_budget)}</strong></div>
          <progress max={c.hours_budget || 1} value={Math.min(c.hours_used, c.hours_budget || 1)} aria-label={`${c.name} hours`} />
          {c.reason && <div className="note">{c.reason}</div>}
        </div>
      ))}
    </aside>
  )
}

function TodayView({ start, user }: { start: Start; user: User }) {
  const [t, setT] = useState<Today | null>(null)
  const [err, setErr] = useState('')
  useEffect(() => { getToday().then(setT, () => setErr('We could not load today. Please check your connection and refresh the page.')) }, [])
  if (err) return <div className="error" role="alert">{err}</div>
  if (!t) return <p className="loading">Loading...</p>
  return (
    <>
      <h1>{new Date().toLocaleDateString(undefined, { weekday: 'long', month: 'long', day: 'numeric' })}</h1>
      <Guide user={user} />
      <FeedbackCard />
      <div className="split">
        <div>
          {t.courses.length === 0 && <p>Nothing to study today. <a href="#/courses">Add a course</a></p>}
          {t.courses.map((c) => (
            <details key={c.subject_id} className="folder" open>
              <summary>{c.name} <small>{c.minutes} min</small></summary>
              <ul className="plain">
                {c.items.length === 0 && <li className="topic"><small>Nothing due in this course today.</small></li>}
                {c.items.map((it) => <TopicRow key={it.concept_id + it.kind} it={it} start={start} />)}
              </ul>
            </details>
          ))}
        </div>
        {t.study_time.length > 0 && <HoursBox rows={t.study_time} title="Study Time" />}
      </div>
    </>
  )
}

function TopicRow({ it, start }: { it: TodayItem; start: Start }) {
  const due = dueText(it.due_at)
  return (
    <li className="topic">
      <div><strong>{it.concept_name}</strong>
        <div><small>{it.kind === 'teach' ? 'Teach' : 'Recall'}, {it.minutes} min. </small>
          <small className={due.soon ? 'soon' : ''}>{due.text}</small></div>
        {it.reason && <div className="note">{it.reason}</div>}</div>
      <StartButton id={it.concept_id} name={it.concept_name} start={start} />
    </li>
  )
}

function Week() {
  const [w, setW] = useState<WeekPlan | null>(null)
  const [err, setErr] = useState('')
  useEffect(() => { getWeek(mondayOf()).then(setW, () => setErr('We could not load this week. Please refresh the page.')) }, [])
  if (err) return <div className="error" role="alert">{err}</div>
  if (!w) return <p className="loading">Loading...</p>
  const first = new Date(w.week_start)
  const last = new Date(first.getTime() + 6 * 86400000)
  const md = { month: 'short', day: 'numeric' } as const
  const now = new Date()
  return (
    <>
      <h1>{first.toLocaleDateString(undefined, md)} to {last.toLocaleDateString(undefined, md)}</h1>
      <HoursBox rows={w.courses} title="Hours per course" />
      <h2>Schedule:</h2>
      <ul className="plain">
        {w.days.map((d) => {
          const today = sameDay(new Date(d.date), now)
          return (
            <li key={d.date} className={`card${today ? ' today' : ''}`} aria-current={today ? 'date' : undefined}>
              <strong>{new Date(d.date).toLocaleDateString(undefined, { weekday: 'long', ...md })}{today && ' (today)'}</strong>
              {d.items.length === 0 ? <div><small>Nothing planned.</small></div> : (
                <ul>{d.items.map((it) => <li key={it.concept_id + it.kind}>{it.concept_name}: {it.kind === 'teach' ? 'teach' : 'recall'}, {it.minutes} min</li>)}</ul>
              )}
            </li>
          )
        })}
      </ul>
      {w.notes.length > 0 && <><h2>Note:</h2><ul>{w.notes.map((n, i) => <li key={i}>{n}</li>)}</ul></>}
    </>
  )
}
