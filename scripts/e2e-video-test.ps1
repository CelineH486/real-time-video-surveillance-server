param(
    [string]$VideoDirectory = (Join-Path $PSScriptRoot "..\testdata\videos"),
    [int]$LoopCount = 3
)

$ErrorActionPreference = "Stop"
$ffmpeg = (Get-Command ffmpeg).Source
$processes = @()

for ($cameraNumber = 1; $cameraNumber -le 9; $cameraNumber++) {
    $cameraId = "cam{0:D2}" -f $cameraNumber
    $videoPath = Join-Path $VideoDirectory "cam$cameraNumber.mp4"
    if (-not (Test-Path -LiteralPath $videoPath)) {
        throw "Missing test video: $videoPath"
    }

    $target = "rtsp://truck001:local-development-publish-password@localhost:8554/truck001/$cameraId"
    $arguments = @(
        "-hide_banner", "-loglevel", "error", "-nostdin",
        "-stream_loop", ($LoopCount - 1), "-re", "-i", ('"' + $videoPath + '"'),
        "-map", "0:v:0", "-c:v", "copy", "-an",
        "-f", "rtsp", "-rtsp_transport", "tcp", "$target/main",
        "-map", "0:v:0", "-vf", "scale=-2:360,fps=12",
        "-c:v", "libx264", "-preset", "veryfast", "-tune", "zerolatency",
        "-pix_fmt", "yuv420p", "-b:v", "500k", "-maxrate", "600k", "-bufsize", "1000k",
        "-g", "24", "-an", "-f", "rtsp", "-rtsp_transport", "tcp", "$target/sub"
    )

    $startInfo = [System.Diagnostics.ProcessStartInfo]::new()
    $startInfo.FileName = $ffmpeg
    $startInfo.Arguments = $arguments -join " "
    $startInfo.UseShellExecute = $false
    $startInfo.CreateNoWindow = $true
    $startInfo.RedirectStandardError = $true
    $process = [System.Diagnostics.Process]::Start($startInfo)
    $processes += [pscustomobject]@{ CameraId = $cameraId; Process = $process }
}

Write-Output "Started 9 main streams and 9 sub streams"
$failed = @()
foreach ($item in $processes) {
    $errorOutput = $item.Process.StandardError.ReadToEnd()
    $item.Process.WaitForExit()
    if ($item.Process.ExitCode -ne 0) {
        $failed += "$($item.CameraId): $errorOutput"
    }
}

if ($failed.Count -gt 0) {
    $failed | ForEach-Object { Write-Error $_ }
    exit 1
}

Write-Output "All test publishers completed successfully"
