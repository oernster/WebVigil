import {DEFAULT_MODE, filledForm, mergeLines, readingNote, SAVE_FIRST} from './reading'
import type {ReadingDTO, SiteFormDTO} from './wire'

const form: SiteFormDTO = {
  id: 0, name: '', url: 'https://shop.example/dp/0123456789', enabled: true, intervalMinutes: 60, timeoutSeconds: 30,
  mode: DEFAULT_MODE, selectors: [], fields: [], excludeSelectors: [], excludePatterns: [], caseInsensitive: false,
  collapseWhitespace: true, sound: true, visual: true, requireAck: true, failureThreshold: 3,
}

const reading: ReadingDTO = {
  found: true, name: 'Shop product page', mode: 'structured_fields',
  fields: [{name: 'Title', selector: '#title', attribute: '', optional: false}],
  excludeSelectors: ['.rank'], excludePatterns: ['x+'],
}

describe('filledForm', () => {
  it('puts a default-mode definition in the reading mode with a copy of its fields', () => {
    const next = filledForm(form, reading)
    expect(next.mode).toBe('structured_fields')
    expect(next.fields).toEqual(reading.fields)
    next.fields[0].selector = 'changed'
    expect(reading.fields[0].selector).toBe('#title')
    expect(next.url).toBe(form.url)
  })
  it('leaves a definition the user moved to another mode alone', () => {
    const own = {...form, mode: 'css_selector'}
    expect(filledForm(own, reading)).toBe(own)
  })
})

describe('mergeLines', () => {
  it('keeps the user rules and adds each missing one once', () => {
    expect(mergeLines('', ['.a', '.b'])).toBe('.a\n.b')
    expect(mergeLines('.mine\n.a\n', ['.a', '.b'])).toBe('.mine\n.a\n.b')
    expect(mergeLines('.mine', [])).toBe('.mine')
  })
})

describe('readingNote', () => {
  it('names the reading and says what it leaves out', () => {
    expect(readingNote('Shop product page', false)).toMatch(/^Shop product page: only these fields are compared/)
    expect(readingNote('x', false)).toContain('adverts and menus')
    expect(readingNote('x', false)).not.toContain(SAVE_FIRST)
  })
  it('tells an existing site its fields are not in use until it is saved', () => {
    expect(readingNote('x', true).endsWith(SAVE_FIRST)).toBe(true)
  })
})
