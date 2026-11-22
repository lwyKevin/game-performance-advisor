//go:build !windows

package system

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func Collect() (Info, error) {
	info := Info{OS: runtime.GOOS + " " + runtime.GOARCH, CollectionOK: true}
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
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					info.CPUModel = strings.TrimSpace(parts[1])
				}
				break
			}
		}
	}
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 2 && fields[0] == "MemTotal:" {
				if kib, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
					info.RAMBytes = kib * 1024
					info.RAMGB = float64(info.RAMBytes) / (1024 * 1024 * 1024)
				}
				break
			}
		}
	}
	info.GPUModels = detectLinuxGPUs()
	return info, nil
}

func detectLinuxGPUs() []string {
	if output, err := exec.Command("nvidia-smi", "--query-gpu=name", "--format=csv,noheader").Output(); err == nil {
		names := []string{}
		for _, name := range strings.Split(strings.TrimSpace(string(output)), "\n") {
			if name = strings.TrimSpace(name); name != "" {
				names = append(names, name)
			}
		}
		if len(names) > 0 {
			return names
		}
	}
	if output, err := exec.Command("rocm-smi", "--showproductname").Output(); err == nil {
		text := strings.TrimSpace(string(output))
		if text != "" {
			return []string{text}
		}
	}
	var cards []string
	if entries, err := filepath.Glob("/sys/class/drm/card*-*/device/vendor"); err == nil {
		for _, path := range entries {
			if _, err := os.ReadFile(path); err == nil {
				cards = append(cards, "GPU detected via sysfs")
				break
			}
		}
	}
	return cards
}
