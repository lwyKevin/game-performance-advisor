package system

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	gpametrics "github.com/example/game-performance-advisor/internal/metrics"
)

//go:embed inventory.ps1
var embeddedAssets embed.FS

type Info struct {
	OS           string   `json:"os"`
	CPUModel     string   `json:"cpu_model"`
	GPUModels    []string `json:"gpu_models"`
	RAMBytes     uint64   `json:"ram_bytes"`
	RAMGB        float64  `json:"ram_gb"`
	Resolution   string   `json:"resolution"`
	RefreshHz    int      `json:"refresh_hz"`
	IsAdmin      bool     `json:"is_admin"`
	CollectionOK bool     `json:"collection_ok"`
}

type powershellResult struct {
	OS            string   `json:"os"`
	CPU           string   `json:"cpu"`
	GPUs          []string `json:"gpus"`
	RAMBytes      uint64   `json:"ram_bytes"`
	Resolution    string   `json:"resolution"`
	RefreshHz     int      `json:"refresh_hz"`
	Administrator bool     `json:"administrator"`
}

// Collect reads machine inventory through Windows management interfaces.
// On unsupported platforms it returns a clearly identified fallback result.
func Collect() (Info, error) {
	info := Info{OS: runtime.GOOS, CollectionOK: false}
	if runtime.GOOS != "windows" {
		if runtime.GOOS != "linux" {
			info.OS = runtime.GOOS + " " + runtime.GOARCH
			return info, fmt.Errorf("system inventory is unavailable on %s", runtime.GOOS)
		}
		return collectUnix(info)
	}

	shell, err := exec.LookPath("powershell.exe")
	if err != nil {
		shell, err = exec.LookPath("powershell")
	}
	if err != nil {
		return info, fmt.Errorf("PowerShell was not found")
	}

	script, err := embeddedAssets.ReadFile("inventory.ps1")
	if err != nil {
		return info, fmt.Errorf("read embedded system inventory script: %w", err)
	}
	temp, err := os.CreateTemp("", "gpa-inventory-*.ps1")
	if err != nil {
		return info, fmt.Errorf("create temporary inventory script: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err := temp.Write(script); err != nil {
		temp.Close()
		return info, fmt.Errorf("write temporary inventory script: %w", err)
	}
	if err := temp.Close(); err != nil {
		return info, fmt.Errorf("close temporary inventory script: %w", err)
	}
	output, err := exec.Command(shell, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", tempPath).Output()
	if err != nil {
		return info, fmt.Errorf("collect Windows system information: %w", err)
	}
	var result powershellResult
	normalized, normalizeErr := gpametrics.NormalizeCollectorJSON(output)
	if normalizeErr != nil {
		preview := strings.TrimSpace(string(output))
		if len(preview) > 500 {
			preview = preview[:500]
		}
		return info, fmt.Errorf("Failed to parse Windows system information output.\nRaw collector: Windows system inventory PowerShell script\nDiagnostic output: %s\nPlease report this issue.", preview)
	}
	if err := json.Unmarshal(normalized, &result); err != nil {
		preview := strings.TrimSpace(string(normalized))
		if len(preview) > 500 {
			preview = preview[:500]
		}
		return info, fmt.Errorf("Failed to parse Windows system information output.\nRaw collector: Windows system inventory PowerShell script\nDiagnostic output: %s\nPlease report this issue.", preview)
	}
	info.OS = result.OS
	info.CPUModel = result.CPU
	info.GPUModels = result.GPUs
	info.RAMBytes = result.RAMBytes
	info.RAMGB = float64(result.RAMBytes) / (1024 * 1024 * 1024)
	info.Resolution = result.Resolution
	info.RefreshHz = result.RefreshHz
	info.IsAdmin = result.Administrator
	info.CollectionOK = true
	return info, nil
}

func collectUnix(info Info) (Info, error) {
	info.OS = runtime.GOOS + " " + runtime.GOARCH
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				info.OS = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), `"`)
				break
			}
		}
	}
	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "model name") || strings.HasPrefix(line, "Hardware") {
				if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
					info.CPUModel = strings.TrimSpace(parts[1])
					break
				}
			}
		}
	}
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if kib, err := parseUint(fields[1]); err == nil {
						info.RAMBytes = kib * 1024
						info.RAMGB = float64(info.RAMBytes) / (1024 * 1024 * 1024)
					}
				}
				break
			}
		}
	}
	if entries, err := filepath.Glob("/sys/class/drm/card*-*/modes"); err == nil && len(entries) > 0 {
		info.Resolution = "detected display"
	}
	info.CollectionOK = true
	return info, nil
}

func parseUint(value string) (uint64, error) {
	var result uint64
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return 0, fmt.Errorf("invalid unsigned integer %q", value)
		}
		result = result*10 + uint64(digit-'0')
	}
	return result, nil
}
