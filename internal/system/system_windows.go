//go:build windows

package system

import (
	"embed"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

//go:embed inventory.ps1
var inventoryScript embed.FS

type powershellResult struct {
	OS            string   `json:"os"`
	CPU           string   `json:"cpu"`
	GPUs          []string `json:"gpus"`
	RAMBytes      uint64   `json:"ram_bytes"`
	Resolution    string   `json:"resolution"`
	RefreshHz     int      `json:"refresh_hz"`
	Administrator bool     `json:"administrator"`
}

func Collect() (Info, error) {
	shell, err := exec.LookPath("powershell.exe")
	if err != nil {
		shell, err = exec.LookPath("powershell")
	}
	if err != nil {
		return Info{}, fmt.Errorf("PowerShell was not found")
	}
	script, err := inventoryScript.ReadFile("inventory.ps1")
	if err != nil {
		return Info{}, err
	}
	cmd := exec.Command(shell, "-NoProfile", "-NonInteractive", "-Command", "-")
	cmd.Stdin = strings.NewReader(string(script))
	output, err := cmd.Output()
	if err != nil {
		return Info{}, fmt.Errorf("collect Windows system information: %w", err)
	}
	var result powershellResult
	text := strings.TrimSpace(strings.TrimPrefix(string(output), "\ufeff"))
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return Info{}, fmt.Errorf("parse Windows system information: %w", err)
	}
	return Info{
		OS: result.OS, CPUModel: result.CPU, GPUModels: result.GPUs,
		RAMBytes: result.RAMBytes, RAMGB: float64(result.RAMBytes) / (1024 * 1024 * 1024),
		Resolution: result.Resolution, RefreshHz: result.RefreshHz,
		IsAdmin: result.Administrator, CollectionOK: true,
	}, nil
}
