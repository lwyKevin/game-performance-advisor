//go:build windows

package platform

import "testing"

func TestWindowsCollectorEnvironment(t *testing.T) {
	if environment := (windowsCollector{}).Environment(); environment != "windows-native" {
		t.Fatalf("environment = %q, want windows-native", environment)
	}
}
