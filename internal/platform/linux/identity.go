//go:build linux

package linux

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/pietjan/bender/internal/platform/appicon"
)

// registerIdentity installs bender's desktop identity for the current
// user — the Linux analogue of the Windows AUMID registration. Taskbars
// and shells resolve a window's app-id ("bender", via g_set_prgname)
// against a .desktop entry and the icon theme; neither exists for an
// app that was never packaged, so install both. Best-effort.
func registerIdentity() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	iconDir := filepath.Join(home, ".local/share/icons/hicolor/256x256/apps")
	if err := os.MkdirAll(iconDir, 0o755); err == nil {
		if err := os.WriteFile(filepath.Join(iconDir, "bender.png"), appicon.PNG, 0o644); err != nil {
			log.Printf("linux: app icon: %v", err)
		}
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	appsDir := filepath.Join(home, ".local/share/applications")
	if err := os.MkdirAll(appsDir, 0o755); err != nil {
		return
	}
	desktop := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Bender
Comment=Multi-service messaging browser
Exec=%s
Icon=bender
Categories=Network;InstantMessaging;
StartupWMClass=bender
`, exe)
	if err := os.WriteFile(filepath.Join(appsDir, "bender.desktop"), []byte(desktop), 0o644); err != nil {
		log.Printf("linux: desktop entry: %v", err)
	}
}
