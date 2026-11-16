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

	procEvtOpenPublisherMetadata = wevtapi.NewProc("EvtOpenPublisherMetadata")
	procEvtFormatMessage         = wevtapi.NewProc("EvtFormatMessage")
)

var _ platform.EventLog = System{}

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
		Level       uint8  `xml:"Level"`
		TimeCreated struct {
			SystemTime string `xml:"SystemTime,attr"`
		} `xml:"TimeCreated"`
	} `xml:"System"`
	Data []struct {
		Name  string `xml:"Name,attr"`
		Value string `xml:",chardata"`
	} `xml:"EventData>Data"`
}

// Recent returns the newest events at or above a level (1 critical,
// 2 error, 3 warning) from every source, with their messages.
func (System) Recent(ctx context.Context, channel string, maxLevel int, since time.Time, max int) ([]platform.Event, error) {
	var levels []string
	for l := 1; l <= maxLevel; l++ {
		levels = append(levels, fmt.Sprintf("Level=%d", l))
	}
	q := fmt.Sprintf("*[System[(%s) and TimeCreated[@SystemTime>='%s']]]",
		strings.Join(levels, " or "), since.UTC().Format("2006-01-02T15:04:05.000Z"))
	publishers := map[string]uintptr{}
	defer func() {
		for _, h := range publishers {
			if h != 0 {
				procEvtClose.Call(h)
			}
		}
	}()
	return query(ctx, channel, q, max, func(h uintptr, e *platform.Event) {
		pub, ok := publishers[e.Provider]
		if !ok {
			name, _ := windows.UTF16PtrFromString(e.Provider)
			pub, _, _ = procEvtOpenPublisherMetadata.Call(0, uintptr(unsafe.Pointer(name)), 0, 0, 0)
			publishers[e.Provider] = pub
		}
		if pub != 0 {
			e.Message = formatMessage(pub, h)
		}
	})
}

// formatMessage renders an event's message text in the user's language.
func formatMessage(pub, event uintptr) string {
	const evtFormatMessageEvent = 1
	var used uint32
	procEvtFormatMessage.Call(pub, event, 0, 0, 0, evtFormatMessageEvent, 0, 0, uintptr(unsafe.Pointer(&used)))
	if used == 0 || used > 1<<16 {
		return ""
	}
	buf := make([]uint16, used)
	if ok, _, _ := procEvtFormatMessage.Call(pub, event, 0, 0, 0, evtFormatMessageEvent, uintptr(used), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&used))); ok == 0 {
		return ""
	}
	return strings.TrimSpace(windows.UTF16ToString(buf))
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

	return query(ctx, channel, q, max, nil)
}

// query runs an XPath query newest first; extra, when set, adds to each
// event while its handle is still open.
func query(ctx context.Context, channel, q string, max int, extra func(h uintptr, e *platform.Event)) ([]platform.Event, error) {
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
				if extra != nil {
					extra(eh, &e)
				}
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
	e := platform.Event{Provider: x.System.Provider.Name, ID: x.System.EventID, Level: x.System.Level, Time: t, Data: map[string]string{}}
	for i, d := range x.Data {
		name := d.Name
		if name == "" {
			name = fmt.Sprintf("param%d", i+1) // WER puts the bugcheck in unnamed fields
		}
		e.Data[name] = strings.TrimSpace(d.Value)
	}
	return e, nil
}
