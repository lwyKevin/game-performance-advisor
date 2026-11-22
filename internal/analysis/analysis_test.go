package analysis

import (
	"math"
	"testing"
)

func TestAnalyzeSmoothData(t *testing.T) {
	values := make([]float64, 100)
	for i := range values {
		values[i] = 16.67
	}
	result, err := Analyze(values, 60, 1.8)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(result.AverageMS-16.67) > 0.001 {
		t.Fatalf("average = %.3f", result.AverageMS)
	}
	if math.Abs(result.CoefficientVar) > 1e-12 {
		t.Fatalf("CV = %.3f, want 0", result.CoefficientVar)
	}
	if result.StutterPercent != 0 {
		t.Fatalf("stutter percent = %.3f", result.StutterPercent)
	}
	if result.HealthScore < 90 {
		t.Fatalf("health score = %d, want at least 90", result.HealthScore)
	}
	if result.Confidence < 80 {
		t.Fatalf("diagnosis confidence = %d, want at least 80", result.Confidence)
	}
}

func TestAnalyzeDetectsOneHitch(t *testing.T) {
	values := []float64{16, 16, 16, 16, 16, 16, 16, 16, 16, 50}
	result, err := Analyze(values, 60, 1.8)
	if err != nil {
		t.Fatal(err)
	}
	if result.MedianMS != 16 {
		t.Fatalf("median = %.2f", result.MedianMS)
	}
	if result.StutterCount != 1 {
		t.Fatalf("stutter count = %d, want 1", result.StutterCount)
	}
	if math.Abs(result.OnePercentLowFPS-20) > 0.001 {
		t.Fatalf("1%% low FPS = %.3f, want 20", result.OnePercentLowFPS)
	}
}

func TestAnalyzeKnownLowPercentiles(t *testing.T) {
	values := make([]float64, 1000)
	for i := range values {
		values[i] = 16.67
	}
	values[len(values)-1] = 100

	result, err := Analyze(values, 60, 1.8)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(result.OnePercentLowFPS-39.995) > 0.01 {
		t.Fatalf("1%% low FPS = %.3f, want about 39.995", result.OnePercentLowFPS)
	}
	if math.Abs(result.PointOnePercentLowFPS-10) > 0.001 {
		t.Fatalf("0.1%% low FPS = %.3f, want 10", result.PointOnePercentLowFPS)
	}
}

func TestAnalyzeKnownFramePacing(t *testing.T) {
	values := make([]float64, 100)
	for i := range values {
		if i%2 == 0 {
			values[i] = 16.67
		} else {
			values[i] = 33.34
		}
	}

	result, err := Analyze(values, 60, 1.8)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(result.AverageMS-25.005) > 0.001 {
		t.Fatalf("average ms = %.3f, want 25.005", result.AverageMS)
	}
	if math.Abs(result.CoefficientVar-33.333) > 0.01 {
		t.Fatalf("coefficient of variation = %.3f, want about 33.333", result.CoefficientVar)
	}
	if result.StutterCount != 0 {
		t.Fatalf("stutter count = %d, want 0", result.StutterCount)
	}
}

func TestAnalyzeRejectsInvalidData(t *testing.T) {
	if _, err := Analyze([]float64{16, 0}, 60, 1.8); err == nil {
		t.Fatal("expected invalid frame time error")
	}
	if _, err := Analyze([]float64{16}, 60, 1); err == nil {
		t.Fatal("expected invalid threshold error")
	}
}

func TestAnalyzeClassifiesOccasionalHitching(t *testing.T) {
	values := make([]float64, 100)
	for i := range values {
		values[i] = 16.67
	}
	values[50] = 50
	values[75] = 50
	result, err := Analyze(values, 60, 1.8)
	if err != nil {
		t.Fatal(err)
	}
	if result.Bottleneck != "Stutter / hitching" {
		t.Fatalf("bottleneck = %q, want Stutter / hitching", result.Bottleneck)
	}
}
