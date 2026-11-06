import {render, screen} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {CloseChoiceDialog} from './CloseChoiceDialog'

function renderDialog(workOpen: boolean) {
  const handlers = {onMinimise: vi.fn(), onQuit: vi.fn(), onCancel: vi.fn()}
  render(<CloseChoiceDialog workOpen={workOpen} {...handlers} />)
  return handlers
}

describe('CloseChoiceDialog', () => {
  it('starts on Minimise to tray and offers Quit', async () => {
    const h = renderDialog(false)
    expect(screen.getByRole('button', {name: 'Minimise to tray'})).toHaveFocus()
    expect(screen.queryByRole('button', {name: 'Go back'})).toBeNull()
    expect(screen.queryByText(/lost if you quit/)).toBeNull()
    await userEvent.click(screen.getByRole('button', {name: 'Quit'}))
    expect(h.onQuit).toHaveBeenCalled()
    await userEvent.click(screen.getByRole('button', {name: 'Minimise to tray'}))
    expect(h.onMinimise).toHaveBeenCalled()
  })
  it('warns of unsaved work and starts on Go back, with Quit still offered', async () => {
    const h = renderDialog(true)
    expect(screen.getByText(/anything unsaved in it is lost if you quit/)).toBeInTheDocument()
    expect(screen.getByRole('button', {name: 'Go back'})).toHaveFocus()
    await userEvent.click(screen.getByRole('button', {name: 'Go back'}))
    expect(h.onCancel).toHaveBeenCalled()
    await userEvent.click(screen.getByRole('button', {name: 'Quit'}))
    expect(h.onQuit).toHaveBeenCalled()
  })
  it('stays open on Escape, cancelling the close', async () => {
    const h = renderDialog(false)
    await userEvent.keyboard('{Escape}')
    expect(h.onCancel).toHaveBeenCalled()
    expect(h.onQuit).not.toHaveBeenCalled()
  })
})
