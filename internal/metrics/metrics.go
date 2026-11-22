package metrics

import "time"

type Snapshot struct {
	CapturedAt            time.Time `json:"captured_at"`
	CPUPercent            float64   `json:"cpu_percent"`
	CPUAvailable          bool      `json:"cpu_available"`
	CPUMeasurementSeconds float64   `json:"cpu_measurement_seconds"`
	GPUPercent            float64   `json:"gpu_percent"`
	GPUAvailable          bool      `json:"gpu_available"`
	RAMUsedBytes          uint64    `json:"ram_used_bytes"`
	RAMPercent            float64   `json:"ram_percent"`
	GameProcesses         []Process `json:"likely_game_processes,omitempty"`
}

type Process struct {
	Name string `json:"name"`
	PID  int    `json:"pid"`
}
