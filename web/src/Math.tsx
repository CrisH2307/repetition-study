import katex from 'katex'
import 'katex/dist/katex.min.css'

/** Renders text where $...$ and $$...$$ are formulas. Bad formulas show in red, never crash. */
export function renderMath(text: string): string {
  const esc = (s: string) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
  return text
    .split(/(\$\$[\s\S]+?\$\$|\$[^$\n]+?\$)/g)
    .map((p, i) => {
      if (i % 2 === 0) return esc(p).replace(/\n/g, '<br>')
      const display = p.startsWith('$$')
      const tex = p.slice(display ? 2 : 1, display ? -2 : -1)
      return katex.renderToString(tex, { displayMode: display, throwOnError: false, output: 'htmlAndMathml', trust: false })
    })
    .join('')
}

export function Math({ text }: { text: string }) {
  return <span dangerouslySetInnerHTML={{ __html: renderMath(text) }} />
}
