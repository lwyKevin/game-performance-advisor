package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	frameanalysis "github.com/example/game-performance-advisor/internal/analysis"
	"github.com/example/game-performance-advisor/internal/capture"
	"github.com/example/game-performance-advisor/internal/input"
	"github.com/example/game-performance-advisor/internal/metrics"
	"github.com/example/game-performance-advisor/internal/recommend"
	"github.com/example/game-performance-advisor/internal/report"
	"github.com/example/game-performance-advisor/internal/system"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr *os.File) error {
	if len(args) == 0 {
		printUsage(stderr)
		return fmt.Errorf("scan, monitor, or analyze command is required")
	}
	switch args[0] {
	case "-h", "--help", "help":
		printUsage(stdout)
		return nil
	case "scan":
		return runScan(args[1:], stdout, stderr)
	case "monitor", "capture":
		if args[0] == "capture" && len(args) > 1 &&
			(args[1] == "-h" || args[1] == "--help" || args[1] == "help") {
			printCaptureUsage(stdout)
			return nil
		}
		if args[0] == "capture" && len(args) > 1 && isCaptureDurationShorthand(args[1]) {
			durationText := strings.TrimSuffix(strings.ToLower(args[1]), "s")
			duration, err := strconv.Atoi(durationText)
			if err != nil {
				return fmt.Errorf("invalid capture duration %q", args[1])
			}
			args = append([]string{"--duration", fmt.Sprintf("%d", duration)}, args[2:]...)
			return runFrameCapture(args, stdout, stderr)
		}
		if args[0] == "capture" {
			return runFrameCapture(args[1:], stdout, stderr)
		}
		return runMonitor(args[1:], stdout, stderr)
	case "analyze":
		return runAnalyze(args[1:], stdout, stderr)
	case "compare":
		return runCompare(args[1:], stdout, stderr)
	default:
		printUsage(stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runScan(args []string, stdout, stderr *os.File) error {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "print machine-readable JSON")
	reportPath := fs.String("report", "", "write a Markdown report")
	htmlPath := fs.String("html", "", "write an HTML report")
	if err := fs.Parse(args); err != nil {
		return err
	}

	machine, err := system.Collect()
	if err != nil && !strings.Contains(err.Error(), "unsupported") {
		return err
	}
	snapshot, err := metrics.Collect()
	if err != nil {
		return err
	}
	result := report.SystemReport{
		Machine:        machine,
		Snapshot:       snapshot,
		Assessment:     frameanalysis.AnalyzeSystem(machine, snapshot),
		Recommendation: recommend.RecommendSystem(machine, frameanalysis.AnalyzeSystem(machine, snapshot)),
	}
	return outputSystemReport(result, *jsonOutput, *reportPath, *htmlPath, stdout)
}

func runFrameCapture(args []string, stdout, stderr *os.File) error {
	fs := flag.NewFlagSet("capture", flag.ContinueOnError)
	fs.SetOutput(stderr)
	duration := fs.Int("duration", 15, "capture duration in seconds")
	manualStop := fs.Bool("manual", false, "wait for ENTER instead of using a fixed duration")
	processName := fs.String("process", "", "exact game process name")
	processID := fs.Int("pid", 0, "exact game process ID")
	presentMonPath := fs.String("presentmon", "", "PresentMon executable path")
	targetFPS := fs.Float64("target-fps", 60, "target average FPS")
	stutterThreshold := fs.Float64("stutter-threshold", 1.8, "stutter threshold")
	outputCSV := fs.String("save", "", "normalized CSV output path")
	jsonOutput := fs.Bool("json", false, "print JSON")
	reportPath := fs.String("report", "", "write a Markdown report")
	htmlPath := fs.String("html", "", "write an HTML report")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *duration < 1 || *duration > 300 {
		return fmt.Errorf("duration must be between 1 and 300 seconds")
	}

	if *processName == "" && *processID <= 0 {
		fmt.Fprintln(stdout, "Waiting for gameplay process...")
		fmt.Fprintln(stdout, "Launch or switch to your game. Press Ctrl+C to cancel.")
		for {
			snapshot, err := metrics.Collect()
			if err != nil {
				return fmt.Errorf("detect active games: %w", err)
			}
			candidates := capture.DetectCandidates(snapshot.Processes, snapshot.ForegroundProcess)
			if len(candidates) == 0 {
				time.Sleep(2 * time.Second)
				continue
			}
			fmt.Fprintln(stdout, "\nDetected candidates:")
			for i, candidate := range candidates {
				if i >= 8 {
					break
				}
				fmt.Fprintf(stdout, "%d. %s.exe (%d%%)\n", i+1, candidate.ProcessName, candidate.Confidence)
				fmt.Fprintf(stdout, "   %s\n", candidate.Reason)
			}
			fmt.Fprintf(stdout, "\nDetected game: %s.exe\n", candidates[0].ProcessName)
			fmt.Fprintf(stdout, "Confidence: %d%%\n", candidates[0].Confidence)
			fmt.Fprint(stdout, "Continue? [Y/n]: ")
			var response string
			if _, err := fmt.Fscan(os.Stdin, &response); err != nil {
				return fmt.Errorf("read confirmation: %w", err)
			}
			response = strings.TrimSpace(strings.ToLower(response))
			if response == "n" || response == "no" {
				fmt.Fprint(stdout, "Select process number: ")
				var selection int
				if _, err := fmt.Fscan(os.Stdin, &selection); err != nil {
					return fmt.Errorf("read process selection: %w", err)
				}
				if selection < 1 || selection > len(candidates) {
					return fmt.Errorf("invalid process selection %d", selection)
				}
				selected := candidates[selection-1]
				*processName = selected.ProcessName
				*processID = selected.ProcessID
			} else {
				*processName = candidates[0].ProcessName
				*processID = candidates[0].ProcessID
			}
			break
		}
	}

	presentMonBinary, err := capture.DetectPresentMon(*presentMonPath)
	if err != nil {
		return err
	}
	if *outputCSV == "" {
		*outputCSV = fmt.Sprintf("capture-%s.csv", time.Now().Format("20060102-150405"))
	}
	provider := capture.PresentMon{Binary: presentMonBinary}
	var result capture.Result
	if *manualStop {
		fmt.Fprintln(stdout, "\nCapture started.")
		fmt.Fprintln(stdout, "Play normally, then press ENTER to stop capture.")
		stop := make(chan struct{})
		done := make(chan struct{})
		go func() {
			var ignored string
			_, _ = fmt.Fscan(os.Stdin, &ignored)
			close(stop)
			close(done)
		}()
		result, err = provider.CaptureUntilStopped(*processName, *processID, *outputCSV, stop)
		<-done
		fmt.Fprintln(stdout, "Capture stopped.")
	} else {
		fmt.Fprintf(stdout, "\nCapture started for %s (%d seconds).\n", *processName, *duration)
		fmt.Fprintln(stdout, "Play normally.")
		result, err = provider.Capture(*processName, *processID, *duration, *outputCSV)
	}
	if err != nil {
		return err
	}
	analysisResult, err := frameanalysis.Analyze(result.FrameTimesMS, *targetFPS, *stutterThreshold)
	if err != nil {
		return err
	}
	recommendation := recommend.Recommend(analysisResult, *targetFPS)
	captureReport := report.CaptureReport{
		Capture: result, Analysis: analysisResult,
		Recommendation: recommendation, TargetFPS: *targetFPS,
	}
	if *jsonOutput {
		data, err := report.CaptureJSON(captureReport)
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprint(stdout, report.CaptureTerminal(captureReport))
	}
	if *reportPath != "" {
		if err := os.WriteFile(*reportPath, []byte(report.CaptureMarkdown(captureReport)), 0644); err != nil {
			return fmt.Errorf("write Markdown capture report %q: %w", *reportPath, err)
		}
		if !*jsonOutput {
			fmt.Fprintf(stdout, "\nMarkdown report written to %s\n", *reportPath)
		}
	}
	if *htmlPath != "" {
		if err := os.WriteFile(*htmlPath, []byte(report.CaptureHTML(captureReport)), 0644); err != nil {
			return fmt.Errorf("write HTML capture report %q: %w", *htmlPath, err)
		}
		if !*jsonOutput {
			fmt.Fprintf(stdout, "\nHTML report written to %s\n", *htmlPath)
		}
	}
	return nil
}
func runMonitor(args []string, stdout, stderr *os.File) error {
	fs := flag.NewFlagSet("monitor", flag.ContinueOnError)
	fs.SetOutput(stderr)
	duration := fs.Int("duration", 15, "monitor duration in seconds")
	interval := fs.Int("interval", 2, "sample interval in seconds")
	jsonOutput := fs.Bool("json", false, "print machine-readable JSON")
	reportPath := fs.String("report", "", "write a Markdown report")
	htmlPath := fs.String("html", "", "write an HTML report")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *duration < 1 || *duration > 300 {
		return fmt.Errorf("duration must be between 1 and 300 seconds")
	}
	if *interval < 1 || *interval > 30 {
		return fmt.Errorf("interval must be between 1 and 30 seconds")
	}

	machine, err := system.Collect()
	if err != nil {
		return err
	}
	samples := make([]metrics.Snapshot, 0)
	for elapsed := 0; elapsed < *duration; elapsed += *interval {
		snapshot, collectErr := metrics.Collect()
		if collectErr != nil {
			return collectErr
		}
		samples = append(samples, snapshot)
		fmt.Fprintf(stdout, "Captured snapshot %d/%d\n", len(samples), (*duration+*interval-1)/(*interval))
		if elapsed+*interval < *duration {
			time.Sleep(time.Duration(*interval) * time.Second)
		}
	}
	average := averageSnapshots(samples)
	assessment := frameanalysis.AnalyzeSystem(machine, average)
	result := report.SystemReport{
		Machine: machine, Snapshot: average,
		Assessment: assessment, Recommendation: recommend.RecommendSystem(machine, assessment),
	}
	return outputSystemReport(result, *jsonOutput, *reportPath, *htmlPath, stdout)
}

func averageSnapshots(samples []metrics.Snapshot) metrics.Snapshot {
	result := metrics.Snapshot{}
	if len(samples) == 0 {
		return result
	}
	for _, sample := range samples {
		result.CPUPercent += sample.CPUPercent
		result.GPUPercent += sample.GPUPercent
		result.RAMPercent += sample.RAMPercent
		result.RAMUsedBytes += sample.RAMUsedBytes
		result.GPUAvailable = result.GPUAvailable || sample.GPUAvailable
		if len(sample.GameProcesses) > 0 {
			result.GameProcesses = append(result.GameProcesses, sample.GameProcesses...)
		}
		result.Processes = sample.Processes
	}
	count := float64(len(samples))
	result.CPUPercent /= count
	result.GPUPercent /= count
	result.RAMPercent /= count
	result.RAMUsedBytes /= uint64(count)
	result.CapturedAt = time.Now()
	return result
}

func outputSystemReport(result report.SystemReport, jsonOutput bool, reportPath, htmlPath string, stdout *os.File) error {
	if jsonOutput {
		data, err := report.JSON(result)
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprint(stdout, report.Terminal(result))
	}
	if reportPath != "" {
		if err := os.WriteFile(reportPath, []byte(report.Markdown(result)), 0644); err != nil {
			return fmt.Errorf("write Markdown report %q: %w", reportPath, err)
		}
		if !jsonOutput {
			fmt.Fprintf(stdout, "\nMarkdown report written to %s\n", reportPath)
		}
	}
	if htmlPath != "" {
		if err := os.WriteFile(htmlPath, []byte(report.HTML(result)), 0644); err != nil {
			return fmt.Errorf("write HTML report %q: %w", htmlPath, err)
		}
		if !jsonOutput {
			fmt.Fprintf(stdout, "\nHTML report written to %s\n", htmlPath)
		}
	}
	return nil
}

func printUsage(w *os.File) {
	fmt.Fprintln(w, "Game Performance Advisor")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  game-performance-advisor scan [--json] [--report report.md] [--html report.html]")
	fmt.Fprintln(w, "  game-performance-advisor monitor [--duration 15] [--interval 2]")
	fmt.Fprintln(w, "  game-performance-advisor capture 15s [--process name] [--target-fps 60]")
	fmt.Fprintln(w, "  game-performance-advisor compare before.csv after.csv [--report comparison.md] [--html comparison.html]")
	fmt.Fprintln(w, "  game-performance-advisor analyze <frame-times.csv> [flags]  (advanced/import mode)")
}

func printCaptureUsage(w *os.File) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  game-performance-advisor capture")
	fmt.Fprintln(w, "  game-performance-advisor capture 15s")
	fmt.Fprintln(w, "  game-performance-advisor capture --manual")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --duration 15")
	fmt.Fprintln(w, "  --manual")
	fmt.Fprintln(w, "  --process <name>")
	fmt.Fprintln(w, "  --pid <id>")
	fmt.Fprintln(w, "  --presentmon <path>")
	fmt.Fprintln(w, "  --target-fps 60")
	fmt.Fprintln(w, "  --report report.md")
	fmt.Fprintln(w, "  --html report.html")
	fmt.Fprintln(w, "  --json")
}

func isCaptureDurationShorthand(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	if len(value) < 2 || value[len(value)-1] != 's' {
		return false
	}
	for _, char := range value[:len(value)-1] {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func runAnalyze(args []string, stdout, stderr *os.File) error {
	if len(args) < 1 {
		return fmt.Errorf("CSV or JSON input is required for analyze")
	}
	fs := flag.NewFlagSet("analyze", flag.ContinueOnError)
	fs.SetOutput(stderr)
	targetFPS := fs.Float64("target-fps", 60, "target FPS")
	threshold := fs.Float64("stutter-threshold", 1.8, "stutter threshold")
	format := fs.String("input-format", "auto", "input format: auto, csv, json")
	jsonOutput := fs.Bool("json", false, "print JSON")
	reportPath := fs.String("report", "", "write a Markdown report")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	values, err := input.Read(args[0], *format)
	if err != nil {
		return err
	}
	result, err := frameanalysis.Analyze(values, *targetFPS, *threshold)
	if err != nil {
		return err
	}
	recommendation := recommend.Recommend(result, *targetFPS)
	payload := struct {
		Source         string               `json:"source"`
		TargetFPS      float64              `json:"target_fps"`
		Analysis       frameanalysis.Result `json:"analysis"`
		Recommendation recommend.Result     `json:"recommendation"`
	}{args[0], *targetFPS, result, recommendation}
	if *jsonOutput {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(payload)
	}
	fmt.Fprintf(stdout, "Game Performance Advisor - Advanced Frame Import\n\n")
	fmt.Fprintf(stdout, "Source              %s\n", args[0])
	fmt.Fprintf(stdout, "Frames              %d\n", result.FrameCount)
	fmt.Fprintf(stdout, "Average FPS         %.1f\n", result.AverageFPS)
	fmt.Fprintf(stdout, "1%% Low FPS          %.1f\n", result.OnePercentLowFPS)
	fmt.Fprintf(stdout, "Frame Health        %d / 100\n", result.HealthScore)
	fmt.Fprintf(stdout, "Diagnosis           %s (%d%% confidence)\n", result.Bottleneck, result.Confidence)
	fmt.Fprintf(stdout, "FG Recommendation   %s (%s)\n", recommendation.FrameGeneration, recommendation.FGMultiplier)
	fmt.Fprintf(stdout, "Latency Impact      %s\n", recommendation.LatencyRangeMS)
	if *reportPath != "" {
		var b strings.Builder
		fmt.Fprintf(&b, "# Game Performance Advisor - Frame Import\n\n")
		fmt.Fprintf(&b, "- Source: `%s`\n- Average FPS: %.1f\n- 1%% Low: %.1f FPS\n",
			args[0], result.AverageFPS, result.OnePercentLowFPS)
		fmt.Fprintf(&b, "- Frame Health: %d / 100\n- Diagnosis: %s\n",
			result.HealthScore, result.Bottleneck)
		fmt.Fprintf(&b, "- Frame Generation: %s (%s)\n", recommendation.FrameGeneration, recommendation.FGMultiplier)
		if err := os.WriteFile(*reportPath, []byte(b.String()), 0644); err != nil {
			return fmt.Errorf("write Markdown report %q: %w", *reportPath, err)
		}
		fmt.Fprintf(stdout, "\nMarkdown report written to %s\n", *reportPath)
	}
	return nil
}

func runCompare(args []string, stdout, stderr *os.File) error {
	fs := flag.NewFlagSet("compare", flag.ContinueOnError)
	fs.SetOutput(stderr)
	beforeLabel := fs.String("before-label", "Before optimization", "before capture label")
	afterLabel := fs.String("after-label", "After optimization", "after capture label")
	targetFPS := fs.Float64("target-fps", 60, "target average FPS")
	reportPath := fs.String("report", "", "write a Markdown comparison")
	htmlPath := fs.String("html", "", "write an HTML comparison")
	positional := make([]string, 0, 2)
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--before-label":
			i++
			if i >= len(args) {
				return fmt.Errorf("--before-label requires a value")
			}
			*beforeLabel = args[i]
		case "--after-label":
			i++
			if i >= len(args) {
				return fmt.Errorf("--after-label requires a value")
			}
			*afterLabel = args[i]
		case "--target-fps":
			i++
			if i >= len(args) {
				return fmt.Errorf("--target-fps requires a value")
			}
			value, err := fmt.Sscanf(args[i], "%f", targetFPS)
			if err != nil || value != 1 {
				return fmt.Errorf("invalid target FPS %q", args[i])
			}
		case "--report":
			i++
			if i >= len(args) {
				return fmt.Errorf("--report requires a value")
			}
			*reportPath = args[i]
		case "--html":
			i++
			if i >= len(args) {
				return fmt.Errorf("--html requires a value")
			}
			*htmlPath = args[i]
		default:
			positional = append(positional, args[i])
		}
	}
	if len(positional) != 2 {
		return fmt.Errorf("compare requires before.csv and after.csv")
	}
	beforeValues, err := input.Read(positional[0], "csv")
	if err != nil {
		return fmt.Errorf("read before capture: %w", err)
	}
	afterValues, err := input.Read(positional[1], "csv")
	if err != nil {
		return fmt.Errorf("read after capture: %w", err)
	}
	beforeAnalysis, err := frameanalysis.Analyze(beforeValues, *targetFPS, 1.8)
	if err != nil {
		return err
	}
	afterAnalysis, err := frameanalysis.Analyze(afterValues, *targetFPS, 1.8)
	if err != nil {
		return err
	}
	beforeRecommendation := recommend.Recommend(beforeAnalysis, *targetFPS)
	afterRecommendation := recommend.Recommend(afterAnalysis, *targetFPS)
	beforeCapture := capture.Result{GameName: *beforeLabel, FrameTimesMS: beforeValues, FrameCount: len(beforeValues)}
	afterCapture := capture.Result{GameName: *afterLabel, FrameTimesMS: afterValues, FrameCount: len(afterValues)}
	beforeReport := report.CaptureReport{Capture: beforeCapture, Analysis: beforeAnalysis, Recommendation: beforeRecommendation, TargetFPS: *targetFPS}
	afterReport := report.CaptureReport{Capture: afterCapture, Analysis: afterAnalysis, Recommendation: afterRecommendation, TargetFPS: *targetFPS}
	output := report.ComparisonMarkdown(beforeReport, afterReport)
	fmt.Fprint(stdout, output)
	if *reportPath != "" {
		if err := os.WriteFile(*reportPath, []byte(output), 0644); err != nil {
			return fmt.Errorf("write comparison report %q: %w", *reportPath, err)
		}
		fmt.Fprintf(stdout, "\nMarkdown report written to %s\n", *reportPath)
	}
	if *htmlPath != "" {
		if err := os.WriteFile(*htmlPath, []byte(report.ComparisonHTML(beforeReport, afterReport)), 0644); err != nil {
			return fmt.Errorf("write HTML comparison report %q: %w", *htmlPath, err)
		}
		fmt.Fprintf(stdout, "\nHTML report written to %s\n", *htmlPath)
	}
	return nil
}
