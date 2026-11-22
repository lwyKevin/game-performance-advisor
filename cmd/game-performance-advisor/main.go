package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"

	frameanalysis "github.com/example/game-performance-advisor/internal/analysis"
	"github.com/example/game-performance-advisor/internal/input"
	"github.com/example/game-performance-advisor/internal/platform"
	"github.com/example/game-performance-advisor/internal/recommend"
	"github.com/example/game-performance-advisor/internal/report"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr *os.File) error {
	if len(args) == 0 {
		printUsage(stdout)
		return nil
	}

	switch args[0] {
	case "-h", "--help", "help":
		printUsage(stdout)
		return nil
	case "scan":
		if len(args) > 1 && (args[1] == "-h" || args[1] == "--help" || args[1] == "help") {
			printScanUsage(stdout)
			return nil
		}
		return runScan(args[1:], stdout, stderr)
	case "analyze":
		if len(args) > 1 && (args[1] == "-h" || args[1] == "--help" || args[1] == "help") {
			printAnalyzeUsage(stdout)
			return nil
		}
		return runAnalyze(args[1:], stdout, stderr)
	default:
		printUsage(stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runScan(args []string, stdout, stderr *os.File) error {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "print machine-readable JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	collector := platform.NewCollector()
	machine, snapshot, err := collector.Collect()
	if err != nil {
		return err
	}
	assessment := frameanalysis.AnalyzeSystem(machine, snapshot)
	recommendation := recommend.RecommendSystem(machine, assessment)
	environment := collector.Environment()
	if environment == "container (linux)" {
		assessment.Limitations = append(assessment.Limitations,
			"CPU/GPU counters reflect the container runtime, not the Windows host; run the Windows adapter outside Docker for host metrics.")
	}
	p := report.Report{
		Command:     "scan",
		Platform:    runtime.GOOS,
		Environment: environment,
		System: &report.SystemPayload{
			Machine: machine, Snapshot: snapshot, Assessment: assessment,
			Recommendation: recommendation, Environment: environment,
		},
	}
	return outputReport(p, *jsonOutput, stdout)
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
	environment := platform.NewCollector().Environment()
	p := report.Report{
		Command:     "analyze",
		Platform:    runtime.GOOS,
		Environment: environment,
		Frame: &report.FramePayload{
			Source:         args[0],
			TargetFPS:      *targetFPS,
			Analysis:       result,
			Recommendation: recommendation,
		},
	}
	return outputReport(p, *jsonOutput, stdout)
}

func outputReport(p report.Report, jsonOutput bool, stdout *os.File) error {
	if jsonOutput {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(p)
	}
	fmt.Fprint(stdout, report.Terminal(p))
	return nil
}

func printUsage(w *os.File) {
	fmt.Fprintln(w, "Game Performance Advisor")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  game-performance-advisor scan [--json]")
	fmt.Fprintln(w, "  game-performance-advisor analyze <frame-times.csv|json> [--target-fps 60] [--stutter-threshold 1.8] [--input-format auto|csv|json] [--json]")
}

func printAnalyzeUsage(w *os.File) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  game-performance-advisor analyze <frame-times.csv|json> [--target-fps 60] [--stutter-threshold 1.8] [--input-format auto|csv|json] [--json]")
}

func printScanUsage(w *os.File) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  game-performance-advisor scan [--json]")
}
