import {useAutoScroll} from '../hooks/useAutoScroll'
import {icons} from '../icons'
import {lightInfo, type Light} from '../logic'
import type {AboutDTO, UpdateStatusDTO} from '../wire'
import {guideSections} from './guideContent'
import {Modal} from './Modal'
import {StatusLight} from './StatusLight'

// The dialogs behind the Help menu, ported from PigeonPost's GuideModal, AboutModal, LicenceModal and
// UpdateModal onto WebVigil's own Modal. Each reading dialog hands its body to useAutoScroll: the body
// is the scroller and Close sits in the footer outside it, so the way out never scrolls away while
// the text reads itself.

const LIGHT_ORDER = Object.keys(lightInfo) as Light[]

const closeButton = (onClose: () => void) => <button className="btn primary" onClick={onClose}>Close</button>

// GuideDialog names the furniture with its real pictures, then the rules behind the behaviour. Its
// words live in guideContent.ts; this only draws them.
export function GuideDialog({onClose}: {onClose: () => void}) {
  const autoScroll = useAutoScroll()
  return (
    <Modal title="How WebVigil works" icon={icons.helpAbout} variant="guide" bodyRef={autoScroll} onClose={onClose} footer={closeButton(onClose)}>
      {guideSections.map((section) => (
        <section className="guide-section" key={section.heading}>
          <h3 className="guide-heading">{section.heading}</h3>
          {section.intro && <p className="guide-intro">{section.intro}</p>}
          {section.entries?.map((entry) => (
            <p className="guide-entry" key={entry.name}>
              <img className="guide-icon" src={entry.icon} alt="" draggable={false} />
              <span><b>{entry.name}</b>: {entry.text}</span>
            </p>
          ))}
          {section.lights && LIGHT_ORDER.map((light) => (
            <p className="guide-entry" key={light}>
              <StatusLight light={light} running={false} pending={false} />
              <span>{lightInfo[light].meaning}.</span>
            </p>
          ))}
          {section.rules?.map((rule) => (
            <p className="guide-rule" key={rule.title}><b>{rule.title}</b> {rule.text}</p>
          ))}
          {section.paragraphs?.map((text) => <p className="guide-para" key={text}>{text}</p>)}
        </section>
      ))}
    </Modal>
  )
}

// AboutDialog states what WebVigil is, who made it, its licence and the open-source work it ships.
export function AboutDialog({about, onClose}: {about: AboutDTO; onClose: () => void}) {
  const autoScroll = useAutoScroll()
  return (
    <Modal title={`About ${about.name}`} icon={icons.helpAbout} narrow bodyRef={autoScroll} onClose={onClose} footer={closeButton(onClose)}>
      <div className="about">
        <img src={icons.appIcon} alt={about.name} />
        <h2>{about.name}</h2>
        <p>{about.tagline}</p>
      </div>
      <div className="about-lines">
        <div><span className="about-label">Version</span>{about.version}</div>
        <div><span className="about-label">Author</span>{about.author}</div>
        <div><span className="about-label">Licence</span>{about.licence}</div>
        <div><span className="about-label">Run log</span><span className="about-path">{about.logPath}</span></div>
        <div className="about-copyright">{about.copyright}</div>
      </div>
      <hr className="about-rule" />
      <h3 className="about-credits-title">Open source credits</h3>
      <ul className="about-credits">
        {about.credits.map((credit) => (
          <li key={credit.name}>
            <span className="credit-name">{credit.name}</span>
            <span className="credit-licence">{credit.licence}</span>
          </li>
        ))}
      </ul>
    </Modal>
  )
}

// LicenceDialog shows the full licence. Its title names the licence, so which one this is stays on
// screen once its opening lines have scrolled away.
export function LicenceDialog({name, text, onClose}: {name: string; text: string; onClose: () => void}) {
  const autoScroll = useAutoScroll()
  return (
    <Modal title={name ? `Licence: ${name}` : 'Licence'} icon={icons.helpAbout} variant="licence" bodyRef={autoScroll} onClose={onClose} footer={closeButton(onClose)}>
      <pre className="licence-text">{text}</pre>
    </Modal>
  )
}

// UpdateDialog is the outcome of an update check. A newer release offers Download, Skip This Version
// and Later; a manual check that found nothing says up to date or unreachable in the same dialog, so
// every outcome of Help > Check for Updates is answered somewhere visible.
export function UpdateDialog({status, onDownload, onSkip, onClose}: {
  status: UpdateStatusDTO
  onDownload: (url: string) => void
  onSkip: (version: string) => void
  onClose: () => void
}) {
  if (!status.updateAvailable) {
    return (
      <Modal title="Check for updates" icon={icons.appIcon} narrow onClose={onClose} footer={closeButton(onClose)}>
        <p>{status.latest
          ? 'You are running the latest version.'
          : 'The update check could not reach GitHub. Please try again later.'}</p>
      </Modal>
    )
  }
  const url = status.downloadUrl || status.pageUrl
  return (
    <Modal title="Update available" icon={icons.appIcon} narrow onClose={onClose} footer={<>
      <button className="btn primary" onClick={() => { if (url) onDownload(url); onClose() }}>Download</button>
      <button className="btn" onClick={() => { onSkip(status.latest); onClose() }}>Skip This Version</button>
      <button className="btn" onClick={onClose}>Later</button>
    </>}>
      <p>WebVigil {status.latest} is available. You are running {status.current}.</p>
    </Modal>
  )
}
