package main

import (
	"strings"
	"testing"
)

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
