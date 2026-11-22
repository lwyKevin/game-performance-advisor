$ErrorActionPreference='SilentlyContinue'
[Console]::OutputEncoding = [Text.UTF8Encoding]::new($false)
$os=(Get-CimInstance Win32_OperatingSystem | Select-Object -First 1)
$cpu=(Get-CimInstance Win32_Processor | Select-Object -First 1)
$gpus=@(Get-CimInstance Win32_VideoController | Where-Object { $_.Name } | Select-Object -ExpandProperty Name)
$display=Get-CimInstance Win32_VideoController | Where-Object { $_.CurrentHorizontalResolution } | Select-Object -First 1
$resolution=''
$refresh=0
if ($display) {
  $resolution="$($display.CurrentHorizontalResolution)x$($display.CurrentVerticalResolution)"
  $refresh=[int]$display.CurrentRefreshRate
}
$principal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
[ordered]@{
 os=if ($os){ $os.Caption + ' ' + $os.Version }else{'Windows'}
 cpu=if ($cpu){$cpu.Name}else{'Unknown CPU'}
 gpus=$gpus
 ram_bytes=if ($os){[uint64]$os.TotalVisibleMemorySize*1024}else{0}
 resolution=$resolution
 refresh_hz=$refresh
 administrator=$principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
} | ConvertTo-Json -Compress
