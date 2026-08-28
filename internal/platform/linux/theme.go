//go:build linux

package linux

import (
	"os"
	"strings"
	"sync"

	"github.com/pietjan/bender/internal/platform/linux/native"
)

// Dark reports whether the app should present dark: BENDER_THEME env
// override, then the GNOME color-scheme setting, defaulting to dark
// (the app's design leans dark, and most desktops without the schema
// give no signal either way).
var Dark = sync.OnceValue(func() bool {
	switch os.Getenv("BENDER_THEME") {
	case "dark":
		return true
	case "light":
		return false
	}
	scheme := native.SettingsString("org.gnome.desktop.interface", "color-scheme")
	if scheme != "" {
		return strings.Contains(scheme, "dark")
	}
	return true
})

// applyTheme makes GTK follow the choice (so pages see the matching
// prefers-color-scheme) and paints exposed window area zinc-900 — the
// Linux analogue of the Win32 class brush.
func applyTheme() {
	if !Dark() {
		return
	}
	if settings := native.GtkSettingsGetDefault(); settings != 0 {
		native.SetBoolProperty(settings, "gtk-application-prefer-dark-theme", true)
	}
	provider := native.GtkCssProviderNew()
	native.GtkCssProviderLoadFromString(provider, "window { background-color: #18181b; }")
	if display := native.GdkDisplayGetDefault(); display != 0 {
		native.GtkStyleContextAddProviderForDisplay(display, provider, native.StyleProviderPriorityApplication)
	}
}
