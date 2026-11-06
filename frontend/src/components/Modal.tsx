import {useEffect, useRef, type ReactNode} from 'react'
import {MODAL_BACKDROP_CLASS, isTopmostModalSurface} from '../autoScrollDom'

interface Props {
  title: string
  icon?: string
  narrow?: boolean
  // variant is an extra class on the dialog for a surface with its own width (the guide, a licence).
  variant?: string
  // bodyRef receives the scrolling body, so a reading dialog can hand it to useAutoScroll. The body
  // is the scroller and the footer sits outside it, so Close stays pinned while the text reads itself.
  bodyRef?: (node: HTMLElement | null) => void
  onClose: () => void
  footer: ReactNode
  children: ReactNode
}

// Modal is a dialog: Escape closes it and focus starts on its first field rather than on a frame.
// The body never takes focus (no tabIndex): a pane is not a stop (noborderfocus).
export function Modal({title, icon, narrow, variant, bodyRef, onClose, footer, children}: Props) {
  const ref = useRef<HTMLDivElement>(null)
  useEffect(() => {
    // Priority, not document order: a field first, else the primary (safe) button, else any button.
    for (const sel of ['input, select, textarea', 'button.primary', 'button']) {
      const el = ref.current?.querySelector<HTMLElement>(sel)
      if (el) {
        el.focus()
        break
      }
    }
    const onKey = (e: KeyboardEvent) => {
      // Only the topmost dialog answers, so Escape closes one layer at a time (an update offer can
      // arrive over the Guide).
      if (e.key === 'Escape' && ref.current && isTopmostModalSurface(ref.current)) onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])
  return (
    <div className={MODAL_BACKDROP_CLASS}>
      <div className={['modal', narrow && 'narrow', variant].filter(Boolean).join(' ')} role="dialog" aria-modal="true" aria-label={title} ref={ref}>
        <header>
          {icon && <img src={icon} alt="" />}
          <h1>{title}</h1>
        </header>
        <div className="content" ref={bodyRef}>{children}</div>
        <footer>{footer}</footer>
      </div>
    </div>
  )
}
