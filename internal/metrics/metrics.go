package metrics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

var writeRawCollectorLog = func(raw []byte, script string) {
	if os.Getenv("GPA_DEBUG_METRICS") == "" {
		return
	}
	path := os.Getenv("GPA_DEBUG_METRICS")
	_ = os.WriteFile(path, raw, 0644)
	_ = os.WriteFile(path+".ps1", []byte(script), 0644)
}

// NormalizeCollectorJSON exposes the same safe decoder to system inventory.
func NormalizeCollectorJSON(raw []byte) ([]byte, error) {
	return normalizeCollectorJSON(raw)
}

type Snapshot struct {
	CapturedAt        time.Time `json:"captured_at"`
	CPUPercent        float64   `json:"cpu_percent"`
	GPUPercent        float64   `json:"gpu_percent"`
	GPUAvailable      bool      `json:"gpu_available"`
	RAMUsedBytes      uint64    `json:"ram_used_bytes"`
	RAMPercent        float64   `json:"ram_percent"`
	Processes         []Process `json:"processes"`
	GameProcesses     []Process `json:"likely_game_processes"`
	ForegroundProcess string    `json:"foreground_process"`
}

type Process struct {
	Name       string  `json:"name"`
	PID        int     `json:"pid"`
	CPUPercent float64 `json:"cpu_percent"`
	MemoryMB   float64 `json:"memory_mb"`
	Executable string  `json:"executable"`
}

type psResult struct {
	CPU               float64   `json:"cpu"`
	RAMUsed           uint64    `json:"ram_used"`
	RAMPercent        float64   `json:"ram_percent"`
	Processes         []Process `json:"processes"`
	GPU               float64   `json:"gpu"`
	GPUAvailable      bool      `json:"gpu_available"`
	ForegroundProcess string    `json:"foreground_process"`
}

func normalizeCollectorJSON(raw []byte) ([]byte, error) {
	text := strings.TrimSpace(string(raw))
	text = strings.TrimPrefix(text, "\ufeff")
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("collector output was empty")
	}
	objects := extractJSONValues(text)
	if len(objects) == 0 {
		return nil, fmt.Errorf("collector output contained no JSON values")
	}
	if len(objects) == 1 {
		if bytes.HasPrefix(objects[0], []byte("[")) {
			var items []json.RawMessage
			if err := json.Unmarshal(objects[0], &items); err != nil {
				return nil, err
			}
			if len(items) == 0 {
				return nil, fmt.Errorf("collector returned an empty JSON array")
			}
			return mergeCollectorObjects(items)
		}
		return objects[0], nil
	}
	return mergeCollectorObjects(objects)
}

func extractJSONValues(text string) []json.RawMessage {
	objects := make([]json.RawMessage, 0)
	decoder := json.NewDecoder(bytes.NewReader([]byte(text)))
	for {
		var value json.RawMessage
		err := decoder.Decode(&value)
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		objects = append(objects, bytes.TrimSpace(value))
	}
	return objects
}

func mergeCollectorObjects(objects []json.RawMessage) ([]byte, error) {
	merged := make(map[string]json.RawMessage)
	for _, object := range objects {
		object = bytes.TrimSpace(object)
		if len(object) == 0 || object[0] != '{' {
			return nil, fmt.Errorf("collector JSON array contained a non-object value")
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(object, &fields); err != nil {
			return nil, err
		}
		for key, value := range fields {
			merged[key] = value
		}
	}
	return json.Marshal(merged)
}

// Collect takes a short live snapshot of CPU, memory, GPU, and processes.
func Collect() (Snapshot, error) {
	snapshot := Snapshot{CapturedAt: time.Now()}
	if runtime.GOOS != "windows" {
		return snapshot, fmt.Errorf("live Windows metrics are unavailable on %s", runtime.GOOS)
	}
	shell, err := exec.LookPath("powershell.exe")
	if err != nil {
		shell, err = exec.LookPath("powershell")
	}
	if err != nil {
		return snapshot, fmt.Errorf("PowerShell was not found")
	}
	script := `
$ErrorActionPreference='SilentlyContinue'
[Console]::OutputEncoding = [Text.UTF8Encoding]::new($false)
Start-Sleep -Milliseconds 250
$cpu=0
try {
  $cpuCounter=Get-Counter '\Processor(_Total)\% Processor Time' -ErrorAction Stop
  if ($cpuCounter -and $cpuCounter.CounterSamples.Count -gt 0) {
    $cpu=[math]::Round([double]$cpuCounter.CounterSamples[0].CookedValue,1)
  }
} catch {}
$ramUsed=[uint64]0
$ramPercent=0
try {
  $os=Get-CimInstance -ClassName Win32_OperatingSystem -ErrorAction Stop
  $ramUsed=[uint64](($os.TotalVisibleMemorySize-$os.FreePhysicalMemory)*1024)
  if ($os.TotalVisibleMemorySize -gt 0) {
    $ramPercent=[math]::Round((1-($os.FreePhysicalMemory/$os.TotalVisibleMemorySize))*100,1)
  }
} catch {}
$processes=@()
try {
  foreach ($process in (Get-Process -ErrorAction Stop | Sort-Object -Property CPU -Descending | Select-Object -First 12)) {
    $processCpu=0
    if ($null -ne $process.CPU) { $processCpu=[math]::Round([double]$process.CPU,1) }
    $memoryMb=[math]::Round([double]$process.WorkingSet64/1MB,1)
    $executable=''
    try { $executable=$process.Path } catch {}
    $processes += [ordered]@{
      name=[string]$process.ProcessName
      pid=[int]$process.Id
      cpu_percent=[double]$processCpu
      memory_mb=[double]$memoryMb
      executable=[string]$executable
    }
  }
} catch {}
$foreground=''
try {
  $signature='using System; using System.Runtime.InteropServices; public class GPAWindow { [DllImport("user32.dll")] public static extern IntPtr GetForegroundWindow(); [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr hWnd, out uint processId); }'
  if (-not ('GPAWindow' -as [type])) { Add-Type -TypeDefinition $signature -ErrorAction Stop }
  $window=[GPAWindow]::GetForegroundWindow()
  $windowProcessId=0
  [GPAWindow]::GetWindowThreadProcessId($window,[ref]$windowProcessId) | Out-Null
  if ($windowProcessId -gt 0) {
    $foregroundProcess=Get-Process -Id $windowProcessId -ErrorAction SilentlyContinue
    if ($foregroundProcess) { $foreground=$foregroundProcess.ProcessName }
  }
} catch {}
$gpu=0.0
$gpuAvailable=$false
try {
  $nvidia=Get-Command nvidia-smi.exe -ErrorAction SilentlyContinue
  if ($nvidia) {
    $gpuText=& $nvidia.Source --query-gpu=utilization.gpu --format=csv,noheader,nounits 2>$null
    if ($gpuText) {
      $gpu=[math]::Round([double]($gpuText | Select-Object -First 1),1)
      $gpuAvailable=$true
    }
  }
} catch {}
[ordered]@{
  cpu=[double]$cpu
  ram_used=[uint64]$ramUsed
  ram_percent=[double]$ramPercent
  processes=$processes
  gpu=[double]$gpu
  gpu_available=[bool]$gpuAvailable
  foreground_process=[string]$foreground
} | ConvertTo-Json -Compress -Depth 6
`
	output, err := exec.Command(shell, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script).Output()
	writeRawCollectorLog(output, script)
	if err != nil {
		return snapshot, fmt.Errorf("collect live Windows metrics: %w", err)
	}
	var result psResult
	normalized, normalizeErr := normalizeCollectorJSON(output)
	if normalizeErr != nil {
		preview := strings.TrimSpace(string(output))
		if len(preview) > 500 {
			preview = preview[:500]
		}
		return snapshot, fmt.Errorf("Failed to parse Windows metrics collection output.\nRaw collector: Windows live metrics PowerShell script\nDiagnostic output: %s\nPlease report this issue.", preview)
	}
	if err := json.Unmarshal(normalized, &result); err != nil {
		preview := strings.TrimSpace(string(normalized))
		return snapshot, fmt.Errorf("Failed to parse Windows metrics collection output.\nRaw collector: Windows live metrics PowerShell script\nDiagnostic output: %s\nPlease report this issue.", preview)
	}
	snapshot.CPUPercent = result.CPU
	snapshot.GPUPercent = result.GPU
	snapshot.GPUAvailable = result.GPUAvailable
	snapshot.RAMUsedBytes = result.RAMUsed
	snapshot.RAMPercent = result.RAMPercent
	snapshot.Processes = result.Processes
	snapshot.ForegroundProcess = result.ForegroundProcess
	snapshot.GameProcesses = detectGames(result.Processes)
	return snapshot, nil
}

func detectGames(processes []Process) []Process {
	names := []string{
		"league of legends", "valorant", "cs2", "fortnite", "minecraft",
		"dota2", "eldenring", "cyberpunk2077", "gta5", "rdr2",
		"deadlock", "overwatch", "roblox", "starcraft2",
		"apexlegends", "r5apex", "warzone", "cod", "modernwarfare",
		"pubg", "borderlands", "destiny", "helldivers", "baldurs",
		"hogwartslegacy", "monsterhunterwilds", "thefinals", "marathon",
	}
	matches := make([]Process, 0)
	for _, process := range processes {
		lower := strings.ToLower(process.Name)
		for _, name := range names {
			if strings.Contains(lower, name) {
				matches = append(matches, process)
				break
			}
		}
	}
	return matches
}

// DetectGames exposes active-process matching to capture and reporting layers.
func DetectGames(processes []Process) []Process {
	return detectGames(processes)
}
