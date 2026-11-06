import {icons} from '../icons'
import {Modal} from './Modal'

interface Props {
  // workOpen is true while a dialog holding unsaved edits sits behind this one.
  workOpen: boolean
  onMinimise: () => void
  onQuit: () => void
  onCancel: () => void
}

// CloseChoiceDialog is ported from PigeonPost's: on Windows the window's close button asks whether
// to keep WebVigil running in the tray or to quit. Escape or Cancel leaves the window open, so an
// accidental click loses nothing. Minimise is the default; while an edit is still open behind it,
// the dialog says so and Go back becomes the default instead, with Quit still on offer.
export function CloseChoiceDialog({workOpen, onMinimise, onQuit, onCancel}: Props) {
  return (
    <Modal title="Close WebVigil" icon={icons.appIcon} narrow onClose={onCancel} footer={<>
      <button className="btn left" onClick={onQuit}>Quit</button>
      {workOpen && <button className="btn primary" onClick={onCancel}>Go back</button>}
      <button className={workOpen ? 'btn' : 'btn primary'} onClick={onMinimise}>Minimise to tray</button>
    </>}>
      <p>Keep WebVigil running in the system tray or quit it? While it runs in the tray, the checks carry
        on and changes still raise the alarm. Quitting stops monitoring until WebVigil is started again.</p>
      {workOpen && <p>A dialog you were editing is still open; anything unsaved in it is lost if you
        quit. Go back to finish or save it first.</p>}
    </Modal>
  )
}
