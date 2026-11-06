//go:build windows

package tray

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// Win32 constants used by the tray.
const (
	wmApp            = 0x8000
	wmTrayCallback   = wmApp + 1
	wmShowBalloon    = wmApp + 2
	wmUpdateTip      = wmApp + 3
	wmNull           = 0x0000
	wmClose          = 0x0010
	wmDestroy        = 0x0002
	wmLButtonUp      = 0x0202
	wmLButtonDblClk  = 0x0203
	wmRButtonUp      = 0x0205
	ninBalloonClick  = 0x0405 // NIN_BALLOONUSERCLICK
	nimAdd           = 0x0
	nimModify        = 0x1
	nimDelete        = 0x2
	nifMessage       = 0x01
	nifIcon          = 0x02
	nifTip           = 0x04
	nifInfo          = 0x10
	niifInfo         = 0x01
	mfString         = 0x0000
	mfSeparator      = 0x0800
	tpmRightButton   = 0x0002
	tpmNonotify      = 0x0080
	tpmReturnCmd     = 0x0100
	idiApplication   = 32512
	trayIconID       = 1
	balloonBuffer    = 16
	idOpen           = 1
	idRefresh        = 2
	idPause          = 3
	idExit           = 4
	lowWordMask      = 0xFFFF
	tipChars         = 128
	infoChars        = 256
	infoTitleChars   = 64
	firstIconIndex   = 0
	iconsToExtract   = 1
	trayClassName    = "WebVigilTray"
	taskbarCreatedID = "TaskbarCreated"
)

var (
	modshell32  = windows.NewLazySystemDLL("shell32.dll")
	modkernel32 = windows.NewLazySystemDLL("kernel32.dll")
	moduser32   = windows.NewLazySystemDLL("user32.dll")

	procShellNotifyIcon     = modshell32.NewProc("Shell_NotifyIconW")
	procExtractIconEx       = modshell32.NewProc("ExtractIconExW")
	procGetModuleHandle     = modkernel32.NewProc("GetModuleHandleW")
	procRegisterClassEx     = moduser32.NewProc("RegisterClassExW")
	procCreateWindowEx      = moduser32.NewProc("CreateWindowExW")
	procDefWindowProc       = moduser32.NewProc("DefWindowProcW")
	procGetMessage          = moduser32.NewProc("GetMessageW")
	procTranslateMessage    = moduser32.NewProc("TranslateMessage")
	procDispatchMessage     = moduser32.NewProc("DispatchMessageW")
	procPostQuitMessage     = moduser32.NewProc("PostQuitMessage")
	procPostMessage         = moduser32.NewProc("PostMessageW")
	procRegisterWindowMsg   = moduser32.NewProc("RegisterWindowMessageW")
	procCreatePopupMenu     = moduser32.NewProc("CreatePopupMenu")
	procAppendMenu          = moduser32.NewProc("AppendMenuW")
	procTrackPopupMenu      = moduser32.NewProc("TrackPopupMenu")
	procDestroyMenu         = moduser32.NewProc("DestroyMenu")
	procGetCursorPos        = moduser32.NewProc("GetCursorPos")
	procLoadIcon            = moduser32.NewProc("LoadIconW")
	procSetForegroundWindow = moduser32.NewProc("SetForegroundWindow")
)

// notifyIconData mirrors NOTIFYICONDATAW; cbSize covers the whole structure (Vista and later).
type notifyIconData struct {
	cbSize            uint32
	hWnd              windows.HWND
	uID               uint32
	uFlags            uint32
	uCallbackMessage  uint32
	hIcon             windows.Handle
	szTip             [tipChars]uint16
	dwState           uint32
	dwStateMask       uint32
	szInfo            [infoChars]uint16
	uVersionOrTimeout uint32
	szInfoTitle       [infoTitleChars]uint16
	dwInfoFlags       uint32
	guidItem          windows.GUID
	hBalloonIcon      windows.Handle
}

// wndClassEx mirrors WNDCLASSEXW.
type wndClassEx struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     windows.Handle
	hIcon         windows.Handle
	hCursor       windows.Handle
	hbrBackground windows.Handle
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       windows.Handle
}

type point struct{ x, y int32 }

// msg mirrors MSG.
type msg struct {
	hwnd     windows.HWND
	message  uint32
	wParam   uintptr
	lParam   uintptr
	time     uint32
	pt       point
	lPrivate uint32
}

// copyUTF16 writes s into a fixed UTF-16 field, always null-terminated.
func copyUTF16(dst []uint16, s string) {
	enc, err := windows.UTF16FromString(s)
	if err != nil {
		return
	}
	if copy(dst, enc) >= len(dst) {
		dst[len(dst)-1] = 0
	}
}

func appendMenuItem(menu uintptr, id uint32, label string) {
	p, err := windows.UTF16PtrFromString(label)
	if err != nil {
		return
	}
	procAppendMenu.Call(menu, mfString, uintptr(id), uintptr(unsafe.Pointer(p)))
}

func appendMenuSeparator(menu uintptr) {
	procAppendMenu.Call(menu, mfSeparator, 0, 0)
}
