// The wire between Go and the page: one interface per DTO in dto.go, field for field.
// tests/structural/wire_test.go compares the two statements and fails on any difference.

export interface SiteDTO {
  id: number
  name: string
  url: string
  enabled: boolean
  light: string
  running: boolean
  outcome: string
  category: string
  httpStatus: number
  lastChecked: string
  lastSuccess: string
  lastChange: string
  nextDue: string
  details: string
  hasBaseline: boolean
  pendingChangeId: number
  consecutiveFailures: number
  mode: string
  intervalMinutes: number
}

export interface FieldRuleDTO {
  name: string
  selector: string
  attribute: string
  optional: boolean
}

export interface SiteFormDTO {
  id: number
  name: string
  url: string
  enabled: boolean
  intervalMinutes: number
  timeoutSeconds: number
  mode: string
  selectors: string[]
  fields: FieldRuleDTO[]
  excludeSelectors: string[]
  excludePatterns: string[]
  caseInsensitive: boolean
  collapseWhitespace: boolean
  sound: boolean
  visual: boolean
  requireAck: boolean
  failureThreshold: number
}

export interface SettingsDTO {
  theme: string
  paused: boolean
  startAtSignIn: boolean
  soundPath: string
  repeatAlarm: boolean
  foregroundDialog: boolean
}

export interface AppStateDTO {
  version: string
  problem: string
  logPath: string
  settings: SettingsDTO
  toasts: boolean
}

export interface CheckDTO {
  at: string
  httpStatus: number
  durationMs: number
  outcome: string
  category: string
  detail: string
  hash: string
  change: string
  manual: boolean
  finalUrl: string
  contentType: string
  bytes: number
}

export interface ChangeDTO {
  id: number
  detectedAt: string
  acknowledgedAt: string
  summary: string
  prevHash: string
  newHash: string
}

export interface HistoryDTO {
  checks: CheckDTO[]
  changes: ChangeDTO[]
}

export interface DiffLineDTO {
  op: string
  text: string
}

export interface FieldChangeDTO {
  name: string
  old: string
  new: string
}

export interface ChangeDetailDTO {
  change: ChangeDTO
  lines: DiffLineDTO[]
  fields: FieldChangeDTO[]
}

export interface FieldValueDTO {
  name: string
  value: string
}

export interface PreviewDTO {
  httpStatus: number
  category: string
  detail: string
  text: string
  fields: FieldValueDTO[]
}

export interface ImportItemDTO {
  index: number
  name: string
  url: string
  duplicate: boolean
  canUpdate: boolean
  action: string
}

export interface InvalidEntryDTO {
  index: number
  name: string
  url: string
  reason: string
}

export interface ImportPreviewDTO {
  path: string
  items: ImportItemDTO[]
  invalid: InvalidEntryDTO[]
}

export interface ImportSummaryDTO {
  imported: number
  updated: number
  skipped: number
  invalid: number
}

export interface ExportResultDTO {
  path: string
  count: number
}

export interface CreditDTO {
  name: string
  licence: string
}

export interface AboutDTO {
  name: string
  tagline: string
  version: string
  author: string
  copyright: string
  licence: string
  logPath: string
  credits: CreditDTO[]
}

export interface UpdateStatusDTO {
  current: string
  latest: string
  updateAvailable: boolean
  downloadUrl: string
  pageUrl: string
}
