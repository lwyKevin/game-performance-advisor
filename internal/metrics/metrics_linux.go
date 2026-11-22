//go:build !windows

package metrics

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func Collect() (Snapshot, error) {
	snapshot := Snapshot{CapturedAt: time.Now()}
	first, err := readCPU()
	if err != nil {
		return snapshot, err
	}
	const measurementWindowSeconds = 1.0
	time.Sleep(time.Second)
	second, err := readCPU()
	if err != nil {
		return snapshot, err
	}
	deltaTotal := second.total - first.total
	deltaIdle := second.idle - first.idle
	if deltaTotal > 0 {
		snapshot.CPUPercent = float64(deltaTotal-deltaIdle) / float64(deltaTotal) * 100
		snapshot.CPUAvailable = snapshot.CPUPercent >= 0 && snapshot.CPUPercent <= 100
		snapshot.CPUMeasurementSeconds = measurementWindowSeconds
	}

	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		var total, available uint64
		for _, line := range strings.Split(string(data), "\n") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			value, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
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
	if output, err := exec.Command("nvidia-smi", "--query-gpu=utilization.gpu", "--format=csv,noheader,nounits").Output(); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
			value, err := strconv.ParseFloat(strings.TrimSpace(line), 64)
			if err == nil {
				snapshot.GPUPercent = value
				snapshot.GPUAvailable = true
				break
			}
		}
	}
	return snapshot, nil
}

type cpuTimes struct {
	total uint64
	idle  uint64
}

func readCPU() (cpuTimes, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return cpuTimes{}, fmt.Errorf("read /proc/stat: %w", err)
	}
	fields := strings.Fields(strings.SplitN(string(data), "\n", 2)[0])
	if len(fields) < 5 || fields[0] != "cpu" {
		return cpuTimes{}, fmt.Errorf("invalid /proc/stat format")
	}
	var result cpuTimes
	for i, field := range fields[1:] {
		value, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return cpuTimes{}, err
		}
		result.total += value
		if i == 3 {
			result.idle = value
		}
	}
	return result, nil
}
