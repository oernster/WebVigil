import {act, renderHook} from '@testing-library/react'
import {api} from '../api'
import type {UpdateStatusDTO} from '../wire'
import {CHECK_INTERVAL_MS, LAUNCH_CHECK_DELAY_MS, SKIPPED_UPDATE_KEY, useUpdateCheck} from './useUpdateCheck'

const newer: UpdateStatusDTO = {current: '1.0.0', latest: '1.1.0', updateAvailable: true, downloadUrl: '', pageUrl: 'https://page'}
const nothing: UpdateStatusDTO = {...newer, updateAvailable: false}

// settle lets the awaited check resolve inside act, so its state update lands.
const settle = () => act(async () => { await Promise.resolve() })

beforeEach(() => {
  vi.useFakeTimers()
  localStorage.clear()
})
afterEach(() => {
  vi.useRealTimers()
  vi.restoreAllMocks()
})

describe('useUpdateCheck', () => {
  it('checks shortly after launch and then daily, passing the skipped version', async () => {
    localStorage.setItem(SKIPPED_UPDATE_KEY, '1.1.0')
    const check = vi.spyOn(api, 'checkForUpdates').mockResolvedValue(nothing)
    renderHook(() => useUpdateCheck(vi.fn()))
    expect(check).not.toHaveBeenCalled()
    await act(async () => { vi.advanceTimersByTime(LAUNCH_CHECK_DELAY_MS) })
    expect(check).toHaveBeenCalledTimes(1)
    expect(check.mock.calls[0][0]).toBe('1.1.0')
    await act(async () => { vi.advanceTimersByTime(CHECK_INTERVAL_MS) })
    expect(check).toHaveBeenCalledTimes(2)
  })

  it('stays silent when an automatic check finds nothing or is refused', async () => {
    const onError = vi.fn()
    const check = vi.spyOn(api, 'checkForUpdates').mockImplementation(async (_s, refused) => { refused('no back end'); return null })
    const {result} = renderHook(() => useUpdateCheck(onError))
    await act(async () => { vi.advanceTimersByTime(LAUNCH_CHECK_DELAY_MS) })
    expect(check).toHaveBeenCalled()
    expect(onError).not.toHaveBeenCalled()
    expect(result.current.updateStatus).toBeNull()
  })

  it('offers a newer release found automatically', async () => {
    vi.spyOn(api, 'checkForUpdates').mockResolvedValue(newer)
    const {result} = renderHook(() => useUpdateCheck(vi.fn()))
    await act(async () => { vi.advanceTimersByTime(LAUNCH_CHECK_DELAY_MS) })
    await settle()
    expect(result.current.updateStatus).toEqual(newer)
    act(() => result.current.closeUpdate())
    expect(result.current.updateStatus).toBeNull()
  })

  it('a manual check ignores the skip and reports every outcome', async () => {
    localStorage.setItem(SKIPPED_UPDATE_KEY, '1.1.0')
    const check = vi.spyOn(api, 'checkForUpdates').mockResolvedValue(nothing)
    const {result} = renderHook(() => useUpdateCheck(vi.fn()))
    act(() => result.current.checkUpdates())
    await settle()
    expect(check.mock.calls[0][0]).toBe('')
    expect(result.current.updateStatus).toEqual(nothing)
  })

  it('a refused manual check says why', async () => {
    const onError = vi.fn()
    vi.spyOn(api, 'checkForUpdates').mockImplementation(async (_s, refused) => { refused('no back end'); return null })
    const {result} = renderHook(() => useUpdateCheck(onError))
    act(() => result.current.checkUpdates())
    await settle()
    expect(onError).toHaveBeenCalledWith('no back end')
    expect(result.current.updateStatus).toBeNull()
  })

  it('remembers a skip, surviving storage that refuses', () => {
    const {result} = renderHook(() => useUpdateCheck(vi.fn()))
    act(() => result.current.skipUpdate('1.1.0'))
    expect(localStorage.getItem(SKIPPED_UPDATE_KEY)).toBe('1.1.0')
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => { throw new Error('denied') })
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => { throw new Error('denied') })
    expect(() => act(() => result.current.skipUpdate('1.2.0'))).not.toThrow()
  })

  it('stops its timers when the window goes away', () => {
    const before = vi.getTimerCount()
    const {unmount} = renderHook(() => useUpdateCheck(vi.fn()))
    // The launch check and the daily one, both armed; measured against a baseline because a timer
    // from an earlier test's unmount can still be on the fake clock.
    expect(vi.getTimerCount()).toBe(before + 2)
    unmount()
    expect(vi.getTimerCount()).toBe(before)
  })
})
