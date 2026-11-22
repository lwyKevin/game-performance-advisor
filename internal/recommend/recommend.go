package recommend

import (
	"fmt"
	"strings"

	"github.com/example/game-performance-advisor/internal/analysis"
)

type Result struct {
	SmoothEnough       bool     `json:"smooth_enough"`
	FrameGeneration    string   `json:"frame_generation"`
	FGMultiplier       string   `json:"fg_multiplier"`
	FGSuitability      int      `json:"fg_suitability_score"`
	FGBenefit          string   `json:"fg_expected_benefit"`
	FGRisk             string   `json:"fg_risk"`
	FGReason           string   `json:"fg_reason"`
	Upscaling          string   `json:"spatial_upscaling"`
	LatencyQualitative string   `json:"latency_impact"`
	AddedLatencyMS     float64  `json:"approx_added_latency_ms"`
	LatencyRangeMS     string   `json:"latency_range_ms"`
	PriorityActions    []string `json:"priority_actions"`
	Summary            string   `json:"summary"`
	Evidence           []string `json:"evidence"`
	ExpectedVisualFPS  float64  `json:"expected_visual_fps"`
}

const (
	// FrameGenerationTargetGapFPS: frame generation is considered only when
	// base FPS is at least 10 FPS below target. This avoids recommending extra
	// latency for a stream that is already close to its goal.
	FrameGenerationTargetGapFPS = 10.0
	FrameGenerationTargetMargin = 0.95
)

func Recommend(a analysis.Result, targetFPS float64) Result {
	oneLow := 0.0
	if a.OnePercentLowFPS != nil {
		oneLow = *a.OnePercentLowFPS
	}
	smooth := a.OnePercentLowFPS != nil &&
		a.AverageFPS >= targetFPS*0.98 &&
		a.CoefficientVar <= 5 &&
		oneLow >= targetFPS*0.9
	upscaling := "Consider Balanced or Quality if additional FPS or resolution is needed."
	var fg, multiplier, latency string
	var added float64
	latencyRange := ""
	suitability := 0
	benefit := "Low"
	risk := "Low"
	reason := ""
	evidence := []string{}
	evidence = append(evidence, fmt.Sprintf("Captured average: %.1f FPS", a.AverageFPS))
	if a.OnePercentLowFPS != nil {
		evidence = append(evidence, fmt.Sprintf("Captured 1%% low: %.1f FPS", *a.OnePercentLowFPS))
	} else {
		evidence = append(evidence, "Captured 1% low: not available")
	}
	evidence = append(evidence, fmt.Sprintf("Captured frametime variance: %.1f%% CV", a.CoefficientVar))
	evidence = append(evidence, fmt.Sprintf("Captured stutter share: %.1f%%", a.StutterPercent))

	isGPU := strings.Contains(a.Bottleneck, "GPU-bound") ||
		strings.Contains(a.Bottleneck, "GPU Limited")
	isSpikeHeavy := strings.Contains(a.Bottleneck, "CPU-bound") ||
		strings.Contains(a.Bottleneck, "mixed")
	isHitching := strings.Contains(a.Bottleneck, "Stutter")
	meetsTarget := targetFPS > 0 && a.AverageFPS >= targetFPS*FrameGenerationTargetMargin

	switch {
	case meetsTarget:
		fg = "Not recommended"
		multiplier = "None"
		latency = "The base stream already meets the target; added FG latency is not justified."
		added = 0
		latencyRange = "+0 ms; no frame-generation benefit"
		suitability = 12
		risk = "Low"
		reason = "The base frame rate meets or is within 5% of the target, so frame generation would add latency without providing a visual refresh-rate benefit."
		upscaling = "Base FPS already meets the target; frame generation is not needed."
	case isSpikeHeavy || a.CoefficientVar >= 18 || a.StutterPercent >= 8:
		fg = "Not recommended yet"
		multiplier = "None"
		latency = "Negative value if spikes remain; FG may amplify perceived stutter."
		added = 0
		latencyRange = "+0 ms; fix pacing first"
		suitability = 18
		risk = "High"
		reason = "Frame pacing is unstable, so generated frames would be likely to amplify visible hitches."
		upscaling = "Enable spatial upscaling first if it raises stable base FPS."
	case smooth:
		fg = "Not recommended"
		multiplier = "None"
		latency = "The base stream is responsive; latency overhead is unlikely to be justified."
		added = 12
		latencyRange = "+8 to +15 ms"
		suitability = 12
		risk = "Low"
		reason = "The base render rate is stable and responsive, so visual smoothness gains are unlikely to justify added latency."
	case isGPU && a.AverageFPS < targetFPS:
		fg = "Likely helpful"
		multiplier = recommendMultiplier(a.AverageFPS)
		suitability = 84
		benefit = "Moderate"
		risk = "Medium"
		reason = "The stream is GPU-limited and has stable pacing, making generated frames less likely to expose pacing errors."
		latency = "Moderate added latency; strongest benefit on a stable GPU-bound frame rate."
		added = latencyForMultiplier(multiplier)
		latencyRange = latencyRangeForMultiplier(multiplier)
		upscaling = "Enable Quality or Balanced spatial upscaling first to raise base FPS."
	case isGPU:
		fg = "Possible, but validate latency tolerance"
		multiplier = "2x"
		suitability = 62
		benefit = "Moderate"
		risk = "Medium"
		reason = "Pacing is stable, but the base frame rate already meets the target and added latency may outweigh the benefit."
		latency = "Moderate added latency."
		added = 12
		latencyRange = "+8 to +15 ms"
	default:
		fg = "Use with caution"
		multiplier = "2x"
		suitability = 38
		risk = "Medium"
		reason = "Pacing is not stable enough to strongly predict that frame generation will improve the experience."
		latency = "Moderate added latency; avoid it until frame pacing improves."
		added = 12
		latencyRange = "+8 to +15 ms"
	}

	actions := make([]string, 0, 4)
	if isSpikeHeavy {
		actions = append(actions, "Fix CPU bottleneck, background activity, or shader/streaming hitches first")
	} else if isHitching {
		actions = append(actions, "Investigate shader loading, streaming, drivers, or isolated background hitches first")
	} else if isGPU && !meetsTarget {
		actions = append(actions, "Enable Quality or Balanced spatial upscaling, then retest base FPS")
	}
	if upscaling != "" && !smooth && !meetsTarget {
		actions = append(actions, "Enable spatial upscaling before frame generation")
	}
	if fg == "Likely helpful" {
		actions = append(actions, fmt.Sprintf("Test %s frame generation and compare input latency", multiplier))
	}
	actions = append(actions, "Retest with the same 1%/0.1% low metrics")

	summary := fmt.Sprintf("Base FPS is %.1f with %.1f%% coefficient of variation; %s.", a.AverageFPS, a.CoefficientVar, strings.ToLower(a.Bottleneck))
	expectedVisualFPS := 0.0
	if multiplier := parseMultiplier(multiplier); multiplier > 1 {
		expectedVisualFPS = a.AverageFPS * float64(multiplier)
	}
	return Result{
		SmoothEnough:       smooth,
		FrameGeneration:    fg,
		FGMultiplier:       multiplier,
		FGSuitability:      suitability,
		FGBenefit:          benefit,
		FGRisk:             risk,
		FGReason:           reason,
		Upscaling:          upscaling,
		LatencyQualitative: latency,
		AddedLatencyMS:     added,
		LatencyRangeMS:     latencyRange,
		PriorityActions:    deduplicate(actions),
		Summary:            summary,
		Evidence:           evidence,
		ExpectedVisualFPS:  expectedVisualFPS,
	}
}

func parseMultiplier(value string) int {
	switch value {
	case "2x":
		return 2
	case "3x":
		return 3
	case "4x":
		return 4
	default:
		return 0
	}
}

func recommendMultiplier(fps float64) string {
	switch {
	case fps >= 50:
		return "2x"
	case fps >= 33:
		return "3x"
	default:
		return "4x"
	}
}

func latencyForMultiplier(multiplier string) float64 {
	switch multiplier {
	case "2x":
		return 12
	case "3x":
		return 22
	case "4x":
		return 32
	default:
		return 0
	}
}

func latencyRangeForMultiplier(multiplier string) string {
	switch multiplier {
	case "2x":
		return "+8 to +15 ms"
	case "3x":
		return "+18 to +28 ms"
	case "4x":
		return "+28 to +38 ms"
	default:
		return "Not estimated"
	}
}

func deduplicate(values []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(values))
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}
