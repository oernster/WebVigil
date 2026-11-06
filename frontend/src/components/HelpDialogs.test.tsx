import {fireEvent, render, screen} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {icons} from '../icons'
import {lightInfo} from '../logic'
import type {AboutDTO, UpdateStatusDTO} from '../wire'
import {guideSections} from './guideContent'
import {AboutDialog, GuideDialog, LicenceDialog, UpdateDialog} from './HelpDialogs'
import {Modal} from './Modal'

const about: AboutDTO = {
  name: 'WebVigil', tagline: 'A tagline', version: '1.2.3', author: 'Someone', copyright: 'Copyright (c) Someone',
  licence: 'GPL-3.0', logPath: 'C:\\logs\\run.log', credits: [{name: 'Go', licence: 'BSD-3-Clause'}],
}

const status = (over: Partial<UpdateStatusDTO>): UpdateStatusDTO => ({
  current: '1.0.0', latest: '1.1.0', updateAvailable: true, downloadUrl: 'https://dl/setup.exe', pageUrl: 'https://page', ...over,
})

describe('GuideDialog', () => {
  it('draws every section and each entry with its real picture', () => {
    const {container} = render(<GuideDialog platform="windows" onClose={() => undefined} />)
    for (const section of guideSections('windows')) expect(screen.getByText(section.heading)).toBeInTheDocument()
    const pictures = Array.from(container.querySelectorAll('img.guide-icon')).map((i) => i.getAttribute('src'))
    for (const entry of guideSections('windows').flatMap((s) => s.entries ?? [])) expect(pictures).toContain(entry.icon)
    expect(pictures).toContain(icons.helpAbout)
  })
  it('draws the status lights from their one home', () => {
    render(<GuideDialog platform="windows" onClose={() => undefined} />)
    for (const info of Object.values(lightInfo)) {
      expect(screen.getByText(info.label)).toBeInTheDocument()
      expect(screen.getByText(`${info.meaning}.`)).toBeInTheDocument()
    }
  })
  it('opens on Close and the body takes no focus (a pane is not a stop)', () => {
    const {container} = render(<GuideDialog platform="windows" onClose={() => undefined} />)
    expect(document.activeElement).toHaveTextContent('Close')
    expect(container.querySelector('.content')).not.toHaveAttribute('tabindex')
  })
  it('tells each platform what closing the window does', () => {
    const {unmount} = render(<GuideDialog platform="windows" onClose={() => undefined} />)
    expect(screen.getByText(/minimise WebVigil to the tray or to quit/)).toBeInTheDocument()
    unmount()
    render(<GuideDialog platform="linux" onClose={() => undefined} />)
    expect(screen.getByText(/Closing the window quits WebVigil/)).toBeInTheDocument()
    expect(screen.queryByText(/tray/)).toBeNull()
  })
})

describe('AboutDialog', () => {
  it('shows the identity, the facts and the credits', () => {
    render(<AboutDialog about={about} onClose={() => undefined} />)
    for (const text of ['About WebVigil', 'A tagline', '1.2.3', 'Someone', 'GPL-3.0', 'C:\\logs\\run.log', 'Go', 'BSD-3-Clause']) {
      expect(screen.getAllByText(text).length).toBeGreaterThan(0)
    }
  })
})

describe('LicenceDialog', () => {
  it('names the licence in its title, falling back to the plain word', () => {
    const {rerender} = render(<LicenceDialog name="GPL-3.0" text="THE TEXT" onClose={() => undefined} />)
    expect(screen.getByRole('dialog', {name: 'Licence: GPL-3.0'})).toBeInTheDocument()
    expect(screen.getByText('THE TEXT')).toBeInTheDocument()
    rerender(<LicenceDialog name="" text="THE TEXT" onClose={() => undefined} />)
    expect(screen.getByRole('dialog', {name: 'Licence'})).toBeInTheDocument()
  })
})

describe('UpdateDialog', () => {
  it('offers the platform download, then closes', async () => {
    const onDownload = vi.fn()
    const onClose = vi.fn()
    render(<UpdateDialog status={status({})} onDownload={onDownload} onSkip={vi.fn()} onClose={onClose} />)
    expect(screen.getByText('WebVigil 1.1.0 is available. You are running 1.0.0.')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', {name: 'Download'}))
    expect(onDownload).toHaveBeenCalledWith('https://dl/setup.exe')
    expect(onClose).toHaveBeenCalled()
  })
  it('falls back to the release page when the release carries no asset for this platform', async () => {
    const onDownload = vi.fn()
    render(<UpdateDialog status={status({downloadUrl: ''})} onDownload={onDownload} onSkip={vi.fn()} onClose={vi.fn()} />)
    await userEvent.click(screen.getByRole('button', {name: 'Download'}))
    expect(onDownload).toHaveBeenCalledWith('https://page')
  })
  it('opens nothing when the release has no address at all', async () => {
    const onDownload = vi.fn()
    render(<UpdateDialog status={status({downloadUrl: '', pageUrl: ''})} onDownload={onDownload} onSkip={vi.fn()} onClose={vi.fn()} />)
    await userEvent.click(screen.getByRole('button', {name: 'Download'}))
    expect(onDownload).not.toHaveBeenCalled()
  })
  it('skips exactly the offered version; Later only closes', async () => {
    const onSkip = vi.fn()
    const onClose = vi.fn()
    const {rerender} = render(<UpdateDialog status={status({})} onDownload={vi.fn()} onSkip={onSkip} onClose={onClose} />)
    await userEvent.click(screen.getByRole('button', {name: 'Skip This Version'}))
    expect(onSkip).toHaveBeenCalledWith('1.1.0')
    rerender(<UpdateDialog status={status({})} onDownload={vi.fn()} onSkip={onSkip} onClose={onClose} />)
    await userEvent.click(screen.getByRole('button', {name: 'Later'}))
    expect(onSkip).toHaveBeenCalledTimes(1)
    expect(onClose).toHaveBeenCalledTimes(2)
  })
  it('answers a manual check that found nothing newer or could not ask', () => {
    const {rerender} = render(<UpdateDialog status={status({updateAvailable: false, latest: '1.0.0'})} onDownload={vi.fn()} onSkip={vi.fn()} onClose={vi.fn()} />)
    expect(screen.getByText('You are running the latest version.')).toBeInTheDocument()
    rerender(<UpdateDialog status={status({updateAvailable: false, latest: ''})} onDownload={vi.fn()} onSkip={vi.fn()} onClose={vi.fn()} />)
    expect(screen.getByText('The update check could not reach GitHub. Please try again later.')).toBeInTheDocument()
  })
})

describe('Modal stacking', () => {
  it('lets only the topmost dialog answer Escape', () => {
    const below = vi.fn()
    const above = vi.fn()
    render(<>
      <Modal title="Below" onClose={below} footer={null}>below</Modal>
      <Modal title="Above" onClose={above} footer={null}>above</Modal>
    </>)
    fireEvent.keyDown(window, {key: 'Escape'})
    expect(above).toHaveBeenCalledTimes(1)
    expect(below).not.toHaveBeenCalled()
  })
})
