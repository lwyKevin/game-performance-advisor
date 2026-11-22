# Game Performance Advisor

Game Performance Advisor (GPA) is a Go application that analyzes gaming performance.

It has two use cases:

1. **Windows system reports**: inspect the PC, detect CPU/GPU/RAM activity, diagnose a likely bottleneck, and generate an HTML report.
2. **Frame-time analysis**: read existing frame-time measurements, calculate FPS lows and stutters, and recommend whether spatial upscaling or frame generation is suitable.

The source code is the product repository. A prebuilt `gpa.exe` is not included in this repository.

## Pick your path

| If you are... | Use this |
|---|---|
| A Windows user who wants the full PC report | Build and double-click `gpa.exe` (Option 1) |
| A Windows user who wants to analyze a frametime CSV | Build `gpa.exe`, then use PowerShell (Option 2) |
| A reviewer or developer on Linux / Mac / Docker | Use Docker (see "Developer: Test with Docker") |

You do not need all three. Pick the one that matches your situation.

## User: Run on Windows

GPA is designed to be run as a Windows executable.

### Option 1: Install Go and build it yourself

Install a Go toolchain compatible with the version declared in `go.mod`, open PowerShell in this repository, and run:

```powershell
go build -trimpath -ldflags="-s -w" -o gpa.exe .\cmd\game-performance-advisor
```

This creates `gpa.exe` in the current folder.

Double-click `gpa.exe`. It will:

1. Collect Windows CPU, GPU, memory, display, and process information.
2. Analyze the snapshot.
3. Create `GPA-Report.html`.
4. Open the HTML report in the default browser.

No CSV file is required for this desktop-style report mode.

### Option 2: Run commands in PowerShell

Keep the PowerShell terminal open and run:

```powershell
.\gpa.exe scan
.\gpa.exe scan --json
.\gpa.exe scan --report scan.md --html scan.html
```

Analyze a frametime CSV:

```powershell
.\gpa.exe analyze .\samples\smooth_60fps.csv --target-fps 60
.\gpa.exe analyze .\samples\smooth_60fps.csv --target-fps 60 --json
.\gpa.exe analyze .\samples\smooth_60fps.csv --target-fps 60 --report frame-analysis.md
```

Compare two recordings:

```powershell
.\gpa.exe compare .\before.csv .\after.csv --target-fps 60
.\gpa.exe compare .\before.csv .\after.csv --target-fps 60 --report comparison.md --html comparison.html
```

Monitor average utilization over time:

```powershell
.\gpa.exe monitor --duration 15 --interval 2
.\gpa.exe monitor --duration 15 --interval 2 --json
.\gpa.exe monitor --duration 15 --interval 2 --report monitor.md --html monitor.html
```

Run `.\gpa.exe --help` to show the available commands.

## Frame-time CSV format

Frame analysis accepts a simple CSV with exactly this kind of header:

```csv
frame,frametime_ms
1,16.67
2,16.71
3,16.69
```

The `frametime_ms` column is required. The `frame` column is accepted but is not needed for the calculations.

Bundled examples are in [`samples/`](samples/):

- `smooth_60fps.csv`
- `gpu_bound_lowfps.csv`
- `mixed_hitching.csv`
- `cpu_stutter.csv`

Tools that export frametime data, such as CapFrameX or RTSS, can produce compatible CSV files after column export or conversion. GPA parses the `frame,frametime_ms` format only; raw multi-column capture exports are not parsed directly.

## Developer: How the project works

This is a layered Go application:

```text
cmd/game-performance-advisor/main.go
  CLI entry point and application modes

internal/system
  Windows inventory collection

internal/metrics
  CPU, GPU, memory, foreground-process, and game-process collection

internal/input
  CSV and JSON frame-time parsing

internal/analysis
  FPS lows, stutter, variation, bottleneck, and health calculations

internal/recommend
  Upscaling and frame-generation recommendations

internal/report
  Terminal, JSON, Markdown, and HTML report generation
```

The CLI starts in [`main.go`](cmd/game-performance-advisor/main.go). It:

1. Reads command-line arguments.
2. Calls the appropriate system, frame, or report package.
3. Prints terminal output or writes a requested report file.
4. Exits with an error message if required input or collection fails.

On Windows, GPA uses embedded PowerShell scripts to collect machine and live metrics. The scripts are compiled into the executable with Go's `embed` package. They are temporarily written to disk only while PowerShell executes them, then deleted.

On Linux and inside Docker, the Windows hardware scan is not available. Docker does not inspect the Windows host or collect Windows WMI/PowerShell metrics.

## Developer: Run tests

Run the complete Go test suite:

```powershell
go test ./...
```

The tests cover frame parsing, frame statistics, bottleneck and recommendation logic, system inventory behavior, report rendering, and command-line argument parsing.

Run one package while developing:

```powershell
go test ./internal/analysis
```

Build a local executable:

```powershell
go build -o gpa.exe .\cmd\game-performance-advisor
```

## Developer: Test with Docker

Docker is a portable test environment for the frame-analysis engine. It is useful when the developer or reviewer does not have Windows, does not want to install Go, or wants to verify a clean build and a known sample.

Docker tests:

- The application builds.
- The Go tests pass.
- The bundled CSV parser works.
- The FPS, stutter, and recommendation logic works.
- The final executable runs independently of Windows hardware APIs.

Docker does **not** test the Windows hardware scan.

Build the image:

```powershell
docker build -t gpa .
```

Run the default smooth-60-FPS sample:

```powershell
docker run --rm gpa
```

Run another bundled sample:

```powershell
docker run --rm gpa analyze /samples/mixed_hitching.csv --target-fps 60
```

Mount and analyze a CSV from the Windows working directory:

```powershell
docker run --rm -v "${PWD}:/data" gpa analyze /data/recording.csv --target-fps 144
```

The Dockerfile is a two-stage build:

1. A Go image runs `go test ./...` and builds the executable.
2. A small scratch image contains the executable and the `samples/` directory.
3. The default container command analyzes `smooth_60fps.csv`.

## Distributing a Windows executable

An executable generated locally is not automatically committed or uploaded to GitHub.

For a normal user, you can distribute `gpa.exe` through a GitHub Release. GitHub Releases are intended for downloadable binaries and install packages. Attach the file to a tagged release and link to that release from the README.

You can also distribute source instructions, as this repository does, so users can build the executable themselves.

## Reports

- `scan` and `monitor` support `--json`, `--report <file.md>`, and `--html <file.html>`.
- `analyze` supports `--json` and `--report <file.md>`; it does not support `--html`.
- `compare` supports `--report <file.md>` and `--html <file.html>`.

The HTML report uses embedded templates and does not require the source repository to remain beside the executable at runtime.
