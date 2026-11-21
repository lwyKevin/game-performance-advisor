
$ErrorActionPreference='SilentlyContinue'
[Console]::OutputEncoding = [Text.UTF8Encoding]::new($false)
Start-Sleep -Milliseconds 250
$cpu=0
try {
  $cpuCounter=Get-Counter '\Processor(_Total)\% Processor Time' -ErrorAction Stop
  if ($cpuCounter -and $cpuCounter.CounterSamples.Count -gt 0) {
    $cpu=[math]::Round([double]$cpuCounter.CounterSamples[0].CookedValue,1)
  }
} catch {}
$ramUsed=[uint64]0
$ramPercent=0
try {
  $os=Get-CimInstance -ClassName Win32_OperatingSystem -ErrorAction Stop
  $ramUsed=[uint64](($os.TotalVisibleMemorySize-$os.FreePhysicalMemory)*1024)
  if ($os.TotalVisibleMemorySize -gt 0) {
    $ramPercent=[math]::Round((1-($os.FreePhysicalMemory/$os.TotalVisibleMemorySize))*100,1)
  }
} catch {}
$processes=@()
try {
  foreach ($process in (Get-Process -ErrorAction Stop | Sort-Object -Property CPU -Descending | Select-Object -First 12)) {
    $processCpu=0
    if ($null -ne $process.CPU) { $processCpu=[math]::Round([double]$process.CPU,1) }
    $memoryMb=[math]::Round([double]$process.WorkingSet64/1MB,1)
    $executable=''
    try { $executable=$process.Path } catch {}
    $processes += [ordered]@{
      name=[string]$process.ProcessName
      pid=[int]$process.Id
      cpu_percent=[double]$processCpu
      memory_mb=[double]$memoryMb
      executable=[string]$executable
    }
  }
} catch {}
$foreground=''
try {
  $signature='using System; using System.Runtime.InteropServices; public class GPAWindow { [DllImport("user32.dll")] public static extern IntPtr GetForegroundWindow(); [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr hWnd, out uint processId); }'
  if (-not ('GPAWindow' -as [type])) { Add-Type -TypeDefinition $signature -ErrorAction Stop }
  $window=[GPAWindow]::GetForegroundWindow()
  $windowProcessId=0
  [GPAWindow]::GetWindowThreadProcessId($window,[ref]$windowProcessId) | Out-Null
  if ($windowProcessId -gt 0) {
    $foregroundProcess=Get-Process -Id $windowProcessId -ErrorAction SilentlyContinue
    if ($foregroundProcess) { $foreground=$foregroundProcess.ProcessName }
  }
} catch {}
$gpu=0.0
$gpuAvailable=$false
try {
  $nvidia=Get-Command nvidia-smi.exe -ErrorAction SilentlyContinue
  if ($nvidia) {
    $gpuText=& $nvidia.Source --query-gpu=utilization.gpu --format=csv,noheader,nounits 2>$null
    if ($gpuText) {
      $gpu=[math]::Round([double]($gpuText | Select-Object -First 1),1)
      $gpuAvailable=$true
    }
  }
} catch {}
[ordered]@{
  cpu=[double]$cpu
  ram_used=[uint64]$ramUsed
  ram_percent=[double]$ramPercent
  processes=$processes
  gpu=[double]$gpu
  gpu_available=[bool]$gpuAvailable
  foreground_process=[string]$foreground
} | ConvertTo-Json -Compress -Depth 6
