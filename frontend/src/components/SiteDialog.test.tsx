import {act, fireEvent, render, screen} from '@testing-library/react'
import type {ReadingDTO, SiteFormDTO} from '../wire'
import {SiteDialog} from './SiteDialog'

const PRODUCT = 'https://shop.example/dp/0123456789'

const blank: SiteFormDTO = {
  id: 0, name: '', url: '', enabled: true, intervalMinutes: 60, timeoutSeconds: 30, mode: 'extracted_text',
  selectors: [], fields: [], excludeSelectors: [], excludePatterns: [], caseInsensitive: false,
  collapseWhitespace: true, sound: true, visual: true, requireAck: true, failureThreshold: 3,
}

const reading: ReadingDTO = {
  found: true, name: 'Shop product page', mode: 'structured_fields',
  fields: [
    {name: 'Title', selector: '#title', attribute: '', optional: false},
    {name: 'Cover', selector: '#cover', attribute: 'src', optional: true},
  ],
  excludeSelectors: ['.rank'], excludePatterns: ['x+'],
}

const none: ReadingDTO = {found: false, name: '', mode: '', fields: [], excludeSelectors: [], excludePatterns: []}

type Answer = (url: string) => Promise<ReadingDTO>

// back fakes the one Go call this behaviour makes; every address gets an answer from answer.
function back(answer: Answer) {
  const calls: string[] = []
  const w = window as unknown as {go?: unknown}
  w.go = {main: {App: {SuggestedReading: (url: string) => { calls.push(url); return answer(url) }}}}
  return calls
}

const instant: Answer = async (url) => (url === PRODUCT ? reading : none)

async function settle() {
  await act(async () => { await Promise.resolve() })
}

function open(initial: SiteFormDTO = blank) {
  render(<SiteDialog initial={initial} onSaved={() => undefined} onCancel={() => undefined} />)
}

async function typeURL(url: string) {
  fireEvent.change(screen.getByLabelText('Website URL'), {target: {value: url}})
  await settle()
}

const mode = () => (screen.getByLabelText('Detection mode') as HTMLSelectElement).value
const fieldNames = () => screen.queryAllByLabelText('Field name').map((i) => (i as HTMLInputElement).value)
const text = (label: string) => (screen.getByLabelText(label) as HTMLTextAreaElement).value

afterEach(() => { delete (window as unknown as {go?: unknown}).go })

describe('SiteDialog ready-made readings', () => {
  it('fills a product page in as its address is entered and says why', async () => {
    back(instant)
    open()
    await typeURL(PRODUCT)
    expect(mode()).toBe('structured_fields')
    expect(fieldNames()).toEqual(['Title', 'Cover'])
    expect(text('Exclude selectors')).toBe('.rank')
    expect(text('Exclude patterns')).toBe('x+')
    expect(screen.getByText(/^Shop product page: only these fields are compared/)).toBeInTheDocument()
    expect(screen.queryByText(/Not saved yet/)).not.toBeInTheDocument()
  })

  it('fills an existing text-mode definition in when it opens, keeping its own rules', async () => {
    back(instant)
    open({...blank, id: 7, name: 'Book', url: PRODUCT, excludeSelectors: ['.mine']})
    await settle()
    expect(mode()).toBe('structured_fields')
    expect(text('Exclude selectors')).toBe('.mine\n.rank')
    expect(screen.getByText(/Not saved yet: press Save to compare these fields instead of the whole page\.$/)).toBeInTheDocument()
  })

  it('changes nothing for a page it has no reading for', async () => {
    back(instant)
    open()
    await typeURL('https://example.com/')
    expect(mode()).toBe('extracted_text')
    expect(text('Exclude selectors')).toBe('')
    expect(screen.queryByText(/only these fields are compared/)).not.toBeInTheDocument()
  })

  it('leaves a definition in a mode the user chose alone, without asking', async () => {
    const calls = back(instant)
    open({...blank, id: 7, url: PRODUCT, mode: 'css_selector', selectors: ['#mine']})
    await settle()
    expect(mode()).toBe('css_selector')
    expect(calls).toEqual([])
  })

  it('keeps text mode when the user moves a filled-in page back to it', async () => {
    back(instant)
    open()
    await typeURL(PRODUCT)
    fireEvent.change(screen.getByLabelText('Detection mode'), {target: {value: 'extracted_text'}})
    await settle()
    expect(mode()).toBe('extracted_text')
    expect(screen.queryByText(/only these fields are compared/)).not.toBeInTheDocument()
  })

  it('drops an answer that arrives for an address the user has since changed', async () => {
    let late: (r: ReadingDTO) => void = () => undefined
    back((url) => (url === PRODUCT ? new Promise((resolve) => { late = resolve }) : Promise.resolve(none)))
    open()
    await typeURL(PRODUCT)
    await typeURL('https://example.com/')
    late(reading)
    await settle()
    expect(mode()).toBe('extracted_text')
    expect(fieldNames()).toEqual([])
  })

  it('stays quiet when the back end cannot answer', async () => {
    open()
    await typeURL(PRODUCT)
    expect(mode()).toBe('extracted_text')
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })
})
