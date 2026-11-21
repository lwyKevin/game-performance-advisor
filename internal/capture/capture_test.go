package capture

import (
	"strings"
	"testing"
)

func TestParsePresentMonFrameTimes(t *testing.T) {
	csvData := "Application,ProcessId,MsBetweenPresents,MsUntilDisplayed\nGame.exe,1234,16.67,18.2\nGame.exe,1234,33.33,35.0\n"
	values, err := ParsePresentMon(strings.NewReader(csvData))
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 || values[0] != 16.67 || values[1] != 33.33 {
		t.Fatalf("unexpected values: %v", values)
	}
}

func TestParsePresentMonRejectsMissingFrameColumn(t *testing.T) {
	_, err := ParsePresentMon(strings.NewReader("Application,ProcessId\nGame.exe,1234\n"))
	if err == nil {
		t.Fatal("expected missing frame-time column error")
	}
}
