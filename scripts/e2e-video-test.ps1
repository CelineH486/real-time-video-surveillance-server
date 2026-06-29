param(
    [string]$VideoDirectory = (Join-Path $PSScriptRoot "..\testdata\videos"),
    [int]$LoopCount = 3
)

$ErrorActionPreference = "Stop"
$ffmpeg = (Get-Command ffmpeg).Source
$processes = @()

function Start-Publisher {
    param(
        [string]$CameraId,
        [string]$VideoPath,
        [string]$TargetUrl,
        [string]$Quality
    )

    $videoOptions = if ($Quality -eq "sub") {
        @("-vf", "scale=-2:360,fps=12", "-b:v", "500k", "-maxrate", "600k", "-bufsize", "1000k", "-g", "24")
    } else {
        @("-b:v", "2500k", "-maxrate", "3000k", "-bufsize", "5000k", "-g", "60")
    }

    $arguments = @(
        "-hide_banner", "-loglevel", "error", "-nostdin",
        "-stream_loop", ($LoopCount - 1), "-re", "-i", ('"' + $VideoPath + '"'),
        "-map", "0:v:0", "-an",
        "-c:v", "libx264", "-preset", "veryfast", "-tune", "zerolatency",
        "-pix_fmt", "yuv420p"
    ) + $videoOptions + @(
        "-f", "rtsp", "-rtsp_transport", "tcp", $TargetUrl
    )

    $startInfo = [System.Diagnostics.ProcessStartInfo]::new()
    $startInfo.FileName = $ffmpeg
    $startInfo.Arguments = $arguments -join " "
    $startInfo.UseShellExecute = $false
    $startInfo.CreateNoWindow = $true
    $startInfo.RedirectStandardError = $true
    $process = [System.Diagnostics.Process]::Start($startInfo)
    return [pscustomobject]@{ CameraId = $CameraId; Quality = $Quality; Process = $process }
}

for ($cameraNumber = 1; $cameraNumber -le 9; $cameraNumber++) {
    $cameraId = "cam{0:D2}" -f $cameraNumber
    $videoPath = Join-Path $VideoDirectory "cam$cameraNumber.mp4"
    if (-not (Test-Path -LiteralPath $videoPath)) {
        throw "Missing test video: $videoPath"
    }

    $target = "rtsp://truck001:local-development-publish-password@localhost:8554/truck001/$cameraId"
    $processes += Start-Publisher -CameraId $cameraId -VideoPath $videoPath -TargetUrl "$target/main" -Quality "main"
    $processes += Start-Publisher -CameraId $cameraId -VideoPath $videoPath -TargetUrl "$target/sub" -Quality "sub"
}

Write-Output "Started 9 main streams and 9 sub streams"
$failed = @()
foreach ($item in $processes) {
    $errorOutput = $item.Process.StandardError.ReadToEnd()
    $item.Process.WaitForExit()
    if ($item.Process.ExitCode -ne 0) {
        $failed += "$($item.CameraId) $($item.Quality): $errorOutput"
    }
}

if ($failed.Count -gt 0) {
    $failed | ForEach-Object { Write-Error $_ }
    exit 1
}

Write-Output "All test publishers completed successfully"
