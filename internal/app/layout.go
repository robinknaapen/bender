package app

import "github.com/pietjan/bender/internal/platform"

// SidebarDIP is the sidebar width in device-independent pixels.
const SidebarDIP = 220

// Layout places the chrome and the active service inside the client area.
type Layout struct {
	Sidebar platform.Rect
	Content platform.Rect
}

// ComputeLayout splits a client area of w×h physical pixels at dpi into
// the sidebar strip and the service content area.
func ComputeLayout(w, h, dpi int) Layout {
	if dpi <= 0 {
		dpi = 96
	}
	sw := SidebarDIP * dpi / 96
	if sw > w {
		sw = w
	}
	return Layout{
		Sidebar: platform.Rect{X: 0, Y: 0, W: sw, H: h},
		Content: platform.Rect{X: sw, Y: 0, W: w - sw, H: h},
	}
}
