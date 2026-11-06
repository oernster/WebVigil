import {icons} from '../icons'
import {Menu, type MenuItem} from './Menu'
import {toggleTarget, type Appearance} from '../logic'

interface Props {
  appearance: Appearance
  paused: boolean
  disabled: boolean
  hasSites: boolean
  note: string
  onAdd: () => void
  onImport: () => void
  onExport: () => void
  onRefreshAll: () => void
  onTogglePause: () => void
  onToggleTheme: (target: Appearance) => void
  onSettings: () => void
  onGuide: () => void
  onAbout: () => void
  onLicence: () => void
  onCheckUpdates: () => void
}

// Toolbar carries the global actions as icon buttons with names and tooltips. The theme toggle
// shows the artwork of the appearance it switches TO (FR-121).
export function Toolbar(p: Props) {
  const target = toggleTarget(p.appearance)
  const themeName = target === 'light' ? 'Switch to light appearance' : 'Switch to dark appearance'
  const pauseName = p.paused ? 'Resume automatic monitoring' : 'Pause automatic monitoring'
  // Guide leads the menu, as in PigeonPost: it is the entry someone meeting the app reaches for.
  const helpItems: MenuItem[] = [
    {label: 'Guide', onClick: p.onGuide},
    {label: '', separator: true},
    {label: 'About WebVigil', onClick: p.onAbout},
    {label: 'Licence', onClick: p.onLicence},
    {label: 'Check for Updates', onClick: p.onCheckUpdates},
  ]
  return (
    <div className="toolbar" role="toolbar" aria-label="WebVigil">
      <button className="icon-btn" title="Add website" aria-label="Add website" disabled={p.disabled} onClick={p.onAdd}>
        <img src={icons.addSite} alt="" />
      </button>
      <button className="icon-btn" title="Import sites" aria-label="Import sites" disabled={p.disabled} onClick={p.onImport}>
        <img src={icons.importSites} alt="" />
      </button>
      <button className="icon-btn" title="Export sites" aria-label="Export sites" disabled={p.disabled || !p.hasSites} onClick={p.onExport}>
        <img src={icons.exportSites} alt="" />
      </button>
      <span className="sep" aria-hidden="true" />
      <button className="icon-btn" title="Refresh all websites now" aria-label="Refresh all" disabled={p.disabled || !p.hasSites} onClick={p.onRefreshAll}>
        <img src={icons.refresh} alt="" />
      </button>
      <button className="icon-btn" title={pauseName} aria-label={pauseName} aria-pressed={p.paused} disabled={p.disabled} onClick={p.onTogglePause}>
        <img src={p.paused ? icons.play : icons.pause} alt="" />
      </button>
      <span className="spacer" />
      {p.note && <span className="note" role="status">{p.note}</span>}
      {p.paused && <span className="note" role="status">Monitoring paused</span>}
      <button className="icon-btn" title="Settings" aria-label="Settings" disabled={p.disabled} onClick={p.onSettings}>
        <img src={icons.settings} alt="" />
      </button>
      <span className="sep" aria-hidden="true" />
      <button className="icon-btn" title={themeName} aria-label={themeName} onClick={() => p.onToggleTheme(target)}>
        <img src={target === 'light' ? icons.lightMode : icons.darkMode} alt="" />
      </button>
      <Menu title="Help" icon={icons.helpAbout} items={helpItems} />
    </div>
  )
}
