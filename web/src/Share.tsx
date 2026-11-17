import { useEffect, useState } from 'react'
import { Math } from './Math'
import { NewShare, PackImport, PackPreview, Subject, getShared, importShared, listCourses, makeShare, revokeShare } from './api'

const msg = (e: unknown) => (e instanceof Error ? e.message : 'Something went wrong. Please try again.')

/** Course screen: make a link to this course's key ideas, copy it, turn it off. The token is shown once, so it lives only in this component. */
export function ShareBox({ course }: { course: string }) {
  const [s, setS] = useState<NewShare | null>(null)
  const [done, setDone] = useState('')
  const [err, setErr] = useState('')
  const link = s ? `${location.origin}/#/shared/${s.token}` : ''
  const run = async (f: () => Promise<void>) => { setErr(''); setDone(''); try { await f() } catch (e) { setErr(msg(e)) } }
  return (
    <section className="card" aria-label="Share key ideas">
      <h3>Share key ideas</h3>
      <p className="note">The link has topic names and key ideas only. Your notes and quotes stay private.</p>
      {s ? (
        <>
          <label>Link <input readOnly value={link} onFocus={(e) => e.target.select()} /></label>
          <div className="row">
            <button onClick={() => run(async () => { await navigator.clipboard.writeText(link); setDone('Link copied.') })}>Copy link</button>
            <button className="plain" onClick={() => run(async () => { await revokeShare(s.id); setS(null); setDone('Link turned off.') })}>Turn off link</button>
          </div>
        </>
      ) : <button onClick={() => run(async () => setS(await makeShare(course)))}>Make a link</button>}
      <div role="status">{done}</div>
      {err && <div className="error" role="alert">{err}</div>}
    </section>
  )
}

/** Opened from a share link. Lists what it holds, then imports into one of my courses. */
export function Shared({ token }: { token: string }) {
  const [p, setP] = useState<PackPreview | null>(null)
  const [courses, setCourses] = useState<Subject[]>([])
  const [to, setTo] = useState('')
  const [res, setRes] = useState<PackImport | null>(null)
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')
  useEffect(() => {
    getShared(token).then(setP, (e) => setErr(msg(e)))
    listCourses().then((l) => { setCourses(l); setTo(l[0]?.id ?? '') }, () => {})
  }, [token])
  if (err && !p) return <><h1>Shared key ideas</h1><div className="error" role="alert">{err}</div></>
  if (!p) return <p className="loading">Loading...</p>
  const go = async () => {
    setBusy(true); setErr('')
    try { setRes(await importShared(token, to)) } catch (e) { setErr(msg(e)) }
    setBusy(false)
  }
  return (
    <>
      <h1>{p.course_name}</h1>
      <p>From {p.owner_name} {p.instructor && <span className="label supported">Instructor approved</span>}</p>
      {p.concepts.map((c) => (
        <div key={c.name} className="card">
          <strong>{c.name}</strong>
          <ul>{c.claims.map((k, i) => <li key={i}><Math text={k.text} /></li>)}</ul>
        </div>
      ))}
      {p.sources.length > 0 && <p><small>Includes notes: {p.sources.join(', ')}</small></p>}
      {res ? (
        <div role="status" className="card">
          <p>Added {res.concepts_added} topics and {res.claims_added} key ideas.</p>
          {res.needs_quote.length > 0 && (
            <>
              <h2>Needs a quote</h2>
              <p className="note">Each key idea needs a quote from your own notes. Add it on the course page.</p>
              <ul>{res.needs_quote.map((n, i) => <li key={i}><Math text={n.claim.text} /></li>)}</ul>
            </>
          )}
          <a className="btn" href={`#/course/${res.subject_id}`}>Open the course</a>
        </div>
      ) : courses.length === 0 ? <p>Add a course first, then open this link again.</p> : (
        <div className="card">
          <label>Add to course <select value={to} onChange={(e) => setTo(e.target.value)}>{courses.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}</select></label>
          {err && <div className="error" role="alert">{err}</div>}
          <button disabled={busy || !to} onClick={go}>{busy ? 'Adding...' : 'Add to my course'}</button>
        </div>
      )}
    </>
  )
}
