# CHANGES

## Removed
 
- Reduced the CLI to the two commands needed for the Linux/Docker submission.
- Removed auxiliary report paths and command-specific output methods.
- The remaining interface uses one terminal serializer and one JSON serializer.

## Rewritten

- Made `analyze` the primary Docker default and the platform-independent core.
- Added a Linux procfs/sysfs platform collector and kept Windows collection as a separate build-tagged bonus adapter.
- Replaced the Windows inventory and metrics temp-file execution with PowerShell stdin execution.
- Added a `platform.Collector` interface so scan collection is separated from report and analysis code.
- Introduced one `report.Report` model with terminal and JSON serializers.
- Reworked frame analysis around explicit constants and known-value tests.
- Added 100-frame and 1000-frame availability limits for 1% and 0.1% lows.
- Added bimodal-distribution limitations and documented candidate stutter/bottleneck thresholds.
- Added an evidence-sufficiency gate for hardware diagnosis so near-zero CPU or unavailable GPU metrics cannot produce a false balanced diagnosis or frame-generation recommendation.
- Added null 0.1% low FPS JSON values and container/native environment metadata for scan output.

## Windows-Only Surface

`scan` can collect Windows inventory and metrics through PowerShell/WMI. This is retained only as a bonus; Linux/Docker frame analysis is the primary runnable path.

## Open Questions

- The stutter threshold `1.8 × median` is an initial heuristic and should be calibrated with larger real datasets.
- GPU utilization on Linux may require NVIDIA or ROCm tools; missing readings remain explicit.
- Recommendation thresholds are deliberately simple and should be tuned against target users and real measurements.
