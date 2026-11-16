//go:build windows

package windows

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cn8001/reforgepulse/internal/platform"
)

var _ platform.Hardware = System{}

// hardwareScript gathers the facts in one PowerShell run; CIM and the
// Storage module have no small Win32 equivalents. Reliability counters need
// administrator rights and stay empty without them.
const hardwareScript = `
$m = @(Get-CimInstance Win32_PhysicalMemory | ForEach-Object { [pscustomobject]@{
  Slot=$_.BankLabel+' '+$_.DeviceLocator; Maker=$_.Manufacturer; Part=($_.PartNumber -as [string]).Trim()
  CapacityBytes=[uint64]$_.Capacity; RatedMHz=[uint32]$_.Speed; ConfiguredMHz=[uint32]$_.ConfiguredClockSpeed } })
$d = @(Get-PhysicalDisk | ForEach-Object {
  $r = $_ | Get-StorageReliabilityCounter -ErrorAction SilentlyContinue
  $n = { param($v) if ($null -eq $v) { -1 } else { [int]$v } }
  [pscustomobject]@{ Name=$_.FriendlyName; Media=[string]$_.MediaType; Health=[string]$_.HealthStatus
    Status=($_.OperationalStatus -join ', '); TempC=(& $n $r.Temperature); TempMaxC=(& $n $r.TemperatureMax)
    Wear=(& $n $r.Wear); ReadErrors=(& $n $r.ReadErrorsUncorrected); WriteErrors=(& $n $r.WriteErrorsUncorrected)
    PowerOnHours=(& $n $r.PowerOnHours) } })
$p = Get-CimInstance Win32_Processor | Select-Object -First 1
$c = [pscustomobject]@{ Name=$p.Name.Trim(); Cores=[int]$p.NumberOfCores; Threads=[int]$p.NumberOfLogicalProcessors
  MaxMHz=[int]$p.MaxClockSpeed; L2KB=[int]$p.L2CacheSize; L3KB=[int]$p.L3CacheSize }
$g = @(Get-CimInstance Win32_VideoController | ForEach-Object { [pscustomobject]@{ Name=$_.Name; DriverVersion=$_.DriverVersion
  DriverDate=$(if ($_.DriverDate) { $_.DriverDate.ToString('yyyy-MM-dd') } else { '' }) } })
$cs = Get-CimInstance Win32_ComputerSystem; $b = Get-CimInstance Win32_BIOS; $bb = Get-CimInstance Win32_BaseBoard
[pscustomobject]@{ CPU=$c; GPUs=$g; Memory=$m; Disks=$d; Hypervisor=[bool]$cs.HypervisorPresent; BIOS=$b.SMBIOSBIOSVersion
  BIOSDate=$b.ReleaseDate.ToString('yyyy-MM-dd'); Board=($bb.Manufacturer+' '+$bb.Product) } | ConvertTo-Json -Depth 4 -Compress`

func (System) HardwareFacts(ctx context.Context) (platform.HardwareFacts, error) {
	out, err := powershell(ctx, hardwareScript)
	if err != nil {
		return platform.HardwareFacts{}, err
	}
	var f platform.HardwareFacts
	if err := json.Unmarshal([]byte(out), &f); err != nil {
		return platform.HardwareFacts{}, fmt.Errorf("read hardware facts: %w", err)
	}
	return f, nil
}
