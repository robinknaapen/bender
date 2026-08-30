package app

import "github.com/pietjan/spectacle"

// Sidebar widths in device-independent pixels.
const (
	SidebarDIP          = 220
	SidebarCollapsedDIP = 56
)

// Layout places the chrome and the active service inside the client area.
type Layout struct {
	Sidebar spectacle.Rect
	Content spectacle.Rect
}

// ComputeLayout splits a client area of w×h physical pixels at dpi into
// the sidebar strip and the service content area.
func ComputeLayout(w, h, dpi int, collapsed bool) Layout {
	if dpi <= 0 {
		dpi = 96
	}
	dip := SidebarDIP
	if collapsed {
		dip = SidebarCollapsedDIP
	}
	sw := min(dip*dpi/96, w)
	return Layout{
		Sidebar: spectacle.Rect{X: 0, Y: 0, W: sw, H: h},
		Content: spectacle.Rect{X: sw, Y: 0, W: w - sw, H: h},
	}
}
