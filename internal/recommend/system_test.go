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

func structSystemInfo(refresh int) system.Info {
	return system.Info{RefreshHz: refresh}
}
