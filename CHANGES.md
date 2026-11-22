# CHANGES

## Removed

- Removed `monitor` and `compare`. They added live monitoring and a second diff report but were not needed for the Linux/Docker submission.
- Removed capture/PresentMon integration and its game-process capture helpers.
- Removed HTML report generation and report templates.
- Removed Markdown report output. The CLI now uses one terminal serializer and one JSON serializer.
- Removed the report format matrix and command-specific output methods.

## Rewritten

- Made `analyze` the primary Docker default and the platform-independent core.
- Added a Linux procfs/sysfs platform collector and kept Windows collection as a separate build-tagged bonus adapter.
- Replaced the Windows inventory and metrics temp-file execution with PowerShell stdin execution.
- Added a `platform.Collector` interface so scan collection is separated from report and analysis code.
- Introduced one `report.Report` model with terminal and JSON serializers.
- Reworked frame analysis around explicit constants and known-value tests.
- Added 100-frame and 1000-frame availability limits for 1% and 0.1% lows.
- Added bimodal-distribution limitations and documented candidate stutter/bottleneck thresholds.

## Windows-Only Surface

`scan` can collect Windows inventory and metrics through PowerShell/WMI. This is retained only as a bonus; Linux/Docker frame analysis is the primary runnable path.

## Open Questions

- The stutter threshold `1.8 × median` is an initial heuristic and should be calibrated with larger real datasets.
- GPU utilization on Linux may require NVIDIA or ROCm tools; missing readings remain explicit.
- Recommendation thresholds are deliberately simple and should be tuned against target users and real measurements.
