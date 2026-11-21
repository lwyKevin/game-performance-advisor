package capture

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Result struct {
	GameName        string    `json:"game_name"`
	ProcessName     string    `json:"process_name"`
	ProcessID       int       `json:"process_id"`
	Provider        string    `json:"provider"`
	ProviderBinary  string    `json:"provider_binary"`
	OutputCSV       string    `json:"output_csv"`
	FrameTimesMS    []float64 `json:"frame_times_ms"`
	CapturedAt      time.Time `json:"captured_at"`
	DurationSeconds int       `json:"duration_seconds"`
	FrameCount      int       `json:"frame_count"`
}

type Provider interface {
	Name() string
	Capture(processName string, processID int, durationSeconds int, outputCSV string) (Result, error)
}

// DetectPresentMon searches PATH and common Windows install locations.
func DetectPresentMon(explicit string) (string, error) {
	if explicit != "" {
		path, err := exec.LookPath(explicit)
		if err == nil {
			return path, nil
		}
		if _, err := os.Stat(explicit); err == nil {
			return explicit, nil
		}
		return "", fmt.Errorf("PresentMon binary was not found at %q", explicit)
	}
	for _, name := range []string{"PresentMon.exe", "PresentMon"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	if runtime.GOOS != "windows" {
		return "", fmt.Errorf("PresentMon support is available on Windows")
	}
	roots := []string{
		os.Getenv("ProgramFiles"),
		os.Getenv("ProgramFiles(x86)"),
		os.Getenv("LOCALAPPDATA"),
	}
	for _, root := range roots {
		if root == "" {
			continue
		}
		candidates := []string{
			filepath.Join(root, "Intel", "PresentMon", "PresentMon.exe"),
			filepath.Join(root, "PresentMon", "PresentMon.exe"),
		}
		for _, candidate := range candidates {
			if _, err := os.Stat(candidate); err == nil {
				return candidate, nil
			}
		}
	}
	return "", fmt.Errorf("PresentMon.exe was not found. Install Intel PresentMon and add it to PATH")
}

type PresentMon struct {
	Binary string
}

// CaptureUntilStopped starts PresentMon without a fixed timer and cancels it
// when the stop channel is closed.
func (p PresentMon) CaptureUntilStopped(processName string, processID int, outputCSV string, stop <-chan struct{}) (Result, error) {
	if processName == "" && processID <= 0 {
		return Result{}, fmt.Errorf("a process name or process ID is required")
	}
	args := []string{"-output_file", outputCSV, "-no_top"}
	if processID > 0 {
		args = append(args, "-process_id", strconv.Itoa(processID))
	} else {
		args = append(args, "-process_name", processName)
	}
	command := exec.Command(p.Binary, args...)
	command.Env = append(os.Environ(), "PYTHONUTF8=1")
	if err := command.Start(); err != nil {
		return Result{}, fmt.Errorf("start PresentMon capture: %w", err)
	}
	<-stop
	if err := command.Process.Kill(); err != nil {
		return Result{}, fmt.Errorf("stop PresentMon capture: %w", err)
	}
	_, _ = command.Process.Wait()

	file, err := os.Open(outputCSV)
	if err != nil {
		return Result{}, fmt.Errorf("open PresentMon CSV %q: %w", outputCSV, err)
	}
	defer file.Close()
	values, err := ParsePresentMon(file)
	if err != nil {
		return Result{}, err
	}
	if len(values) == 0 {
		return Result{}, fmt.Errorf("PresentMon produced no frame-time samples")
	}
	normalized, err := writeNormalizedCSV(outputCSV, values)
	if err != nil {
		return Result{}, err
	}
	return Result{
		GameName: processName, ProcessName: processName, ProcessID: processID,
		Provider: p.Name(), ProviderBinary: p.Binary, OutputCSV: normalized,
		FrameTimesMS: values, CapturedAt: time.Now(),
		FrameCount: len(values),
	}, nil
}

func (p PresentMon) Name() string {
	return "Intel PresentMon"
}

func (p PresentMon) Capture(processName string, processID int, durationSeconds int, outputCSV string) (Result, error) {
	if processName == "" && processID <= 0 {
		return Result{}, fmt.Errorf("a process name or process ID is required")
	}
	if durationSeconds < 1 {
		return Result{}, fmt.Errorf("capture duration must be at least one second")
	}

	args := []string{
		"-terminate_after_t", strconv.Itoa(durationSeconds),
		"-output_file", outputCSV,
		"-no_top",
	}
	if processID > 0 {
		args = append(args, "-process_id", strconv.Itoa(processID))
	} else {
		args = append(args, "-process_name", processName)
	}
	command := exec.Command(p.Binary, args...)
	command.Env = append(os.Environ(), "PYTHONUTF8=1")
	output, err := command.CombinedOutput()
	if err != nil {
		text := strings.TrimSpace(string(output))
		if text == "" {
			text = err.Error()
		}
		return Result{}, fmt.Errorf("PresentMon capture failed: %s", text)
	}

	file, err := os.Open(outputCSV)
	if err != nil {
		return Result{}, fmt.Errorf("open PresentMon CSV %q: %w", outputCSV, err)
	}
	defer file.Close()
	values, err := ParsePresentMon(file)
	if err != nil {
		return Result{}, err
	}
	if len(values) == 0 {
		return Result{}, fmt.Errorf("PresentMon produced no frame-time samples")
	}
	normalized, err := writeNormalizedCSV(outputCSV, values)
	if err != nil {
		return Result{}, err
	}
	return Result{
		GameName: processName, ProcessName: processName, ProcessID: processID,
		Provider: p.Name(), ProviderBinary: p.Binary, OutputCSV: normalized,
		FrameTimesMS: values, CapturedAt: time.Now(),
		DurationSeconds: durationSeconds, FrameCount: len(values),
	}, nil
}

// ParsePresentMon accepts common PresentMon frame-time columns and normalizes them.
func ParsePresentMon(r io.Reader) ([]float64, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read PresentMon CSV: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("PresentMon CSV contains no frame records")
	}
	column := -1
	for i, header := range records[0] {
		name := strings.ToLower(strings.Trim(strings.TrimSpace(header), "\ufeff"))
		name = strings.ReplaceAll(name, " ", "")
		name = strings.ReplaceAll(name, "_", "")
		if name == "msbetweenpresents" || name == "frametime" || name == "frametimems" ||
			name == "msperframe" || name == "msuntildisplayed" {
			column = i
			break
		}
	}
	if column < 0 {
		return nil, fmt.Errorf("PresentMon CSV does not contain a recognized frame-time column")
	}
	values := make([]float64, 0, len(records)-1)
	for i, record := range records[1:] {
		if column >= len(record) {
			continue
		}
		value, err := strconv.ParseFloat(strings.TrimSpace(record[column]), 64)
		if err != nil || value <= 0 {
			continue
		}
		if value > 1000 {
			continue
		}
		values = append(values, value)
		_ = i
	}
	return values, nil
}

func writeNormalizedCSV(outputCSV string, values []float64) (string, error) {
	normalized := outputCSV
	file, err := os.Create(normalized)
	if err != nil {
		return "", fmt.Errorf("create normalized frame CSV %q: %w", normalized, err)
	}
	writer := csv.NewWriter(file)
	if err := writer.Write([]string{"frame", "frametime_ms"}); err != nil {
		file.Close()
		return "", err
	}
	for i, value := range values {
		if err := writer.Write([]string{strconv.Itoa(i + 1), fmt.Sprintf("%.6f", value)}); err != nil {
			file.Close()
			return "", err
		}
		_ = i
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return normalized, nil
}
