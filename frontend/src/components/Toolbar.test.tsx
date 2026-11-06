import {render, screen} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {icons} from '../icons'
import {Toolbar} from './Toolbar'

const noop = () => undefined
const props = {
  paused: false, disabled: false, hasSites: true, hasSelection: true, note: '',
  onAdd: noop, onEdit: noop, onDelete: noop, onImport: noop, onExport: noop,
  onRefreshAll: noop, onTogglePause: noop, onSettings: noop,
  onGuide: noop, onAbout: noop, onLicence: noop, onCheckUpdates: noop,
}

describe('Toolbar', () => {
  it('shows the artwork of the appearance it switches to (FR-121)', async () => {
    const onToggle = vi.fn()
    const {rerender} = render(<Toolbar {...props} appearance="dark" onToggleTheme={onToggle} />)
    const toLight = screen.getByRole('button', {name: 'Switch to light appearance'})
    expect(toLight.querySelector('img')).toHaveAttribute('src', icons.lightMode)
    await userEvent.click(toLight)
    expect(onToggle).toHaveBeenCalledWith('light')
    rerender(<Toolbar {...props} appearance="light" onToggleTheme={onToggle} />)
    expect(screen.getByRole('button', {name: 'Switch to dark appearance'}).querySelector('img')).toHaveAttribute('src', icons.darkMode)
  })
  it('uses the supplied icons with accessible names', () => {
    render(<Toolbar {...props} appearance="dark" onToggleTheme={noop} />)
    for (const [name, src] of [['Add website', icons.addSite], ['Edit selected website', icons.editSite],
      ['Delete selected website', icons.deleteSite], ['Import sites', icons.importSites],
      ['Export sites', icons.exportSites], ['Refresh all', icons.refresh], ['Help', icons.helpAbout],
      ['Pause automatic monitoring', icons.pause], ['Settings', icons.settings]]) {
      expect(screen.getByRole('button', {name}).querySelector('img')).toHaveAttribute('src', src)
    }
  })
  it('offers Guide, About, Licence and Check for Updates on the Help menu, Guide first', async () => {
    const calls = {onGuide: vi.fn(), onAbout: vi.fn(), onLicence: vi.fn(), onCheckUpdates: vi.fn()}
    render(<Toolbar {...props} {...calls} appearance="dark" onToggleTheme={noop} />)
    await userEvent.click(screen.getByRole('button', {name: 'Help'}))
    const labels = screen.getAllByRole('menuitem').map((m) => m.textContent)
    expect(labels).toEqual(['Guide', 'About WebVigil', 'Licence', 'Check for Updates'])
    for (const [label, fn] of [['Guide', calls.onGuide], ['About WebVigil', calls.onAbout],
      ['Licence', calls.onLicence], ['Check for Updates', calls.onCheckUpdates]] as const) {
      if (!screen.queryByRole('menu')) await userEvent.click(screen.getByRole('button', {name: 'Help'}))
      await userEvent.click(screen.getByRole('menuitem', {name: label}))
      expect(fn).toHaveBeenCalledTimes(1)
    }
  })
  it('names the pause button by the action it will take', () => {
    render(<Toolbar {...props} paused appearance="dark" onToggleTheme={noop} />)
    const resume = screen.getByRole('button', {name: 'Resume automatic monitoring'})
    expect(resume).toHaveAttribute('aria-pressed', 'true')
    expect(resume.querySelector('img')).toHaveAttribute('src', icons.play)
    expect(screen.getByText('Monitoring paused')).toBeInTheDocument()
  })
  it('puts Edit and Delete straight after Add, then a divider before the rest', () => {
    const {container} = render(<Toolbar {...props} appearance="dark" onToggleTheme={noop} />)
    const order = Array.from(container.querySelector('.toolbar')!.children).slice(0, 5)
      .map((el) => el.classList.contains('sep') ? 'sep' : el.getAttribute('aria-label'))
    expect(order).toEqual(['Add website', 'Edit selected website', 'Delete selected website', 'sep', 'Import sites'])
  })
  it('acts on the selected site only while one is selected', async () => {
    const onEdit = vi.fn()
    const onDelete = vi.fn()
    const {rerender} = render(<Toolbar {...props} onEdit={onEdit} onDelete={onDelete} appearance="dark" onToggleTheme={noop} />)
    await userEvent.click(screen.getByRole('button', {name: 'Edit selected website'}))
    await userEvent.click(screen.getByRole('button', {name: 'Delete selected website'}))
    expect(onEdit).toHaveBeenCalledTimes(1)
    expect(onDelete).toHaveBeenCalledTimes(1)
    rerender(<Toolbar {...props} hasSelection={false} onEdit={onEdit} onDelete={onDelete} appearance="dark" onToggleTheme={noop} />)
    expect(screen.getByRole('button', {name: 'Edit selected website'})).toBeDisabled()
    expect(screen.getByRole('button', {name: 'Delete selected website'})).toBeDisabled()
  })
  it('disables what cannot be done', () => {
    render(<Toolbar {...props} hasSites={false} appearance="dark" onToggleTheme={noop} />)
    expect(screen.getByRole('button', {name: 'Export sites'})).toBeDisabled()
    expect(screen.getByRole('button', {name: 'Refresh all'})).toBeDisabled()
    expect(screen.getByRole('button', {name: 'Add website'})).toBeEnabled()
  })
})
