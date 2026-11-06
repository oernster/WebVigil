// Every call into Go goes through here. Each takes a refusal handler as its last argument and
// resolves to null when Go refused, never rejecting: a call site cannot forget the failure path,
// because there is no promise rejection left to forget. null means "no answer"; an empty list is
// an answer.
import type {
  AboutDTO, AppStateDTO, ChangeDetailDTO, ExportResultDTO, HistoryDTO, ImportPreviewDTO, ImportSummaryDTO,
  PreviewDTO, SettingsDTO, SiteDTO, SiteFormDTO, UpdateStatusDTO,
} from './wire'

export type Refused = (message: string) => void

type Bound = Record<string, (...args: unknown[]) => Promise<unknown>>

interface WailsWindow {
  go?: {main?: {App?: Bound}}
  runtime?: {
    EventsOn: (name: string, cb: (...data: unknown[]) => void) => () => void
  }
}

const w = window as unknown as WailsWindow

function bound(): Bound | null {
  return w.go?.main?.App ?? null
}

export function errorText(e: unknown): string {
  if (typeof e === 'string') return e
  if (e instanceof Error) return e.message
  return 'An unexpected error occurred'
}

async function call<T>(method: string, args: unknown[], onRefused: Refused): Promise<T | null> {
  const app = bound()
  if (!app || typeof app[method] !== 'function') {
    onRefused('WebVigil is not connected to its back end')
    return null
  }
  try {
    return (await app[method](...args)) as T
  } catch (e) {
    onRefused(errorText(e))
    return null
  }
}

export const api = {
  state: (r: Refused) => call<AppStateDTO>('State', [], r),
  listSites: (r: Refused) => call<SiteDTO[]>('ListSites', [], r),
  newSiteForm: (r: Refused) => call<SiteFormDTO>('NewSiteForm', [], r),
  getSite: (id: number, r: Refused) => call<SiteFormDTO>('GetSite', [id], r),
  addSite: (f: SiteFormDTO, allowDuplicate: boolean, r: Refused) => call<number>('AddSite', [f, allowDuplicate], r),
  updateSite: (f: SiteFormDTO, rebaseline: boolean, allowDuplicate: boolean, r: Refused) =>
    okCall('UpdateSite', [f, rebaseline, allowDuplicate], r),
  deleteSite: (id: number, r: Refused) => okCall('DeleteSite', [id], r),
  setEnabled: (id: number, enabled: boolean, r: Refused) => okCall('SetEnabled', [id, enabled], r),
  refresh: (id: number, r: Refused) => okCall('Refresh', [id], r),
  refreshAll: (r: Refused) => call<number>('RefreshAll', [], r),
  acknowledge: (id: number, r: Refused) => okCall('Acknowledge', [id], r),
  openWebsite: (id: number, r: Refused) => okCall('OpenWebsite', [id], r),
  openExternal: (address: string, r: Refused) => okCall('OpenExternal', [address], r),
  preview: (f: SiteFormDTO, r: Refused) => call<PreviewDTO>('Preview', [f], r),
  history: (id: number, r: Refused) => call<HistoryDTO>('History', [id], r),
  changeDetail: (id: number, r: Refused) => call<ChangeDetailDTO>('ChangeDetail', [id], r),
  exportSites: (ids: number[], r: Refused) => call<ExportResultDTO>('ExportSites', [ids], r),
  previewImport: (r: Refused) => call<ImportPreviewDTO>('PreviewImport', [], r),
  confirmImport: (path: string, actions: Record<number, string>, r: Refused) =>
    call<ImportSummaryDTO>('ConfirmImport', [path, actions], r),
  saveSettings: (s: SettingsDTO, r: Refused) => okCall('SaveSettings', [s], r),
  setPaused: (paused: boolean, r: Refused) => okCall('SetPaused', [paused], r),
  setTheme: (theme: string, r: Refused) => okCall('SetTheme', [theme], r),
  chooseSound: (r: Refused) => call<string>('ChooseSound', [], r),
  testSound: (path: string, r: Refused) => okCall('TestSound', [path], r),
  quit: (r: Refused) => okCall('Quit', [], r),
  about: (r: Refused) => call<AboutDTO>('About', [], r),
  licenceText: (r: Refused) => call<string>('LicenceText', [], r),
  checkForUpdates: (skippedVersion: string, r: Refused) => call<UpdateStatusDTO>('CheckForUpdates', [skippedVersion], r),
}

// okCall is for a Go method with no result. Such a method resolves to null on success, so the
// refusal is tracked explicitly: true when it succeeded, null when it was refused.
async function okCall(method: string, args: unknown[], onRefused: Refused): Promise<true | null> {
  let refused = false
  await call<unknown>(method, args, (m) => {
    refused = true
    onRefused(m)
  })
  return refused ? null : true
}

// onEvent subscribes to a Go event and returns the unsubscribe function.
export function onEvent(name: string, cb: (...data: unknown[]) => void): () => void {
  return w.runtime?.EventsOn(name, cb) ?? (() => undefined)
}
