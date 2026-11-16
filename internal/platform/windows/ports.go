//go:build windows

package windows

import (
	"context"
	"encoding/binary"
	"fmt"
	"net/netip"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/cn8001/reforgepulse/internal/platform"
)

var (
	iphlpapi                = windows.NewLazySystemDLL("iphlpapi.dll")
	procGetExtendedTcpTable = iphlpapi.NewProc("GetExtendedTcpTable")
	procGetExtendedUdpTable = iphlpapi.NewProc("GetExtendedUdpTable")
)

const (
	tcpTableOwnerPIDAll = 5
	udpTableOwnerPID    = 1
)

var tcpStates = map[uint32]string{1: "closed", 2: "listen", 3: "syn-sent", 4: "syn-received", 5: "established",
	6: "fin-wait-1", 7: "fin-wait-2", 8: "close-wait", 9: "closing", 10: "last-ack", 11: "time-wait", 12: "delete-tcb"}

// table describes one MIB_*ROW_OWNER_PID layout: where each field sits, so
// one reader covers TCP and UDP over IPv4 and IPv6.
type table struct {
	proto                     string
	udp                       bool
	family                    uintptr
	rowSize                   int
	local, localPort, pid     int
	remote, remotePort, state int // -1 when the row has no such field
	addrLen                   int
}

var tables = []table{
	{"tcp", false, windows.AF_INET, 24, 4, 8, 20, 12, 16, 0, 4},
	{"tcp6", false, windows.AF_INET6, 56, 0, 20, 52, 24, 44, 48, 16},
	{"udp", true, windows.AF_INET, 12, 0, 4, 8, -1, -1, -1, 4},
	{"udp6", true, windows.AF_INET6, 28, 0, 20, 24, -1, -1, -1, 16},
}

func (System) Ports(ctx context.Context) ([]platform.PortInfo, error) {
	var out []platform.PortInfo
	for _, t := range tables {
		rows, err := t.read()
		if err != nil {
			return out, err
		}
		out = append(out, rows...)
	}
	return out, nil
}

func (t table) read() ([]platform.PortInfo, error) {
	proc, class := procGetExtendedTcpTable, uintptr(tcpTableOwnerPIDAll)
	if t.udp {
		proc, class = procGetExtendedUdpTable, udpTableOwnerPID
	}
	var size uint32
	proc.Call(0, uintptr(unsafe.Pointer(&size)), 0, t.family, class, 0)
	if size == 0 {
		return nil, nil
	}
	buf := make([]byte, size)
	if r, _, _ := proc.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)), 1, t.family, class, 0); r != 0 {
		return nil, fmt.Errorf("read %s table: error %d", t.proto, r)
	}
	n := int(binary.LittleEndian.Uint32(buf))
	out := make([]platform.PortInfo, 0, n)
	for i := 0; i < n; i++ {
		row := buf[4+i*t.rowSize:]
		if len(row) < t.rowSize {
			break
		}
		p := platform.PortInfo{
			Proto: t.proto,
			PID:   binary.LittleEndian.Uint32(row[t.pid:]),
			Local: addr(row[t.local : t.local+t.addrLen]),
			Port:  port(row[t.localPort:]),
		}
		if t.remote >= 0 {
			p.Remote = fmt.Sprintf("%s:%d", addr(row[t.remote:t.remote+t.addrLen]), port(row[t.remotePort:]))
			p.State = tcpStates[binary.LittleEndian.Uint32(row[t.state:])]
		}
		out = append(out, p)
	}
	return out, nil
}

func addr(b []byte) string {
	a, _ := netip.AddrFromSlice(b)
	return a.String()
}

// Ports sit in the low word of a DWORD in network byte order.
func port(b []byte) uint16 { return binary.BigEndian.Uint16(b[:2]) }
