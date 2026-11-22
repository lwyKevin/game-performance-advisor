package main

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestHardwarePlatformDetection(t *testing.T) {
	tests := []struct {
		goos    string
		windows bool
	}{
		{goos: "windows", windows: true},
		{goos: "linux", windows: false},
		{goos: "darwin", windows: false},
	}

	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			if got := isWindowsHardwarePlatform(tt.goos); got != tt.windows {
				t.Errorf("isWindowsHardwarePlatform(%q) = %v, want %v", tt.goos, got, tt.windows)
			}
		})
	}
}

func TestRunHardwareCommandsFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("non-Windows fallback behavior")
	}

	for _, command := range []string{"scan", "monitor"} {
		t.Run(command, func(t *testing.T) {
			err := run([]string{command}, os.Stdout, os.Stderr)
			if err == nil {
				t.Fatalf("run(%q) returned no error", command)
			}
			if err.Error() != hardwareScanUnavailableMessage() {
				t.Fatalf("run(%q) error = %q, want %q", command, err.Error(), hardwareScanUnavailableMessage())
			}
		})
	}
}

func TestParseCaptureArgs(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantProcess  string
		wantPID      int
		wantDuration int
		wantManual   bool
		wantErr      string
	}{
		{
			name:         "process before duration",
			args:         []string{"--process", "Cyberpunk2077.exe", "--duration", "15"},
			wantProcess:  "Cyberpunk2077.exe",
			wantDuration: 15,
		},
		{
			name:         "duration before process",
			args:         []string{"--duration", "15", "--process", "Cyberpunk2077.exe"},
			wantProcess:  "Cyberpunk2077.exe",
			wantDuration: 15,
		},
		{
			name:         "positional seconds",
			args:         []string{"15s"},
			wantDuration: 15,
		},
		{
			name:         "positional seconds without suffix",
			args:         []string{"15"},
			wantDuration: 15,
		},
		{
			name:         "manual",
			args:         []string{"--manual"},
			wantDuration: 15,
			wantManual:   true,
		},
		{
			name:         "duration with pid",
			args:         []string{"--duration", "15", "--pid", "1234"},
			wantPID:      1234,
			wantDuration: 15,
		},
		{
			name:    "duration not numeric",
			args:    []string{"--duration", "notanumber"},
			wantErr: "invalid value",
		},
		{
			name:    "duration negative",
			args:    []string{"--duration", "-5"},
			wantErr: "duration must be between",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options, err := parseCaptureArgs(tt.args)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("parseCaptureArgs(%v) returned no error", tt.args)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("parseCaptureArgs(%v) error = %q, want substring %q", tt.args, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseCaptureArgs(%v) error = %v", tt.args, err)
			}
			if options.ProcessName != tt.wantProcess {
				t.Errorf("ProcessName = %q, want %q", options.ProcessName, tt.wantProcess)
			}
			if options.ProcessID != tt.wantPID {
				t.Errorf("ProcessID = %d, want %d", options.ProcessID, tt.wantPID)
			}
			if options.Duration != tt.wantDuration {
				t.Errorf("Duration = %d, want %d", options.Duration, tt.wantDuration)
			}
			if options.Manual != tt.wantManual {
				t.Errorf("Manual = %v, want %v", options.Manual, tt.wantManual)
			}
		})
	}
}
