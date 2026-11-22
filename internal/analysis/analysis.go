package analysis

import (
	"fmt"
	"math"
	"sort"
)

type Result struct {
	FrameCount                  int      `json:"frame_count"`
	AverageMS                   float64  `json:"average_frametime_ms"`
	AverageFPS                  float64  `json:"average_fps"`
	MedianMS                    float64  `json:"median_ms"`
	OnePercentLowFPS            *float64 `json:"1_percent_low_fps"`
	OnePercentLowAvailable      bool     `json:"1_percent_low_available"`
	OnePercentLowReason         string   `json:"1_percent_low_reason,omitempty"`
	PointOnePercentLowFPS       *float64 `json:"0.1_percent_low_fps"`
	PointOnePercentLowAvailable bool     `json:"0.1_percent_low_available"`
	PointOnePercentLowReason    string   `json:"0.1_percent_low_reason,omitempty"`
	StdDevMS                    float64  `json:"standard_deviation_ms"`
	CoefficientVar              float64  `json:"coefficient_of_variation_percent"`
	StutterPercent              float64  `json:"stutter_percent"`
	StutterCount                int      `json:"stutter_count"`
	Bottleneck                  string   `json:"bottleneck"`
	Confidence                  int      `json:"diagnosis_confidence"`
	Alternative                 string   `json:"alternative_diagnosis"`
	Evidence                    []string `json:"diagnosis_evidence"`
	HealthScore                 int      `json:"frame_health_score"`
	SmoothnessScore             int      `json:"smoothness_score"`
	ConsistencyScore            int      `json:"consistency_score"`
	LatencyPotentialScore       int      `json:"latency_potential_score"`
	StutterResistanceScore      int      `json:"stutter_resistance_score"`
}

const (
	// StutterThresholdK defines a median-relative stutter heuristic. 1.8 is
	// deliberately conservative: it counts clear pacing spikes but may miss
	// very small regular alternation. It is a candidate for tuning.
	StutterThresholdK = 1.8
	// OnePercentFrameCountMinimum is the minimum capture size used for a
	// reported 1% low. Below this size, the percentile is statistically weak.
	OnePercentFrameCountMinimum = 100
	// PointOnePercentFrameCountMinimum is the minimum capture size used for a
	// reported 0.1% low. Below this size, the result represents only a few frames.
	PointOnePercentFrameCountMinimum = 1000
)

// Analyze calculates percentiles by averaging the slowest 1%/0.1% of frames,
// then converting their average frametime into frames per second.
func Analyze(frameTimes []float64, targetFPS float64, stutterThreshold float64) (Result, error) {
	if len(frameTimes) == 0 {
		return Result{}, fmt.Errorf("at least one frame time is required")
	}
	if targetFPS <= 0 {
		return Result{}, fmt.Errorf("target FPS must be greater than zero")
	}
	if stutterThreshold <= 1 {
		return Result{}, fmt.Errorf("stutter threshold must be greater than 1")
	}

	sum := 0.0
	copied := make([]float64, len(frameTimes))
	for i, value := range frameTimes {
		if value <= 0 {
			return Result{}, fmt.Errorf("frame time at index %d must be greater than zero", i)
		}
		sum += value
		copied[i] = value
	}
	average := sum / float64(len(copied))

	varianceSum := 0.0
	for _, value := range copied {
		varianceSum += (value - average) * (value - average)
	}
	stdDev := math.Sqrt(varianceSum / float64(len(copied)))
	cv := stdDev / average * 100

	sort.Float64s(copied)
	median := percentileValue(copied, 0.50)
	stutterCount := 0
	for _, value := range copied {
		if value > median*stutterThreshold {
			stutterCount++
		}
	}
	stutterPercent := float64(stutterCount) / float64(len(copied)) * 100

	// copied is ascending, so take the slowest frames from the end.
	oneLowAvailable := len(copied) >= OnePercentFrameCountMinimum
	oneLowFPS := 0.0
	if oneLowAvailable {
		oneLowFPS = slowAverageFPS(copied, 0.01)
	}
	pointOneAvailable := len(copied) >= PointOnePercentFrameCountMinimum
	pointOneLowFPS := 0.0
	if pointOneAvailable {
		pointOneLowFPS = slowAverageFPS(copied, 0.001)
	}
	var oneLowPointer *float64
	oneLowReason := ""
	if oneLowAvailable {
		value := oneLowFPS
		oneLowPointer = &value
	} else {
		oneLowReason = "not available: at least 100 frames are required"
	}
	var pointOnePointer *float64
	pointOneReason := ""
	if pointOneAvailable {
		value := pointOneLowFPS
		pointOnePointer = &value
	} else {
		pointOneReason = "not available: at least 1000 frames are required"
	}

	bottleneck, confidence, alternative, evidence := diagnose(average, targetFPS, cv, stutterPercent)
	if cv >= 15 && stutterPercent < 1 {
		evidence = append(evidence,
			"Frame distribution is widely split or bimodal; median-relative stutter detection may undercount sustained alternation")
	}
	if !oneLowAvailable {
		evidence = append(evidence, "1% low FPS is not reported: at least 100 frames are required")
	}
	if !pointOneAvailable {
		evidence = append(evidence, "0.1% low FPS is not reported: at least 1000 frames are required")
	}
	health, smoothness, consistency, latencyPotential, stutterResistance :=
		scoreHealth(average, targetFPS, cv, stutterPercent, oneLowFPS, oneLowAvailable)
	return Result{
		FrameCount:                  len(frameTimes),
		AverageMS:                   average,
		AverageFPS:                  1000 / average,
		MedianMS:                    median,
		OnePercentLowFPS:            oneLowPointer,
		OnePercentLowAvailable:      oneLowAvailable,
		OnePercentLowReason:         oneLowReason,
		PointOnePercentLowFPS:       pointOnePointer,
		PointOnePercentLowAvailable: pointOneAvailable,
		PointOnePercentLowReason:    pointOneReason,
		StdDevMS:                    stdDev,
		CoefficientVar:              cv,
		StutterPercent:              stutterPercent,
		StutterCount:                stutterCount,
		Bottleneck:                  bottleneck,
		Confidence:                  confidence,
		Alternative:                 alternative,
		Evidence:                    evidence,
		HealthScore:                 health,
		SmoothnessScore:             smoothness,
		ConsistencyScore:            consistency,
		LatencyPotentialScore:       latencyPotential,
		StutterResistanceScore:      stutterResistance,
	}, nil
}

func percentileValue(sorted []float64, p float64) float64 {
	index := int(math.Round(p * float64(len(sorted)-1)))
	return sorted[index]
}

func slowAverageFPS(sorted []float64, fraction float64) float64 {
	count := int(math.Ceil(float64(len(sorted)) * fraction))
	if count < 1 {
		count = 1
	}
	if count > len(sorted) {
		count = len(sorted)
	}
	sum := 0.0
	start := len(sorted) - count
	for _, value := range sorted[start:] {
		sum += value
	}
	return 1000 / (sum / float64(count))
}

func classify(averageMS, targetFPS, cv, stutterPercent float64) string {
	// These are intentionally simple, transparent thresholds.
	averageFPS := 1000 / averageMS
	if averageFPS >= targetFPS*0.98 && cv <= 5 && stutterPercent <= 1 {
		return "Smooth / no clear bottleneck"
	}
	if averageMS >= 16.67 && cv <= 12 && stutterPercent <= 3 {
		return "Likely GPU-bound"
	}
	if stutterPercent >= 8 {
		return "Likely CPU-bound or mixed with frequent spikes"
	}
	if stutterPercent >= 1 {
		return "Stutter / hitching"
	}
	if cv >= 18 {
		return "Likely CPU-bound or mixed with frequent spikes"
	}
	if cv >= 8 && stutterPercent >= 2 {
		return "Likely mixed bottleneck"
	}
	if stutterPercent >= 1 {
		return "Stutter / hitching"
	}
	return "Unclassified / mixed"
}

func diagnose(averageMS, targetFPS, cv, stutterPercent float64) (string, int, string, []string) {
	averageFPS := 1000 / averageMS
	evidence := make([]string, 0, 4)

	if averageFPS >= targetFPS*0.98 && cv <= 5 && stutterPercent <= 1 {
		confidence := clampInt(84+int((5-cv)*2), 80, 96)
		evidence = append(evidence,
			fmt.Sprintf("Average frame time is %.1f ms", averageMS),
			fmt.Sprintf("Frame variance is low (%.1f%% CV)", cv),
			"No meaningful stutter pattern detected",
		)
		return "Balanced / excellent pacing", confidence, "Minor GPU or mixed bottleneck", evidence
	}
	if stutterPercent >= 8 {
		confidence := clampInt(62+int(stutterPercent), 65, 92)
		evidence = append(evidence,
			fmt.Sprintf("%.1f%% of frames exceed the stutter threshold", stutterPercent),
			fmt.Sprintf("Frame variance is %.1f%% CV", cv),
			"Frequent spikes are consistent with CPU, streaming, or mixed-pressure hitches",
		)
		return "CPU Limited / mixed-pressure spikes", confidence, "Mixed bottleneck", evidence
	}
	if averageMS >= 16.67 && cv <= 12 {
		confidence := clampInt(68+int((12-cv)*2), 70, 88)
		evidence = append(evidence,
			fmt.Sprintf("Average frame time is %.1f ms", averageMS),
			fmt.Sprintf("Frame variance remains low (%.1f%% CV)", cv),
			fmt.Sprintf("Only %.1f%% of frames are classified as stutters", stutterPercent),
		)
		return "GPU Limited", confidence, "Mixed bottleneck", evidence
	}
	if stutterPercent >= 1 {
		confidence := clampInt(58+int(stutterPercent*3), 60, 82)
		evidence = append(evidence,
			fmt.Sprintf("%.1f%% of frames are isolated hitches", stutterPercent),
			fmt.Sprintf("Frame variance is %.1f%% CV", cv),
			"Pacing varies despite portions of the stream remaining stable",
		)
		return "Stutter / hitching", confidence, "Mixed bottleneck", evidence
	}
	confidence := 55
	evidence = append(evidence,
		fmt.Sprintf("Average frame time is %.1f ms", averageMS),
		fmt.Sprintf("Frame variance is %.1f%% CV", cv),
		"No single bottleneck clearly dominates",
	)
	return "Mixed / unclassified", confidence, "GPU or CPU limiting", evidence
}

func scoreHealth(averageMS, targetFPS, cv, stutterPercent, oneLowFPS float64, oneLowAvailable bool) (int, int, int, int, int) {
	averageFPS := 1000 / averageMS
	smoothness := clampInt(int(100-lowPenalty(averageFPS/targetFPS)*35-stutterPercent*2.5), 0, 100)
	consistency := clampInt(int(100-cv*3.2), 0, 100)
	latencyPotential := clampInt(int(100-averageFPS*0.25), 0, 100)
	if averageFPS >= targetFPS*0.98 {
		latencyPotential = clampInt(latencyPotential+10, 0, 100)
	}
	stutterResistance := clampInt(int(100-stutterPercent*7), 0, 100)
	lowFPSRatio := oneLowFPS / targetFPS
	if oneLowAvailable && lowFPSRatio < 1 {
		smoothness = clampInt(smoothness-int((1-lowFPSRatio)*55), 0, 100)
	}
	health := (smoothness*35 + consistency*25 + latencyPotential*15 + stutterResistance*25) / 100
	return clampInt(health, 0, 100), smoothness, consistency, latencyPotential, stutterResistance
}

func lowPenalty(ratio float64) float64 {
	if ratio >= 1 {
		return 0
	}
	return 1 - ratio
}

func clampInt(value, minimum, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}
