import {ago, asLight, httpText, lines, patternLines, pendingSummary, resolveTheme, sortSites, toggleTarget} from './logic'
import type {SiteDTO} from './wire'

const base: SiteDTO = {
  id: 1, name: 'b', url: 'https://b/', enabled: true, light: 'green', running: false, outcome: 'success', category: '',
  httpStatus: 200, lastChecked: '', lastSuccess: '', lastChange: '', nextDue: '', details: '', hasBaseline: true,
  pendingChangeId: 0, consecutiveFailures: 0, mode: 'extracted_text', intervalMinutes: 60,
}

describe('logic', () => {
  it('formats relative times', () => {
    const now = new Date('2026-11-06T12:00:00Z')
    expect(ago('', now)).toBe('Never')
    expect(ago('2026-11-06T11:59:40Z', now)).toBe('just now')
    expect(ago('2026-11-06T11:55:00Z', now)).toBe('5 min ago')
    expect(ago('2026-11-06T09:00:00Z', now)).toBe('3 h ago')
  })
  it('never invents an HTTP status', () => {
    expect(httpText(0)).toBe('None')
    expect(httpText(503)).toBe('503')
  })
  it('sorts the status column by severity', () => {
    const sites = [{...base, id: 1, light: 'green'}, {...base, id: 2, light: 'red'}, {...base, id: 3, light: 'amber'}]
    expect(sortSites(sites, 'light', true).map((s) => s.id)).toEqual([2, 3, 1])
    expect(sortSites(sites, 'light', false).map((s) => s.id)).toEqual([1, 3, 2])
    expect(sortSites([{...base, name: 'Z'}, {...base, id: 2, name: 'a'}], 'name', true)[0].name).toBe('a')
  })
  it('resolves the theme and its toggle target', () => {
    expect(resolveTheme('system', true)).toBe('dark')
    expect(resolveTheme('system', false)).toBe('light')
    expect(resolveTheme('light', true)).toBe('light')
    expect(toggleTarget('dark')).toBe('light')
    expect(toggleTarget('light')).toBe('dark')
  })
  it('reads rule lists', () => {
    expect(lines(' a \n\n b ')).toEqual(['a', 'b'])
    expect(patternLines(' a \n\nb')).toEqual([' a ', 'b'])
    expect(asLight('purple')).toBe('grey')
  })
  it('counts pending changes', () => {
    expect(pendingSummary([base])).toBe('')
    expect(pendingSummary([{...base, pendingChangeId: 3}])).toBe('1 change to acknowledge')
    expect(pendingSummary([{...base, pendingChangeId: 3}, {...base, pendingChangeId: 4}])).toBe('2 changes to acknowledge')
  })
})
