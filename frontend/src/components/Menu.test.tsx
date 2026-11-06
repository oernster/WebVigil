// Ported from PigeonPost: the menu opens on hover as well as click, keeps a grace period on leave and
// owns its keyboard contract.
import {act, cleanup, fireEvent, render, screen} from '@testing-library/react'
import {HOVER_CLOSE_DELAY_MS, Menu} from './Menu'

function renderMenu() {
  const first = vi.fn()
  const second = vi.fn()
  const view = render(<Menu title="Help" icon="H" items={[
    {label: 'Guide', onClick: first}, {label: '', separator: true}, {label: 'About', onClick: second},
  ]} />)
  const root = view.container.querySelector('.menu') as HTMLElement
  const trigger = screen.getByRole('button', {name: 'Help'})
  return {...view, root, trigger, first, second}
}

beforeEach(() => { vi.useFakeTimers() })
afterEach(() => { vi.useRealTimers(); cleanup() })

describe('Menu hover', () => {
  it('opens on hover, headed with the menu name', () => {
    const {root} = renderMenu()
    expect(screen.queryByRole('menu')).toBeNull()
    fireEvent.mouseEnter(root)
    expect(screen.getByRole('menu')).toBeInTheDocument()
    expect(screen.getByText('Guide')).toBeInTheDocument()
    expect(screen.getByRole('separator')).toBeInTheDocument()
  })

  it('closes only after the pointer has been gone for the grace period', () => {
    const {root} = renderMenu()
    fireEvent.mouseEnter(root)
    fireEvent.mouseLeave(root)
    expect(screen.getByRole('menu')).toBeInTheDocument()
    act(() => { vi.advanceTimersByTime(HOVER_CLOSE_DELAY_MS) })
    expect(screen.queryByRole('menu')).toBeNull()
  })

  it('coming back within the grace period keeps the menu open', () => {
    const {root} = renderMenu()
    fireEvent.mouseEnter(root)
    fireEvent.mouseLeave(root)
    fireEvent.mouseEnter(root)
    act(() => { vi.advanceTimersByTime(HOVER_CLOSE_DELAY_MS * 2) })
    expect(screen.getByRole('menu')).toBeInTheDocument()
  })

  it('choosing an item runs it and closes the menu', () => {
    const {root, first} = renderMenu()
    fireEvent.mouseEnter(root)
    fireEvent.click(screen.getByText('Guide'))
    expect(first).toHaveBeenCalled()
    expect(screen.queryByRole('menu')).toBeNull()
  })

  it('a click on a menu the pointer has just opened keeps it open; a second click closes it', () => {
    const {root, trigger} = renderMenu()
    fireEvent.mouseEnter(root)
    fireEvent.click(trigger)
    expect(screen.getByRole('menu')).toBeInTheDocument()
    fireEvent.click(trigger)
    expect(screen.queryByRole('menu')).toBeNull()
  })

  it('an outside press closes it', () => {
    const {trigger} = renderMenu()
    fireEvent.click(trigger)
    fireEvent.mouseDown(document.body)
    expect(screen.queryByRole('menu')).toBeNull()
  })
})

describe('Menu keyboard', () => {
  it('opens onto the first item and walks the items, wrapping', () => {
    const {trigger} = renderMenu()
    fireEvent.keyDown(trigger, {key: 'ArrowDown'})
    const menu = screen.getByRole('menu')
    expect(document.activeElement).toHaveTextContent('Guide')
    fireEvent.keyDown(menu, {key: 'ArrowDown'})
    expect(document.activeElement).toHaveTextContent('About')
    fireEvent.keyDown(menu, {key: 'ArrowDown'})
    expect(document.activeElement).toHaveTextContent('Guide')
    fireEvent.keyDown(menu, {key: 'ArrowUp'})
    expect(document.activeElement).toHaveTextContent('About')
    fireEvent.keyDown(menu, {key: 'Home'})
    expect(document.activeElement).toHaveTextContent('Guide')
    fireEvent.keyDown(menu, {key: 'End'})
    expect(document.activeElement).toHaveTextContent('About')
  })

  it('Down on the trigger of an open menu steps into the items', () => {
    const {trigger} = renderMenu()
    fireEvent.click(trigger)
    expect(document.activeElement?.getAttribute('role')).not.toBe('menuitem')
    fireEvent.keyDown(trigger, {key: 'ArrowDown'})
    expect(document.activeElement).toHaveTextContent('Guide')
  })

  it.each(['Escape', 'Tab', 'ArrowLeft', 'ArrowRight'])('%s collapses back to the trigger', (key) => {
    const {trigger} = renderMenu()
    fireEvent.keyDown(trigger, {key: 'Enter'})
    fireEvent.keyDown(screen.getByRole('menu'), {key})
    expect(screen.queryByRole('menu')).toBeNull()
    expect(document.activeElement).toBe(trigger)
  })

  it('ignores a key it does not own', () => {
    const {trigger} = renderMenu()
    fireEvent.keyDown(trigger, {key: 'a'})
    expect(screen.queryByRole('menu')).toBeNull()
    fireEvent.keyDown(trigger, {key: ' '})
    fireEvent.keyDown(screen.getByRole('menu'), {key: 'a'})
    expect(screen.getByRole('menu')).toBeInTheDocument()
  })
})
