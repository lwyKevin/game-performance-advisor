//go:build windows

package metrics

import (
	"bytes"
	"embed"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
	"unicode/utf16"
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
	cmd := exec.Command(shell, "-NoProfile", "-NonInteractive", "-EncodedCommand", encodePowerShell(script))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail != "" {
			return snapshot, fmt.Errorf("collect Windows metrics: %w: %s", err, detail)
		}
		return snapshot, fmt.Errorf("collect Windows metrics: %w", err)
	}
	var result psResult
	text := strings.TrimSpace(strings.TrimPrefix(string(output), "\ufeff"))
	if text == "" {
		return snapshot, fmt.Errorf("collect Windows metrics: PowerShell returned no JSON output")
	}
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

func encodePowerShell(script []byte) string {
	runes := utf16.Encode([]rune(string(script)))
	encoded := make([]byte, len(runes)*2)
	for i, value := range runes {
		binary.LittleEndian.PutUint16(encoded[i*2:], value)
	}
	return base64.StdEncoding.EncodeToString(encoded)
}
