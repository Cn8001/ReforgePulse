package dumps

import (
	"bytes"
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func header(code uint32, p1 uint64) []byte {
	b := make([]byte, 0x2000)
	copy(b, "PAGEDU64")
	binary.LittleEndian.PutUint32(b[0x38:], code)
	binary.LittleEndian.PutUint64(b[0x40:], p1)
	return b
}

func TestHeader_ReadsStopCode(t *testing.T) {
	// Act
	code, params := Header(bytes.NewReader(header(0x3B, 0xC0000005)))

	// Assert
	if code != "0x3B" || len(params) != 4 || params[0] != "0xC0000005" {
		t.Fatalf("code = %s, params = %v", code, params)
	}
}

func TestHeader_IgnoresOtherFiles(t *testing.T) {
	// Act
	code, _ := Header(bytes.NewReader(make([]byte, 0x100)))

	// Assert
	if code != "" {
		t.Fatalf("code = %q for a non-dump", code)
	}
}

func TestInspector_ListNamesCodes(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "111326-1.dmp"), header(0x124, 0), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o644)
	in := New(Deps{Dir: dir, Names: map[string]string{"0x124": "WHEA_UNCORRECTABLE_ERROR"}})

	// Act
	got, err := in.List(context.Background())

	// Assert
	if err != nil || len(got) != 1 || got[0].CodeName != "WHEA_UNCORRECTABLE_ERROR" {
		t.Fatalf("got %+v, err = %v", got, err)
	}
}

func TestInspector_AnalyzeRefusesPaths(t *testing.T) {
	// Arrange
	in := New(Deps{Dir: t.TempDir()})

	// Act
	_, err := in.Analyze(context.Background(), `..\..\Windows\System32\config\SAM.dmp`)

	// Assert
	if err == nil {
		t.Fatal("a path outside the dump folder was accepted")
	}
}

func TestParse_PullsTheVerdict(t *testing.T) {
	// Arrange
	out := "Loading symbols...\nMODULE_NAME: nvlddmkm\nIMAGE_NAME:  nvlddmkm.sys\nPROCESS_NAME:  System\nSYMBOL_NAME:  nvlddmkm+0x1234\nFAILURE_BUCKET_ID:  0x3B_c0000005_nvlddmkm!unknown_function\n"

	// Act
	a := Parse(out)

	// Assert
	if a.Module != "nvlddmkm" || a.Image != "nvlddmkm.sys" || a.Process != "System" || a.Bucket == "" {
		t.Fatalf("analysis = %+v", a)
	}
}
