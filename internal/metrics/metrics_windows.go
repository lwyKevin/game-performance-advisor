//go:build windows

package metrics

import (
	"embed"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

//go:embed collector.ps1
var collectorScript embed.FS

type psResult struct {
	CPU          float64 `json:"cpu"`
	RAMUsed      uint64  `json:"ram_used"`
	RAMPercent   float64 `json:"ram_percent"`
	GPU          float64 `json:"gpu"`
	GPUAvailable bool    `json:"gpu_available"`
}

func Collect() (Snapshot, error) {
	snapshot := Snapshot{CapturedAt: time.Now()}
	shell, err := exec.LookPath("powershell.exe")
	if err != nil {
		shell, err = exec.LookPath("powershell")
	}
	if err != nil {
		return snapshot, fmt.Errorf("PowerShell was not found")
	}
	script, err := collectorScript.ReadFile("collector.ps1")
	if err != nil {
		return snapshot, fmt.Errorf("read embedded Windows metrics script: %w", err)
	}
	cmd := exec.Command(shell, "-NoProfile", "-NonInteractive", "-Command", "-")
	cmd.Stdin = strings.NewReader(string(script))
	output, err := cmd.Output()
	if err != nil {
		return snapshot, fmt.Errorf("collect Windows metrics: %w", err)
	}
	var result psResult
	text := strings.TrimSpace(strings.TrimPrefix(string(output), "\ufeff"))
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return snapshot, fmt.Errorf("parse Windows metrics: %w", err)
	}
	snapshot.CPUPercent = result.CPU
	snapshot.CPUAvailable = result.CPU >= 0 && result.CPU <= 100
	snapshot.CPUMeasurementSeconds = 1
	snapshot.GPUPercent = result.GPU
	snapshot.GPUAvailable = result.GPUAvailable
	snapshot.RAMUsedBytes = result.RAMUsed
	snapshot.RAMPercent = result.RAMPercent
	return snapshot, nil
}
