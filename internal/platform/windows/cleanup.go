//go:build windows

package windows

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows"

	"github.com/cn8001/reforgepulse/internal/platform"
)

const day = 24 * time.Hour

// CleanupLocations are places whose contents Windows or the owning app
// rebuilds. Minidumps stay: Issues reads them as evidence.
func (s System) CleanupLocations() []platform.Location {
	win, local, roaming, data := os.Getenv("SystemRoot"), os.Getenv("LOCALAPPDATA"), os.Getenv("APPDATA"), os.Getenv("ProgramData")
	j := filepath.Join
	browser := func(base string, dirs ...string) []string {
		var out []string
		for _, d := range dirs {
			out = append(out, j(base, "*", d))
		}
		return out
	}
	return []platform.Location{
		{ID: "user-temp", Name: "Temporary files", Note: "Files apps left in your temp folder. Only files untouched for a day.",
			Paths: []string{os.TempDir()}, MinAge: day},
		{ID: "windows-temp", Name: "Windows temporary files", Note: "Leftovers from installers and Windows. Only files untouched for a day.",
			Paths: []string{j(win, "Temp")}, MinAge: day},
		{ID: "recycle-bin", Name: "Recycle Bin", Note: "Everything in your Recycle Bin, on every drive. Deleted for good.",
			Paths: s.recycleBins()},
		{ID: "browser-cache", Name: "Browser caches", Note: "Chrome, Edge, Brave and Firefox page caches. Logins, history and bookmarks stay. An open browser keeps its files.",
			Paths: append(append(append(
				browser(j(local, `Google\Chrome\User Data`), "Cache", "Code Cache", "GPUCache"),
				browser(j(local, `Microsoft\Edge\User Data`), "Cache", "Code Cache", "GPUCache")...),
				browser(j(local, `BraveSoftware\Brave-Browser\User Data`), "Cache", "Code Cache", "GPUCache")...),
				j(local, `Mozilla\Firefox\Profiles\*\cache2`))},
		{ID: "app-cache", Name: "App caches", Note: "Discord, Slack, Teams, Spotify and VS Code caches. The apps download them again as needed.",
			Paths: []string{
				j(roaming, `discord\Cache`), j(roaming, `discord\Code Cache`), j(roaming, `Slack\Cache`), j(roaming, `Slack\Code Cache`),
				j(roaming, `Microsoft\Teams\Cache`), j(local, `Spotify\Data`), j(roaming, `Code\Cache`), j(roaming, `Code\CachedData`),
			}},
		{ID: "dev-cache", Name: "Developer caches", Note: "npm, Yarn, pip and Go build caches. The next build downloads or rebuilds what it needs.",
			Paths: []string{j(local, `npm-cache\_cacache`), j(local, `Yarn\Cache`), j(local, `pip\Cache`), j(local, "go-build")}},
		{ID: "shader-cache", Name: "GPU shader caches", Note: "Compiled shaders. Games rebuild them, so the first launch after may stutter briefly.",
			Paths: []string{j(local, "D3DSCache"), j(local, `NVIDIA\DXCache`), j(local, `NVIDIA\GLCache`), j(local, `AMD\DxCache`)}},
		{ID: "error-reports", Name: "Old error reports", Note: "Windows Error Reporting files older than a week. Recent ones stay for Issues.",
			Paths: []string{j(data, `Microsoft\Windows\WER\ReportArchive`), j(data, `Microsoft\Windows\WER\ReportQueue`),
				j(local, `Microsoft\Windows\WER\ReportArchive`)}, MinAge: 7 * day},
		{ID: "windows-logs", Name: "Old Windows logs", Note: "Servicing and setup logs older than a month.",
			Paths: []string{j(win, `Logs\CBS`), j(win, `Logs\DISM`)}, MinAge: 30 * day},
		{ID: "update-cache", Name: "Windows Update downloads", Note: "Installed update packages. Windows downloads one again if it still needs it.",
			Paths: []string{j(win, `SoftwareDistribution\Download`)}, MinAge: 3 * day},
		{ID: "delivery-cache", Name: "Delivery Optimization cache", Note: "Update pieces shared with other PCs. Rebuilt on demand.",
			Paths: []string{j(win, `ServiceProfiles\NetworkService\AppData\Local\Microsoft\Windows\DeliveryOptimization\Cache`)}},
	}
}

// recycleBins are this user's Recycle Bin folders, one per fixed drive,
// named after the user's SID.
func (s System) recycleBins() []string {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil
	}
	sid := user.User.Sid.String()
	vols, _ := s.Volumes(context.Background())
	var out []string
	for _, v := range vols {
		out = append(out, filepath.Join(v.Root, "$Recycle.Bin", sid))
	}
	return out
}
