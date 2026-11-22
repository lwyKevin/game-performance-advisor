package metrics

import (
	"bytes"
	"os"
	"testing"
)

func TestCreateTemporaryPowerShellScriptCleansUp(t *testing.T) {
	path, cleanup, err := CreateTemporaryPowerShellScript([]byte("Write-Output test"), "gpa-test-*.ps1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("temporary script was not created: %v", err)
	}
	cleanup()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("temporary script was not removed: %v", err)
	}
}

func TestNormalizeCollectorJSONSingleObject(t *testing.T) {
	raw := []byte(`{"cpu":25.5,"gpu_available":true}`)
	output, err := normalizeCollectorJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(output, []byte(`"cpu":25.5`)) {
		t.Fatalf("unexpected normalized output: %s", output)
	}
}

func TestNormalizeCollectorJSONArray(t *testing.T) {
	raw := []byte(`[{"cpu":25},{"foreground_process":"game.exe"}]`)
	output, err := normalizeCollectorJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(output, []byte(`"cpu":25`)) ||
		!bytes.Contains(output, []byte(`"foreground_process":"game.exe"`)) {
		t.Fatalf("unexpected merged array: %s", output)
	}
}

func TestNormalizeCollectorJSONMultiObject(t *testing.T) {
	raw := []byte("{\"cpu\":25}\n{\"gpu\":90,\"gpu_available\":true}")
	output, err := normalizeCollectorJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(output, []byte(`"gpu":90`)) ||
		!bytes.Contains(output, []byte(`"cpu":25`)) {
		t.Fatalf("unexpected merged objects: %s", output)
	}
}

func TestNormalizeCollectorJSONTrailingWhitespaceAndBOM(t *testing.T) {
	raw := []byte("\ufeff  {\"cpu\":12.5} \r\n\t")
	output, err := normalizeCollectorJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(output, []byte(`"cpu":12.5`)) {
		t.Fatalf("unexpected normalized output: %s", output)
	}
}

func TestNormalizeCollectorJSONLocalizedUTF8(t *testing.T) {
	raw := []byte(`{"foreground_process":"GenshinImpact.exe","processes":[{"name":"League of Legends.exe","executable":"C:/法國/League.exe"}]}`)
	output, err := normalizeCollectorJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(output, []byte("法國")) {
		t.Fatalf("UTF-8 text was changed: %s", output)
	}
}

func TestNormalizeCollectorJSONRejectsEmptyOutput(t *testing.T) {
	if _, err := normalizeCollectorJSON([]byte("\r\n \t")); err == nil {
		t.Fatal("expected empty collector output error")
	}
}
