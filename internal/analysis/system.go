package analysis

import (
	"fmt"

	"github.com/example/game-performance-advisor/internal/metrics"
	"github.com/example/game-performance-advisor/internal/system"
)

type SystemAssessment struct {
	PerformanceScore int      `json:"performance_score"`
	SmoothnessScore  int      `json:"smoothness_score"`
	FGReadiness      int      `json:"frame_generation_readiness"`
	Confidence       int      `json:"confidence"`
	Diagnosis        string   `json:"primary_diagnosis"`
	Alternative      string   `json:"alternative_diagnosis"`
	Evidence         []string `json:"evidence"`
	Limitations      []string `json:"limitations"`
}

const (
	// A near-zero CPU reading is treated as a measurement artifact rather than
	// evidence that the system is idle. Candidate confidence requires at least
	// one second of measurement and CPU utilization above 5%.
	MinimumUsableCPUPercent = 5.0
	MinimumCPUWindowSeconds = 1.0
	// GPU utilization at or above 95% with CPU below 85% is treated as a
	// candidate GPU-limited diagnosis. Both thresholds are transparent tunable
	// defaults, not measurements of a specific game's engine.
	GPUUtilizationLimitedThreshold     = 95.0
	CPUUtilizationNotDominantThreshold = 85.0
	// CPU utilization at or above 92% is the candidate CPU-limited threshold.
	CPUUtilizationLimitedThreshold = 92.0
	// When CPU and GPU are both in the 80-95% region, the diagnosis is
	// intentionally ambiguous and remains low confidence.
	MixedAmbiguityGPUThreshold = 80.0
	MixedAmbiguityCPUThreshold = 55.0
)

// AnalyzeSystem produces transparent system-pressure diagnostics.
// It deliberately avoids treating one metric as certainty.
func AnalyzeSystem(machine system.Info, snapshot metrics.Snapshot) SystemAssessment {
	evidence := make([]string, 0, 6)
	limitations := make([]string, 0, 3)

	cpuMeasured := snapshot.CPUAvailable &&
		snapshot.CPUMeasurementSeconds >= MinimumCPUWindowSeconds &&
		snapshot.CPUPercent >= MinimumUsableCPUPercent &&
		snapshot.CPUPercent <= 100

	if !cpuMeasured {
		switch {
		case snapshot.CPUMeasurementSeconds < MinimumCPUWindowSeconds:
			limitations = append(limitations, "CPU utilization was not measured over a 1-second or longer window")
		case snapshot.CPUPercent <= 0:
			limitations = append(limitations, "CPU utilization is 0%; it is not yet usable as evidence")
		case snapshot.CPUPercent < MinimumUsableCPUPercent:
			limitations = append(limitations, fmt.Sprintf("CPU utilization was %.0f%%; a near-zero reading is treated as a measurement artifact", snapshot.CPUPercent))
		default:
			limitations = append(limitations, "CPU utilization was outside the plausible 0-100% range")
		}
	}
	if !snapshot.GPUAvailable {
		limitations = append(limitations, "GPU utilization was not available; CPU-only diagnosis is not enabled")
	}
	if !cpuMeasured || !snapshot.GPUAvailable {
		limitations = append(limitations, "Insufficient evidence for bottleneck diagnosis")
		return SystemAssessment{
			PerformanceScore: 0,
			SmoothnessScore:  0,
			FGReadiness:      0,
			Confidence:       20,
			Diagnosis:        "Insufficient evidence for bottleneck diagnosis",
			Alternative:      "No bottleneck diagnosis available",
			Evidence:         evidence,
			Limitations:      limitations,
		}
	}
	gpuLimiting := snapshot.GPUAvailable && snapshot.GPUPercent >= GPUUtilizationLimitedThreshold
	cpuHigh := snapshot.CPUPercent >= CPUUtilizationLimitedThreshold
	cpuModerate := snapshot.CPUPercent >= MixedAmbiguityCPUThreshold
	memoryHigh := snapshot.RAMPercent >= 85

	if snapshot.GPUAvailable {
		evidence = append(evidence, fmt.Sprintf("GPU utilization was %.0f%% during the snapshot", snapshot.GPUPercent))
	} else {
		limitations = append(limitations, "GPU utilization was not available; vendor-specific tools may provide more detail")
	}
	evidence = append(evidence, fmt.Sprintf("CPU utilization was %.0f%%", snapshot.CPUPercent))
	evidence = append(evidence, fmt.Sprintf("Memory utilization was %.0f%%", snapshot.RAMPercent))
	if len(snapshot.GameProcesses) > 0 {
		evidence = append(evidence, fmt.Sprintf("Detected likely game process: %s", snapshot.GameProcesses[0].Name))
	} else {
		limitations = append(limitations, "No likely game process was detected; this snapshot may reflect the desktop or a non-matched launcher")
	}
	if machine.RefreshHz > 0 && machine.Resolution != "" {
		evidence = append(evidence, fmt.Sprintf("Display is %s at %d Hz", machine.Resolution, machine.RefreshHz))
	} else {
		limitations = append(limitations, "Display refresh or resolution could not be confirmed")
	}

	diagnosis := "Balanced / no clear bottleneck"
	alternative := "Mixed bottleneck"
	confidence := 55
	switch {
	case gpuLimiting && snapshot.CPUPercent < CPUUtilizationNotDominantThreshold:
		diagnosis = "GPU Limited"
		confidence = 82
		alternative = "GPU and CPU constrained"
	case cpuHigh:
		diagnosis = "CPU Limited"
		confidence = 78
		alternative = "CPU and memory pressure"
	case memoryHigh:
		diagnosis = "Memory Pressure"
		confidence = 72
		alternative = "CPU or storage pressure"
	case snapshot.GPUAvailable &&
		snapshot.GPUPercent >= MixedAmbiguityGPUThreshold &&
		snapshot.CPUPercent >= MixedAmbiguityCPUThreshold &&
		snapshot.CPUPercent < CPUUtilizationNotDominantThreshold:
		diagnosis = "Mixed / ambiguous bottleneck"
		confidence = 58
		alternative = "GPU Limited or CPU / mixed pressure"
	case cpuModerate && snapshot.GPUAvailable && snapshot.GPUPercent < 80:
		diagnosis = "CPU / mixed bottleneck"
		confidence = 68
		alternative = "GPU Limited"
	}

	performance := clampScore(100 - int(snapshot.RAMPercent)*20/100 - int(snapshot.CPUPercent)*25/100)
	if snapshot.GPUAvailable {
		gpuHeadroom := 100 - int(snapshot.GPUPercent)
		performance = (performance + clampScore(100-gpuHeadroom/2)) / 2
	}
	smoothness := clampScore(100 - int(snapshot.CPUPercent)*30/100 - int(snapshot.RAMPercent)*20/100)
	fgReadiness := 25
	switch diagnosis {
	case "GPU Limited":
		fgReadiness = 90
	case "Balanced / no clear bottleneck":
		fgReadiness = 55
	case "CPU / mixed bottleneck", "CPU Limited":
		fgReadiness = 25
	case "Memory Pressure":
		fgReadiness = 20
	}

	return SystemAssessment{
		PerformanceScore: performance,
		SmoothnessScore:  smoothness,
		FGReadiness:      fgReadiness,
		Confidence:       confidence,
		Diagnosis:        diagnosis,
		Alternative:      alternative,
		Evidence:         evidence,
		Limitations:      limitations,
	}
}

func clampScore(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}
