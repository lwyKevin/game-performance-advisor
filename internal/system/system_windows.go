//go:build windows

package system

import (
	"bytes"
	"embed"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"unicode/utf16"
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
	cmd := exec.Command(shell, "-NoProfile", "-NonInteractive", "-EncodedCommand", encodePowerShell(script))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail != "" {
			return Info{}, fmt.Errorf("collect Windows system information: %w: %s", err, detail)
		}
		return Info{}, fmt.Errorf("collect Windows system information: %w", err)
	}
	var result powershellResult
	text := strings.TrimSpace(strings.TrimPrefix(string(output), "\ufeff"))
	if text == "" {
		return Info{}, fmt.Errorf("collect Windows system information: PowerShell returned no JSON output")
	}
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

func encodePowerShell(script []byte) string {
	runes := utf16.Encode([]rune(string(script)))
	encoded := make([]byte, len(runes)*2)
	for i, value := range runes {
		binary.LittleEndian.PutUint16(encoded[i*2:], value)
	}
	return base64.StdEncoding.EncodeToString(encoded)
}
