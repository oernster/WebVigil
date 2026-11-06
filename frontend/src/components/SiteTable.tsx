import {useRef, useState, type KeyboardEvent, type MouseEvent} from 'react'
import {icons} from '../icons'
import {ago, full, httpText, sortSites, type SortKey} from '../logic'
import type {SiteDTO} from '../wire'
import {StatusLight} from './StatusLight'

interface Column {
  key: SortKey | 'actions'
  title: string
  width: number
}

const COLUMNS: Column[] = [
  {key: 'light', title: 'Status', width: 120},
  {key: 'name', title: 'Website', width: 180},
  {key: 'url', title: 'URL', width: 260},
  {key: 'httpStatus', title: 'HTTP', width: 64},
  {key: 'lastChecked', title: 'Last Checked', width: 110},
  {key: 'lastSuccess', title: 'Last Success', width: 110},
  {key: 'lastChange', title: 'Last Change', width: 110},
  {key: 'details', title: 'Details', width: 320},
  {key: 'actions', title: 'Actions', width: 110},
]

const MIN_COLUMN = 48

interface Props {
  sites: SiteDTO[]
  selected: number
  now: Date
  onSelect: (id: number) => void
  onRefresh: (id: number) => void
  onEdit: (id: number) => void
  onDelete: (site: SiteDTO) => void
}

// SiteTable is the main list: sortable, resizable columns, horizontal scrolling and a tooltip on
// every cell that may be cut short, so no diagnostic is lost to an ellipsis.
export function SiteTable({sites, selected, now, onSelect, onRefresh, onEdit, onDelete}: Props) {
  const [sort, setSort] = useState<{key: SortKey; asc: boolean}>({key: 'light', asc: true})
  const [widths, setWidths] = useState(() => COLUMNS.map((c) => c.width))
  const drag = useRef<{index: number; startX: number; startW: number} | null>(null)
  const rows = sortSites(sites, sort.key, sort.asc)

  const startResize = (index: number) => (e: MouseEvent) => {
    e.preventDefault()
    drag.current = {index, startX: e.clientX, startW: widths[index]}
    const move = (ev: globalThis.MouseEvent) => {
      const d = drag.current
      if (!d) return
      setWidths((w) => w.map((v, i) => (i === d.index ? Math.max(MIN_COLUMN, d.startW + ev.clientX - d.startX) : v)))
    }
    const up = () => {
      drag.current = null
      window.removeEventListener('mousemove', move)
      window.removeEventListener('mouseup', up)
    }
    window.addEventListener('mousemove', move)
    window.addEventListener('mouseup', up)
  }

  const onKey = (e: KeyboardEvent<HTMLTableRowElement>, index: number) => {
    const step = e.key === 'ArrowDown' ? 1 : e.key === 'ArrowUp' ? -1 : 0
    if (step !== 0) {
      e.preventDefault()
      const next = rows[Math.min(rows.length - 1, Math.max(0, index + step))]
      onSelect(next.id)
      const el = e.currentTarget.parentElement?.children[rows.indexOf(next)] as HTMLElement | undefined
      el?.focus()
    } else if (e.key === 'Enter') {
      onSelect(rows[index].id)
    }
  }

  return (
    <div className="table-wrap">
      <table className="sites" style={{width: widths.reduce((a, b) => a + b, 0)}}>
        <colgroup>{widths.map((w, i) => <col key={COLUMNS[i].key} style={{width: w}} />)}</colgroup>
        <thead>
          <tr>
            {COLUMNS.map((c, i) => (
              <th key={c.key} style={{position: 'sticky'}} aria-sort={sort.key === c.key ? (sort.asc ? 'ascending' : 'descending') : undefined}>
                {c.key === 'actions' ? c.title : (
                  <button className="sort" onClick={() => setSort((s) => ({key: c.key as SortKey, asc: s.key === c.key ? !s.asc : true}))}>
                    {c.title}{sort.key === c.key ? (sort.asc ? ' ▲' : ' ▼') : ''}
                  </button>
                )}
                <span className="grip" onMouseDown={startResize(i)} aria-hidden="true" />
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.length === 0 && (
            <tr><td colSpan={COLUMNS.length}>No websites yet. Use Add Website or Import Sites to begin.</td></tr>
          )}
          {rows.map((s, i) => (
            <tr key={s.id} tabIndex={0} className={`row${s.id === selected ? ' selected' : ''}${s.enabled ? '' : ' disabled'}`}
              aria-selected={s.id === selected} onClick={() => onSelect(s.id)} onKeyDown={(e) => onKey(e, i)}>
              <td><StatusLight light={s.light} running={s.running} pending={s.pendingChangeId !== 0} /></td>
              <td title={s.name}>{s.name}{s.enabled ? '' : ' (disabled)'}</td>
              <td title={s.url}>{s.url}</td>
              <td title={s.httpStatus === 0 ? 'No HTTP response was received' : `HTTP ${s.httpStatus}`}>{httpText(s.httpStatus)}</td>
              <td title={full(s.lastChecked)}>{ago(s.lastChecked, now)}</td>
              <td title={full(s.lastSuccess)}>{ago(s.lastSuccess, now)}</td>
              <td title={full(s.lastChange)}>{ago(s.lastChange, now)}</td>
              <td title={s.details}>{s.details || (s.hasBaseline ? '' : 'Waiting for the first check')}</td>
              <td className="actions">
                <button className="icon-btn small" title={`Refresh ${s.name}`} aria-label={`Refresh ${s.name}`} disabled={s.running}
                  onClick={(e) => { e.stopPropagation(); onRefresh(s.id) }}>
                  <img src={icons.refresh} alt="" />
                </button>
                <button className="icon-btn small" title={`Edit ${s.name}`} aria-label={`Edit ${s.name}`}
                  onClick={(e) => { e.stopPropagation(); onEdit(s.id) }}>
                  <span className="label">Edit</span>
                </button>
                <button className="icon-btn small" title={`Delete ${s.name}`} aria-label={`Delete ${s.name}`}
                  onClick={(e) => { e.stopPropagation(); onDelete(s) }}>
                  <img src={icons.deleteSite} alt="" />
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
