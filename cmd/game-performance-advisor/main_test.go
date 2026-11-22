package main

import (
	"os"
	"strings"
	"testing"
)

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
