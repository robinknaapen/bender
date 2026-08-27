package app

import "testing"

func TestComputeLayout(t *testing.T) {
	l := ComputeLayout(1000, 600, 96, false)
	if l.Sidebar.W != SidebarDIP || l.Sidebar.H != 600 {
		t.Fatalf("sidebar %+v", l.Sidebar)
	}
	if l.Content.X != SidebarDIP || l.Content.W != 1000-SidebarDIP || l.Content.H != 600 {
		t.Fatalf("content %+v", l.Content)
	}
}

func TestComputeLayoutScalesWithDPI(t *testing.T) {
	l := ComputeLayout(2000, 1200, 192, false) // 200% scaling
	if l.Sidebar.W != SidebarDIP*2 {
		t.Fatalf("sidebar at 192dpi: %+v", l.Sidebar)
	}
}

func TestComputeLayoutTinyWindow(t *testing.T) {
	l := ComputeLayout(100, 100, 96, false)
	if l.Sidebar.W != 100 || l.Content.W != 0 {
		t.Fatalf("sidebar must clamp: %+v %+v", l.Sidebar, l.Content)
	}
	// Zero dpi (window not yet measured) must not divide by zero.
	if l := ComputeLayout(500, 500, 0, false); l.Sidebar.W != SidebarDIP {
		t.Fatalf("dpi fallback: %+v", l.Sidebar)
	}
}

func TestComputeLayoutCollapsed(t *testing.T) {
	l := ComputeLayout(1000, 600, 96, true)
	if l.Sidebar.W != SidebarCollapsedDIP || l.Content.X != SidebarCollapsedDIP {
		t.Fatalf("collapsed layout: %+v", l)
	}
}
