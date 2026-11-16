//go:build windows

package windows

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"github.com/cn8001/reforgepulse/internal/platform"
)

var _ platform.Startup = System{}

const approvedBase = `Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\`

// startupSource is one place programs register to start, and the key where
// Task Manager stores whether that entry is enabled.
type startupSource struct {
	id, label string
	root      registry.Key
	runKey    string // a Run key, or empty for a folder
	folder    string
	approved  string
}

func startupSources() []startupSource {
	return []startupSource{
		{"user-run", "Registry (you)", registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, "", approvedBase + "Run"},
		{"machine-run", "Registry (all users)", registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Run`, "", approvedBase + "Run"},
		{"machine-run32", "Registry, 32-bit (all users)", registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Run`, "", approvedBase + "Run32"},
		{"user-folder", "Startup folder (you)", registry.CURRENT_USER, "", filepath.Join(os.Getenv("APPDATA"), `Microsoft\Windows\Start Menu\Programs\Startup`), approvedBase + "StartupFolder"},
		{"common-folder", "Startup folder (all users)", registry.LOCAL_MACHINE, "", filepath.Join(os.Getenv("ProgramData"), `Microsoft\Windows\Start Menu\Programs\StartUp`), approvedBase + "StartupFolder"},
	}
}

func (System) StartupEntries(ctx context.Context) ([]platform.StartupEntry, error) {
	var out []platform.StartupEntry
	for _, s := range startupSources() {
		names, commands := s.list()
		for i, name := range names {
			out = append(out, platform.StartupEntry{
				ID: s.id + "|" + name, Name: strings.TrimSuffix(name, ".lnk"), Command: commands[i],
				Source: s.label, Machine: s.root == registry.LOCAL_MACHINE, Enabled: s.enabled(name),
			})
		}
	}
	return out, nil
}

func (s startupSource) list() (names, commands []string) {
	if s.runKey == "" {
		entries, _ := os.ReadDir(s.folder)
		for _, e := range entries {
			if !e.IsDir() && !strings.EqualFold(e.Name(), "desktop.ini") {
				names = append(names, e.Name())
				commands = append(commands, filepath.Join(s.folder, e.Name()))
			}
		}
		return names, commands
	}
	k, err := registry.OpenKey(s.root, s.runKey, registry.QUERY_VALUE)
	if err != nil {
		return nil, nil
	}
	defer k.Close()
	vals, _ := k.ReadValueNames(-1)
	for _, v := range vals {
		if cmd, _, err := k.GetStringValue(v); err == nil {
			names = append(names, v)
			commands = append(commands, cmd)
		}
	}
	return names, commands
}

// enabled reads Task Manager's flag: an odd first byte means disabled, and
// no value at all means it was never touched, so it starts.
func (s startupSource) enabled(name string) bool {
	k, err := registry.OpenKey(s.root, s.approved, registry.QUERY_VALUE)
	if err != nil {
		return true
	}
	defer k.Close()
	b, _, err := k.GetBinaryValue(name)
	return err != nil || len(b) == 0 || b[0]&1 == 0
}

func (System) SetStartupEnabled(ctx context.Context, id string, enabled bool) error {
	srcID, name, ok := strings.Cut(id, "|")
	if !ok {
		return fmt.Errorf("bad startup id %q", id)
	}
	for _, s := range startupSources() {
		if s.id != srcID {
			continue
		}
		k, _, err := registry.CreateKey(s.root, s.approved, registry.SET_VALUE)
		if err != nil {
			return fmt.Errorf("open startup approvals: %w", err)
		}
		defer k.Close()
		// 12 bytes: the flag, three zero bytes, then when it was changed.
		v := make([]byte, 12)
		v[0] = 0x03
		if enabled {
			v[0] = 0x02
		}
		ft := windows.NsecToFiletime(time.Now().UnixNano())
		binary.LittleEndian.PutUint32(v[4:], ft.LowDateTime)
		binary.LittleEndian.PutUint32(v[8:], ft.HighDateTime)
		if err := k.SetBinaryValue(name, v); err != nil {
			return fmt.Errorf("set startup flag: %w", err)
		}
		return nil
	}
	return errors.New("unknown startup source " + srcID)
}
