import {useState} from 'react'
import {api} from '../api'
import {icons} from '../icons'
import type {ImportPreviewDTO, SettingsDTO} from '../wire'
import {Modal} from './Modal'

// ImportDialog previews an import and lets the user choose per duplicate; Skip is the default
// (FR-014). Nothing is written until Import is pressed.
export function ImportDialog({preview, onDone, onCancel}: {preview: ImportPreviewDTO; onDone: (msg: string) => void; onCancel: () => void}) {
  const [actions, setActions] = useState<Record<number, string>>(() => Object.fromEntries(preview.items.map((i) => [i.index, i.action])))
  const [error, setError] = useState('')
  const confirm = async () => {
    const s = await api.confirmImport(preview.path, actions, setError)
    if (s) onDone(`Imported ${s.imported}, updated ${s.updated}, skipped ${s.skipped}, invalid ${s.invalid}.`)
  }
  return (
    <Modal title="Import sites" icon={icons.importSites} onClose={onCancel} footer={<>
      <button className="btn" onClick={onCancel}>Cancel</button>
      <button className="btn primary" onClick={confirm}>Import</button>
    </>}>
      <p>{preview.path}</p>
      {error && <p className="error-text" role="alert">{error}</p>}
      <table className="list">
        <thead><tr><th>#</th><th>Website</th><th>URL</th><th>Action</th></tr></thead>
        <tbody>{preview.items.map((it) => (
          <tr key={it.index}>
            <td>{it.index}</td><td>{it.name}</td><td className="wrap">{it.url}</td>
            <td>{it.duplicate ? (
              <select aria-label={`Action for ${it.name}`} value={actions[it.index]} onChange={(e) => setActions((a) => ({...a, [it.index]: e.target.value}))}>
                <option value="skip">Skip (already exists)</option>
                {it.canUpdate && <option value="update">Update existing</option>}
                <option value="separate">Import as separate</option>
              </select>
            ) : 'Add'}</td>
          </tr>
        ))}</tbody>
      </table>
      {preview.invalid.length > 0 && <>
        <p className="failed">{preview.invalid.length} entries are invalid and will not be imported:</p>
        <ul>{preview.invalid.map((v) => <li key={v.index}>Entry {v.index} {v.name || v.url}: {v.reason}</li>)}</ul>
      </>}
    </Modal>
  )
}

// SettingsDialog edits the application settings.
export function SettingsDialog({initial, onSaved, onCancel}: {initial: SettingsDTO; onSaved: () => void; onCancel: () => void}) {
  const [s, setS] = useState(initial)
  const [error, setError] = useState('')
  const set = <K extends keyof SettingsDTO>(k: K, v: SettingsDTO[K]) => setS((o) => ({...o, [k]: v}))
  const save = async () => { if (await api.saveSettings(s, setError)) onSaved() }
  const choose = async () => { const p = await api.chooseSound(setError); if (p) set('soundPath', p) }
  return (
    <Modal title="Settings" icon={icons.appIcon} onClose={onCancel} footer={<>
      <button className="btn" onClick={onCancel}>Cancel</button>
      <button className="btn primary" onClick={save}>Save</button>
    </>}>
      {error && <p className="error-text" role="alert">{error}</p>}
      <div className="form">
        <label htmlFor="s-theme">Appearance</label>
        <select id="s-theme" value={s.theme} onChange={(e) => set('theme', e.target.value)}>
          <option value="system">Follow Windows</option><option value="light">Light</option><option value="dark">Dark</option>
        </select>
        <label htmlFor="s-sound">Alarm sound</label>
        <div className="checks">
          <span id="s-sound">{s.soundPath || 'Built-in chime'}</span>
          <button className="btn" onClick={choose}>Choose WAV</button>
          {s.soundPath && <button className="btn" onClick={() => set('soundPath', '')}>Use built-in</button>}
          <button className="btn" onClick={() => api.testSound(s.soundPath, setError)}>Test sound</button>
        </div>
        <span />
        <div className="checks">
          <label><input type="checkbox" checked={s.repeatAlarm} onChange={(e) => set('repeatAlarm', e.target.checked)} /> Repeat the alarm until acknowledged</label>
          <label><input type="checkbox" checked={s.foregroundDialog} onChange={(e) => set('foregroundDialog', e.target.checked)} /> Bring WebVigil forward on a change</label>
          <label><input type="checkbox" checked={s.startAtSignIn} onChange={(e) => set('startAtSignIn', e.target.checked)} /> Start WebVigil when I sign in to Windows</label>
        </div>
      </div>
    </Modal>
  )
}
