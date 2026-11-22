package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/example/game-performance-advisor/internal/platform"
	"github.com/example/game-performance-advisor/internal/report"
)

func TestAnalyzeJSONIncludesEnvironment(t *testing.T) {
	stdout, err := os.CreateTemp("", "gpa-analyze-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(stdout.Name())
	defer stdout.Close()

	if err := run([]string{"analyze", "../../samples/smooth_60fps.csv", "--target-fps", "60", "--json"}, stdout, os.Stderr); err != nil {
		t.Fatal(err)
	}
	if _, err := stdout.Seek(0, 0); err != nil {
		t.Fatal(err)
	}

	var payload report.Report
	if err := json.NewDecoder(stdout).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	want := platform.NewCollector().Environment()
	if payload.Environment != want {
		t.Fatalf("analyze environment = %q, want %q", payload.Environment, want)
	}
}

func TestRemovedCommandsAreRejected(t *testing.T) {
	for _, command := range []string{"monitor", "compare", "capture"} {
		err := run([]string{command}, os.Stdout, os.Stderr)
		if err == nil {
			t.Fatalf("removed command %q was accepted", command)
		}
		if !strings.Contains(err.Error(), "unknown command") {
			t.Fatalf("removed command %q returned unexpected error: %v", command, err)
		}
	}
}

func TestNoArgumentsPrintsUsage(t *testing.T) {
	stdout, err := os.CreateTemp("", "gpa-usage-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(stdout.Name())
	defer stdout.Close()
	if err := run(nil, stdout, os.Stderr); err != nil {
		t.Fatal(err)
	}
	if _, err := stdout.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	data := make([]byte, 1024)
	n, err := stdout.Read(data)
	if err != nil && n == 0 {
		t.Fatal(err)
	}
	if !strings.Contains(string(data[:n]), "game-performance-advisor analyze") {
		t.Fatalf("usage did not contain analyze command: %s", data[:n])
	}
}

func TestHelpAndREADMEKeepOnlyDocumentedCommands(t *testing.T) {
	readme, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatal(err)
	}
	text := string(readme)
	for _, required := range []string{
		"game-performance-advisor analyze",
		"game-performance-advisor scan",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("README CLI block missing %q", required)
		}
	}
	for _, removed := range []string{"compare", "--html", "--report", "monitor"} {
		if strings.Contains(text, removed) {
			t.Fatalf("README still documents removed command or flag %q", removed)
		}
	}
	for _, command := range []string{"scan", "analyze"} {
		stdout, err := os.CreateTemp("", "gpa-help-*.txt")
		if err != nil {
			t.Fatal(err)
		}
		if err := run([]string{command, "--help"}, stdout, os.Stderr); err != nil {
			stdout.Close()
			os.Remove(stdout.Name())
			t.Fatal(err)
		}
		if _, err := stdout.Seek(0, 0); err != nil {
			stdout.Close()
			os.Remove(stdout.Name())
			t.Fatal(err)
		}
		data := make([]byte, 1024)
		n, readErr := stdout.Read(data)
		stdout.Close()
		os.Remove(stdout.Name())
		if readErr != nil && n == 0 {
			t.Fatal(readErr)
		}
		if !strings.Contains(string(data[:n]), command) {
			t.Fatalf("help for %s is inconsistent", command)
		}
	}
}
