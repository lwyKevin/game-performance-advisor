package system

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	gpametrics "github.com/example/game-performance-advisor/internal/metrics"
)

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
		info.OS = runtime.GOOS + " " + runtime.GOARCH
		return info, fmt.Errorf("system inventory is implemented for Windows; detected %s", runtime.GOOS)
	}

	shell, err := exec.LookPath("powershell.exe")
	if err != nil {
		shell, err = exec.LookPath("powershell")
	}
	if err != nil {
		return info, fmt.Errorf("PowerShell was not found")
	}

	script := `
$ErrorActionPreference='SilentlyContinue'
[Console]::OutputEncoding = [Text.UTF8Encoding]::new($false)
$os=(Get-CimInstance Win32_OperatingSystem | Select-Object -First 1)
$cpu=(Get-CimInstance Win32_Processor | Select-Object -First 1)
$gpus=@(Get-CimInstance Win32_VideoController | Where-Object { $_.Name } | Select-Object -ExpandProperty Name)
$display=Get-CimInstance Win32_VideoController | Where-Object { $_.CurrentHorizontalResolution } | Select-Object -First 1
$resolution=''
$refresh=0
if ($display) {
  $resolution="$($display.CurrentHorizontalResolution)x$($display.CurrentVerticalResolution)"
  $refresh=[int]$display.CurrentRefreshRate
}
$principal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
[ordered]@{
 os=if ($os){ $os.Caption + ' ' + $os.Version }else{'Windows'}
 cpu=if ($cpu){$cpu.Name}else{'Unknown CPU'}
 gpus=$gpus
 ram_bytes=if ($os){[uint64]$os.TotalVisibleMemorySize*1024}else{0}
 resolution=$resolution
 refresh_hz=$refresh
 administrator=$principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
} | ConvertTo-Json -Compress
`
	output, err := exec.Command(shell, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script).Output()
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
