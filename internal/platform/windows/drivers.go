//go:build windows

package windows

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/cn8001/reforgepulse/internal/platform"
)

var _ platform.Drivers = System{}

// Code 45 is a device that is not plugged in right now, which is not a
// problem worth showing.
const problemScript = `ConvertTo-Json -Compress -InputObject @(Get-CimInstance Win32_PnPEntity |
  Where-Object { $_.ConfigManagerErrorCode -ne 0 -and $_.ConfigManagerErrorCode -ne 45 } |
  ForEach-Object { [pscustomobject]@{ name=[string]$_.Name; class=[string]$_.PNPClass; hardwareId=[string]@($_.HardwareID)[0]
    deviceId=[string]$_.DeviceID; errorCode=[int]$_.ConfigManagerErrorCode } })`

// searchScript asks Windows Update Agent, the engine behind Settings >
// Windows Update, for drivers it offers and this PC does not have yet.
const searchScript = `$s = New-Object -ComObject Microsoft.Update.Session; $s.ClientApplicationID = 'ReforgePulse'
$r = $s.CreateUpdateSearcher().Search("IsInstalled=0 and Type='Driver' and IsHidden=0")
ConvertTo-Json -Compress -InputObject @($r.Updates | ForEach-Object { [pscustomobject]@{
  id=$_.Identity.UpdateID; title=$_.Title; manufacturer=[string]$_.DriverManufacturer; model=[string]$_.DriverModel
  class=[string]$_.DriverClass; date=$(if ($_.DriverVerDate) { $_.DriverVerDate.ToString('yyyy-MM-dd') } else { '' })
  sizeMb=[math]::Round($_.MaxDownloadSize / 1MB, 1); hardwareId=[string]$_.DriverHardwareID } })`

func (System) ProblemDevices(ctx context.Context) ([]platform.ProblemDevice, error) {
	var out []platform.ProblemDevice
	return out, runJSON(ctx, problemScript, &out)
}

func (System) DriverUpdates(ctx context.Context) ([]platform.DriverUpdate, error) {
	var out []platform.DriverUpdate
	return out, runJSON(ctx, searchScript, &out)
}

// OpenDriverPlace opens one of two fixed places; nothing from the window
// reaches the command line.
func (System) OpenDriverPlace(place string) error {
	var cmd *exec.Cmd
	switch place {
	case "windows-update":
		cmd = exec.Command("explorer.exe", "ms-settings:windowsupdate-optionalupdates")
	case "device-manager":
		cmd = exec.Command("mmc.exe", "devmgmt.msc")
	default:
		return fmt.Errorf("unknown place %q", place)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open %s: %w", place, err)
	}
	return nil
}

func runJSON(ctx context.Context, script string, v any) error {
	out, err := powershell(ctx, script)
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(out), v); err != nil {
		return fmt.Errorf("read powershell output: %w", err)
	}
	return nil
}
