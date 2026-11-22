# Game Performance Advisor

## System Summary

| Item | Detected |
|---|---|
| CPU | {{.Machine.CPUModel}} |
| GPU | {{range $i, $gpu := .Machine.GPUModels}}{{if $i}}, {{end}}{{$gpu}}{{end}} |
| RAM | {{printf "%.1f" .Machine.RAMGB}} GB |
| Display | {{.Machine.Resolution}} @ {{.Machine.RefreshHz}} Hz |
| OS | {{.Machine.OS}} |

## Performance Snapshot

- GPU: {{printf "%.0f" .Snapshot.GPUPercent}}%
- GPU peak / p95: {{printf "%.0f" .Snapshot.GPUPeakPercent}}% / {{printf "%.0f" .Snapshot.GPUP95Percent}}%
- CPU: {{printf "%.0f" .Snapshot.CPUPercent}}%
- CPU peak / p95: {{printf "%.0f" .Snapshot.CPUPeakPercent}}% / {{printf "%.0f" .Snapshot.CPUP95Percent}}%
- Memory: {{printf "%.0f" .Snapshot.RAMPercent}}%
- Memory peak / p95: {{printf "%.0f" .Snapshot.RAMPeakPercent}}% / {{printf "%.0f" .Snapshot.RAMP95Percent}}%

## Assessment

- Performance score: **{{.Assessment.PerformanceScore}} / 100**
- Smoothness score: **{{.Assessment.SmoothnessScore}} / 100**
- Frame Generation readiness: **{{.Assessment.FGReadiness}} / 100**
- Confidence: **{{.Assessment.Confidence}} / 100**
- Primary diagnosis: **{{.Assessment.Diagnosis}}**
- Alternative: {{.Assessment.Alternative}}

### Evidence
{{range .Assessment.Evidence}}- {{.}}
{{end}}{{range .Assessment.Limitations}}- {{.}}
{{end}}
## Recommendation

- Upscaling: {{.Recommendation.Upscaling}}
- Frame Generation: {{.Recommendation.FrameGeneration}} ({{.Recommendation.FGMultiplier}})
- Estimated benefit: {{.Recommendation.FGBenefit}}
- Latency impact: {{.Recommendation.LatencyImpact}}

### Suggested Actions
{{range .Recommendation.PriorityActions}}- {{.}}
{{end}}
