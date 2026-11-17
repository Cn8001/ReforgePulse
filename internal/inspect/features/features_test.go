package features

import (
	"context"
	"testing"

	"github.com/cn8001/reforgepulse/internal/platform"
)

type fakeFeatures []platform.Feature

func (f fakeFeatures) OptionalFeatures(context.Context) ([]platform.Feature, error) { return f, nil }
func (f fakeFeatures) SetFeature(context.Context, string, bool) (bool, error)       { return false, nil }

func TestInspector_SuggestsOnlyCatalogued(t *testing.T) {
	// Arrange
	in := New(fakeFeatures{
		{Name: "VirtualMachinePlatform", Enabled: true},
		{Name: "SMB1Protocol", Enabled: true},
		{Name: "SomethingNew", Enabled: true},
		{Name: "TelnetClient", Enabled: false},
	})

	// Act
	got, err := in.List(context.Background())

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].Name != "SMB1Protocol" || !got[0].Suggest {
		t.Fatalf("got %+v", got)
	}
	for _, f := range got {
		if f.Name == "VirtualMachinePlatform" && (f.Suggest || !f.Protected) {
			t.Fatalf("WSL's platform was suggested: %+v", f)
		}
		if f.Name == "SomethingNew" && f.Suggest {
			t.Fatal("an unknown feature was suggested")
		}
	}
}

func TestInspector_ListsFeaturesWaitingForARestart(t *testing.T) {
	// Arrange
	in := New(fakeFeatures{
		{Name: "WorkFolders-Client", Pending: "off"},
		{Name: "TelnetClient"},
	})

	// Act
	got, err := in.List(context.Background())

	// Assert
	if err != nil || len(got) != 1 || got[0].Name != "WorkFolders-Client" || got[0].Pending != "off" || got[0].Enabled {
		t.Fatalf("got %+v, %v", got, err)
	}
}
