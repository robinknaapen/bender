//go:build windows

package webview2

import (
	"unsafe"

	"github.com/pietjan/bender/internal/platform/win/w32"
)

// Controller wraps ICoreWebView2Controller: the windowed host of one
// webview — bounds, visibility, focus, lifetime.
type Controller struct {
	vtbl *controllerVtbl
}

type controllerVtbl struct {
	iUnknownVtbl
	GetIsVisible                      uintptr
	PutIsVisible                      uintptr
	GetBounds                         uintptr
	PutBounds                         uintptr
	GetZoomFactor                     uintptr
	PutZoomFactor                     uintptr
	AddZoomFactorChanged              uintptr
	RemoveZoomFactorChanged           uintptr
	SetBoundsAndZoomFactor            uintptr
	MoveFocus                         uintptr
	AddMoveFocusRequested             uintptr
	RemoveMoveFocusRequested          uintptr
	AddGotFocus                       uintptr
	RemoveGotFocus                    uintptr
	AddLostFocus                      uintptr
	RemoveLostFocus                   uintptr
	AddAcceleratorKeyPressed          uintptr
	RemoveAcceleratorKeyPressed       uintptr
	GetParentWindow                   uintptr
	PutParentWindow                   uintptr
	NotifyParentWindowPositionChanged uintptr
	Close                             uintptr
	GetCoreWebView2                   uintptr
}

// moveFocusReasonProgrammatic is COREWEBVIEW2_MOVE_FOCUS_REASON_PROGRAMMATIC.
const moveFocusReasonProgrammatic = 0

// SetBounds positions the webview within its parent's client area.
// (Win64 passes >8-byte structs by reference, hence the pointer.)
func (c *Controller) SetBounds(r w32.Rect) error {
	hr := call(c.vtbl.PutBounds, unsafe.Pointer(c), uintptr(unsafe.Pointer(&r)))
	return checkHR("put_Bounds", hr)
}

// SetVisible shows or hides the webview.
func (c *Controller) SetVisible(visible bool) error {
	v := uintptr(0)
	if visible {
		v = 1
	}
	return checkHR("put_IsVisible", call(c.vtbl.PutIsVisible, unsafe.Pointer(c), v))
}

// Bounds returns the controller's current bounds in the parent window.
func (c *Controller) Bounds() (w32.Rect, error) {
	var r w32.Rect
	hr := call(c.vtbl.GetBounds, unsafe.Pointer(c), uintptr(unsafe.Pointer(&r)))
	return r, checkHR("get_Bounds", hr)
}

// IsVisible reports the controller's visibility flag.
func (c *Controller) IsVisible() bool {
	var v int32
	call(c.vtbl.GetIsVisible, unsafe.Pointer(c), uintptr(unsafe.Pointer(&v)))
	return v != 0
}

// MoveFocus gives the webview keyboard focus.
func (c *Controller) MoveFocus() error {
	return checkHR("MoveFocus", call(c.vtbl.MoveFocus, unsafe.Pointer(c), moveFocusReasonProgrammatic))
}

// NotifyParentWindowPositionChanged must be called on WM_MOVE so popups
// and accessibility track the window.
func (c *Controller) NotifyParentWindowPositionChanged() {
	call(c.vtbl.NotifyParentWindowPositionChanged, unsafe.Pointer(c))
}

// Close shuts the webview down and flushes its profile; the controller is
// unusable afterwards.
func (c *Controller) Close() {
	call(c.vtbl.Close, unsafe.Pointer(c))
	release(unsafe.Pointer(c))
}

// CoreWebView2 returns the content API of this controller.
func (c *Controller) CoreWebView2() (*CoreWebView2, error) {
	var out unsafe.Pointer
	hr := call(c.vtbl.GetCoreWebView2, unsafe.Pointer(c), uintptr(unsafe.Pointer(&out)))
	if err := checkHR("get_CoreWebView2", hr); err != nil {
		return nil, err
	}
	return (*CoreWebView2)(out), nil
}
