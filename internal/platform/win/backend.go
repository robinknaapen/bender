//go:build windows

package win

import (
	"fmt"
	"sync"
	"syscall"
	"unsafe"

	"github.com/wailsapp/go-webview2/webviewloader"

	"github.com/pietjan/bender/internal/platform"
	"github.com/pietjan/bender/internal/platform/win/w32"
	"github.com/pietjan/bender/internal/platform/win/webview2"
)

// Backend is the Windows implementation of platform.Backend: one UI
// thread, a Win32 message loop, and WebView2 for the webviews.
type Backend struct {
	userDataFolder string
	debug          bool

	main *Window

	// env is the shared WebView2 environment (profile-capable runtimes).
	// fallbackEnvs holds one environment per profile for old runtimes
	// without ICoreWebView2Environment10.
	env          *webview2.Environment
	fallbackEnvs map[string]*webview2.Environment

	mu    sync.Mutex
	queue []func()
}

// New prepares the backend. Call from the OS-locked main goroutine.
// userDataFolder hosts the browser profiles; debug enables DevTools.
func New(userDataFolder string, debug bool) (*Backend, error) {
	// Per-monitor-v2 DPI awareness must be set before any window exists.
	w32.SetProcessDpiAwarenessCtx.Call(w32.DpiAwarenessContextPerMonitorAwareV2)
	if r, _, _ := w32.CoInitializeEx.Call(0, w32.CoinitApartmentThreaded); int32(r) < 0 {
		return nil, fmt.Errorf("win: CoInitializeEx failed: 0x%08x", uint32(r))
	}
	if err := ensureRuntime(); err != nil {
		return nil, err
	}
	return &Backend{
		userDataFolder: userDataFolder,
		debug:          debug,
		fallbackEnvs:   map[string]*webview2.Environment{},
	}, nil
}

// ensureRuntime verifies the WebView2 evergreen runtime is installed; if
// not, it tells the user and points them at the bootstrapper.
func ensureRuntime() error {
	if _, err := webviewloader.GetAvailableCoreWebView2BrowserVersionString(""); err == nil {
		return nil
	}
	const bootstrapper = "https://go.microsoft.com/fwlink/p/?LinkId=2124703"
	msg := "bender needs the Microsoft WebView2 Runtime.\n\nA download page will open; run the installer and start bender again."
	w32.MessageBox.Call(0,
		uintptr(unsafe.Pointer(utf16Ptr(msg))),
		uintptr(unsafe.Pointer(utf16Ptr("bender"))),
		w32.MbIconError|w32.MbOK)
	w32.ShellExecute.Call(0,
		uintptr(unsafe.Pointer(utf16Ptr("open"))),
		uintptr(unsafe.Pointer(utf16Ptr(bootstrapper))), 0, 0, w32.SwShowNormal)
	return fmt.Errorf("win: WebView2 runtime not installed")
}

// NewWindow creates the main window (hidden until Show).
func (b *Backend) NewWindow(title string, bounds platform.Rect) (platform.Window, error) {
	w, err := newWindow(b, title, bounds)
	if err != nil {
		return nil, err
	}
	if b.main == nil {
		b.main = w
	}
	return w, nil
}

// NewWebView creates a webview on w in the given profile.
func (b *Backend) NewWebView(pw platform.Window, profile string) (platform.WebView, error) {
	w, ok := pw.(*Window)
	if !ok {
		return nil, fmt.Errorf("win: foreign window %T", pw)
	}
	ctrl, err := b.controllerFor(w.hwnd, profile)
	if err != nil {
		return nil, err
	}
	return newWebView(w, ctrl, b.debug)
}

// controllerFor isolates the profile strategy: one shared environment
// with named profiles on current runtimes, one environment (and user
// data folder) per profile on runtimes too old for Environment10.
func (b *Backend) controllerFor(hwnd uintptr, profile string) (*webview2.Controller, error) {
	if b.env == nil {
		env, err := webview2.NewEnvironment(b.userDataFolder)
		if err != nil {
			return nil, err
		}
		b.env = env
	}
	if profile == "" {
		return b.env.CreateController(hwnd)
	}
	if b.env.SupportsProfiles() {
		return b.env.CreateControllerWithProfile(hwnd, profile)
	}
	env, ok := b.fallbackEnvs[profile]
	if !ok {
		var err error
		env, err = webview2.NewEnvironment(b.userDataFolder + `\profile-` + profile)
		if err != nil {
			return nil, err
		}
		b.fallbackEnvs[profile] = env
	}
	return env.CreateController(hwnd)
}

// Run pumps the message loop until Quit.
func (b *Backend) Run() error {
	var msg w32.Msg
	for {
		r, _, _ := w32.GetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		switch int32(r) {
		case -1:
			return fmt.Errorf("win: GetMessage failed")
		case 0:
			return nil // WM_QUIT
		}
		w32.TranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		w32.DispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

// Dispatch schedules f on the UI thread. Safe from any goroutine once
// the main window exists.
func (b *Backend) Dispatch(f func()) {
	b.mu.Lock()
	b.queue = append(b.queue, f)
	b.mu.Unlock()
	if b.main != nil {
		w32.PostMessage.Call(b.main.hwnd, wmAppDispatch, 0, 0)
	}
}

func (b *Backend) drainDispatch() {
	for {
		b.mu.Lock()
		if len(b.queue) == 0 {
			b.mu.Unlock()
			return
		}
		f := b.queue[0]
		b.queue = b.queue[1:]
		b.mu.Unlock()
		f()
	}
}

// Quit removes the tray icon and ends the message loop.
func (b *Backend) Quit() {
	if b.main != nil && b.main.tray != nil {
		b.main.tray.remove()
	}
	w32.PostQuitMessage.Call(0)
}

func utf16FromString(s string) ([]uint16, error) {
	return syscall.UTF16FromString(s)
}
