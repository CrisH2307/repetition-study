import { FormEvent, useEffect, useState } from 'react'
import { Math } from './Math'
import { WORDS } from './Teach'
import { Start, StartButton } from './App'
import {
  ClaimKind, CourseView, Page, SourceBlock, StudyCard, Subject, Upload,
  addClaim, addConcept, addCourse, courseView, getPage, getUpload, listBlocks, listCourses, listPages, markReady, savePage, uploadNotes,
} from './api'

const msg = (e: unknown) => (e instanceof Error ? e.message : 'Something went wrong. Please try again.')
const pct = (n: number | null, none: string) => (n === null ? none : `${(n * 100).toFixed(0)}%`)
const STATUS: Record<Upload['status'], string> = { reading: 'Reading the file...', needs_checking: 'Pages need checking', ready: 'Ready' }

export function Courses() {
  const [list, setList] = useState<Subject[] | null>(null)
  const [name, setName] = useState('')
  const [h, setH] = useState('3')
  const [err, setErr] = useState('')
  const load = () => listCourses().then(setList, (e) => setErr(msg(e)))
  useEffect(() => { load() }, [])
  const add = async (e: FormEvent) => {
    e.preventDefault(); setErr('')
    try { await addCourse(name.trim(), Number(h) || 0); setName(''); load() } catch (e) { setErr(msg(e)) }
  }
  return (
    <>
      <h1>Courses</h1>
      {err && <div className="error" role="alert">{err}</div>}
      {list?.length === 0 && <p>No courses yet. Add your first one below.</p>}
      <ul className="plain">{list?.map((c) => <li key={c.id} className="card"><a href={`#/course/${c.id}`}>{c.name}</a> <small>{c.weekly_hours} h a week</small></li>)}</ul>
      <form className="card" onSubmit={add}>
        <h2>Add a course</h2>
        <label>Course name <input value={name} onChange={(e) => setName(e.target.value)} required /></label>
        <label>Hours a week <input type="number" min="0" step="0.5" value={h} onChange={(e) => setH(e.target.value)} /></label>
        <button disabled={!name.trim()}>Add course</button>
      </form>
    </>
  )
}

export function Course({ id, start }: { id: string; start: Start }) {
  const [v, setV] = useState<CourseView | null>(null)
  const [err, setErr] = useState('')
  const load = () => courseView(id).then(setV, (e) => setErr(msg(e)))
  useEffect(() => { load() }, [id])
  // ponytail: re-reads the whole course view while a file is being read; poll /uploads/{id} if this gets heavy.
  const reading = v?.notes.some((n) => n.status === 'reading')
  useEffect(() => { if (reading) { const t = setTimeout(load, 1500); return () => clearTimeout(t) } }, [v])
  if (err && !v) return <div className="error" role="alert">{err}</div>
  if (!v) return <p>Loading...</p>
  const names = Object.fromEntries(v.cards.map((c) => [c.concept_id, c.name]))
  return (
    <>
      <h1>{v.course.name}</h1>
      <section>
        <h2>Course notes</h2>
        <p className="note">Only ready notes are quoted in corrections.</p>
        <ul className="plain">
          {v.notes.map((n) => (
            <li key={n.id} className="topic">
              <div><strong>{n.filename}</strong>
                <div><small>{STATUS[n.status]}{n.status !== 'reading' && `, ${n.pages_ready} of ${n.pages_total} pages ready`}</small></div>
                {n.error && <div className="error" role="alert">{n.error}</div>}</div>
              {n.status !== 'reading' && n.pages_total > 0 && <a className="btn plain" href={`#/upload/${n.id}`}>Check pages</a>}
            </li>
          ))}
        </ul>
        <UploadForm course={id} onDone={load} />
      </section>
      <section>
        <h2>Study cards</h2>
        {v.cards.map((c) => <Card key={c.concept_id} c={c} course={id} start={start} onChange={load} />)}
        <AddTopic course={id} onDone={load} />
      </section>
      <section>
        <h2>Mistakes</h2>
        {v.mistakes.length === 0 ? <p><small>No mistakes found yet.</small></p> : (
          <div className="scroll">
            <table>
              <thead><tr><th>Topic</th><th>You said</th><th>Result</th><th>Passage</th></tr></thead>
              <tbody>
                {v.mistakes.map((m, i) => (
                  <tr key={i}>
                    <td>{names[m.concept_id] ?? ''}</td>
                    <td>{m.student_text ? <Math text={m.student_text} /> : <small>Not mentioned</small>}</td>
                    <td><span className={`label ${m.label}`}>{WORDS[m.label]}</span></td>
                    <td>&ldquo;{m.quote}&rdquo; {m.page_id && <a href={`#/page/${m.page_id}`}>Page {m.page}</a>}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </>
  )
}

function UploadForm({ course, onDone }: { course: string; onDone: () => void }) {
  const [file, setFile] = useState<File | null>(null)
  const [text, setText] = useState('')
  const [title, setTitle] = useState('')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')
  const send = async (e: FormEvent) => {
    e.preventDefault(); setErr('')
    if (file && file.size > 20 * 1024 * 1024) return setErr('This file is bigger than 20 MB. Please split it and try again.')
    const f = new FormData()
    if (file) { f.append('filename', file.name); f.append('file', file) }
    else { f.append('filename', title.trim() || 'Pasted notes'); f.append('text', text) }
    setBusy(true)
    try { await uploadNotes(course, f); setFile(null); setText(''); setTitle(''); (e.target as HTMLFormElement).reset(); onDone() }
    catch (e) { setErr(msg(e)) }
    setBusy(false)
  }
  return (
    <form className="card" onSubmit={send}>
      <h3>Add notes</h3>
      <label>A PDF file (up to 20 MB) <input type="file" accept="application/pdf" onChange={(e) => setFile(e.target.files?.[0] ?? null)} /></label>
      <p><small>Or paste your notes as text:</small></p>
      <label>Title <input value={title} onChange={(e) => setTitle(e.target.value)} disabled={!!file} /></label>
      <label>Notes <textarea className="short" value={text} onChange={(e) => setText(e.target.value)} disabled={!!file} /></label>
      {err && <div className="error" role="alert">{err}</div>}
      <button disabled={busy || (!file && !text.trim())}>{busy ? 'Sending...' : 'Add notes'}</button>
    </form>
  )
}

function Card({ c, course, start, onChange }: { c: StudyCard; course: string; start: Start; onChange: () => void }) {
  return (
    <div className="card">
      <div className="topic">
        <div><strong>{c.name}</strong>
          <div><small>Recall: {pct(c.recall, 'no reviews yet')}. Explanation: {pct(c.explanation, 'not taught yet')}.</small></div>
          {c.next_due && <div><small>Next review {new Date(c.next_due).toLocaleDateString(undefined, { weekday: 'short', month: 'short', day: 'numeric' })}</small></div>}</div>
        <StartButton id={c.concept_id} name={c.name} start={start} />
      </div>
      <details>
        <summary>Key ideas ({c.claims.length})</summary>
        <ul>{c.claims.map((k) => <li key={k.id}><Math text={k.text} /> <small>({k.locator}{k.required ? '' : ', optional'})</small></li>)}</ul>
        <AddClaim concept={c.concept_id} course={course} onDone={onChange} />
      </details>
    </div>
  )
}

const KINDS: [ClaimKind, string][] = [['definition', 'Definition'], ['statement', 'Statement'], ['condition', 'Condition'], ['method', 'Method']]

function AddClaim({ concept, course, onDone }: { concept: string; course: string; onDone: () => void }) {
  const [blocks, setBlocks] = useState<SourceBlock[] | null>(null)
  const [text, setText] = useState('')
  const [kind, setKind] = useState<ClaimKind>('statement')
  const [required, setRequired] = useState(true)
  const [block, setBlock] = useState('')
  const [quote, setQuote] = useState('')
  const [err, setErr] = useState('')
  useEffect(() => { listBlocks(course).then(setBlocks, () => setBlocks([])) }, [course])
  if (blocks === null) return null
  if (!blocks.length) return <p><small>To add a key idea, first check a page of your notes and mark it Ready.</small></p>
  const pick = (id: string) => { setBlock(id); setQuote(blocks.find((b) => b.id === id)?.text_md ?? '') }
  const send = async (e: FormEvent) => {
    e.preventDefault(); setErr('')
    try {
      await addClaim(concept, { text: text.trim(), kind, required, anchor: { block_id: block, quote } })
      setText(''); setBlock(''); setQuote(''); onDone()
    } catch (e) { setErr(msg(e)) }
  }
  const id = `c-${concept}`
  return (
    <form onSubmit={send} aria-label="Add a key idea">
      <h4>Add a key idea</h4>
      <label>The idea, in your words <textarea className="short" value={text} onChange={(e) => setText(e.target.value)} /></label>
      <div className="row">
        <label>Kind <select value={kind} onChange={(e) => setKind(e.target.value as ClaimKind)}>{KINDS.map(([k, w]) => <option key={k} value={k}>{w}</option>)}</select></label>
        <label className="check"><input type="checkbox" checked={required} onChange={(e) => setRequired(e.target.checked)} /> Must be explained</label>
      </div>
      <label htmlFor={`${id}-b`}>Where it is in your notes</label>
      <select id={`${id}-b`} value={block} onChange={(e) => pick(e.target.value)}>
        <option value="">Choose a passage</option>
        {blocks.map((b) => <option key={b.id} value={b.id}>{b.page ? `Page ${b.page}: ` : ''}{b.text_md.slice(0, 70)}</option>)}
      </select>
      <label>Exact words from that passage <textarea className="short" value={quote} onChange={(e) => setQuote(e.target.value)} /></label>
      <div className="note">Keep only the words that back up the idea. They must match the notes exactly.</div>
      {err && <div className="error" role="alert">{err}</div>}
      <button disabled={!text.trim() || !block || !quote.trim()}>Add key idea</button>
    </form>
  )
}

function AddTopic({ course, onDone }: { course: string; onDone: () => void }) {
  const [name, setName] = useState('')
  const [err, setErr] = useState('')
  const send = async (e: FormEvent) => {
    e.preventDefault(); setErr('')
    try { await addConcept(course, name.trim()); setName(''); onDone() } catch (e) { setErr(msg(e)) }
  }
  return (
    <form className="row" onSubmit={send}>
      <label>New topic <input value={name} onChange={(e) => setName(e.target.value)} /></label>
      <button disabled={!name.trim()}>Add topic</button>
      {err && <div className="error" role="alert">{err}</div>}
    </form>
  )
}

/** Page check: the page picture next to the text we read, so the student can fix it and mark it Ready. */
export function PageCheck({ uploadId, pageId }: { uploadId?: string; pageId?: string }) {
  const [up, setUp] = useState<Upload | null>(null)
  const [pages, setPages] = useState<Page[]>([])
  const [at, setAt] = useState(0)
  const [text, setText] = useState('')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')
  useEffect(() => {
    (async () => {
      const uid = uploadId ?? (await getPage(pageId!)).upload_id
      const [u, ps] = await Promise.all([getUpload(uid), listPages(uid)])
      const i = pageId ? ps.findIndex((p) => p.id === pageId) : ps.findIndex((p) => p.status !== 'ready')
      setUp(u); setPages(ps); setAt(i < 0 ? 0 : i); setText(ps[i < 0 ? 0 : i]?.edited_text ?? '')
    })().catch((e) => setErr(msg(e)))
  }, [uploadId, pageId])
  if (err && !up) return <div className="error" role="alert">{err}</div>
  if (!up) return <p>Loading...</p>
  const p = pages[at]
  const go = (i: number) => { setAt(i); setText(pages[i].edited_text); setErr('') }
  const act = async (ready: boolean) => {
    setBusy(true); setErr('')
    try {
      let np = p
      if (text !== p.edited_text) np = await savePage(p.id, text)
      if (ready !== (np.status === 'ready')) np = await markReady(p.id, ready)
      setPages(pages.map((x) => (x.id === np.id ? np : x)))
    } catch (e) { setErr(msg(e)) }
    setBusy(false)
  }
  return (
    <>
      <p><a href={`#/course/${up.subject_id}`}>Back to the course</a></p>
      <h1>{up.filename}</h1>
      <p>Check that the text matches the page. Fix anything that is wrong, then mark the page Ready.</p>
      <nav className="row" aria-label="Pages">
        {pages.map((x, i) => (
          <button key={x.id} className={i === at ? '' : 'plain'} aria-current={i === at || undefined} onClick={() => go(i)}>
            Page {x.number}{x.status === 'ready' ? ' ✓' : ''}
          </button>
        ))}
      </nav>
      {p && (
        <div className="pagecheck">
          <div className="card">
            {p.has_image ? <img src={`/v1/pages/${p.id}/image`} alt={`Page ${p.number} of the file`} /> : <small>No picture for this page. It came from pasted text.</small>}
          </div>
          <div>
            <p><strong>Page {p.number}: {p.status === 'ready' ? 'Ready' : 'Needs checking'}</strong></p>
            <label htmlFor="pt">Text we read from the page</label>
            <textarea id="pt" value={text} readOnly={p.status === 'ready'} onChange={(e) => setText(e.target.value)} />
            {p.status === 'ready' && <div className="note">To change a Ready page, mark it as needing checks first.</div>}
            <div className="card preview" aria-label="Preview"><Math text={text} /></div>
            {err && <div className="error" role="alert">{err}</div>}
            <div className="row">
              {p.status === 'ready'
                ? <button className="plain" disabled={busy} onClick={() => act(false)}>Needs checking again</button>
                : <button disabled={busy} onClick={() => act(true)}>Mark Ready</button>}
            </div>
          </div>
        </div>
      )}
    </>
  )
}
