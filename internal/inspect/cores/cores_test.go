package cores

import (
	"testing"

	"github.com/cn8001/reforgepulse/internal/platform"
)

func TestMap_HybridCoresAndErrors(t *testing.T) {
	// Arrange: two threads of a P-core and one E-core, as on a 14900KS.
	lps := []platform.LogicalCPU{
		{Index: 8, Core: 4, Efficiency: 1, SMT: true, APIC: 32},
		{Index: 9, Core: 4, Efficiency: 1, SMT: true, APIC: 33},
		{Index: 16, Core: 8, Efficiency: 0, APIC: 64},
	}

	// Act
	got := Map(lps, map[uint32]int{32: 4, 33: 1})

	// Assert
	if got[0].Kind != "P" || got[0].Sibling != 9 || got[0].WHEA != 4 {
		t.Fatalf("lp8 = %+v", got[0])
	}
	if got[2].Kind != "E" || got[2].Sibling != -1 || got[2].WHEA != 0 {
		t.Fatalf("lp16 = %+v", got[2])
	}
}

func TestMap_NonHybridHasNoKind(t *testing.T) {
	// Act
	got := Map([]platform.LogicalCPU{{Index: 0, Core: 0}, {Index: 1, Core: 1}}, nil)

	// Assert
	if got[0].Kind != "" || got[1].Kind != "" {
		t.Fatalf("got %+v", got)
	}
}
