// Pure logic for the interface: no React, no DOM, so every rule here is a unit test.
import type {SiteDTO} from './wire'

export type Light = 'grey' | 'green' | 'amber' | 'red'

// Status is never colour alone (FR-072): every light has a label and its own shape.
export const lightInfo: Record<Light, {label: string; shape: string; meaning: string}> = {
  grey: {label: 'Pending', shape: 'ring', meaning: 'No successful baseline yet'},
  green: {label: 'Healthy', shape: 'circle', meaning: 'Checked and compared; nothing new'},
  amber: {label: 'Changed', shape: 'diamond', meaning: 'A change awaits acknowledgement'},
  red: {label: 'Failed', shape: 'square', meaning: 'The latest check failed'},
}

export function asLight(v: string): Light {
  return v === 'green' || v === 'amber' || v === 'red' ? v : 'grey'
}

const MS_PER_SECOND = 1000
const SECONDS_PER_MINUTE = 60
const MINUTES_PER_HOUR = 60
const HOURS_PER_DAY = 24

// ago renders a stamp relative to now: "just now", "5 min ago", "3 h ago", else the date.
export function ago(stamp: string, now: Date): string {
  if (!stamp) return 'Never'
  const t = new Date(stamp)
  const seconds = Math.round((now.getTime() - t.getTime()) / MS_PER_SECOND)
  if (seconds < SECONDS_PER_MINUTE) return 'just now'
  const minutes = Math.floor(seconds / SECONDS_PER_MINUTE)
  if (minutes < MINUTES_PER_HOUR) return `${minutes} min ago`
  const hours = Math.floor(minutes / MINUTES_PER_HOUR)
  if (hours < HOURS_PER_DAY) return `${hours} h ago`
  return t.toLocaleDateString()
}

// full renders a stamp in the user's locale, for tooltips and the details panel.
export function full(stamp: string): string {
  return stamp ? new Date(stamp).toLocaleString() : 'Never'
}

export function httpText(status: number): string {
  return status === 0 ? 'None' : String(status)
}

export type SortKey = 'light' | 'name' | 'url' | 'httpStatus' | 'lastChecked' | 'lastSuccess' | 'lastChange' | 'details'

// Severity orders the status column so the rows needing attention rise to the top.
const severity: Record<Light, number> = {red: 0, amber: 1, grey: 2, green: 3}

export function sortSites(sites: SiteDTO[], key: SortKey, ascending: boolean): SiteDTO[] {
  const dir = ascending ? 1 : -1
  const value = (s: SiteDTO): string | number =>
    key === 'light' ? severity[asLight(s.light)] : key === 'httpStatus' ? s.httpStatus : String(s[key]).toLowerCase()
  return [...sites].sort((a, b) => {
    const va = value(a)
    const vb = value(b)
    if (va < vb) return -dir
    if (va > vb) return dir
    return a.name.localeCompare(b.name)
  })
}

export type Appearance = 'light' | 'dark'

// resolveTheme turns a preference into the appearance to paint.
export function resolveTheme(pref: string, systemDark: boolean): Appearance {
  if (pref === 'light' || pref === 'dark') return pref
  return systemDark ? 'dark' : 'light'
}

// toggleTarget is the appearance the toolbar toggle switches to, which is what it shows (FR-121).
export function toggleTarget(current: Appearance): Appearance {
  return current === 'dark' ? 'light' : 'dark'
}

// Interval presets offered by the dialog, in minutes (REQUIREMENTS.md FR-084).
export const intervalPresets = [5, 15, 30, 60, 120]

// lines splits a textarea into trimmed, non-empty rules.
export function lines(text: string): string[] {
  return text.split('\n').map((l) => l.trim()).filter((l) => l !== '')
}

// patternLines keeps surrounding spaces, which can matter in a regular expression.
export function patternLines(text: string): string[] {
  return text.split('\n').filter((l) => l !== '')
}

// pendingSummary is the count shown beside the toolbar when changes await acknowledgement.
export function pendingSummary(sites: SiteDTO[]): string {
  const n = sites.filter((s) => s.pendingChangeId !== 0).length
  if (n === 0) return ''
  return n === 1 ? '1 change to acknowledge' : `${n} changes to acknowledge`
}
