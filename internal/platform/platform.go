// Package platform defines the small, platform-neutral surface that the
// application core is written against. Each operating system provides a
// Backend implementation (Windows/WebView2 today; WebKitGTK and WKWebView
// later) behind build tags; nothing above this package may import
// platform-specific code.
package platform

// Rect is a rectangle in physical pixels, relative to the window's client
// area origin.
type Rect struct {
	X, Y, W, H int
}

// MenuItem is one entry in a tray menu. A zero Label renders a separator.
type MenuItem struct {
	Label   string
	OnClick func()
}

// Backend owns the UI thread, the native event loop, and webview creation.
//
// Every method except Dispatch must be called on the UI thread: either
// before Run (during startup) or from within a callback. Dispatch is the
// only door in from other goroutines.
type Backend interface {
	// NewWindow creates the (hidden) main window.
	NewWindow(title string, bounds Rect) (Window, error)
	// NewWebView creates a webview as a child of w. profile selects an
	// isolated browsing session; webviews sharing a profile share cookies
	// and storage. The empty profile is the default session.
	NewWebView(w Window, profile string) (WebView, error)
	// Run enters the native event loop and blocks until Quit.
	Run() error
	// Dispatch schedules f on the UI thread. Safe from any goroutine.
	Dispatch(f func())
	// Quit ends the event loop and releases native resources.
	Quit()
}

// Window is the native top-level window.
type Window interface {
	SetBounds(Rect)
	Bounds() Rect
	Show()
	Hide()
	IsVisible() bool
	// OnResize reports the client area size in physical pixels and the
	// current DPI whenever either changes.
	OnResize(func(w, h, dpi int))
	// OnCloseRequest is consulted when the user asks to close the window.
	// Returning false vetoes the close (the app may hide to tray instead).
	OnCloseRequest(func() bool)
	Tray() Tray
}

// WebView is one embedded browser view inside a Window.
type WebView interface {
	Navigate(url string)
	// NavigateHTML renders the given document directly, without a server.
	NavigateHTML(html string)
	// InitScript registers js to run in every document before it loads.
	// Must be called before the first navigation to take effect there.
	InitScript(js string)
	// PostJSON delivers a JSON document to the page; the page receives it
	// via its message event. OnMessage receives JSON the page posts back.
	PostJSON(json string)
	OnMessage(func(json string))
	OnTitleChanged(func(title string))
	SetBounds(Rect)
	SetVisible(bool)
	Focus()
	// Close releases the webview and flushes its session to disk.
	Close()
}

// Tray is the system notification-area icon.
type Tray interface {
	SetTooltip(string)
	SetMenu([]MenuItem)
	// Notify shows a desktop notification attributed to the app.
	Notify(title, body string)
	// OnActivate fires when the user clicks the tray icon.
	OnActivate(func())
}
