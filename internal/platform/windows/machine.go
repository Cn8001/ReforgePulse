//go:build windows

package windows

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"

	"github.com/cn8001/reforgepulse/internal/platform"
)

var _ platform.Machine = System{}

// Microcode reads the running revision. The value is REG_BINARY with the
// revision in the high dword on most builds, as crash-diag.ps1 found.
func (System) Microcode(ctx context.Context) (uint32, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `HARDWARE\DESCRIPTION\System\CentralProcessor\0`, registry.QUERY_VALUE)
	if err != nil {
		return 0, fmt.Errorf("open cpu key: %w", err)
	}
	defer k.Close()
	b, _, err := k.GetBinaryValue("Update Revision")
	if err != nil {
		return 0, fmt.Errorf("read microcode: %w", err)
	}
	switch {
	case len(b) >= 8:
		return binary.LittleEndian.Uint32(b[4:8]), nil
	case len(b) >= 4:
		return binary.LittleEndian.Uint32(b[:4]), nil
	}
	return 0, fmt.Errorf("microcode value has %d bytes", len(b))
}

// Services reads every driver and service registration from the registry,
// which covers kernel drivers the service manager API leaves out.
func (System) Services(ctx context.Context) ([]platform.Service, error) {
	root, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Services`, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return nil, fmt.Errorf("open services: %w", err)
	}
	defer root.Close()
	names, err := root.ReadSubKeyNames(-1)
	if err != nil {
		return nil, fmt.Errorf("list services: %w", err)
	}
	var out []platform.Service
	for _, n := range names {
		k, err := registry.OpenKey(root, n, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		s := platform.Service{Name: n}
		s.DisplayName, _, _ = k.GetStringValue("DisplayName")
		s.ImagePath, _, _ = k.GetStringValue("ImagePath")
		start, _, serr := k.GetIntegerValue("Start")
		typ, _, terr := k.GetIntegerValue("Type")
		k.Close()
		if serr != nil || terr != nil {
			continue // not a service, just a settings key
		}
		s.Start, s.Driver = uint32(start), typ&0x3 != 0 // kernel or file system driver
		out = append(out, s)
	}
	return out, nil
}

// Programs lists uninstall entries for the machine, 32-bit programs and the
// current user.
func (System) Programs(ctx context.Context) ([]platform.Program, error) {
	const path = `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`
	roots := []struct {
		key  registry.Key
		path string
	}{
		{registry.LOCAL_MACHINE, path},
		{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`},
		{registry.CURRENT_USER, path},
	}
	seen := map[string]bool{}
	var out []platform.Program
	for _, r := range roots {
		root, err := registry.OpenKey(r.key, r.path, registry.ENUMERATE_SUB_KEYS)
		if err != nil {
			continue
		}
		names, _ := root.ReadSubKeyNames(-1)
		for _, n := range names {
			k, err := registry.OpenKey(root, n, registry.QUERY_VALUE)
			if err != nil {
				continue
			}
			var p platform.Program
			p.Name, _, _ = k.GetStringValue("DisplayName")
			p.Version, _, _ = k.GetStringValue("DisplayVersion")
			p.Publisher, _, _ = k.GetStringValue("Publisher")
			p.InstallDate, _, _ = k.GetStringValue("InstallDate")
			p.InstallLocation, _, _ = k.GetStringValue("InstallLocation")
			p.UninstallString, _, _ = k.GetStringValue("UninstallString")
			sys, _, _ := k.GetIntegerValue("SystemComponent")
			k.Close()
			if p.Name == "" || sys == 1 || seen[p.Name+p.Version] {
				continue // updates and components without a name are not programs
			}
			seen[p.Name+p.Version] = true
			out = append(out, p)
		}
		root.Close()
	}
	return out, nil
}

func (System) DumpDir() string {
	return filepath.Join(os.Getenv("SystemRoot"), "Minidump")
}
