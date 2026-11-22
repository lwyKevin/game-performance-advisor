package analysis

import (
	"testing"

	"github.com/example/game-performance-advisor/internal/metrics"
	"github.com/example/game-performance-advisor/internal/system"
)

func TestAnalyzeSystemGPULimited(t *testing.T) {
	result := AnalyzeSystem(system.Info{Resolution: "2560x1440", RefreshHz: 165},
		metrics.Snapshot{CPUPercent: 48, CPUAvailable: true, CPUMeasurementSeconds: 1, GPUPercent: 96, GPUAvailable: true, RAMPercent: 55})
	if result.Diagnosis != "GPU Limited" {
		t.Fatalf("diagnosis = %q", result.Diagnosis)
	}
	if result.Confidence < 80 || result.FGReadiness < 80 {
		t.Fatalf("unexpected confidence/readiness: %d/%d", result.Confidence, result.FGReadiness)
	}
}

func TestAnalyzeSystemCPULimited(t *testing.T) {
	result := AnalyzeSystem(system.Info{}, metrics.Snapshot{CPUPercent: 92, CPUAvailable: true, CPUMeasurementSeconds: 1, GPUPercent: 60, GPUAvailable: true, RAMPercent: 50})
	if result.Diagnosis != "CPU Limited" {
		t.Fatalf("diagnosis = %q", result.Diagnosis)
	}
}

func TestAnalyzeSystemUnavailableGPU(t *testing.T) {
	result := AnalyzeSystem(system.Info{}, metrics.Snapshot{CPUPercent: 35, CPUAvailable: true, CPUMeasurementSeconds: 1, RAMPercent: 45})
	if len(result.Limitations) == 0 {
		t.Fatal("expected limitations when GPU metrics are unavailable")
	}
	if result.Diagnosis != "Insufficient evidence for bottleneck diagnosis" {
		t.Fatalf("diagnosis = %q", result.Diagnosis)
	}
}

func TestAnalyzeSystemInsufficientEvidenceLowCPU(t *testing.T) {
	result := AnalyzeSystem(system.Info{}, metrics.Snapshot{
		CPUPercent: 1, CPUAvailable: true, CPUMeasurementSeconds: 1,
		GPUPercent: 0, GPUAvailable: false, RAMPercent: 11,
	})
	if result.Diagnosis != "Insufficient evidence for bottleneck diagnosis" {
		t.Fatalf("diagnosis = %q", result.Diagnosis)
	}
	if result.Confidence > 20 {
		t.Fatalf("confidence = %d, want <=20", result.Confidence)
	}
}

func TestAnalyzeSystemZeroCPU(t *testing.T) {
	result := AnalyzeSystem(system.Info{}, metrics.Snapshot{
		CPUPercent: 0, CPUAvailable: true, CPUMeasurementSeconds: 1,
		GPUPercent: 50, GPUAvailable: true,
	})
	if result.Diagnosis != "Insufficient evidence for bottleneck diagnosis" {
		t.Fatalf("diagnosis = %q", result.Diagnosis)
	}
	if result.Diagnosis == "Balanced / no clear bottleneck" {
		t.Fatal("zero CPU must never produce balanced diagnosis")
	}
}

func TestAnalyzeSystemAcceptanceSnapshots(t *testing.T) {
	tests := []struct {
		name      string
		cpu       float64
		gpu       float64
		gpuOK     bool
		diagnosis string
	}{
		{name: "GPU limited", cpu: 60, gpu: 97, gpuOK: true, diagnosis: "GPU Limited"},
		{name: "CPU limited", cpu: 94, gpu: 50, gpuOK: true, diagnosis: "CPU Limited"},
		{name: "mixed ambiguous", cpu: 70, gpu: 82, gpuOK: true, diagnosis: "Mixed / ambiguous bottleneck"},
		{name: "balanced enough evidence", cpu: 50, gpu: 50, gpuOK: true, diagnosis: "Balanced / no clear bottleneck"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AnalyzeSystem(system.Info{}, metrics.Snapshot{
				CPUPercent: tt.cpu, CPUAvailable: true, CPUMeasurementSeconds: 1,
				GPUPercent: tt.gpu, GPUAvailable: tt.gpuOK,
			})
			if result.Diagnosis != tt.diagnosis {
				t.Fatalf("diagnosis = %q, want %q", result.Diagnosis, tt.diagnosis)
			}
		})
	}
}
