import {useEffect, useRef, useState} from 'react'
import type {KeyboardEvent as ReactKeyboardEvent} from 'react'

// Menu is ported from PigeonPost's title-tray dropdown, trimmed to what WebVigil's one menu (Help)
// uses: leaf items and dividers, no flyouts, ticks or accelerator hints.

// MenuItem is one entry: a leaf with its action; else a divider between groups.
export interface MenuItem {
  label: string
  onClick?: () => void
  separator?: boolean
}

interface MenuProps {
  // title is the trigger's tooltip, accessible name and the dropdown's header.
  title: string
  // icon is the supplied artwork shown on the trigger.
  icon: string
  items: MenuItem[]
}

// HOVER_CLOSE_DELAY_MS is the grace the pointer gets after leaving a hover-opened menu before it
// closes: long enough to travel from the trigger into the dropdown without it slamming shut.
export const HOVER_CLOSE_DELAY_MS = 200

// directItems returns the item buttons of an open dropdown, in view order.
function directItems(panel: HTMLElement | null): HTMLButtonElement[] {
  if (!panel) return []
  return Array.from(panel.querySelectorAll<HTMLButtonElement>(':scope > .menu-item:not([disabled])'))
}

// Menu is a picture trigger that opens a list of items. Hovering the trigger opens it too (the
// dropdown is headed with the menu's name, since the picture alone does not carry it); it closes once
// the pointer has left both, on an outside click, on Escape and after an item is chosen. Keyboard:
// Down, Enter or Space on the trigger opens the menu onto its first item; Up and Down (wrapping) plus
// Home and End walk the items; Escape, Tab, Left or Right collapse back to the trigger so the window's
// focus ring carries on from there.
export function Menu({title, icon, items}: MenuProps) {
  const [open, setOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const dropdownRef = useRef<HTMLDivElement>(null)
  // openedByKeyboard records a keyboard open, so the first item is focused; a mouse open leaves focus
  // on the trigger (no stray ring) and Down from there still steps into the items.
  const openedByKeyboard = useRef(false)
  // openedByHover records that the pointer opened the menu on its way to the trigger, so the click
  // that follows keeps it open rather than toggling it shut under the pointer; a second click closes.
  const openedByHover = useRef(false)
  const hoverCloseTimer = useRef<number | null>(null)

  const cancelHoverClose = () => {
    if (hoverCloseTimer.current !== null) {
      window.clearTimeout(hoverCloseTimer.current)
      hoverCloseTimer.current = null
    }
  }

  useEffect(() => cancelHoverClose, [])

  useEffect(() => {
    if (open && openedByKeyboard.current) directItems(dropdownRef.current)[0]?.focus()
    if (!open) {
      openedByKeyboard.current = false
      openedByHover.current = false
    }
  }, [open])

  useEffect(() => {
    if (!open) return
    const onDocDown = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDocDown)
    return () => document.removeEventListener('mousedown', onDocDown)
  }, [open])

  const closeToTrigger = () => {
    setOpen(false)
    triggerRef.current?.focus()
  }

  const choose = (item: MenuItem) => {
    if (!item.onClick) return
    setOpen(false)
    item.onClick()
  }

  // Enter and Space also fire the trigger's native click (which toggles), so the open is handled here
  // and the key consumed; otherwise the click would toggle it straight back shut.
  const onTriggerKeyDown = (e: ReactKeyboardEvent) => {
    if (e.key === 'ArrowDown' || e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      e.stopPropagation()
      if (open) {
        directItems(dropdownRef.current)[0]?.focus()
      } else {
        openedByKeyboard.current = true
        setOpen(true)
      }
    }
  }

  const onDropdownKeyDown = (e: ReactKeyboardEvent) => {
    const list = directItems(dropdownRef.current)
    const index = list.indexOf(document.activeElement as HTMLButtonElement)
    const moves: Record<string, () => number> = {
      ArrowDown: () => (index + 1) % list.length,
      ArrowUp: () => (index - 1 + list.length) % list.length,
      Home: () => 0,
      End: () => list.length - 1,
    }
    if (moves[e.key]) {
      e.preventDefault()
      e.stopPropagation()
      list[moves[e.key]()]?.focus()
    } else if (e.key === 'Escape' || e.key === 'Tab' || e.key === 'ArrowLeft' || e.key === 'ArrowRight') {
      e.preventDefault()
      e.stopPropagation()
      closeToTrigger()
    }
  }

  return (
    <div
      className="menu"
      ref={menuRef}
      onMouseEnter={() => {
        cancelHoverClose()
        if (!open) openedByHover.current = true
        setOpen(true)
      }}
      onMouseLeave={() => {
        cancelHoverClose()
        hoverCloseTimer.current = window.setTimeout(() => setOpen(false), HOVER_CLOSE_DELAY_MS)
      }}
    >
      <button
        ref={triggerRef}
        className={'icon-btn menu-title' + (open ? ' active' : '')}
        title={open ? undefined : title}
        aria-label={title}
        aria-haspopup="menu"
        aria-expanded={open}
        onClick={() => {
          const keep = openedByHover.current
          openedByHover.current = false
          setOpen((v) => keep || !v)
        }}
        onKeyDown={onTriggerKeyDown}
      >
        <img src={icon} alt="" draggable={false} />
      </button>
      {open && (
        <div className="menu-dropdown" role="menu" aria-label={title} ref={dropdownRef} onKeyDown={onDropdownKeyDown}>
          <div className="menu-header" aria-hidden="true">{title}</div>
          {items.map((item, i) => item.separator
            ? <div key={i} className="menu-sep" role="separator" />
            : <button key={i} className="menu-item" role="menuitem" onClick={() => choose(item)}>{item.label}</button>)}
        </div>
      )}
    </div>
  )
}
