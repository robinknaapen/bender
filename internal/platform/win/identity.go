//go:build windows

package win

import (
	_ "embed"
	"log"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows/registry"

	"github.com/pietjan/bender/internal/platform/win/w32"
)

// appUserModelID is bender's explicit app identity. The shell shows it
// (via the registration below) as the notification header.
const appUserModelID = "Bender"

// Same art as the winres icon resources; a copy on disk is needed
// because the AppUserModelId IconUri registry value wants a file path.
//
//go:embed appicon.png
var appIconPNG []byte

// registerIdentity sets the process AUMID and registers its display
// name and icon, which is what puts a name and icon on toasts from an
// app that was never installed. dataDir hosts the icon file. Best-effort.
func registerIdentity(dataDir string) {
	w32.SetAppUserModelID.Call(uintptr(unsafe.Pointer(utf16Ptr(appUserModelID))))

	iconPath := filepath.Join(dataDir, "appicon.png")
	if err := os.WriteFile(iconPath, appIconPNG, 0o644); err != nil {
		log.Printf("win: app icon: %v", err)
		return
	}
	k, _, err := registry.CreateKey(registry.CURRENT_USER,
		`Software\Classes\AppUserModelId\`+appUserModelID, registry.SET_VALUE)
	if err != nil {
		log.Printf("win: app identity: %v", err)
		return
	}
	defer k.Close()
	k.SetStringValue("DisplayName", "Bender")
	k.SetStringValue("IconUri", iconPath)
}
