//go:build windows

package windows

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"syscall"

	"github.com/cn8001/reforgepulse/internal/platform"
)

var _ platform.Features = System{}

// featureName is all a feature name may contain, so it can sit inside a
// fixed PowerShell command without quoting tricks.
var featureName = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

func (System) OptionalFeatures(ctx context.Context) ([]platform.Feature, error) {
	out, err := powershell(ctx, `Get-WindowsOptionalFeature -Online | Select-Object FeatureName, @{n='Enabled';e={$_.State -eq 'Enabled'}} | ConvertTo-Json -Compress`)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		FeatureName string
		Enabled     bool
	}
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		return nil, fmt.Errorf("read feature list: %w", err)
	}
	fs := make([]platform.Feature, len(rows))
	for i, r := range rows {
		fs[i] = platform.Feature{Name: r.FeatureName, Enabled: r.Enabled}
	}
	return fs, nil
}

func (System) SetFeature(ctx context.Context, name string, enabled bool) (bool, error) {
	if !featureName.MatchString(name) {
		return false, fmt.Errorf("bad feature name %q", name)
	}
	cmd := fmt.Sprintf("(Disable-WindowsOptionalFeature -Online -FeatureName '%s' -NoRestart).RestartNeeded", name)
	if enabled {
		cmd = fmt.Sprintf("(Enable-WindowsOptionalFeature -Online -FeatureName '%s' -All -NoRestart).RestartNeeded", name)
	}
	out, err := powershell(ctx, cmd)
	if err != nil {
		return false, err
	}
	return strings.Contains(strings.ToLower(out), "true"), nil
}

// powershell runs one fixed command without a profile or a window.
func powershell(ctx context.Context, command string) (string, error) {
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command",
		"$ErrorActionPreference='Stop'; [Console]::OutputEncoding=[Text.Encoding]::UTF8; "+command)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			return "", fmt.Errorf("powershell: %s", strings.TrimSpace(string(ee.Stderr)))
		}
		return "", fmt.Errorf("powershell: %w", err)
	}
	return string(out), nil
}
