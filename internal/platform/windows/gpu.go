//go:build windows

package windows

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sys/windows/registry"
)

// displayClass is the device class key every display adapter registers under.
const displayClass = `SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}`

// GPU names the adapter with the most dedicated memory, so a desktop with
// both an integrated and a discrete GPU reports the one that games use.
func (System) GPU(ctx context.Context) (string, error) {
	class, err := registry.OpenKey(registry.LOCAL_MACHINE, displayClass, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return "", fmt.Errorf("open display class: %w", err)
	}
	defer class.Close()
	subs, err := class.ReadSubKeyNames(-1)
	if err != nil {
		return "", fmt.Errorf("list adapters: %w", err)
	}
	var best string
	var bestMem uint64
	for _, sub := range subs {
		k, err := registry.OpenKey(class, sub, registry.QUERY_VALUE)
		if err != nil {
			continue // "Properties" and similar are not adapters
		}
		name, _, nerr := k.GetStringValue("DriverDesc")
		mem, _, _ := k.GetIntegerValue("HardwareInformation.qwMemorySize")
		k.Close()
		if nerr == nil && (best == "" || mem > bestMem) {
			best, bestMem = name, mem
		}
	}
	if best == "" {
		return "", errors.New("no display adapter found")
	}
	return best, nil
}
