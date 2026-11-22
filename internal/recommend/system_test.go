package recommend

import (
	"testing"

	"github.com/example/game-performance-advisor/internal/analysis"
	"github.com/example/game-performance-advisor/internal/system"
)

func TestSystemRecommendationGPULimited(t *testing.T) {
	result := RecommendSystem(structSystemInfo(165), analysis.SystemAssessment{Diagnosis: "GPU Limited"})
	if result.FrameGeneration != "Likely useful" || result.FGMultiplier != "2x" {
		t.Fatalf("unexpected recommendation: %+v", result)
	}
}

func TestSystemRecommendationCPULimited(t *testing.T) {
	result := RecommendSystem(structSystemInfo(144), analysis.SystemAssessment{Diagnosis: "CPU Limited"})
	if result.FrameGeneration != "Not recommended yet" {
		t.Fatalf("unexpected recommendation: %+v", result)
	}
}

func TestSystemRecommendationInsufficientEvidence(t *testing.T) {
	result := RecommendSystem(structSystemInfo(165), analysis.SystemAssessment{
		Diagnosis: "Insufficient evidence for bottleneck diagnosis",
	})
	if result.FrameGeneration != "Defer: insufficient bottleneck evidence" {
		t.Fatalf("frame generation = %q", result.FrameGeneration)
	}
	if result.FGMultiplier != "None" {
		t.Fatalf("FG multiplier = %q, want None", result.FGMultiplier)
	}
}

func structSystemInfo(refresh int) system.Info {
	return system.Info{RefreshHz: refresh}
}
