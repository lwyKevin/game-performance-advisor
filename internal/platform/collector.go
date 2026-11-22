package platform

import (
	"github.com/example/game-performance-advisor/internal/metrics"
	"github.com/example/game-performance-advisor/internal/system"
)

// Collector is the OS adapter boundary. Both scan implementations return
// inventory and one live snapshot, marking unavailable values explicitly.
type Collector interface {
	Collect() (system.Info, metrics.Snapshot, error)
}
