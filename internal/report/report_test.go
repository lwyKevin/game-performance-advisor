package report

import (
	"strings"
	"testing"

	"github.com/example/game-performance-advisor/internal/analysis"
	"github.com/example/game-performance-advisor/internal/metrics"
	"github.com/example/game-performance-advisor/internal/recommend"
	"github.com/example/game-performance-advisor/internal/system"
)

func testReport() SystemReport {
	machine := system.Info{
		CPUModel: "Test CPU", GPUModels: []string{"Test GPU"}, RAMGB: 32,
		Resolution: "2560x1440", RefreshHz: 165, OS: "Test OS",
	}
	snapshot := metrics.Snapshot{CPUPercent: 48, GPUPercent: 96, GPUAvailable: true, RAMPercent: 55}
	assessment := analysis.AnalyzeSystem(machine, snapshot)
	return SystemReport{
		Machine: machine, Snapshot: snapshot,
		Assessment: assessment, Recommendation: recommend.RecommendSystem(machine, assessment),
	}
}

func TestTerminalReportIncludesDiagnosis(t *testing.T) {
	output := Terminal(testReport())
	if !strings.Contains(output, "Primary Diagnosis") || !strings.Contains(output, "GPU Limited") {
		t.Fatalf("terminal report is missing diagnosis:\n%s", output)
	}
}

func TestMarkdownReportIncludesScores(t *testing.T) {
	output := Markdown(testReport())
	if !strings.Contains(output, "Performance score") ||
		!strings.Contains(output, "Frame Generation readiness") {
		t.Fatalf("Markdown report is missing scores:\n%s", output)
	}
}

func TestHTMLReportIsPortfolioStyled(t *testing.T) {
	output := HTML(testReport())
	if !strings.Contains(output, "<!doctype html") ||
		!strings.Contains(output, "GAME PERFORMANCE ADVISOR") {
		t.Fatal("HTML report is missing expected structure")
	}

}
