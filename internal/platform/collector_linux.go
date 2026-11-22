//go:build !windows

package platform

import (
	"os"
	"strings"

	"github.com/example/game-performance-advisor/internal/metrics"
	"github.com/example/game-performance-advisor/internal/system"
)

type linuxCollector struct{}

func NewCollector() Collector {
	return linuxCollector{}
}

func (linuxCollector) Environment() string {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return "container (linux)"
	}
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		text := string(data)
		if strings.Contains(text, "docker") || strings.Contains(text, "containerd") || strings.Contains(text, "kubepods") {
			return "container (linux)"
		}
	}
	return "linux-native"
}

func (linuxCollector) Collect() (system.Info, metrics.Snapshot, error) {
	machine, err := system.Collect()
	if err != nil {
		return machine, metrics.Snapshot{}, err
	}
	snapshot, err := metrics.Collect()
	return machine, snapshot, err
}
