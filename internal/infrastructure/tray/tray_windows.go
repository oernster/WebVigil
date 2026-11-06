//go:build windows

package tray

import (
	"log"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Tray is the notification-area icon. It owns a hidden window on its own locked OS thread, since
// tray callbacks arrive as window messages on the thread that created the icon.
type Tray struct {
	actions  Actions
	balloons chan [2]string
	hwnd     atomic.Uintptr
	icon     windows.Handle
	paused   atomic.Bool
	mu       sync.Mutex
	tip      string
	done     chan struct{}
}

// activeTray is reached by the window procedure, a C callback that cannot carry a receiver. There
// is one tray per process.
var activeTray *Tray

var taskbarCreated uint32

var wndProc = syscall.NewCallback(trayWndProc)

// New returns a tray with the given initial tooltip.
func New(tip string) *Tray {
	return &Tray{balloons: make(chan [2]string, balloonBuffer), tip: tip, done: make(chan struct{})}
}

// Start records the callbacks and starts the tray thread.
func (t *Tray) Start(actions Actions) {
	t.actions = actions
	go t.run()
}

// Stop removes the icon and ends the tray thread, waiting for it to finish.
func (t *Tray) Stop() {
	h := t.hwnd.Load()
	if h == 0 {
		return
	}
	procPostMessage.Call(h, wmClose, 0, 0)
	<-t.done
}

// SetState updates the tooltip and the pause item's label.
func (t *Tray) SetState(tip string, paused bool) {
	t.mu.Lock()
	t.tip = tip
	t.mu.Unlock()
	t.paused.Store(paused)
	if h := t.hwnd.Load(); h != 0 {
		procPostMessage.Call(h, wmUpdateTip, 0, 0)
	}
}

// Balloon shows a notification balloon on the icon; Windows renders it as a toast.
func (t *Tray) Balloon(title, body string) {
	h := t.hwnd.Load()
	if h == 0 {
		return
	}
	select {
	case t.balloons <- [2]string{title, body}:
		procPostMessage.Call(h, wmShowBalloon, 0, 0)
	default:
	}
}

func (t *Tray) run() {
	defer close(t.done)
	defer func() {
		if p := recover(); p != nil {
			log.Printf("tray thread panicked: %v", p)
		}
	}()
	runtime.LockOSThread()
	activeTray = t
	hinst, _, _ := procGetModuleHandle.Call(0)
	className, _ := windows.UTF16PtrFromString(trayClassName)
	wc := wndClassEx{lpfnWndProc: wndProc, hInstance: windows.Handle(hinst), lpszClassName: className}
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))
	hwnd, _, err := procCreateWindowEx.Call(0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(className)),
		0, 0, 0, 0, 0, 0, 0, hinst, 0)
	if hwnd == 0 {
		log.Printf("tray: create window: %v", err)
		return
	}
	t.hwnd.Store(hwnd)
	if name, err := windows.UTF16PtrFromString(taskbarCreatedID); err == nil {
		id, _, _ := procRegisterWindowMsg.Call(uintptr(unsafe.Pointer(name)))
		taskbarCreated = uint32(id)
	}
	t.icon = loadIcon()
	t.addIcon()
	var m msg
	for {
		r, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}
	t.deleteIcon()
}

// loadIcon reads the small icon out of WebVigil.exe itself, so the tray, taskbar and shortcuts all
// show the one icon placed on the executable. The generic application icon is the last resort.
func loadIcon() windows.Handle {
	if exe, err := os.Executable(); err == nil {
		if p, err := windows.UTF16PtrFromString(exe); err == nil {
			var large, small windows.Handle
			n, _, _ := procExtractIconEx.Call(uintptr(unsafe.Pointer(p)), firstIconIndex,
				uintptr(unsafe.Pointer(&large)), uintptr(unsafe.Pointer(&small)), iconsToExtract)
			if n > 0 && small != 0 {
				return small
			}
			if large != 0 {
				return large
			}
		}
	}
	h, _, _ := procLoadIcon.Call(0, idiApplication)
	return windows.Handle(h)
}

func trayWndProc(hwnd, message, wParam, lParam uintptr) uintptr {
	t := activeTray
	if t != nil && taskbarCreated != 0 && uint32(message) == taskbarCreated {
		t.addIcon() // Explorer restarted: put the icon back
		return 0
	}
	if t != nil {
		switch uint32(message) {
		case wmTrayCallback:
			switch lParam & lowWordMask {
			case wmLButtonUp, wmLButtonDblClk:
				invoke(t.actions.Open)
			case wmRButtonUp:
				t.showMenu()
			case ninBalloonClick:
				invoke(t.actions.BalloonClicked)
			}
			return 0
		case wmShowBalloon:
			t.drainBalloons()
			return 0
		case wmUpdateTip:
			t.modifyTip()
			return 0
		case wmDestroy:
			procPostQuitMessage.Call(0)
			return 0
		}
	}
	r, _, _ := procDefWindowProc.Call(hwnd, message, wParam, lParam)
	return r
}

// showMenu pops up the menu at the cursor; the foreground and trailing null message are the
// standard idiom that makes the menu close on an outside click.
func (t *Tray) showMenu() {
	h := t.hwnd.Load()
	procSetForegroundWindow.Call(h)
	menu, _, _ := procCreatePopupMenu.Call()
	appendMenuItem(menu, idOpen, labelOpen)
	appendMenuItem(menu, idRefresh, labelRefresh)
	appendMenuItem(menu, idPause, pauseLabel(t.paused.Load()))
	appendMenuSeparator(menu)
	appendMenuItem(menu, idExit, labelExit)
	var pt point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	cmd, _, _ := procTrackPopupMenu.Call(menu, tpmReturnCmd|tpmRightButton|tpmNonotify, uintptr(pt.x), uintptr(pt.y), 0, h, 0)
	procPostMessage.Call(h, wmNull, 0, 0)
	procDestroyMenu.Call(menu)
	switch cmd {
	case idOpen:
		invoke(t.actions.Open)
	case idRefresh:
		invoke(t.actions.RefreshAll)
	case idPause:
		invoke(t.actions.TogglePause)
	case idExit:
		invoke(t.actions.Exit)
	}
}

func (t *Tray) drainBalloons() {
	for {
		select {
		case b := <-t.balloons:
			n := t.nid()
			n.uFlags = nifInfo
			n.dwInfoFlags = niifInfo
			copyUTF16(n.szInfoTitle[:], b[0])
			copyUTF16(n.szInfo[:], b[1])
			procShellNotifyIcon.Call(nimModify, uintptr(unsafe.Pointer(&n)))
		default:
			return
		}
	}
}

func (t *Tray) nid() notifyIconData {
	var n notifyIconData
	n.cbSize = uint32(unsafe.Sizeof(n))
	n.hWnd = windows.HWND(t.hwnd.Load())
	n.uID = trayIconID
	return n
}

func (t *Tray) currentTip() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.tip
}

func (t *Tray) addIcon() {
	n := t.nid()
	n.uFlags = nifMessage | nifIcon | nifTip
	n.uCallbackMessage = wmTrayCallback
	n.hIcon = t.icon
	copyUTF16(n.szTip[:], t.currentTip())
	procShellNotifyIcon.Call(nimAdd, uintptr(unsafe.Pointer(&n)))
}

func (t *Tray) modifyTip() {
	n := t.nid()
	n.uFlags = nifTip
	copyUTF16(n.szTip[:], t.currentTip())
	procShellNotifyIcon.Call(nimModify, uintptr(unsafe.Pointer(&n)))
}

func (t *Tray) deleteIcon() {
	n := t.nid()
	procShellNotifyIcon.Call(nimDelete, uintptr(unsafe.Pointer(&n)))
}
