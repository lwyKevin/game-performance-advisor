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
