import {useCallback, useEffect, useMemo, useState} from 'react'
import {api, onEvent} from './api'
import {BottomBar} from './components/BottomBar'
import {ConfirmDelete} from './components/ConfirmDelete'
import {Details} from './components/Details'
import {ImportDialog, SettingsDialog} from './components/Dialogs'
import {AboutDialog, GuideDialog, LicenceDialog, UpdateDialog} from './components/HelpDialogs'
import {useUpdateCheck} from './hooks/useUpdateCheck'
import {SiteDialog} from './components/SiteDialog'
import {SiteTable} from './components/SiteTable'
import {Toolbar} from './components/Toolbar'
import {pendingSummary, resolveTheme, type Appearance} from './logic'
import type {AboutDTO, AppStateDTO, ImportPreviewDTO, SiteDTO, SiteFormDTO} from './wire'

const CLOCK_TICK_MS = 30_000
const DARK_QUERY = '(prefers-color-scheme: dark)'

type Dialog =
  | {kind: 'site'; form: SiteFormDTO}
  | {kind: 'delete'; site: SiteDTO}
  | {kind: 'import'; preview: ImportPreviewDTO}
  | {kind: 'settings'}
  | {kind: 'guide'}
  | {kind: 'about'; about: AboutDTO}
  | {kind: 'licence'; name: string; text: string}
  | null

export function App() {
  const [state, setState] = useState<AppStateDTO | null>(null)
  const [sites, setSites] = useState<SiteDTO[]>([])
  const [selected, setSelected] = useState(0)
  const [tab, setTab] = useState<'status' | 'changes' | 'history'>('status')
  const [dialog, setDialog] = useState<Dialog>(null)
  const [message, setMessage] = useState<{text: string; error: boolean} | null>(null)
  const [version, setVersion] = useState(0)
  const [now, setNow] = useState(() => new Date())
  const [systemDark, setSystemDark] = useState(() => window.matchMedia?.(DARK_QUERY).matches ?? true)

  const fail = useCallback((text: string) => setMessage({text, error: true}), [])
  const reload = useCallback(async () => {
    const s = await api.state(fail)
    if (s) setState(s)
    if (s && !s.problem) {
      const list = await api.listSites(fail)
      if (list) setSites(list)
    }
    setVersion((v) => v + 1)
  }, [fail])

  useEffect(() => {
    reload()
    const offs = [
      onEvent('webvigil:changed', () => reload()),
      onEvent('webvigil:problem', (t) => fail(String(t))),
      onEvent('webvigil:show-changes', (id) => { setSelected(Number(id)); setTab('changes') }),
    ]
    const timer = window.setInterval(() => setNow(new Date()), CLOCK_TICK_MS)
    const mq = window.matchMedia?.(DARK_QUERY)
    const onScheme = (e: MediaQueryListEvent) => setSystemDark(e.matches)
    mq?.addEventListener('change', onScheme)
    return () => { offs.forEach((off) => off()); window.clearInterval(timer); mq?.removeEventListener('change', onScheme) }
  }, [reload, fail])

  const appearance: Appearance = resolveTheme(state?.settings.theme ?? 'system', systemDark)
  useEffect(() => { document.documentElement.dataset.theme = appearance }, [appearance])

  const site = useMemo(() => sites.find((s) => s.id === selected) ?? null, [sites, selected])
  const unavailable = !state || state.problem !== ''

  const openEdit = async (id: number) => {
    const form = await api.getSite(id, fail)
    if (form) setDialog({kind: 'site', form})
  }
  const openAdd = async () => {
    const form = await api.newSiteForm(fail)
    if (form) setDialog({kind: 'site', form})
  }
  const doExport = async () => {
    const r = await api.exportSites([], fail)
    if (r && r.path) setMessage({text: `Exported ${r.count} sites to ${r.path}`, error: false})
  }
  const doImport = async () => {
    const p = await api.previewImport(fail)
    if (p && p.path) setDialog({kind: 'import', preview: p})
  }
  // The licence is titled by About's licence name, its one home, rather than a guess from the text; a
  // failed About read leaves the plain title rather than keeping the licence from opening.
  const openAbout = async () => {
    const about = await api.about(fail)
    if (about) setDialog({kind: 'about', about})
  }
  const openLicence = async () => {
    const text = await api.licenceText(fail)
    if (text === null) return
    const about = await api.about(() => undefined)
    setDialog({kind: 'licence', name: about?.licence ?? '', text})
  }
  const updates = useUpdateCheck(fail)
  const close = () => setDialog(null)

  return (
    <div className="shell">
      <Toolbar appearance={appearance} paused={state?.settings.paused ?? false} disabled={unavailable}
        hasSites={sites.length > 0} note={pendingSummary(sites)}
        onAdd={openAdd} onImport={doImport} onExport={doExport}
        onRefreshAll={() => api.refreshAll(fail)}
        onTogglePause={() => api.setPaused(!state?.settings.paused, fail).then(reload)}
        onToggleTheme={(t) => {
          if (unavailable) { document.documentElement.dataset.theme = t; return }
          api.setTheme(t, fail).then(reload)
        }}
        onSettings={() => setDialog({kind: 'settings'})} onGuide={() => setDialog({kind: 'guide'})}
        onAbout={openAbout} onLicence={openLicence} onCheckUpdates={updates.checkUpdates} />
      {state?.problem && <div className="banner error" role="alert">{state.problem}. The run log is at {state.logPath}.</div>}
      {message && <div className={message.error ? 'banner error' : 'banner'} role={message.error ? 'alert' : 'status'}>
        {message.text}<button className="btn" onClick={() => setMessage(null)}>Dismiss</button>
      </div>}
      <main className="main">
        <SiteTable sites={sites} selected={selected} now={now} onSelect={setSelected}
          onRefresh={(id) => api.refresh(id, fail)} onEdit={openEdit} onDelete={(s) => setDialog({kind: 'delete', site: s})} />
        {site && <Details site={site} tab={tab} version={version} onTab={setTab} onError={fail} />}
      </main>
      <BottomBar onError={fail} />
      {dialog?.kind === 'site' && <SiteDialog initial={dialog.form} onCancel={close} onSaved={() => { close(); reload() }} />}
      {dialog?.kind === 'delete' && <ConfirmDelete site={dialog.site} onCancel={close}
        onDelete={() => { api.deleteSite(dialog.site.id, fail).then(reload); close() }}
        onDisableInstead={() => { api.setEnabled(dialog.site.id, false, fail).then(reload); close() }} />}
      {dialog?.kind === 'import' && <ImportDialog preview={dialog.preview} onCancel={close}
        onDone={(text) => { close(); setMessage({text, error: false}); reload() }} />}
      {dialog?.kind === 'settings' && state && <SettingsDialog initial={state.settings} onCancel={close} onSaved={() => { close(); reload() }} />}
      {dialog?.kind === 'guide' && <GuideDialog onClose={close} />}
      {dialog?.kind === 'about' && <AboutDialog about={dialog.about} onClose={close} />}
      {dialog?.kind === 'licence' && <LicenceDialog name={dialog.name} text={dialog.text} onClose={close} />}
      {updates.updateStatus && <UpdateDialog status={updates.updateStatus} onClose={updates.closeUpdate}
        onSkip={updates.skipUpdate} onDownload={(url) => api.openExternal(url, fail)} />}
    </div>
  )
}
