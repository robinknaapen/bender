//go:build windows

package win

import (
	"log"
	"unsafe"

	"github.com/pietjan/bender/internal/platform"
	"github.com/pietjan/bender/internal/platform/win/w32"
)

const trayIconID = 1

// Tray is the notification-area icon, its menu, and balloon notifications.
type Tray struct {
	win           *Window
	menu          []platform.MenuItem
	onActivate    func()
	onNotifyClick func()
}

func newTray(w *Window) *Tray {
	t := &Tray{win: w}
	data := t.data()
	data.UFlags = w32.NifMessage | w32.NifIcon | w32.NifTip
	data.UCallbackMessage = wmAppTray
	icon, _, _ := w32.LoadIcon.Call(0, uintptr(w32.IdiApplication))
	data.HIcon = icon
	w32.ShellNotifyIcon.Call(w32.NimAdd, uintptr(unsafe.Pointer(data)))
	data.UVersion = w32.NotifyIconVersion4
	w32.ShellNotifyIcon.Call(w32.NimSetVersion, uintptr(unsafe.Pointer(data)))
	return t
}

func (t *Tray) data() *w32.NotifyIconData {
	d := &w32.NotifyIconData{HWnd: t.win.hwnd, UID: trayIconID}
	d.CbSize = uint32(unsafe.Sizeof(*d))
	return d
}

// SetTooltip sets the hover text.
func (t *Tray) SetTooltip(tip string) {
	d := t.data()
	d.UFlags = w32.NifTip
	copyUTF16(d.SzTip[:], tip)
	w32.ShellNotifyIcon.Call(w32.NimModify, uintptr(unsafe.Pointer(d)))
}

// SetMenu sets the right-click menu.
func (t *Tray) SetMenu(items []platform.MenuItem) { t.menu = items }

// Notify shows a balloon notification (a native toast on Windows 10/11).
func (t *Tray) Notify(title, body string) {
	d := t.data()
	d.UFlags = w32.NifInfo
	d.DwInfoFlags = w32.NiifInfo
	copyUTF16(d.SzInfoTitle[:], title)
	copyUTF16(d.SzInfo[:], body)
	r, _, err := w32.ShellNotifyIcon.Call(w32.NimModify, uintptr(unsafe.Pointer(d)))
	if r == 0 {
		log.Printf("win: tray notify failed: %v", err)
	}
}

// OnActivate registers the icon-click handler.
func (t *Tray) OnActivate(fn func()) { t.onActivate = fn }

// OnNotificationClick registers the notification-click handler.
func (t *Tray) OnNotificationClick(fn func()) { t.onNotifyClick = fn }

// onEvent handles the tray callback message (version-4 encoding: the
// event lives in LOWORD(lParam)).
func (t *Tray) onEvent(event uint16) {
	const wmContextMenu = 0x007B
	switch event {
	case w32.WmLButtonUp:
		if t.onActivate != nil {
			t.onActivate()
		}
	case w32.NinBalloonUserClick:
		switch {
		case t.onNotifyClick != nil:
			t.onNotifyClick()
		case t.onActivate != nil:
			t.onActivate()
		}
	case w32.WmRButtonUp, wmContextMenu:
		t.showMenu()
	}
}

func (t *Tray) showMenu() {
	if len(t.menu) == 0 {
		return
	}
	menu, _, _ := w32.CreatePopupMenu.Call()
	defer w32.DestroyMenu.Call(menu)
	for i, item := range t.menu {
		if item.Label == "" {
			w32.AppendMenu.Call(menu, w32.MfSeparator, 0, 0)
			continue
		}
		w32.AppendMenu.Call(menu, w32.MfString, uintptr(i+1),
			uintptr(unsafe.Pointer(utf16Ptr(item.Label))))
	}
	// Required quirk: without foregrounding the window first, the menu
	// won't dismiss when the user clicks away.
	w32.SetForegroundWindow.Call(t.win.hwnd)
	var pt w32.Point
	w32.GetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	cmd, _, _ := w32.TrackPopupMenu.Call(menu,
		w32.TpmReturnCmd|w32.TpmRightButton,
		uintptr(pt.X), uintptr(pt.Y), 0, t.win.hwnd, 0)
	if cmd == 0 {
		return
	}
	item := t.menu[cmd-1]
	if item.OnClick != nil {
		item.OnClick()
	}
}

// remove deletes the icon; call on shutdown.
func (t *Tray) remove() {
	w32.ShellNotifyIcon.Call(w32.NimDelete, uintptr(unsafe.Pointer(t.data())))
}

func copyUTF16(dst []uint16, s string) {
	u, err := utf16FromString(s)
	if err != nil {
		return
	}
	n := copy(dst, u)
	if n == len(dst) {
		dst[len(dst)-1] = 0
	}
}
