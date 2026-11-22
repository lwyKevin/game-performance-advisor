package report

import (
	"embed"
	"encoding/json"
	"fmt"
	"html"
	htmltemplate "html/template"
	"strings"
	"text/template"

	"github.com/example/game-performance-advisor/internal/analysis"
	"github.com/example/game-performance-advisor/internal/metrics"
	"github.com/example/game-performance-advisor/internal/recommend"
	"github.com/example/game-performance-advisor/internal/system"
)

//go:embed templates/*.html templates/*.md
var embeddedTemplates embed.FS

type SystemReport struct {
	Machine        system.Info               `json:"system"`
	Snapshot       metrics.Snapshot          `json:"performance_snapshot"`
	Assessment     analysis.SystemAssessment `json:"assessment"`
	Recommendation recommend.SystemResult    `json:"recommendation"`
}

func JSON(r SystemReport) ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

func Terminal(r SystemReport) string {
	m, s, a, rec := r.Machine, r.Snapshot, r.Assessment, r.Recommendation
	var b strings.Builder
	fmt.Fprintln(&b, "┌────────────────────────────────────────────┐")
	fmt.Fprintln(&b, "│ GAME PERFORMANCE ADVISOR                   │")
	fmt.Fprintln(&b, "└────────────────────────────────────────────┘")
	fmt.Fprintln(&b, "\nSystem Summary")
	fmt.Fprintf(&b, "  Detected CPU       %s\n", first(m.CPUModel, "Not detected"))
	if len(m.GPUModels) > 0 {
		fmt.Fprintf(&b, "  Detected GPU       %s\n", strings.Join(m.GPUModels, ", "))
	} else {
		fmt.Fprintln(&b, "  Detected GPU       Not detected")
	}
	fmt.Fprintf(&b, "  Memory             %.1f GB\n", m.RAMGB)
	fmt.Fprintf(&b, "  Display            %s", m.Resolution)
	if m.RefreshHz > 0 {
		fmt.Fprintf(&b, " @ %d Hz", m.RefreshHz)
	}
	fmt.Fprintf(&b, "\n  Operating System   %s\n", m.OS)

	fmt.Fprintln(&b, "\nPerformance Snapshot")
	fmt.Fprintf(&b, "  GPU utilization    %.0f%%", s.GPUPercent)
	if !s.GPUAvailable {
		fmt.Fprint(&b, " (not available)")
	}
	fmt.Fprintf(&b, "\n  GPU peak / p95     %.0f%% / %.0f%%", s.GPUPeakPercent, s.GPUP95Percent)
	fmt.Fprintf(&b, "\n  CPU utilization    %.0f%%\n", s.CPUPercent)
	fmt.Fprintf(&b, "  CPU peak / p95     %.0f%% / %.0f%%\n", s.CPUPeakPercent, s.CPUP95Percent)
	fmt.Fprintf(&b, "  Memory utilization %.0f%%\n", s.RAMPercent)
	fmt.Fprintf(&b, "  Memory peak / p95  %.0f%% / %.0f%%\n", s.RAMPeakPercent, s.RAMP95Percent)
	if len(s.GameProcesses) > 0 {
		fmt.Fprintf(&b, "  Likely game        %s\n", s.GameProcesses[0].Name)
	}

	fmt.Fprintln(&b, "\nPerformance Assessment")
	fmt.Fprintf(&b, "  Performance Score          %d / 100\n", a.PerformanceScore)
	fmt.Fprintf(&b, "  Smoothness Score           %d / 100\n", a.SmoothnessScore)
	fmt.Fprintf(&b, "  FG Readiness               %d / 100\n", a.FGReadiness)
	fmt.Fprintf(&b, "  Confidence                 %d / 100\n", a.Confidence)
	fmt.Fprintf(&b, "\n  Primary Diagnosis          %s\n", a.Diagnosis)
	fmt.Fprintf(&b, "  Alternative                %s\n", a.Alternative)

	fmt.Fprintln(&b, "\nEvidence")
	for _, evidence := range a.Evidence {
		fmt.Fprintf(&b, "  • %s\n", evidence)
	}
	for _, limitation := range a.Limitations {
		fmt.Fprintf(&b, "  ! %s\n", limitation)
	}

	fmt.Fprintln(&b, "\nRecommendation")
	fmt.Fprintf(&b, "  Upscaling                  %s\n", rec.Upscaling)
	fmt.Fprintf(&b, "  Frame Generation           %s", rec.FrameGeneration)
	if rec.FGMultiplier != "" {
		fmt.Fprintf(&b, " (%s)", rec.FGMultiplier)
	}
	fmt.Fprintf(&b, "\n  Estimated Benefit          %s\n", rec.FGBenefit)
	fmt.Fprintf(&b, "  Latency Impact             %s\n", rec.LatencyImpact)
	fmt.Fprintln(&b, "\nSuggested Actions")
	for i, action := range rec.PriorityActions {
		fmt.Fprintf(&b, "  %d. %s\n", i+1, action)
	}
	return b.String()
}

func Markdown(r SystemReport) string {
	m, s, a, rec := r.Machine, r.Snapshot, r.Assessment, r.Recommendation
	raw, err := embeddedTemplates.ReadFile("templates/system.md")
	if err != nil {
		return ""
	}
	tmpl, err := template.New("system").Parse(string(raw))
	if err != nil {
		return ""
	}
	data := struct {
		Machine        system.Info
		Snapshot       metrics.Snapshot
		Assessment     analysis.SystemAssessment
		Recommendation recommend.SystemResult
	}{
		Machine: m, Snapshot: s, Assessment: a, Recommendation: rec,
	}
	var b strings.Builder
	if err := tmpl.Execute(&b, data); err != nil {
		return ""
	}
	return b.String()
}

func HTML(r SystemReport) string {
	m, s, a, rec := r.Machine, r.Snapshot, r.Assessment, r.Recommendation
	gpus := strings.Join(m.GPUModels, ", ")
	if gpus == "" {
		gpus = "Not detected"
	}
	raw, err := embeddedTemplates.ReadFile("templates/system.html")
	if err != nil {
		return ""
	}
	tmpl, err := htmltemplate.New("system").Parse(string(raw))
	if err != nil {
		return ""
	}
	data := struct {
		Machine        system.Info
		Snapshot       metrics.Snapshot
		Assessment     analysis.SystemAssessment
		Recommendation recommend.SystemResult
		GPUs           string
	}{
		Machine: m, Snapshot: s, Assessment: a, Recommendation: rec, GPUs: gpus,
	}
	var b strings.Builder
	if err := tmpl.Execute(&b, data); err != nil {
		return ""
	}
	return b.String()
}

func evidenceHTML(values []string) string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		items = append(items, "<li>"+html.EscapeString(value)+"</li>")
	}
	return strings.Join(items, "")
}

func actionsHTML(values []string) string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		items = append(items, "<li>"+html.EscapeString(value)+"</li>")
	}
	return strings.Join(items, "")
}

func first(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
