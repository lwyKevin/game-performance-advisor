package report

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/example/game-performance-advisor/internal/analysis"
	"github.com/example/game-performance-advisor/internal/metrics"
	"github.com/example/game-performance-advisor/internal/recommend"
	"github.com/example/game-performance-advisor/internal/system"
)

type Report struct {
	Command     string         `json:"command"`
	Platform    string         `json:"platform"`
	Environment string         `json:"environment"`
	System      *SystemPayload `json:"system,omitempty"`
	Frame       *FramePayload  `json:"frame,omitempty"`
}

type SystemPayload struct {
	Machine        system.Info               `json:"machine"`
	Snapshot       metrics.Snapshot          `json:"performance_snapshot"`
	Assessment     analysis.SystemAssessment `json:"assessment"`
	Recommendation recommend.SystemResult    `json:"recommendation"`
	Environment    string                    `json:"environment"`
}

type FramePayload struct {
	Source         string           `json:"source"`
	TargetFPS      float64          `json:"target_fps"`
	Analysis       analysis.Result  `json:"analysis"`
	Recommendation recommend.Result `json:"recommendation"`
}

func JSON(r Report) ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

func Terminal(r Report) string {
	if r.System != nil {
		return terminalSystem(r.System)
	}
	if r.Frame != nil {
		return terminalFrame(r.Frame)
	}
	return ""
}

func terminalFrame(p *FramePayload) string {
	a, rec := p.Analysis, p.Recommendation
	var b strings.Builder
	fmt.Fprintln(&b, "Game Performance Advisor - Frame Analysis")
	fmt.Fprintf(&b, "\nSource              %s\n", p.Source)
	fmt.Fprintf(&b, "Frames              %d\n", a.FrameCount)
	fmt.Fprintf(&b, "Average FPS         %.1f\n", a.AverageFPS)
	if a.OnePercentLowAvailable {
		fmt.Fprintf(&b, "1%% Low FPS          %.1f\n", *a.OnePercentLowFPS)
	} else {
		fmt.Fprintln(&b, "1% Low FPS          Not available (<100 frames)")
	}
	if a.PointOnePercentLowAvailable {
		fmt.Fprintf(&b, "0.1%% Low FPS        %.1f\n", *a.PointOnePercentLowFPS)
	} else {
		fmt.Fprintln(&b, "0.1% Low FPS        Not available (<1000 frames)")
	}
	fmt.Fprintf(&b, "Frame Health        %d / 100\n", a.HealthScore)
	fmt.Fprintf(&b, "Diagnosis           %s (%d%% confidence)\n", a.Bottleneck, a.Confidence)
	fmt.Fprintf(&b, "FG Recommendation   %s (%s)\n", rec.FrameGeneration, rec.FGMultiplier)
	fmt.Fprintf(&b, "Latency Impact      %s\n", rec.LatencyRangeMS)
	return b.String()
}

func terminalSystem(p *SystemPayload) string {
	m, s, a, rec := p.Machine, p.Snapshot, p.Assessment, p.Recommendation
	var b strings.Builder
	fmt.Fprintln(&b, "Game Performance Advisor - Hardware Scan")
	fmt.Fprintf(&b, "Environment         %s\n", p.Environment)
	fmt.Fprintf(&b, "\nSystem Summary\n")
	fmt.Fprintf(&b, "  CPU                 %s\n", first(m.CPUModel, "Not detected"))
	fmt.Fprintf(&b, "  GPU                 %s\n", strings.Join(m.GPUModels, ", "))
	fmt.Fprintf(&b, "  Memory              %.1f GB\n", m.RAMGB)
	fmt.Fprintf(&b, "  Operating System    %s\n", m.OS)
	fmt.Fprintf(&b, "\nPerformance Snapshot\n")
	fmt.Fprintf(&b, "  CPU                 %.0f%%\n", s.CPUPercent)
	fmt.Fprintf(&b, "  GPU                 %.0f%%", s.GPUPercent)
	if !s.GPUAvailable {
		fmt.Fprint(&b, " (not available)")
	}
	fmt.Fprintf(&b, "\n  Memory              %.0f%%\n", s.RAMPercent)
	fmt.Fprintf(&b, "\nDiagnosis\n")
	fmt.Fprintf(&b, "  %s (%d%% confidence)\n", a.Diagnosis, a.Confidence)
	for _, evidence := range a.Evidence {
		fmt.Fprintf(&b, "  Evidence: %s\n", evidence)
	}
	for _, limitation := range a.Limitations {
		fmt.Fprintf(&b, "  Not available: %s\n", limitation)
	}
	fmt.Fprintf(&b, "\nRecommendation\n")
	fmt.Fprintf(&b, "  Upscaling: %s\n", rec.Upscaling)
	fmt.Fprintf(&b, "  Frame Generation: %s\n", rec.FrameGeneration)
	return b.String()
}

func first(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
