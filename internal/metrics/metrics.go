package metrics

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed collector.ps1
var embeddedAssets embed.FS

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

// CreateTemporaryPowerShellScript writes an embedded script for execution and
// returns a cleanup function that removes the temporary file.
func CreateTemporaryPowerShellScript(script []byte, pattern string) (string, func(), error) {
	temp, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", nil, err
	}
	tempPath := temp.Name()
	cleanup := func() {
		_ = temp.Close()
		_ = os.Remove(tempPath)
	}
	if _, err := temp.Write(script); err != nil {
		cleanup()
		return "", nil, err
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempPath)
		return "", nil, err
	}
	return tempPath, cleanup, nil
}

type Snapshot struct {
	CapturedAt        time.Time `json:"captured_at"`
	CPUPercent        float64   `json:"cpu_percent"`
	GPUPercent        float64   `json:"gpu_percent"`
	GPUAvailable      bool      `json:"gpu_available"`
	RAMUsedBytes      uint64    `json:"ram_used_bytes"`
	RAMPercent        float64   `json:"ram_percent"`
	CPUP95Percent     float64   `json:"cpu_p95_percent"`
	CPUPeakPercent    float64   `json:"cpu_peak_percent"`
	GPUP95Percent     float64   `json:"gpu_p95_percent"`
	GPUPeakPercent    float64   `json:"gpu_peak_percent"`
	RAMP95Percent     float64   `json:"ram_p95_percent"`
	RAMPeakPercent    float64   `json:"ram_peak_percent"`
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
		if runtime.GOOS == "linux" {
			return collectLinux(snapshot)
		}
		return snapshot, fmt.Errorf("live metrics are unavailable on %s", runtime.GOOS)
	}
	shell, err := exec.LookPath("powershell.exe")
	if err != nil {
		shell, err = exec.LookPath("powershell")
	}
	if err != nil {
		return snapshot, fmt.Errorf("PowerShell was not found")
	}
	script, err := embeddedAssets.ReadFile("collector.ps1")
	if err != nil {
		return snapshot, fmt.Errorf("read embedded Windows metrics script: %w", err)
	}
	tempPath, cleanup, err := CreateTemporaryPowerShellScript(script, "gpa-metrics-*.ps1")
	if err != nil {
		return snapshot, fmt.Errorf("prepare temporary metrics script: %w", err)
	}
	defer cleanup()
	output, err := exec.Command(shell, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", tempPath).Output()
	writeRawCollectorLog(output, string(script))
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

func collectLinux(snapshot Snapshot) (Snapshot, error) {
	firstCPU, err := readLinuxCPU()
	if err != nil {
		return snapshot, fmt.Errorf("read Linux CPU metrics: %w", err)
	}
	time.Sleep(250 * time.Millisecond)
	secondCPU, err := readLinuxCPU()
	if err != nil {
		return snapshot, fmt.Errorf("read Linux CPU metrics: %w", err)
	}
	totalDelta := secondCPU.total - firstCPU.total
	idleDelta := secondCPU.idle - firstCPU.idle
	if totalDelta > 0 {
		snapshot.CPUPercent = float64(totalDelta-idleDelta) / float64(totalDelta) * 100
	}
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		var total, available uint64
		for _, line := range strings.Split(string(data), "\n") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			value, parseErr := strconv.ParseUint(fields[1], 10, 64)
			if parseErr != nil {
				continue
			}
			switch fields[0] {
			case "MemTotal:":
				total = value * 1024
			case "MemAvailable:":
				available = value * 1024
			}
		}
		if total > 0 {
			snapshot.RAMUsedBytes = total - available
			snapshot.RAMPercent = float64(snapshot.RAMUsedBytes) / float64(total) * 100
		}
	}
	snapshot.Processes = linuxProcesses()
	snapshot.GameProcesses = detectGames(snapshot.Processes)
	return snapshot, nil
}

type linuxCPU struct {
	total uint64
	idle  uint64
}

func readLinuxCPU() (linuxCPU, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return linuxCPU{}, err
	}
	fields := strings.Fields(strings.SplitN(string(data), "\n", 2)[0])
	if len(fields) < 5 || fields[0] != "cpu" {
		return linuxCPU{}, fmt.Errorf("invalid /proc/stat format")
	}
	var result linuxCPU
	for i, field := range fields[1:] {
		value, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return linuxCPU{}, err
		}
		result.total += value
		if i == 3 {
			result.idle = value
		}
	}
	return result, nil
}

func linuxProcesses() []Process {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	processes := make([]Process, 0, 12)
	pageSize := int64(os.Getpagesize())
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || !entry.IsDir() {
			continue
		}
		nameData, err := os.ReadFile("/proc/" + entry.Name() + "/comm")
		if err != nil {
			continue
		}
		process := Process{Name: strings.TrimSpace(string(nameData)), PID: pid}
		if statm, err := os.ReadFile("/proc/" + entry.Name() + "/statm"); err == nil {
			fields := strings.Fields(string(statm))
			if len(fields) > 1 {
				if resident, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
					process.MemoryMB = float64(resident*pageSize) / (1024 * 1024)
				}
			}
		}
		if executable, err := os.Readlink("/proc/" + entry.Name() + "/exe"); err == nil {
			process.Executable = executable
		}
		processes = append(processes, process)
	}
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].MemoryMB > processes[j].MemoryMB
	})
	if len(processes) > 12 {
		processes = processes[:12]
	}
	return processes
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
