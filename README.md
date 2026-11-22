# Game Performance Advisor

**The primary product is the Windows desktop app (`gpa.exe`).** It scans live PC hardware and opens a professional HTML report. Docker and CSV analysis are for platform-independent testing and advanced users with existing frame-time data.

*"The primary product is the Windows desktop app (`gpa.exe`). Docker and CSV analysis are for platform-independent testing and advanced users with existing frame-time data."*

There is no installer and no need to install Go, Python, or external scripts for the Windows app. Double-clicking `gpa.exe` requires no CSV and does not expose any of the internal data collection details to the user.

## Quick Start

**Option 1: Double-Click (Easiest)**

Simply double-click `gpa.exe`. It will scan your PC, generate a dashboard, and automatically open it in your web browser.

**Option 2: Command Line (Advanced)**

If you want to capture live frames or analyze CSVs, use the terminal:

1. Download `gpa.exe` from the Releases page.
2. Open PowerShell or Command Prompt.
3. Run:

```powershell
.\gpa.exe scan --html report.html
```

You can also use:

```powershell
.\gpa.exe scan
.\gpa.exe scan --report scan.md --html scan.html
.\gpa.exe analyze samples\smooth_60fps.csv --target-fps 60 --report frame-analysis.md
.\gpa.exe analyze any-captured-frame-times.csv --target-fps 144 --html frame-analysis.html
```

`analyze` works on frame-time CSV or JSON data and does not require elevation.

## Docker / CSV Testing

Docker is a testing, CI, and demonstration environment for the platform-independent frame analysis engine. It does not replace the Windows hardware scan and cannot access Windows WMI or PowerShell counters.

Frame analysis accepts CSV files with `frame,frametime_ms` columns. Samples are provided in `samples/`. Tools that export frametime data (e.g., CapFrameX, RTSS) can be used to produce compatible files.

Build the image and run the bundled example automatically:

```bash
docker build -t gpa .
docker run --rm gpa
```

This runs `analyze` on a sample CSV bundled with the image. No Windows installation or user-provided CSV is required for the reviewer demo.

With a real existing frame-time CSV, mount it into the container:

```bash
docker run --rm -v "${PWD}:/data" gpa analyze /data/my-recording.csv --target-fps 60
```

The Docker workflow demonstrates that the Go analysis and recommendation math is platform-independent. Normal users should double-click `gpa.exe` instead of using Docker or creating CSV files.

## Commands

```text
gpa.exe scan [--json] [--report report.md] [--html report.html]
gpa.exe monitor [--duration 15] [--interval 2]
gpa.exe capture [15s] [--manual] [--process name] [--pid id]
gpa.exe analyze <frame-times.csv|json> [--target-fps 60]
gpa.exe compare before.csv after.csv [--report comparison.md] [--html comparison.html]
```

## Reports

- `--json` emits machine-readable system or capture data.
- `--report <file.md>` writes a Markdown report.
- `--html <file.html>` writes a standalone HTML report.

## Architecture

```text
cmd/game-performance-advisor/
internal/
  system/       Linux /proc inventory and embedded Windows/CIM script
  metrics/      Linux /proc metrics and embedded Windows metrics script
  analysis/     Bottleneck, confidence, and score calculations
  recommend/    Explainable action and frame-generation rules
  report/       Embedded HTML/Markdown templates and JSON rendering
  capture/      Capture support
  input/        CSV/JSON frame imports
```

PowerShell scripts and report templates are compiled into the executable with Go's `embed` package. Temporary copies of embedded scripts are created only while PowerShell executes them and are deleted afterward.

## Diagnostics

- GPU-limited: GPU utilization is high while CPU utilization is not dominant.
- CPU-limited/mixed: CPU utilization is high or utilization patterns suggest mixed pressure.
- Memory pressure: memory utilization is at or above a high-pressure threshold.
- Balanced: no dominant pressure is detected.

Confidence and recommendations are transparent; missing GPU telemetry, display refresh, or game-process detection reduces confidence and is stated in the report.

## Advanced / Developer

This section is for building or contributing from source.

Build the final standalone executable with no debug symbols or trim paths:

```powershell
go build -trimpath -ldflags="-s -w" -o gpa.exe ./cmd/game-performance-advisor
```

Run source tests:

```powershell
go test ./...
go vet ./...
```

Optional developer requirements:

- Go for building from source
- Docker verifies the platform-independent CSV analysis engine and CI workflows; it does not provide the Windows hardware scan
PowerShell is used by the native Windows build to query Windows CIM/WMI inventory and performance counters. NVIDIA utilization is enhanced with `nvidia-smi` when available.

## AI-assisted development notes

**Tools used:** GitHub Copilot

**Approach:** Plan-first workflow, package boundaries, targeted tests, and review of generated diffs.

I designed the architecture to use Go's `embed` package for a zero-dependency single binary.

Key verification areas include frame-time calculations, CPU/GPU bottleneck classification, report generation, capture argument parsing, and embedded script execution.
