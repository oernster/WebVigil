import type {ChangeDetailDTO} from '../wire'

// DiffView shows a change as text only. Fetched content is never rendered as HTML (NFR-SEC-002):
// every value goes through React's text escaping.
export function DiffView({detail}: {detail: ChangeDetailDTO}) {
  if (detail.fields.length > 0) {
    return (
      <table className="list">
        <thead><tr><th>Field</th><th>Before</th><th>After</th></tr></thead>
        <tbody>
          {detail.fields.map((f) => (
            <tr key={f.name}>
              <td>{f.name}</td>
              <td className="wrap diff"><span className="removed">{f.old || '(empty)'}</span></td>
              <td className="wrap diff"><span className="added">{f.new || '(empty)'}</span></td>
            </tr>
          ))}
        </tbody>
      </table>
    )
  }
  if (detail.lines.length === 0) return <p>No line differences were recorded.</p>
  return (
    <div className="diff" role="list" aria-label="Changed lines">
      {detail.lines.map((l, i) => {
        const mark = l.op === 'added' ? '+' : l.op === 'removed' ? '-' : ' '
        const label = l.op === 'added' ? 'added' : l.op === 'removed' ? 'removed' : 'unchanged'
        return (
          <div key={i} role="listitem" className={l.op === 'equal' ? '' : l.op} aria-label={`${label}: ${l.text}`}>
            <span className="op">{mark}</span>{l.text}
          </div>
        )
      })}
    </div>
  )
}
