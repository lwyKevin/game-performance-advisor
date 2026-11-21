package report

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"

	"github.com/example/game-performance-advisor/internal/analysis"
	"github.com/example/game-performance-advisor/internal/metrics"
	"github.com/example/game-performance-advisor/internal/recommend"
	"github.com/example/game-performance-advisor/internal/system"
)

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
	fmt.Fprintf(&b, "\n  CPU utilization    %.0f%%\n", s.CPUPercent)
	fmt.Fprintf(&b, "  Memory utilization %.0f%%\n", s.RAMPercent)
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
	var b strings.Builder
	fmt.Fprintln(&b, "# Game Performance Advisor")
	fmt.Fprintln(&b, "\n## System Summary")
	fmt.Fprintf(&b, "\n| Item | Detected |\n|---|---|\n")
	fmt.Fprintf(&b, "| CPU | %s |\n", m.CPUModel)
	fmt.Fprintf(&b, "| GPU | %s |\n", strings.Join(m.GPUModels, ", "))
	fmt.Fprintf(&b, "| RAM | %.1f GB |\n", m.RAMGB)
	fmt.Fprintf(&b, "| Display | %s @ %d Hz |\n", m.Resolution, m.RefreshHz)
	fmt.Fprintf(&b, "| OS | %s |\n", m.OS)
	fmt.Fprintln(&b, "\n## Performance Snapshot")
	fmt.Fprintf(&b, "\n- GPU: %.0f%%\n- CPU: %.0f%%\n- Memory: %.0f%%\n",
		s.GPUPercent, s.CPUPercent, s.RAMPercent)
	fmt.Fprintln(&b, "\n## Assessment")
	fmt.Fprintf(&b, "\n- Performance score: **%d / 100**\n", a.PerformanceScore)
	fmt.Fprintf(&b, "- Smoothness score: **%d / 100**\n", a.SmoothnessScore)
	fmt.Fprintf(&b, "- Frame Generation readiness: **%d / 100**\n", a.FGReadiness)
	fmt.Fprintf(&b, "- Confidence: **%d / 100**\n", a.Confidence)
	fmt.Fprintf(&b, "- Primary diagnosis: **%s**\n", a.Diagnosis)
	fmt.Fprintf(&b, "- Alternative: %s\n", a.Alternative)
	fmt.Fprintln(&b, "\n### Evidence")
	for _, evidence := range append(a.Evidence, a.Limitations...) {
		fmt.Fprintf(&b, "- %s\n", evidence)
	}
	fmt.Fprintln(&b, "\n## Recommendation")
	fmt.Fprintf(&b, "\n- Upscaling: %s\n", rec.Upscaling)
	fmt.Fprintf(&b, "- Frame Generation: %s (%s)\n", rec.FrameGeneration, rec.FGMultiplier)
	fmt.Fprintf(&b, "- Estimated benefit: %s\n", rec.FGBenefit)
	fmt.Fprintf(&b, "- Latency impact: %s\n", rec.LatencyImpact)
	fmt.Fprintln(&b, "\n### Suggested Actions")
	for i, action := range rec.PriorityActions {
		fmt.Fprintf(&b, "%d. %s\n", i+1, action)
	}
	return b.String()
}

func HTML(r SystemReport) string {
	m, s, a, rec := r.Machine, r.Snapshot, r.Assessment, r.Recommendation
	gpus := strings.Join(m.GPUModels, ", ")
	if gpus == "" {
		gpus = "Not detected"
	}
	return fmt.Sprintf(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width">
<title>Game Performance Advisor</title><style>
:root{--bg:#070b14;--panel:#111827;--line:#263449;--text:#edf4ff;--muted:#9aa9bc;--cyan:#66d9ff;--green:#67dda7;--yellow:#ffd166}
*{box-sizing:border-box}body{margin:0;background:radial-gradient(circle at top right,#17243a,#070b14 48%%);color:var(--text);font-family:Inter,Segoe UI,system-ui,sans-serif}.shell{max-width:1120px;margin:auto;padding:36px 24px}.top{display:flex;justify-content:space-between;align-items:end}.top h1{margin:0}.badge{color:var(--green);border:1px solid #355c50;padding:7px 10px;border-radius:999px}.grid{display:grid;grid-template-columns:repeat(4,1fr);gap:12px;margin-top:20px}.panel,.metric{background:rgba(17,24,39,.94);border:1px solid var(--line);border-radius:14px}.metric{padding:16px}.metric span{display:block;color:var(--muted);font-size:12px}.metric strong{display:block;font-size:28px;margin-top:8px}.main{display:grid;grid-template-columns:1fr 1fr;gap:12px;margin-top:12px}.panel{padding:20px}.row{display:flex;justify-content:space-between;border-top:1px solid var(--line);padding:9px 0}.row:first-of-type{border-top:0}.muted{color:var(--muted)}.cyan{color:var(--cyan)}.green{color:var(--green)}.priority{line-height:1.8}@media(max-width:800px){.grid{grid-template-columns:1fr 1fr}.main{grid-template-columns:1fr}}
</style></head><body><main class="shell">
<div class="top"><h1>GAME PERFORMANCE ADVISOR</h1><span class="badge">Live system snapshot</span></div>
<div class="grid">
<div class="metric"><span>Performance Score</span><strong>%d / 100</strong></div>
<div class="metric"><span>Smoothness Score</span><strong>%d / 100</strong></div>
<div class="metric"><span>FG Readiness</span><strong>%d / 100</strong></div>
<div class="metric"><span>Confidence</span><strong>%d / 100</strong></div></div>
<div class="main">
<section class="panel"><h2>System Summary</h2>
<div class="row"><span>CPU</span><strong>%s</strong></div><div class="row"><span>GPU</span><strong>%s</strong></div>
<div class="row"><span>RAM</span><strong>%.1f GB</strong></div><div class="row"><span>Display</span><strong>%s @ %d Hz</strong></div>
<div class="row"><span>Operating System</span><strong>%s</strong></div></section>
<section class="panel"><h2>Performance Snapshot</h2>
<div class="row"><span>CPU utilization</span><strong>%.0f%%</strong></div><div class="row"><span>GPU utilization</span><strong>%.0f%%</strong></div>
<div class="row"><span>Memory utilization</span><strong>%.0f%%</strong></div></section></div>
<div class="main"><section class="panel"><h2>Diagnosis</h2><h3 class="cyan">%s</h3>
<p class="muted">Alternative: %s</p><h3>Evidence</h3><ul>%s</ul></section>
<section class="panel"><h2>Recommendation</h2><h3 class="green">%s</h3>
<div class="row"><span>Upscaling</span><strong>%s</strong></div><div class="row"><span>Frame Generation</span><strong>%s (%s)</strong></div>
<div class="row"><span>Estimated Benefit</span><strong>%s</strong></div><div class="row"><span>Latency Impact</span><strong>%s</strong></div></section></div>
<section class="panel" style="margin-top:12px"><h2>Action Plan</h2><ol class="priority">%s</ol></section>
</main></body></html>`,
		a.PerformanceScore, a.SmoothnessScore, a.FGReadiness, a.Confidence,
		html.EscapeString(m.CPUModel), html.EscapeString(gpus), m.RAMGB,
		html.EscapeString(m.Resolution), m.RefreshHz, html.EscapeString(m.OS),
		s.CPUPercent, s.GPUPercent, s.RAMPercent,
		html.EscapeString(a.Diagnosis), html.EscapeString(a.Alternative), evidenceHTML(a.Evidence),
		html.EscapeString(rec.FrameGeneration), html.EscapeString(rec.Upscaling),
		html.EscapeString(rec.FrameGeneration), html.EscapeString(rec.FGMultiplier),
		html.EscapeString(rec.FGBenefit), html.EscapeString(rec.LatencyImpact),
		actionsHTML(rec.PriorityActions))
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
