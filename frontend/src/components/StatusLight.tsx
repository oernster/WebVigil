import {asLight, lightInfo} from '../logic'

interface Props {
  light: string
  running: boolean
  pending: boolean
}

// StatusLight shows the traffic light as colour, shape and word together, plus a separate badge
// for a change awaiting acknowledgement, so a failure never hides a pending change (FR-065, FR-072).
export function StatusLight({light, running, pending}: Props) {
  const l = asLight(light)
  const info = lightInfo[l]
  const title = info.meaning + (pending && l !== 'amber' ? '; a change also awaits acknowledgement' : '')
  return (
    <span className={`light ${l}`} title={title}>
      <span className={`mark ${info.shape}`} aria-hidden="true" />
      <span className="label">{info.label}</span>
      {pending && l !== 'amber' && <span className="badge" aria-label="change awaiting acknowledgement">!</span>}
      {running && <span className="spinner" role="status" aria-label="checking" />}
    </span>
  )
}
