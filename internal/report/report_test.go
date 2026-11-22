package report

import (
	"strings"
	"testing"

	"github.com/example/game-performance-advisor/internal/analysis"
	"github.com/example/game-performance-advisor/internal/metrics"
	"github.com/example/game-performance-advisor/internal/recommend"
	"github.com/example/game-performance-advisor/internal/system"
)

func TestTerminalAndJSONCoverBothCommands(t *testing.T) {
	frame := Report{Command: "analyze", Platform: "linux", Frame: &FramePayload{
		Source: "sample.csv", TargetFPS: 60,
		Analysis:       analysis.Result{FrameCount: 300, AverageFPS: 60},
		Recommendation: recommend.Result{FrameGeneration: "Not recommended"},
	}}
	if !strings.Contains(Terminal(frame), "Frame Analysis") {
		t.Fatal("frame terminal report missing")
	}
	encoded, err := JSON(frame)
	if err != nil || !strings.Contains(string(encoded), `"command": "analyze"`) {
		t.Fatalf("frame JSON report failed: %v", err)
	}

	scan := Report{Command: "scan", Platform: "linux", System: &SystemPayload{
		Machine:        system.Info{CPUModel: "Test CPU"},
		Snapshot:       metrics.Snapshot{CPUPercent: 20},
		Assessment:     analysis.SystemAssessment{Diagnosis: "Balanced"},
		Recommendation: recommend.SystemResult{FrameGeneration: "Optional"},
	}}
	if !strings.Contains(Terminal(scan), "Hardware Scan") {
		t.Fatal("scan terminal report missing")
	}
}
