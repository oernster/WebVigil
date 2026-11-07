import {useEffect, useRef, useState} from 'react'
import {api} from '../api'
import {icons} from '../icons'
import {intervalPresets, lines, patternLines} from '../logic'
import {DEFAULT_MODE, filledForm, mergeLines, readingNote} from '../reading'
import type {FieldRuleDTO, PreviewDTO, SiteFormDTO} from '../wire'
import {Modal} from './Modal'

// The application's refusal texts the dialog answers with a question rather than an error.
const DUPLICATE = 'already exists'
const REBASELINE = 'needs a new baseline'

const MODES: [string, string][] = [
  ['extracted_text', 'Extracted text (visible page text)'],
  ['css_selector', 'CSS selector'],
  ['structured_fields', 'Structured fields'],
  ['raw', 'Raw response'],
]

interface Props {
  initial: SiteFormDTO
  onSaved: () => void
  onCancel: () => void
}

type Ask = 'duplicate' | 'rebaseline' | null

// SiteDialog adds or edits a definition. Two refusals come back as questions: a duplicate URL and
// an edit that changes what content means, which needs a new baseline (FR-004, FR-006).
export function SiteDialog({initial, onSaved, onCancel}: Props) {
  const [f, setF] = useState<SiteFormDTO>(initial)
  const [selectors, setSelectors] = useState(initial.selectors.join('\n'))
  const [excludeSel, setExcludeSel] = useState(initial.excludeSelectors.join('\n'))
  const [excludePat, setExcludePat] = useState(initial.excludePatterns.join('\n'))
  const [error, setError] = useState('')
  const [ask, setAsk] = useState<Ask>(null)
  const [busy, setBusy] = useState(false)
  const [preview, setPreview] = useState<PreviewDTO | null>(null)
  const [filledFrom, setFilledFrom] = useState('')
  const filledFor = useRef('')
  const editing = initial.id !== 0
  const set = <K extends keyof SiteFormDTO>(k: K, v: SiteFormDTO[K]) => setF((o) => ({...o, [k]: v}))
  // A page WebVigil has a ready-made reading for is filled in from it as its address is entered
  // or when its definition opens, while the definition is still in the default mode (FR-058). An
  // answer for an address the user has since changed is dropped. An address is filled in once:
  // a user who moves it back to text mode keeps text mode.
  useEffect(() => {
    if (f.mode !== DEFAULT_MODE || f.url === filledFor.current) return
    let live = true
    void api.suggestedReading(f.url, () => undefined).then((reading) => {
      if (!live || !reading?.found) return
      filledFor.current = f.url
      setF((o) => filledForm(o, reading))
      setExcludeSel((t) => mergeLines(t, reading.excludeSelectors))
      setExcludePat((t) => mergeLines(t, reading.excludePatterns))
      setFilledFrom(reading.name)
    })
    return () => { live = false }
  }, [f.url, f.mode])

  const form = (): SiteFormDTO => ({...f, selectors: lines(selectors), excludeSelectors: lines(excludeSel), excludePatterns: patternLines(excludePat)})

  const save = async (allowDuplicate: boolean, rebaseline: boolean) => {
    setBusy(true)
    setError('')
    let refusal = ''
    const r = (m: string) => { refusal = m }
    const done = editing
      ? await api.updateSite(form(), rebaseline, allowDuplicate, r)
      : await api.addSite(form(), allowDuplicate, r)
    setBusy(false)
    if (done !== null) return onSaved()
    if (refusal.includes(DUPLICATE)) return setAsk('duplicate')
    if (refusal.includes(REBASELINE)) return setAsk('rebaseline')
    setError(refusal)
  }

  const runPreview = async () => {
    setBusy(true)
    setPreview(await api.preview(form(), (m) => setError(m)))
    setBusy(false)
  }

  const setField = (i: number, patch: Partial<FieldRuleDTO>) =>
    set('fields', f.fields.map((x, j) => (j === i ? {...x, ...patch} : x)))

  const footer = ask === null ? <>
    <button className="btn left" disabled={busy} onClick={runPreview}>Preview extraction</button>
    <button className="btn" onClick={onCancel}>Cancel</button>
    <button className="btn primary" disabled={busy} onClick={() => save(false, false)}>{editing ? 'Save' : 'Add website'}</button>
  </> : <>
    <button className="btn" onClick={() => setAsk(null)}>Go back</button>
    <button className="btn primary" onClick={() => save(ask === 'duplicate', ask === 'rebaseline')}>
      {ask === 'duplicate' ? 'Add anyway' : 'Save and start a new baseline'}
    </button>
  </>

  return (
    <Modal title={editing ? `Edit ${initial.name}` : 'Add website'} icon={icons.addSite} onClose={onCancel} footer={footer}>
      {ask === 'duplicate' && <p className="alert">Another site already monitors this address. Add a second definition for it anyway?</p>}
      {ask === 'rebaseline' && <p className="alert">These changes alter what WebVigil compares. The next successful check will
        establish a new baseline without an alarm; earlier history is kept.</p>}
      {error && <p className="error-text" role="alert">{error}</p>}
      <div className="form" hidden={ask !== null}>
        <label htmlFor="f-name">Friendly name</label>
        <input id="f-name" type="text" value={f.name} onChange={(e) => set('name', e.target.value)} placeholder="Taken from the address when empty" />
        <label htmlFor="f-url">Website URL</label>
        <input id="f-url" type="text" value={f.url} onChange={(e) => set('url', e.target.value)} placeholder="https://example.com/page" />
        <label htmlFor="f-interval">Check every</label>
        <select id="f-interval" value={f.intervalMinutes} onChange={(e) => set('intervalMinutes', Number(e.target.value))}>
          {[...new Set([...intervalPresets, f.intervalMinutes])].sort((a, b) => a - b).map((m) => <option key={m} value={m}>{m} minutes</option>)}
        </select>
        <label htmlFor="f-timeout">Timeout (seconds)</label>
        <input id="f-timeout" type="number" value={f.timeoutSeconds} onChange={(e) => set('timeoutSeconds', Number(e.target.value))} />
        <label htmlFor="f-mode">Detection mode</label>
        <select id="f-mode" value={f.mode} onChange={(e) => set('mode', e.target.value)}>
          {MODES.map(([v, t]) => <option key={v} value={v}>{t}</option>)}
        </select>
        {filledFrom && f.mode === 'structured_fields' && <p className="hint">{readingNote(filledFrom)}</p>}
        {f.mode === 'css_selector' && <>
          <label htmlFor="f-sel">CSS selectors</label>
          <textarea id="f-sel" value={selectors} onChange={(e) => setSelectors(e.target.value)} placeholder="One per line; each must match" />
        </>}
        {f.mode === 'structured_fields' && <fieldset>
          <legend>Fields (each compared separately)</legend>
          {f.fields.map((x, i) => (
            <div className="field-row" key={i}>
              <input type="text" aria-label="Field name" placeholder="Name" value={x.name} onChange={(e) => setField(i, {name: e.target.value})} />
              <input type="text" aria-label="Field selector" placeholder="CSS selector" value={x.selector} onChange={(e) => setField(i, {selector: e.target.value})} />
              <input type="text" aria-label="Attribute" placeholder="Attribute (optional)" value={x.attribute} onChange={(e) => setField(i, {attribute: e.target.value})} />
              <label><input type="checkbox" checked={x.optional} onChange={(e) => setField(i, {optional: e.target.checked})} /> Optional</label>
              <button className="btn" aria-label={`Remove field ${x.name}`} onClick={() => set('fields', f.fields.filter((_, j) => j !== i))}>Remove</button>
            </div>
          ))}
          <button className="btn" onClick={() => set('fields', [...f.fields, {name: '', selector: '', attribute: '', optional: false}])}>Add field</button>
        </fieldset>}
        <fieldset>
          <legend>Ignore rules</legend>
          <div className="form">
            <label htmlFor="f-xsel">Exclude selectors</label>
            <textarea id="f-xsel" value={excludeSel} onChange={(e) => setExcludeSel(e.target.value)} placeholder="One per line, e.g. .advert" />
            <label htmlFor="f-xpat">Exclude patterns</label>
            <textarea id="f-xpat" value={excludePat} onChange={(e) => setExcludePat(e.target.value)} placeholder="Regular expressions, one per line" />
            <span />
            <div className="checks">
              <label><input type="checkbox" checked={f.collapseWhitespace} onChange={(e) => set('collapseWhitespace', e.target.checked)} /> Normalise whitespace</label>
              <label><input type="checkbox" checked={f.caseInsensitive} onChange={(e) => set('caseInsensitive', e.target.checked)} /> Ignore case</label>
            </div>
          </div>
        </fieldset>
        <fieldset>
          <legend>Notifications</legend>
          <div className="checks">
            <label><input type="checkbox" checked={f.enabled} onChange={(e) => set('enabled', e.target.checked)} /> Monitoring enabled</label>
            <label><input type="checkbox" checked={f.sound} onChange={(e) => set('sound', e.target.checked)} /> Audible alert</label>
            <label><input type="checkbox" checked={f.visual} onChange={(e) => set('visual', e.target.checked)} /> Visual notification</label>
            <label><input type="checkbox" checked={f.requireAck} onChange={(e) => set('requireAck', e.target.checked)} /> Require acknowledgement</label>
          </div>
          <div className="form" style={{marginTop: 8}}>
            <label htmlFor="f-thresh">Warn after failures</label>
            <input id="f-thresh" type="number" value={f.failureThreshold} onChange={(e) => set('failureThreshold', Number(e.target.value))} />
          </div>
        </fieldset>
        {preview && <div className="full">
          <p>{preview.httpStatus ? `HTTP ${preview.httpStatus}. ` : 'No HTTP response. '}
            {preview.category ? <span className="failed">{preview.detail}</span> : 'Extraction succeeded.'}</p>
          {!preview.category && <div className="preview">
            {preview.fields.length > 0 ? preview.fields.map((x) => `${x.name}: ${x.value}`).join('\n') : preview.text}
          </div>}
        </div>}
      </div>
    </Modal>
  )
}
