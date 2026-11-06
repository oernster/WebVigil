import {render, screen} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type {SiteDTO} from '../wire'
import {ConfirmDelete} from './ConfirmDelete'

const site = {id: 4, name: 'Shop page', url: 'https://example.com/item', enabled: true} as SiteDTO

describe('ConfirmDelete (FR-008)', () => {
  it('names the site, states the history goes and cancels cleanly', async () => {
    const onDelete = vi.fn()
    const onCancel = vi.fn()
    render(<ConfirmDelete site={site} onDelete={onDelete} onDisableInstead={vi.fn()} onCancel={onCancel} />)
    expect(screen.getByText('Shop page')).toBeInTheDocument()
    expect(screen.getByText(/check history, snapshots and change records/)).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', {name: 'Cancel'}))
    expect(onCancel).toHaveBeenCalled()
    expect(onDelete).not.toHaveBeenCalled()
  })
  it('starts on Cancel, not on Delete', () => {
    render(<ConfirmDelete site={site} onDelete={vi.fn()} onDisableInstead={vi.fn()} onCancel={vi.fn()} />)
    expect(screen.getByRole('button', {name: 'Cancel'})).toHaveFocus()
  })
  it('offers disabling instead', async () => {
    const onDisable = vi.fn()
    render(<ConfirmDelete site={site} onDelete={vi.fn()} onDisableInstead={onDisable} onCancel={vi.fn()} />)
    await userEvent.click(screen.getByRole('button', {name: 'Disable instead'}))
    expect(onDisable).toHaveBeenCalled()
  })
})
