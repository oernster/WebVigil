import {
  TICK_MS,
  autoScrollTick,
  initialAutoScrollState,
  suspended,
} from './autoScroll'

// autoScrollDom is the DOM half of the self-reading cycle, ported from PigeonPost's useAutoScroll and
// lifted out of the hook so it has one home for two callers: the hook (the application's dialogs) and
// the setup program's licence screen. That page has no build step of its own, so build.ps1 bundles
// this module into a plain script beside it; nothing is copied by hand.

// MANUAL_EVENTS are the inputs that count as reading by hand and so suspend the cycle. mousedown
// covers a press on the native scrollbar as well as on the content; focusin covers keyboard
// navigation arriving in the surface: focus landing here is someone about to read, like a scroll.
const MANUAL_EVENTS = ['wheel', 'mousedown', 'touchstart', 'keydown', 'focusin'] as const

// MODAL_BACKDROP_CLASS is the class Modal puts on its backdrop: the one home for the name, so the
// freeze below and the dialogs it watches cannot drift apart.
export const MODAL_BACKDROP_CLASS = 'backdrop'

// isTopmostModalSurface reports whether the surface is the one the reader is actually looking at.
// Two surfaces reading at once compete for the eye, so a surface under another modal stops moving
// until that modal closes. A surface outside any modal reads only while no modal is open at all.
export function isTopmostModalSurface(node: HTMLElement): boolean {
  const selector = `.${MODAL_BACKDROP_CLASS}`
  const backdrops = document.querySelectorAll(selector)
  const own = node.closest(selector)
  if (!own) {
    return backdrops.length === 0
  }
  return backdrops[backdrops.length - 1] === own
}

// attachAutoScroll starts a fresh cycle on the element that actually scrolls and answers the function
// that stops it. A surface under another modal is FROZEN rather than suspended: the tick is skipped
// entirely, so phase, position and the remaining hold are all still there when the modal closes.
//
// The cycle is NOT gated on prefers-reduced-motion. On Windows that query follows the general
// "Animation effects" switch, which people turn off for performance rather than motion sensitivity,
// so gating on it silently removes the feature. Anyone who does not want the pane to read itself
// stops it by touching it, which is what the manual suspension is for.
export function attachAutoScroll(node: HTMLElement): () => void {
  let state = initialAutoScrollState()
  const onManualInput = () => {
    state = suspended(state)
  }
  for (const type of MANUAL_EVENTS) {
    node.addEventListener(type, onManualInput, {passive: true})
  }
  const timer = window.setInterval(() => {
    if (!isTopmostModalSurface(node)) {
      return
    }
    const view = {
      scrollTop: node.scrollTop,
      maxScrollTop: node.scrollHeight - node.clientHeight,
    }
    const {state: next, delta} = autoScrollTick(state, view)
    state = next
    if (delta !== 0) {
      node.scrollTop = view.scrollTop + delta
    }
  }, TICK_MS)
  return () => {
    window.clearInterval(timer)
    for (const type of MANUAL_EVENTS) {
      node.removeEventListener(type, onManualInput)
    }
  }
}
