package analysis

import (
	"encoding/json"
	"math"
	"strings"
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
	if result.OnePercentLowAvailable {
		t.Fatal("1% low should be unavailable for 10 frames")
	}
	if result.OnePercentLowFPS != nil {
		t.Fatalf("1%% low FPS = %.3f, want null when unavailable", *result.OnePercentLowFPS)
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
	if result.OnePercentLowFPS == nil || math.Abs(*result.OnePercentLowFPS-39.995) > 0.01 {
		t.Fatalf("1%% low FPS = %v, want about 39.995", result.OnePercentLowFPS)
	}
	if result.PointOnePercentLowFPS == nil || math.Abs(*result.PointOnePercentLowFPS-10) > 0.001 {
		t.Fatalf("0.1%% low FPS = %v, want 10", result.PointOnePercentLowFPS)
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

func TestAnalyzeRejectsNoMeaningfulSmallSampleLowPercentiles(t *testing.T) {
	values := make([]float64, 200)
	for i := range values {
		values[i] = 16.67
	}
	result, err := Analyze(values, 60, 1.8)
	if err != nil {
		t.Fatal(err)
	}
	if !result.OnePercentLowAvailable {
		t.Fatal("1% low should be available for 200 frames")
	}
	if result.PointOnePercentLowAvailable {
		t.Fatal("0.1% low should be unavailable for 200 frames")
	}
	if result.PointOnePercentLowFPS != nil {
		t.Fatalf("0.1%% low FPS = %.3f, want null", *result.PointOnePercentLowFPS)
	}
	if result.PointOnePercentLowReason == "" {
		t.Fatal("0.1% low reason must be provided")
	}
}

func TestAnalyzeNinetyNineFramesLowReasons(t *testing.T) {
	values := make([]float64, 99)
	for i := range values {
		values[i] = 16.67
	}
	result, err := Analyze(values, 60, 1.8)
	if err != nil {
		t.Fatal(err)
	}
	if result.OnePercentLowAvailable {
		t.Fatal("1% low should be unavailable below 100 frames")
	}
	if result.OnePercentLowFPS != nil {
		t.Fatal("1% low FPS must be null below 100 frames")
	}
	if result.OnePercentLowReason == "" {
		t.Fatal("1% low reason must be provided")
	}
}

func TestAnalyzeWarnsAboutBimodalDistribution(t *testing.T) {
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
	foundWarning := false
	for _, evidence := range result.Evidence {
		if strings.Contains(evidence, "bimodal") {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Fatalf("expected bimodal limitation evidence, got %v", result.Evidence)
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

func TestPercentileLowUsesFrameTimesNotInstantaneousFPS(t *testing.T) {
	values := make([]float64, 1000)
	for i := range values {
		values[i] = 16.67
	}
	for i := 990; i < 999; i++ {
		values[i] = 100
	}
	values[999] = 200
	result, err := Analyze(values, 60, 1.8)
	if err != nil {
		t.Fatal(err)
	}
	if result.OnePercentLowFPS == nil || math.Abs(*result.OnePercentLowFPS-9.0909) > 0.02 {
		t.Fatalf("1%% low = %v FPS, want 9.091 FPS from percentile frame times", result.OnePercentLowFPS)
	}
	// The naive implementation averages the lowest 10 instantaneous FPS values:
	// nine 10 FPS values and one 5 FPS value = 9.5 FPS.
	// It must not be used because lows are percentiles of frame times.
	naiveFPSAverage := (9*10.0 + 5.0) / 10.0
	if result.OnePercentLowFPS != nil && math.Abs(*result.OnePercentLowFPS-naiveFPSAverage) < 0.1 {
		t.Fatal("test did not distinguish frametime percentile from instantaneous-FPS percentile")
	}
}

func TestAnalyzeEveryTenthFrameBimodal(t *testing.T) {
	values := make([]float64, 100)
	for i := range values {
		values[i] = 16.67
		if i%10 == 9 {
			values[i] = 50.01
		}
	}
	result, err := Analyze(values, 60, 1.8)
	if err != nil {
		t.Fatal(err)
	}
	if result.StutterCount != 10 {
		t.Fatalf("stutter count = %d, want 10", result.StutterCount)
	}
}

func TestAnalyzeMonotonicRamp(t *testing.T) {
	values := make([]float64, 100)
	for i := range values {
		values[i] = 10 + float64(i)/10
	}
	result, err := Analyze(values, 60, 1.8)
	if err != nil {
		t.Fatal(err)
	}
	if result.FrameCount != 100 || result.MedianMS <= 0 {
		t.Fatalf("unexpected ramp result: %+v", result)
	}
}

func TestUnavailableLowFPSJSONIsNull(t *testing.T) {
	values := make([]float64, 300)
	for i := range values {
		values[i] = 16.67
	}
	result, err := Analyze(values, 60, 1.8)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	if !strings.Contains(text, `"0.1_percent_low_fps":null`) {
		t.Fatalf("JSON did not encode unavailable 0.1%% low as null: %s", text)
	}
	if !strings.Contains(text, `"0.1_percent_low_available":false`) {
		t.Fatalf("JSON missing unavailable 0.1%% low flag: %s", text)
	}
	if result.PointOnePercentLowReason == "" {
		t.Fatal("0.1% low reason must be provided")
	}
}
