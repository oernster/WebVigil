import {useEffect, useState} from 'react'
import {api} from '../api'
import {full, httpText, lightInfo, asLight} from '../logic'
import type {ChangeDetailDTO, HistoryDTO, SiteDTO} from '../wire'
import {DiffView} from './DiffView'

type Tab = 'status' | 'changes' | 'history'

interface Props {
  site: SiteDTO
  tab: Tab
  version: number // bumps whenever the site's data changed, so the panel re-reads it
  onTab: (t: Tab) => void
  onError: (m: string) => void
}

// Details shows one site's configuration, current state and history. The current failure and a
// pending change are stated separately, so neither fact hides the other (FR-065).
export function Details({site, tab, version, onTab, onError}: Props) {
  const [history, setHistory] = useState<HistoryDTO | null>(null)
  const [detail, setDetail] = useState<ChangeDetailDTO | null>(null)
  useEffect(() => {
    let live = true
    api.history(site.id, onError).then((h) => {
      if (!live || !h) return
      setHistory(h)
      const first = h.changes[0]
      if (first) api.changeDetail(first.id, onError).then((d) => live && setDetail(d))
      else setDetail(null)
    })
    return () => { live = false }
  }, [site.id, version, onError])

  const act = (f: (r: (m: string) => void) => Promise<unknown>) => () => { f(onError) }
  const light = lightInfo[asLight(site.light)]
  return (
    <section className="details" aria-label={`Details for ${site.name}`}>
      <header>
        <h2 title={site.name}>{site.name}</h2>
        {site.pendingChangeId !== 0 && <button className="btn primary" onClick={act((r) => api.acknowledge(site.id, r))}>Acknowledge change</button>}
        <button className="btn" onClick={act((r) => api.setEnabled(site.id, !site.enabled, r))}>{site.enabled ? 'Disable monitoring' : 'Enable monitoring'}</button>
        <button className="btn" onClick={act((r) => api.openWebsite(site.id, r))}>Open website</button>
      </header>
      <div className="tabs" role="tablist">
        {(['status', 'changes', 'history'] as Tab[]).map((t) => (
          <button key={t} role="tab" className="btn" aria-selected={tab === t} onClick={() => onTab(t)}>
            {t === 'status' ? 'Status' : t === 'changes' ? 'View changes' : 'Check history'}
          </button>
        ))}
      </div>
      <div className="body">
        {tab === 'status' && <dl className="facts">
          <dt>Status</dt><dd>{light.label}: {light.meaning}{site.running ? ' (checking now)' : ''}</dd>
          {site.outcome === 'failure' && <><dt>Latest failure</dt><dd className="failed">{site.details}</dd></>}
          {site.pendingChangeId !== 0 && <><dt>Pending change</dt><dd className="alert">A change detected {full(site.lastChange)} awaits acknowledgement</dd></>}
          <dt>HTTP status</dt><dd>{httpText(site.httpStatus)}</dd>
          <dt>Address</dt><dd>{site.url}</dd>
          <dt>Monitoring</dt><dd>{site.enabled ? `Every ${site.intervalMinutes} minutes` : 'Disabled'}; mode {site.mode.replace('_', ' ')}</dd>
          <dt>Baseline</dt><dd>{site.hasBaseline ? 'Established' : 'Not yet: the next successful check creates it'}</dd>
          <dt>Last checked</dt><dd>{full(site.lastChecked)}</dd>
          <dt>Last success</dt><dd>{full(site.lastSuccess)}</dd>
          <dt>Last change</dt><dd>{full(site.lastChange)}</dd>
          <dt>Next check</dt><dd>{site.enabled ? full(site.nextDue) : 'Not scheduled'}</dd>
          <dt>Failures in a row</dt><dd>{site.consecutiveFailures}</dd>
        </dl>}
        {tab === 'changes' && <>
          {history && history.changes.length === 0 && <p>No changes recorded yet.</p>}
          {history && history.changes.length > 0 && <>
            <label>Change: <select value={detail?.change.id ?? ''} onChange={(e) => api.changeDetail(Number(e.target.value), onError).then(setDetail)}>
              {history.changes.map((c) => <option key={c.id} value={c.id}>
                {full(c.detectedAt)}: {c.summary}{c.acknowledgedAt ? '' : ' (not acknowledged)'}
              </option>)}
            </select></label>
            {detail && <DiffView detail={detail} />}
          </>}
        </>}
        {tab === 'history' && history && <table className="list">
          <thead><tr><th>Checked</th><th>Outcome</th><th>HTTP</th><th>Time</th><th>Result</th><th>Kind</th></tr></thead>
          <tbody>{history.checks.map((c, i) => (
            <tr key={i}>
              <td>{full(c.at)}</td>
              <td className={c.outcome === 'failure' ? 'failed' : ''}>{c.outcome === 'failure' ? 'Failed' : 'Success'}</td>
              <td>{httpText(c.httpStatus)}</td>
              <td>{c.durationMs} ms</td>
              <td className="wrap" title={c.finalUrl}>{c.detail}</td>
              <td>{c.manual ? 'Manual' : 'Scheduled'}</td>
            </tr>
          ))}</tbody>
        </table>}
      </div>
    </section>
  )
}
