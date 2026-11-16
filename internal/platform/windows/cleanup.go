//go:build windows

package windows

import (
	"os"
	"path/filepath"
	"time"

	"github.com/cn8001/reforgepulse/internal/platform"
)

// CleanupLocations are the places Disk Cleanup itself treats as disposable.
// Crash dumps and error reports stay: Fix Issues reads them as evidence.
func (System) CleanupLocations() []platform.Location {
	win := os.Getenv("SystemRoot")
	local := os.Getenv("LOCALAPPDATA")
	return []platform.Location{
		{ID: "user-temp", Name: "Temporary files", Note: "Files apps left in your temp folder. Only files untouched for a day.",
			Paths: []string{os.TempDir()}, MinAge: 24 * time.Hour},
		{ID: "windows-temp", Name: "Windows temporary files", Note: "Leftovers from installers and Windows. Only files untouched for a day.",
			Paths: []string{filepath.Join(win, "Temp")}, MinAge: 24 * time.Hour},
		{ID: "update-cache", Name: "Windows Update downloads", Note: "Installed update packages. Windows downloads them again if it still needs one.",
			Paths: []string{filepath.Join(win, "SoftwareDistribution", "Download")}, MinAge: 72 * time.Hour},
		{ID: "delivery-cache", Name: "Delivery Optimization cache", Note: "Update pieces shared with other PCs. Rebuilt on demand.",
			Paths: []string{filepath.Join(win, `ServiceProfiles\NetworkService\AppData\Local\Microsoft\Windows\DeliveryOptimization\Cache`)}},
		{ID: "shader-cache", Name: "GPU shader caches", Note: "Compiled shaders. Games rebuild them, so the first launch after may stutter briefly.",
			Paths: []string{
				filepath.Join(local, "D3DSCache"),
				filepath.Join(local, "NVIDIA", "DXCache"),
				filepath.Join(local, "NVIDIA", "GLCache"),
				filepath.Join(local, "AMD", "DxCache"),
			}},
	}
}
