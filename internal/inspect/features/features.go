// Package features describes Windows optional features and picks the few
// that most people are better off without. Anything not in the catalog is
// listed but never suggested.
package features

import (
	"context"
	"fmt"
	"sort"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/platform"
)

type entry struct {
	title, note        string
	suggest, protected bool
}

var catalog = map[string]entry{
	"SMB1Protocol":                         {"SMB 1.0 file sharing", "The old file-sharing protocol WannaCry spread through. Only very old NAS boxes and printers need it.", true, false},
	"SMB1Protocol-Client":                  {"SMB 1.0 client", "Part of SMB 1.0; see above.", true, false},
	"SMB1Protocol-Server":                  {"SMB 1.0 server", "Part of SMB 1.0; see above.", true, false},
	"MicrosoftWindowsPowerShellV2Root":     {"PowerShell 2.0", "Deprecated, and used by attackers to slip past PowerShell's security logging.", true, false},
	"MicrosoftWindowsPowerShellV2":         {"PowerShell 2.0 engine", "Part of PowerShell 2.0; see above.", true, false},
	"Printing-XPSServices-Features":        {"XPS printing", "Prints to the XPS format almost nobody uses any more.", true, false},
	"WorkFolders-Client":                   {"Work Folders", "Syncs files with a company Work Folders server.", true, false},
	"Internet-Explorer-Optional-amd64":     {"Internet Explorer 11", "Edge's IE mode does not need it.", true, false},
	"TelnetClient":                         {"Telnet client", "Sends everything, passwords included, in plain text.", true, false},
	"TFTP":                                 {"TFTP client", "An unauthenticated file transfer tool for network gear.", true, false},
	"MediaPlayback":                        {"Media features", "Windows Media Player legacy and codecs some apps use.", false, false},
	"Printing-PrintToPDFServices-Features": {"Microsoft Print to PDF", "Saves anything printable as a PDF.", false, false},
	"Microsoft-Windows-Subsystem-Linux":    {"Windows Subsystem for Linux", "Runs Linux; WSL needs it.", false, true},
	"VirtualMachinePlatform":               {"Virtual Machine Platform", "WSL 2, Docker Desktop and Android apps need it.", false, true},
	"Microsoft-Hyper-V-All":                {"Hyper-V", "Virtual machines; Docker and WSL may rely on it.", false, true},
	"HypervisorPlatform":                   {"Windows Hypervisor Platform", "Lets VMware, VirtualBox and emulators run beside Hyper-V.", false, true},
	"Containers":                           {"Containers", "Windows containers for Docker.", false, true},
	"Containers-DisposableClientVM":        {"Windows Sandbox", "A throwaway desktop for testing programs.", false, true},
	"NetFx3":                               {".NET Framework 3.5", "Older apps and games need it.", false, true},
}

type Inspector struct{ src platform.Features }

func New(src platform.Features) *Inspector { return &Inspector{src: src} }

// List returns enabled features, suggestions first.
func (in *Inspector) List(ctx context.Context) ([]domain.WindowsFeature, error) {
	fs, err := in.src.OptionalFeatures(ctx)
	if err != nil {
		return nil, fmt.Errorf("list features: %w", err)
	}
	var out []domain.WindowsFeature
	for _, f := range fs {
		if !f.Enabled {
			continue // only what is on can be turned off
		}
		e, known := catalog[f.Name]
		w := domain.WindowsFeature{Name: f.Name, Title: f.Name, Enabled: true}
		if known {
			w.Title, w.Note, w.Suggest, w.Protected = e.title, e.note, e.suggest, e.protected
		}
		out = append(out, w)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Suggest != out[j].Suggest {
			return out[i].Suggest
		}
		return out[i].Title < out[j].Title
	})
	return out, nil
}

// Known reports whether Windows lists a feature by this name, so a name
// from the window is checked against the real list before it is used.
func (in *Inspector) Known(ctx context.Context, name string) (bool, error) {
	fs, err := in.src.OptionalFeatures(ctx)
	if err != nil {
		return false, fmt.Errorf("list features: %w", err)
	}
	for _, f := range fs {
		if f.Name == name {
			return true, nil
		}
	}
	return false, nil
}
