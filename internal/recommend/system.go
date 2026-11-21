package recommend

import (
	"strconv"

	"github.com/example/game-performance-advisor/internal/analysis"
	"github.com/example/game-performance-advisor/internal/system"
)

type SystemResult struct {
	Upscaling        string   `json:"spatial_upscaling"`
	FrameGeneration  string   `json:"frame_generation"`
	FGMultiplier     string   `json:"fg_multiplier"`
	FGBenefit        string   `json:"estimated_benefit"`
	LatencyImpact    string   `json:"latency_impact"`
	PriorityActions  []string `json:"priority_actions"`
	AssessmentReason []string `json:"assessment_reason"`
}

// RecommendSystem turns a system assessment into an explainable action plan.
func RecommendSystem(machine system.Info, assessment analysis.SystemAssessment) SystemResult {
	result := SystemResult{
		Upscaling:        "Monitor target FPS before changing upscaling",
		FrameGeneration:  "Evaluate only after base frame pacing is confirmed",
		FGMultiplier:     "2x",
		FGBenefit:        "Uncertain",
		LatencyImpact:    "Expected to add roughly 8-15 ms",
		PriorityActions:  []string{},
		AssessmentReason: []string{},
	}

	switch assessment.Diagnosis {
	case "GPU Limited":
		result.Upscaling = "Enable DLSS/FSR/XeSS Quality first"
		result.FrameGeneration = "Likely useful"
		result.FGMultiplier = "2x"
		result.FGBenefit = "Moderate to High"
		result.AssessmentReason = append(result.AssessmentReason,
			"GPU utilization was consistently high",
			"CPU utilization did not indicate a dominant CPU bottleneck",
			"Upscaling may raise rendering throughput before frame generation")
		result.PriorityActions = []string{
			"Enable DLSS/FSR/XeSS Quality",
			"Re-test GPU and CPU utilization",
			"Consider 2x Frame Generation",
		}
	case "CPU Limited", "CPU / mixed bottleneck", "Memory Pressure":
		result.Upscaling = "Use Balanced or Quality only if stable frame time is preserved"
		result.FrameGeneration = "Not recommended yet"
		result.FGMultiplier = "None"
		result.FGBenefit = "Low or uncertain"
		result.LatencyImpact = "Fix pacing first; enabling FG may amplify hitches"
		result.AssessmentReason = append(result.AssessmentReason,
			"The snapshot shows CPU or memory pressure",
			"Frame generation may expose or amplify frame-pacing problems")
		result.PriorityActions = []string{
			"Reduce CPU-heavy settings or background workload",
			"Verify memory capacity and paging",
			"Re-test performance before enabling Frame Generation",
		}
	default:
		result.Upscaling = "Try Quality or Balanced and compare target-FPS latency"
		result.FrameGeneration = "Optional"
		result.FGMultiplier = "2x, if supported"
		result.FGBenefit = "Moderate"
		result.AssessmentReason = append(result.AssessmentReason,
			"No dominant bottleneck was confirmed",
			"Stable utilization is required before treating FG as beneficial")
		result.PriorityActions = []string{
			"Establish a target FPS and latency budget",
			"Test Quality upscaling",
			"Consider 2x Frame Generation only after a stable re-test",
		}
	}
	if machine.RefreshHz > 0 {
		result.AssessmentReason = append(result.AssessmentReason,
			"Display refresh is "+itoa(machine.RefreshHz)+" Hz")
	}
	return result
}

func itoa(value int) string {
	return strconv.Itoa(value)
}
