//go:build windows

package platform

import (
	"github.com/example/game-performance-advisor/internal/metrics"
	"github.com/example/game-performance-advisor/internal/system"
)

type windowsCollector struct{}

func NewCollector() Collector {
	return windowsCollector{}
}

func (windowsCollector) Collect() (system.Info, metrics.Snapshot, error) {
	machine, err := system.Collect()
	if err != nil {
		return machine, metrics.Snapshot{}, err
	}
	snapshot, err := metrics.Collect()
	return machine, snapshot, err
}
