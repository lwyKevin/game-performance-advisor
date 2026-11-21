package analysis

import (
	"testing"

	"github.com/example/game-performance-advisor/internal/metrics"
	"github.com/example/game-performance-advisor/internal/system"
)

func TestAnalyzeSystemGPULimited(t *testing.T) {
	result := AnalyzeSystem(system.Info{Resolution: "2560x1440", RefreshHz: 165},
		metrics.Snapshot{CPUPercent: 48, GPUPercent: 96, GPUAvailable: true, RAMPercent: 55})
	if result.Diagnosis != "GPU Limited" {
		t.Fatalf("diagnosis = %q", result.Diagnosis)
	}
	if result.Confidence < 80 || result.FGReadiness < 80 {
		t.Fatalf("unexpected confidence/readiness: %d/%d", result.Confidence, result.FGReadiness)
	}
}

func TestAnalyzeSystemCPULimited(t *testing.T) {
	result := AnalyzeSystem(system.Info{}, metrics.Snapshot{CPUPercent: 92, GPUPercent: 60, GPUAvailable: true, RAMPercent: 50})
	if result.Diagnosis != "CPU Limited" {
		t.Fatalf("diagnosis = %q", result.Diagnosis)
	}
}

func TestAnalyzeSystemUnavailableGPU(t *testing.T) {
	result := AnalyzeSystem(system.Info{}, metrics.Snapshot{CPUPercent: 35, RAMPercent: 45})
	if len(result.Limitations) == 0 {
		t.Fatal("expected limitations when GPU metrics are unavailable")
	}
}
