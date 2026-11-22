package recommend

import (
	"testing"

	"github.com/example/game-performance-advisor/internal/analysis"
)

func fpsPointer(value float64) *float64 {
	return &value
}

func TestSmoothRecommendation(t *testing.T) {
	r := Recommend(analysis.Result{
		AverageFPS:       60,
		OnePercentLowFPS: fpsPointer(58),
		CoefficientVar:   3,
		StutterPercent:   0,
		Bottleneck:       "Smooth / no clear bottleneck",
	}, 60)
	if !r.SmoothEnough {
		t.Fatal("expected smooth experience")
	}
	if r.FGMultiplier != "None" {
		t.Fatalf("FG multiplier = %s", r.FGMultiplier)
	}
	if r.FrameGeneration != "Not recommended" {
		t.Fatalf("frame generation = %s", r.FrameGeneration)
	}
}

func TestStable60FPSAtTargetNotRecommended(t *testing.T) {
	r := Recommend(analysis.Result{
		AverageFPS:       60,
		OnePercentLowFPS: fpsPointer(49.8),
		CoefficientVar:   2,
		StutterPercent:   0,
		Bottleneck:       "Likely GPU-bound",
	}, 60)
	if r.FrameGeneration != "Not recommended" {
		t.Fatalf("frame generation = %s, want Not recommended", r.FrameGeneration)
	}
	if r.FGMultiplier != "None" {
		t.Fatalf("FG multiplier = %s, want None", r.FGMultiplier)
	}
}

func TestGPUBoundRecommendation(t *testing.T) {
	r := Recommend(analysis.Result{
		AverageFPS:       40,
		OnePercentLowFPS: fpsPointer(35),
		CoefficientVar:   8,
		StutterPercent:   1,
		Bottleneck:       "Likely GPU-bound",
	}, 60)
	if r.FrameGeneration != "Likely helpful" {
		t.Fatalf("frame generation = %s", r.FrameGeneration)
	}
	if r.FGMultiplier != "3x" {
		t.Fatalf("FG multiplier = %s, want 3x", r.FGMultiplier)
	}
}

func TestSpikeHeavyRecommendation(t *testing.T) {
	r := Recommend(analysis.Result{
		AverageFPS:     55,
		CoefficientVar: 25,
		StutterPercent: 12,
		Bottleneck:     "Likely CPU-bound or mixed with frequent spikes",
	}, 60)
	if r.FrameGeneration != "Not recommended yet" {
		t.Fatalf("frame generation = %s", r.FrameGeneration)
	}
}

func TestHitchingRecommendation(t *testing.T) {
	r := Recommend(analysis.Result{
		AverageFPS:     47,
		CoefficientVar: 88,
		StutterPercent: 4,
		Bottleneck:     "Stutter / hitching",
	}, 60)
	if len(r.PriorityActions) == 0 {
		t.Fatal("expected priority actions")
	}
	if r.PriorityActions[0] != "Investigate shader loading, streaming, drivers, or isolated background hitches first" {
		t.Fatalf("unexpected first action: %s", r.PriorityActions[0])
	}
}
