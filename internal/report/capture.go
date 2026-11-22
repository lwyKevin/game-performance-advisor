package report

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"

	"github.com/example/game-performance-advisor/internal/analysis"
	"github.com/example/game-performance-advisor/internal/capture"
	"github.com/example/game-performance-advisor/internal/recommend"
)

type CaptureReport struct {
	Capture        capture.Result   `json:"capture"`
	Analysis       analysis.Result  `json:"analysis"`
	Recommendation recommend.Result `json:"recommendation"`
	TargetFPS      float64          `json:"target_fps"`
}

type ComparisonReport struct {
	Before CaptureReport `json:"before"`
	After  CaptureReport `json:"after"`
}

func CaptureJSON(r CaptureReport) ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

func CaptureTerminal(r CaptureReport) string {
	a, rec := r.Analysis, r.Recommendation
	var b strings.Builder
	fmt.Fprintln(&b, "┌────────────────────────────────────────────┐")
	fmt.Fprintln(&b, "│ GAME PERFORMANCE ADVISOR - FRAME CAPTURE   │")
	fmt.Fprintln(&b, "└────────────────────────────────────────────┘")
	fmt.Fprintf(&b, "\nCapture\n  Game                 %s\n", r.Capture.GameName)
	fmt.Fprintf(&b, "  Provider              %s\n", r.Capture.Provider)
	fmt.Fprintf(&b, "  Frames                %d\n", a.FrameCount)
	fmt.Fprintf(&b, "  Duration              %d sec\n", r.Capture.DurationSeconds)
	fmt.Fprintf(&b, "  Captured CSV          %s\n", r.Capture.OutputCSV)
	fmt.Fprintln(&b, "\nFrame Performance")
	fmt.Fprintf(&b, "  Average FPS           %.1f\n", a.AverageFPS)
	if a.OnePercentLowAvailable {
		fmt.Fprintf(&b, "  1%% Low FPS            %.1f\n", a.OnePercentLowFPS)
	} else {
		fmt.Fprintln(&b, "  1% Low FPS            Not available (<100 frames)")
	}
	if a.PointOnePercentLowAvailable {
		fmt.Fprintf(&b, "  0.1%% Low FPS          %.1f\n", a.PointOnePercentLowFPS)
	} else {
		fmt.Fprintln(&b, "  0.1% Low FPS          Not available (<1000 frames)")
	}
	fmt.Fprintf(&b, "  Average frametime     %.2f ms\n", a.AverageMS)
	fmt.Fprintf(&b, "  Stutters              %.1f%% (%d frames)\n", a.StutterPercent, a.StutterCount)
	fmt.Fprintln(&b, "\nFrame Health")
	fmt.Fprintf(&b, "  Overall               %d / 100\n", a.HealthScore)
	fmt.Fprintf(&b, "  Smoothness            %d / 100\n", a.SmoothnessScore)
	fmt.Fprintf(&b, "  Consistency           %d / 100\n", a.ConsistencyScore)
	fmt.Fprintf(&b, "  Stutter resistance    %d / 100\n", a.StutterResistanceScore)
	fmt.Fprintln(&b, "\nDiagnosis")
	fmt.Fprintf(&b, "  Primary               %s\n", a.Bottleneck)
	fmt.Fprintf(&b, "  Confidence            %d%%\n", a.Confidence)
	fmt.Fprintf(&b, "  Alternative           %s\n", a.Alternative)
	fmt.Fprintln(&b, "\nEvidence")
	for _, evidence := range a.Evidence {
		fmt.Fprintf(&b, "  • %s\n", evidence)
	}
	fmt.Fprintln(&b, "\nRecommendation")
	fmt.Fprintf(&b, "  Should enable upscaling?  %s\n", rec.Upscaling)
	fmt.Fprintf(&b, "  Frame Generation          %s", rec.FrameGeneration)
	if rec.FGMultiplier != "None" {
		fmt.Fprintf(&b, " (%s)", rec.FGMultiplier)
	}
	fmt.Fprintf(&b, "\n  FG suitability            %d / 100\n", rec.FGSuitability)
	fmt.Fprintf(&b, "  Estimated latency         %s\n", rec.LatencyRangeMS)
	fmt.Fprintf(&b, "  Reason                    %s\n", rec.FGReason)
	if len(rec.Evidence) > 0 {
		fmt.Fprintln(&b, "\nEvidence")
		for _, evidence := range rec.Evidence {
			fmt.Fprintf(&b, "  • %s\n", evidence)
		}
	}
	fmt.Fprintln(&b, "\nSuggested Actions")
	for i, action := range rec.PriorityActions {
		fmt.Fprintf(&b, "  %d. %s\n", i+1, action)
	}
	return b.String()
}

func CaptureMarkdown(r CaptureReport) string {
	a, rec := r.Analysis, r.Recommendation
	var b strings.Builder
	fmt.Fprintf(&b, "# %s - Frame Capture Report\n\n", escapeMarkdown(r.Capture.GameName))
	fmt.Fprintf(&b, "## Capture\n\n- Provider: %s\n- Frames: %d\n- Duration: %d seconds\n- CSV: `%s`\n\n",
		r.Capture.Provider, a.FrameCount, r.Capture.DurationSeconds, r.Capture.OutputCSV)
	fmt.Fprintln(&b, "## Frame Performance")
	fmt.Fprintf(&b, "\n| Metric | Value |\n|---|---:|\n")
	fmt.Fprintf(&b, "| Average FPS | %.1f |\n", a.AverageFPS)
	if a.OnePercentLowAvailable {
		fmt.Fprintf(&b, "| 1%% Low FPS | %.1f |\n", a.OnePercentLowFPS)
	} else {
		fmt.Fprintln(&b, "| 1% Low FPS | Not available (<100 frames) |")
	}
	if a.PointOnePercentLowAvailable {
		fmt.Fprintf(&b, "| 0.1%% Low FPS | %.1f |\n", a.PointOnePercentLowFPS)
	} else {
		fmt.Fprintln(&b, "| 0.1% Low FPS | Not available (<1000 frames) |")
	}
	fmt.Fprintf(&b, "| Average frametime | %.2f ms |\n| Stutters | %.1f%% (%d frames) |\n",
		a.AverageMS, a.StutterPercent, a.StutterCount)
	fmt.Fprintf(&b, "\n## Frame Health\n\n- Overall: **%d / 100**\n- Smoothness: %d / 100\n- Consistency: %d / 100\n- Stutter resistance: %d / 100\n",
		a.HealthScore, a.SmoothnessScore, a.ConsistencyScore, a.StutterResistanceScore)
	fmt.Fprintf(&b, "\n## Diagnosis\n\n- Primary: **%s** (%d%% confidence)\n- Alternative: %s\n\n### Evidence\n",
		a.Bottleneck, a.Confidence, a.Alternative)
	for _, evidence := range a.Evidence {
		fmt.Fprintf(&b, "\n- %s", evidence)
	}
	fmt.Fprintf(&b, "\n\n## Recommendation\n\n- Upscaling: %s\n- Frame Generation: %s (%s)\n- FG suitability: %d / 100\n- Estimated latency: %s\n- Reason: %s\n\n### Suggested Actions\n",
		rec.Upscaling, rec.FrameGeneration, rec.FGMultiplier, rec.FGSuitability, rec.LatencyRangeMS, rec.FGReason)
	for i, action := range rec.PriorityActions {
		fmt.Fprintf(&b, "\n%d. %s", i+1, action)
	}
	return b.String()
}

func lowFPSMarkdown(a analysis.Result, pointOne bool) string {
	if pointOne {
		if !a.PointOnePercentLowAvailable {
			return "N/A"
		}
		return fmt.Sprintf("%.1f", a.PointOnePercentLowFPS)
	}
	if !a.OnePercentLowAvailable {
		return "N/A"
	}
	return fmt.Sprintf("%.1f", a.OnePercentLowFPS)
}

func lowFPSChangeMarkdown(before, after analysis.Result, pointOne bool) string {
	if pointOne {
		if !before.PointOnePercentLowAvailable || !after.PointOnePercentLowAvailable {
			return "N/A"
		}
		return fmt.Sprintf("%+.1f", after.PointOnePercentLowFPS-before.PointOnePercentLowFPS)
	}
	if !before.OnePercentLowAvailable || !after.OnePercentLowAvailable {
		return "N/A"
	}
	return fmt.Sprintf("%+.1f", after.OnePercentLowFPS-before.OnePercentLowFPS)
}

func ComparisonMarkdown(before, after CaptureReport) string {
	beforeA, afterA := before.Analysis, after.Analysis
	var b strings.Builder
	fmt.Fprintln(&b, "# Before / After Performance Comparison")
	fmt.Fprintf(&b, "\n| Metric | Before | After | Change |\n|---|---:|---:|---:|\n")
	fmt.Fprintf(&b, "| Average FPS | %.1f | %.1f | %+.1f |\n", beforeA.AverageFPS, afterA.AverageFPS, afterA.AverageFPS-beforeA.AverageFPS)
	fmt.Fprintf(&b, "| 1%% Low FPS | %s | %s | %s |\n",
		lowFPSMarkdown(beforeA, false), lowFPSMarkdown(afterA, false),
		lowFPSChangeMarkdown(beforeA, afterA, false))
	fmt.Fprintf(&b, "| 0.1%% Low FPS | %s | %s | %s |\n",
		lowFPSMarkdown(beforeA, true), lowFPSMarkdown(afterA, true),
		lowFPSChangeMarkdown(beforeA, afterA, true))
	fmt.Fprintf(&b, "| Frame Health | %d | %d | %+d |\n", beforeA.HealthScore, afterA.HealthScore, afterA.HealthScore-beforeA.HealthScore)
	fmt.Fprintf(&b, "| Stutters | %.1f%% | %.1f%% | %+.1f pp |\n", beforeA.StutterPercent, afterA.StutterPercent, afterA.StutterPercent-beforeA.StutterPercent)
	fmt.Fprintf(&b, "\n## Before\n\n- Diagnosis: %s\n- Recommendation: %s (%s)\n",
		beforeA.Bottleneck, before.Recommendation.FrameGeneration, before.Recommendation.FGMultiplier)
	fmt.Fprintf(&b, "\n## After\n\n- Diagnosis: %s\n- Recommendation: %s (%s)\n",
		afterA.Bottleneck, after.Recommendation.FrameGeneration, after.Recommendation.FGMultiplier)
	return b.String()
}

func ComparisonHTML(before, after CaptureReport) string {
	beforeA, afterA := before.Analysis, after.Analysis
	return fmt.Sprintf(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width">
<title>Before / After Performance Comparison</title><style>
:root{--bg:#070b14;--panel:#111827;--line:#263449;--text:#edf4ff;--muted:#9aa9bc;--cyan:#66d9ff;--green:#67dda7}*{box-sizing:border-box}body{margin:0;background:radial-gradient(circle at top right,#17243a,#070b14 48%%);color:var(--text);font-family:Inter,Segoe UI,system-ui,sans-serif}.shell{max-width:900px;margin:auto;padding:36px 24px}.grid{display:grid;grid-template-columns:1fr 1fr 1fr;gap:12px}.panel{background:var(--panel);border:1px solid var(--line);border-radius:14px;padding:16px}.panel span{display:block;color:var(--muted)}.panel strong{display:block;font-size:26px;margin-top:8px}.delta{color:var(--green)}table{width:100%%;border-collapse:collapse;margin-top:18px}th,td{text-align:right;border-bottom:1px solid var(--line);padding:12px}th:first-child,td:first-child{text-align:left}
</style></head><body><main class="shell"><h1>Before / After Performance Comparison</h1>
<div class="grid"><div class="panel"><span>Average FPS</span><strong>%.1f</strong></div><div class="panel"><span>After</span><strong>%.1f</strong></div><div class="panel"><span>Change</span><strong class="delta">%+.1f</strong></div></div>
<table><thead><tr><th>Metric</th><th>Before</th><th>After</th></tr></thead><tbody>
<tr><td>1%% Low FPS</td><td>%s</td><td>%s</td></tr>
<tr><td>0.1%% Low FPS</td><td>%s</td><td>%s</td></tr>
<tr><td>Frame Health</td><td>%d</td><td>%d</td></tr>
<tr><td>Stutters</td><td>%.1f%%</td><td>%.1f%%</td></tr>
</tbody></table></main></body></html>`,
		beforeA.AverageFPS, afterA.AverageFPS, afterA.AverageFPS-beforeA.AverageFPS,
		lowFPSMarkdown(beforeA, false), lowFPSMarkdown(afterA, false),
		lowFPSMarkdown(beforeA, true), lowFPSMarkdown(afterA, true),
		beforeA.HealthScore, afterA.HealthScore,
		beforeA.StutterPercent, afterA.StutterPercent)
}

func CaptureHTML(r CaptureReport) string {
	a, rec := r.Analysis, r.Recommendation
	return fmt.Sprintf(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width">
<title>%s - Frame Capture</title><style>
:root{--bg:#070b14;--panel:#111827;--line:#263449;--text:#edf4ff;--muted:#9aa9bc;--cyan:#66d9ff;--green:#67dda7}*{box-sizing:border-box}body{margin:0;background:radial-gradient(circle at top right,#17243a,#070b14 48%%);color:var(--text);font-family:Inter,Segoe UI,system-ui,sans-serif}.shell{max-width:1120px;margin:auto;padding:36px 24px}.grid{display:grid;grid-template-columns:repeat(4,1fr);gap:12px}.main{display:grid;grid-template-columns:1.4fr 1fr;gap:12px;margin-top:12px}.panel,.metric{background:var(--panel);border:1px solid var(--line);border-radius:14px}.metric{padding:16px}.metric span{display:block;color:var(--muted);font-size:12px}.metric strong{display:block;font-size:26px;margin-top:8px}.panel{padding:20px}.row{display:flex;justify-content:space-between;border-top:1px solid var(--line);padding:9px 0}.cyan{color:var(--cyan)}.green{color:var(--green)}.chart{width:100%%;height:260px}.priority{line-height:1.8}@media(max-width:800px){.grid{grid-template-columns:1fr 1fr}.main{grid-template-columns:1fr}}
</style></head><body><main class="shell"><h1>%s</h1>
<div class="grid"><div class="metric"><span>Average FPS</span><strong>%.1f</strong></div><div class="metric"><span>1%% Low FPS</span><strong>%.1f</strong></div><div class="metric"><span>Frame Health</span><strong>%d / 100</strong></div><div class="metric"><span>Stutters</span><strong>%.1f%%</strong></div></div>
<div class="main"><section class="panel"><h2>Frame Time Chart</h2>%s</section><section class="panel"><h2>Diagnosis</h2><h3 class="cyan">%s</h3><p>Confidence: %d%%</p><p>Alternative: %s</p><ul>%s</ul></section></div>
<div class="main"><section class="panel"><h2>Frame Health</h2><div class="row"><span>Smoothness</span><strong>%d</strong></div><div class="row"><span>Consistency</span><strong>%d</strong></div><div class="row"><span>Stutter resistance</span><strong>%d</strong></div></section>
<section class="panel"><h2>Recommendation</h2><h3 class="green">%s</h3><p>%s</p><div class="row"><span>Upscaling</span><strong>%s</strong></div><div class="row"><span>FG suitability</span><strong>%d / 100</strong></div><div class="row"><span>Latency</span><strong>%s</strong></div></section></div>
<section class="panel" style="margin-top:12px"><h2>Suggested Actions</h2><ol class="priority">%s</ol></section></main></body></html>`,
		html.EscapeString(r.Capture.GameName), html.EscapeString(r.Capture.GameName),
		a.AverageFPS, a.OnePercentLowFPS, a.HealthScore, a.StutterPercent,
		captureChart(r.Capture.FrameTimesMS),
		html.EscapeString(a.Bottleneck), a.Confidence, html.EscapeString(a.Alternative), captureEvidenceHTML(a.Evidence),
		a.SmoothnessScore, a.ConsistencyScore, a.StutterResistanceScore,
		html.EscapeString(rec.FrameGeneration), html.EscapeString(rec.FGReason), html.EscapeString(rec.Upscaling),
		rec.FGSuitability, html.EscapeString(rec.LatencyRangeMS), captureActionsHTML(rec.PriorityActions))
}

func captureChart(values []float64) string {
	if len(values) == 0 {
		return `<p>No frame samples.</p>`
	}
	step := len(values)/300 + 1
	sampled := make([]float64, 0)
	for i := 0; i < len(values); i += step {
		sampled = append(sampled, values[i])
	}
	maxMS := 20.0
	for _, v := range sampled {
		if v > maxMS {
			maxMS = v
		}
	}
	maxMS += 5
	points := make([]string, 0, len(sampled))
	for i, v := range sampled {
		x := 20.0
		if len(sampled) > 1 {
			x = 20 + float64(i)/float64(len(sampled)-1)*760
		}
		y := 220 - (v/maxMS)*200
		points = append(points, fmt.Sprintf("%.1f,%.1f", x, y))
	}
	return fmt.Sprintf(`<svg class="chart" viewBox="0 0 800 260"><line x1="20" y1="230" x2="780" y2="230" stroke="#263449"/><polyline points="%s" fill="none" stroke="#66d9ff" stroke-width="2"/></svg>`, strings.Join(points, " "))
}

func escapeMarkdown(value string) string {
	return strings.ReplaceAll(value, "|", "\\|")
}

func captureEvidenceHTML(values []string) string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		items = append(items, "<li>"+html.EscapeString(value)+"</li>")
	}
	return strings.Join(items, "")
}

func captureActionsHTML(values []string) string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		items = append(items, "<li>"+html.EscapeString(value)+"</li>")
	}
	return strings.Join(items, "")
}
