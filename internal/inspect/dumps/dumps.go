// Package dumps reads kernel crash dumps: the stop code and parameters from
// the header without any tools, and the debugger's verdict when asked.
package dumps

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/platform"
)

// Deps are where dumps live and how to analyse them.
type Deps struct {
	Dir      string // %SystemRoot%\Minidump
	Symbols  string // the local symbol cache
	Debugger platform.Debugger
	Names    map[string]string // stop code to name
}

type Inspector struct{ d Deps }

func New(d Deps) *Inspector { return &Inspector{d: d} }

func (in *Inspector) List(ctx context.Context) ([]domain.Dump, error) {
	entries, err := os.ReadDir(in.d.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []domain.Dump{}, nil
		}
		return nil, fmt.Errorf("read %s: %w", in.d.Dir, err)
	}
	var out []domain.Dump
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".dmp") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		d := domain.Dump{Name: e.Name(), Time: info.ModTime(), Size: info.Size()}
		if f, err := os.Open(filepath.Join(in.d.Dir, e.Name())); err == nil {
			d.Code, d.Params = Header(f)
			f.Close()
		}
		d.CodeName = in.d.Names[d.Code]
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time.After(out[j].Time) })
	return out, nil
}

// Header reads a 64-bit kernel dump header (DUMP_HEADER64): signature
// "PAGEDU64", the stop code at 0x38 and its four parameters from 0x40.
func Header(r io.ReaderAt) (code string, params []string) {
	b := make([]byte, 0x60)
	if _, err := r.ReadAt(b, 0); err != nil || string(b[:8]) != "PAGEDU64" {
		return "", nil
	}
	code = fmt.Sprintf("0x%X", binary.LittleEndian.Uint32(b[0x38:]))
	for i := 0; i < 4; i++ {
		params = append(params, fmt.Sprintf("0x%X", binary.LittleEndian.Uint64(b[0x40+8*i:])))
	}
	return code, params
}

// Analyze runs the debugger on a dump named by the window. The name must be
// a plain file name in the dump folder, so no other file can be opened.
func (in *Inspector) Analyze(ctx context.Context, name string) (domain.DumpAnalysis, error) {
	if name != filepath.Base(name) || !strings.EqualFold(filepath.Ext(name), ".dmp") {
		return domain.DumpAnalysis{}, fmt.Errorf("%q is not a dump name", name)
	}
	path := filepath.Join(in.d.Dir, name)
	if _, err := os.Stat(path); err != nil {
		return domain.DumpAnalysis{}, fmt.Errorf("dump %s: %w", name, err)
	}
	out, err := in.d.Debugger.Analyze(ctx, path, in.d.Symbols)
	if err != nil {
		return domain.DumpAnalysis{}, err
	}
	a := Parse(out)
	a.Name = name
	return a, nil
}

// Parse pulls the summary fields out of !analyze -v output.
func Parse(out string) domain.DumpAnalysis {
	var a domain.DumpAnalysis
	fields := map[string]*string{
		"MODULE_NAME": &a.Module, "IMAGE_NAME": &a.Image, "SYMBOL_NAME": &a.Symbol,
		"PROCESS_NAME": &a.Process, "FAILURE_BUCKET_ID": &a.Bucket,
	}
	sc := bufio.NewScanner(strings.NewReader(out))
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	var lines []string
	for sc.Scan() {
		line := sc.Text()
		lines = append(lines, line)
		if k, v, ok := strings.Cut(line, ":"); ok {
			if p, want := fields[strings.TrimSpace(k)]; want && *p == "" {
				*p = strings.TrimSpace(v)
			}
		}
	}
	// The verdict block sits at the end; the start is symbol loading noise.
	if len(lines) > 80 {
		lines = lines[len(lines)-80:]
	}
	a.Output = strings.Join(lines, "\n")
	return a
}

func (in *Inspector) HasDebugger() bool { return in.d.Debugger.DebuggerPath() != "" }
