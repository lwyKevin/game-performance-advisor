package capture

import (
	"testing"

	"github.com/example/game-performance-advisor/internal/metrics"
)

func TestKnownGameCandidateConfidence(t *testing.T) {
	candidates := DetectCandidates([]metrics.Process{{
		Name: "Cyberpunk2077.exe", PID: 1234,
		Executable: `C:\SteamLibrary\steamapps\common\Cyberpunk 2077\Cyberpunk2077.exe`,
	}}, "Cyberpunk2077")
	if len(candidates) != 1 {
		t.Fatalf("got %d candidates", len(candidates))
	}
	if candidates[0].Confidence < 95 {
		t.Fatalf("confidence = %d, want at least 95", candidates[0].Confidence)
	}
}

func TestDeveloperToolsRemainLowConfidence(t *testing.T) {
	candidates := DetectCandidates([]metrics.Process{
		{Name: "Code.exe", PID: 10, Executable: `C:\Microsoft VS Code\Code.exe`},
		{Name: "chrome.exe", PID: 11, Executable: `C:\Program Files\Google\Chrome.exe`},
	}, "Code")
	if len(candidates) != 2 {
		t.Fatalf("got %d candidates", len(candidates))
	}
	if candidates[0].Confidence > 10 {
		t.Fatalf("low-confidence application ranked too high: %+v", candidates)
	}
}

func TestLaunchersAreExcluded(t *testing.T) {
	candidates := DetectCandidates([]metrics.Process{
		{Name: "Steam.exe", PID: 20},
		{Name: "LeagueClient.exe", PID: 21},
	}, "")
	if len(candidates) != 0 {
		t.Fatalf("launchers should not become candidates: %+v", candidates)
	}
}
