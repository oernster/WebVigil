import {useCallback, useEffect, useState} from 'react'
import {api, type Refused} from '../api'
import type {UpdateStatusDTO} from '../wire'

// Ported from PigeonPost. The localStorage key holding the release the user chose to skip. It is a
// presentation preference held page-side; the backend stays stateless about it and receives the
// skipped tag with each automatic check.
export const SKIPPED_UPDATE_KEY = 'skippedUpdateVersion'

// LAUNCH_CHECK_DELAY_MS keeps the automatic launch check clear of startup.
export const LAUNCH_CHECK_DELAY_MS = 3000
// One re-check per day while the app stays running.
const CHECK_INTERVAL_HOURS = 24
const MS_PER_HOUR = 60 * 60 * 1000
export const CHECK_INTERVAL_MS = CHECK_INTERVAL_HOURS * MS_PER_HOUR

// Storage can be refused (a locked-down profile); a skip then lasts only until the window closes,
// which is the right way to be wrong for a preference.
function readSkipped(): string {
  try {
    return localStorage.getItem(SKIPPED_UPDATE_KEY) ?? ''
  } catch {
    return ''
  }
}

function writeSkipped(version: string): void {
  try {
    localStorage.setItem(SKIPPED_UPDATE_KEY, version)
  } catch {
    // See readSkipped.
  }
}

const silent: Refused = () => undefined

// useUpdateCheck owns the update-check triggers and the dialog state. An automatic check (shortly
// after launch, then daily) shows the dialog only for a newer, non-skipped release and says nothing
// when it cannot ask. The manual Help > Check for Updates ignores the skip and shows every outcome;
// a manual call the back end refused reports why through onError rather than doing nothing.
export function useUpdateCheck(onError: Refused) {
  const [updateStatus, setUpdateStatus] = useState<UpdateStatusDTO | null>(null)

  const runCheck = useCallback(async (manual: boolean) => {
    const status = await api.checkForUpdates(manual ? '' : readSkipped(), manual ? onError : silent)
    if (status && (status.updateAvailable || manual)) setUpdateStatus(status)
  }, [onError])

  const checkUpdates = useCallback(() => { void runCheck(true) }, [runCheck])

  useEffect(() => {
    const launch = window.setTimeout(() => void runCheck(false), LAUNCH_CHECK_DELAY_MS)
    const periodic = window.setInterval(() => void runCheck(false), CHECK_INTERVAL_MS)
    return () => {
      window.clearTimeout(launch)
      window.clearInterval(periodic)
    }
  }, [runCheck])

  const skipUpdate = useCallback((version: string) => writeSkipped(version), [])
  const closeUpdate = useCallback(() => setUpdateStatus(null), [])

  return {updateStatus, checkUpdates, skipUpdate, closeUpdate}
}
