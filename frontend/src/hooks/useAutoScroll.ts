import {useCallback, useEffect, useState} from 'react'
import {attachAutoScroll} from '../autoScrollDom'

// useAutoScroll gives a scrollable surface the self-reading cycle (autoScroll.ts, driven against the
// element by autoScrollDom.ts). It returns a ref callback to put on the element that actually
// scrolls, so a surface mounted and unmounted with its dialog starts a fresh cycle each time it opens.
export function useAutoScroll(): (node: HTMLElement | null) => void {
  const [node, setNode] = useState<HTMLElement | null>(null)
  useEffect(() => (node ? attachAutoScroll(node) : undefined), [node])
  return useCallback((next: HTMLElement | null) => setNode(next), [])
}
