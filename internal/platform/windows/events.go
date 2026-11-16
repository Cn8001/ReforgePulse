//go:build windows

package windows

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/cn8001/reforgepulse/internal/platform"
)

var (
	wevtapi       = windows.NewLazySystemDLL("wevtapi.dll")
	procEvtQuery  = wevtapi.NewProc("EvtQuery")
	procEvtNext   = wevtapi.NewProc("EvtNext")
	procEvtRender = wevtapi.NewProc("EvtRender")
	procEvtClose  = wevtapi.NewProc("EvtClose")
)

const (
	evtQueryChannelPath      = 0x1
	evtQueryReverseDirection = 0x200
	evtRenderEventXML        = 1
	errNoMoreItems           = windows.Errno(259)
)

// eventXML is the part of an event's XML the analysis needs. EventData
// stays structured (bugcheck codes, APIC IDs), so no localized message text
// has to be parsed.
type eventXML struct {
	System struct {
		Provider struct {
			Name string `xml:"Name,attr"`
		} `xml:"Provider"`
		EventID     uint32 `xml:"EventID"`
		TimeCreated struct {
			SystemTime string `xml:"SystemTime,attr"`
		} `xml:"TimeCreated"`
	} `xml:"System"`
	Data []struct {
		Name  string `xml:"Name,attr"`
		Value string `xml:",chardata"`
	} `xml:"EventData>Data"`
}

// Events returns the newest events first, at most max of them.
func (System) Events(ctx context.Context, channel, provider string, ids []uint32, since time.Time, max int) ([]platform.Event, error) {
	q := fmt.Sprintf("*[System[Provider[@Name='%s']", provider)
	if len(ids) > 0 {
		parts := make([]string, len(ids))
		for i, id := range ids {
			parts[i] = fmt.Sprintf("EventID=%d", id)
		}
		q += " and (" + strings.Join(parts, " or ") + ")"
	}
	q += fmt.Sprintf(" and TimeCreated[@SystemTime>='%s']]]", since.UTC().Format("2006-01-02T15:04:05.000Z"))

	ch, _ := windows.UTF16PtrFromString(channel)
	qp, _ := windows.UTF16PtrFromString(q)
	h, _, err := procEvtQuery.Call(0, uintptr(unsafe.Pointer(ch)), uintptr(unsafe.Pointer(qp)), evtQueryChannelPath|evtQueryReverseDirection)
	if h == 0 {
		return nil, fmt.Errorf("query %s: %w", channel, err)
	}
	defer procEvtClose.Call(h)

	var out []platform.Event
	handles := make([]uintptr, 32)
	for len(out) < max {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		var n uint32
		ok, _, err := procEvtNext.Call(h, uintptr(len(handles)), uintptr(unsafe.Pointer(&handles[0])), 0, 0, uintptr(unsafe.Pointer(&n)))
		if ok == 0 {
			if errors.Is(err, errNoMoreItems) {
				break
			}
			return out, fmt.Errorf("read %s: %w", channel, err)
		}
		for _, eh := range handles[:n] {
			if e, err := render(eh); err == nil && len(out) < max {
				out = append(out, e)
			}
			procEvtClose.Call(eh)
		}
	}
	return out, nil
}

func render(h uintptr) (platform.Event, error) {
	var used, props uint32
	procEvtRender.Call(0, h, evtRenderEventXML, 0, 0, uintptr(unsafe.Pointer(&used)), uintptr(unsafe.Pointer(&props)))
	if used == 0 {
		return platform.Event{}, errors.New("empty event")
	}
	buf := make([]uint16, used/2+1)
	if ok, _, err := procEvtRender.Call(0, h, evtRenderEventXML, uintptr(used), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&used)), uintptr(unsafe.Pointer(&props))); ok == 0 {
		return platform.Event{}, fmt.Errorf("render event: %w", err)
	}
	var x eventXML
	if err := xml.Unmarshal([]byte(windows.UTF16ToString(buf)), &x); err != nil {
		return platform.Event{}, fmt.Errorf("parse event: %w", err)
	}
	t, _ := time.Parse(time.RFC3339Nano, x.System.TimeCreated.SystemTime)
	e := platform.Event{Provider: x.System.Provider.Name, ID: x.System.EventID, Time: t, Data: map[string]string{}}
	for i, d := range x.Data {
		name := d.Name
		if name == "" {
			name = fmt.Sprintf("param%d", i+1) // WER puts the bugcheck in unnamed fields
		}
		e.Data[name] = strings.TrimSpace(d.Value)
	}
	return e, nil
}
