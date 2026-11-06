import donateIcon from '../assets/donate.png'
import {api, type Refused} from '../api'

// DONATE_URL is the payment page the donate button opens. It is the one home for the address; no
// other surface links to it. WebVigil never fetches it: the address is handed to the browser.
export const DONATE_URL = 'https://www.paypal.com/ncp/payment/4JDW9BNNG97CE'

export const DONATE_TIP = 'Donate to support WebVigil (opens your browser)'

// BottomBar is the tray at the foot of the window, built on the toolbar's own class so the two bars
// read as a matched pair (as in PigeonPost). It carries the donate button at the far left. WebVigil
// is free and stays free; nothing is held back behind a donation.
export function BottomBar({onError}: {onError: Refused}) {
  return (
    <footer className="toolbar bottombar">
      <button className="icon-btn donate" title={DONATE_TIP} aria-label={DONATE_TIP}
        onClick={() => api.openExternal(DONATE_URL, onError)}>
        <img src={donateIcon} alt="" draggable={false} />
      </button>
    </footer>
  )
}
