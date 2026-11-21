package capture

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/example/game-performance-advisor/internal/metrics"
)

type Candidate struct {
	ProcessName string `json:"process_name"`
	ProcessID   int    `json:"process_id"`
	Confidence  int    `json:"confidence"`
	Executable  string `json:"executable"`
	Reason      string `json:"reason"`
}

var gameNames = []string{
	"cyberpunk2077", "league of legends", "leagueclient",
	"genshinimpact", "hoyoplay", "robloxplayerbeta", "eldenring",
	"valorant", "valorant-win64-shipping", "cs2", "fortnite",
	"minecraft", "dota2", "gta5", "rdr2", "deadlock", "overwatch",
	"apexlegends", "r5apex", "destiny", "helldivers", "hogwartslegacy",
	"baldursgate", "monsterhunterwilds", "thefinals", "starcraft2",
}

var launcherNames = map[string]bool{
	"steam": true, "steamwebhelper": true, "leagueclient": true,
	"hoyoplay": true, "epicgameslauncher": true, "battle.net": true,
	"origin": true, "eadesktop": true, "ubisoftconnect": true,
	"xbox": true, "riotclient": true,
}

var lowConfidenceNames = map[string]bool{
	"code": true, "devenv": true, "chrome": true, "msedge": true,
	"firefox": true, "outlook": true, "winword": true, "excel": true,
	"powershell": true, "windowsterminal": true, "cmd": true,
	"obs": true, "obs64": true, "blender": true, "unrealeditor": true,
}

func DetectCandidates(processes []metrics.Process, foregroundName string) []Candidate {
	candidates := make([]Candidate, 0)
	seen := make(map[string]bool)
	for _, process := range processes {
		name := strings.TrimSpace(process.Name)
		key := strings.ToLower(name)
		if key == "" || seen[key] || launcherNames[key] {
			continue
		}
		seen[key] = true
		confidence, reason := candidateScore(name, process, foregroundName)
		if confidence < 1 {
			continue
		}
		candidates = append(candidates, Candidate{
			ProcessName: name,
			ProcessID:   process.PID,
			Confidence:  confidence,
			Executable:  process.Executable,
			Reason:      reason,
		})
	}
	sortCandidates(candidates)
	return candidates
}

func candidateScore(name string, process metrics.Process, foregroundName string) (int, string) {
	lower := strings.ToLower(strings.TrimSuffix(name, filepath.Ext(name)))
	if lowConfidenceNames[lower] {
		return 2, "Low-confidence developer, browser, office, or capture application"
	}
	if launcherNames[lower] {
		return 0, "Launcher process"
	}
	for _, known := range gameNames {
		if strings.Contains(lower, strings.ReplaceAll(known, " ", "")) ||
			strings.Contains(lower, known) {
			confidence := 95
			reason := "Recognized game executable"
			if lower == "league of legends" || strings.Contains(lower, "league") {
				confidence = 95
			}
			if lower == strings.ToLower(strings.TrimSuffix(filepath.Base(process.Executable), filepath.Ext(process.Executable))) {
				confidence = 97
				reason = "Recognized game executable and verified executable path"
			}
			if strings.EqualFold(name, foregroundName) {
				confidence += 2
				reason = "Known game executable and foreground application"
			}
			if confidence > 99 {
				confidence = 99
			}
			return confidence, reason
		}
	}
	confidence := 8
	reason := "Unknown executable"
	executableLower := strings.ToLower(process.Executable)
	for _, marker := range []string{"\\steam\\", "\\steamapps\\", "\\epic games\\", "\\xboxgames\\",
		"\\riot games\\", "\\battle.net\\", "\\ubisoft\\", "\\ea games\\", "\\hoyoverse\\", "\\genshin impact\\"} {
		if strings.Contains(executableLower, marker) {
			confidence = 88
			reason = "Executable is installed under a game distribution directory"
			break
		}
	}
	if strings.EqualFold(name, foregroundName) {
		confidence += 10
		reason += "; foreground application"
	}
	if process.CPUPercent >= 5 {
		confidence += 3
		reason += "; significant process activity"
	}
	if confidence > 99 {
		confidence = 99
	}
	return confidence, strings.TrimPrefix(reason, "; ")
}

func sortCandidates(candidates []Candidate) {
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Confidence > candidates[j].Confidence
	})
}
