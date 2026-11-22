//go:build !windows

package platform

import "testing"

func TestLinuxCollectorEnvironment(t *testing.T) {
	environment := (linuxCollector{}).Environment()
	if environment != "container (linux)" && environment != "linux-native" {
		t.Fatalf("unexpected Linux environment: %q", environment)
	}
}
