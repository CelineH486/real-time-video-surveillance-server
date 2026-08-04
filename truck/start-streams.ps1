param(
    [string]$ConfigPath = (Join-Path $PSScriptRoot "streams.json")
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path -LiteralPath $ConfigPath)) {
    throw "Missing config: $ConfigPath. Copy streams.example.json to streams.json first."
}

$config = Get-Content -LiteralPath $ConfigPath -Raw | ConvertFrom-Json
if ($config.cameras.Count -ne 9) {
    throw "Exactly 9 cameras are required; found $($config.cameras.Count)."
}

$server = $config.mediaServer.TrimEnd("/") -replace '^rtsp://', ''
$user = [Uri]::EscapeDataString($config.truckId)
$password = [Uri]::EscapeDataString($config.publishPassword)
$logDirectory = Join-Path $PSScriptRoot "logs"
New-Item -ItemType Directory -Path $logDirectory -Force | Out-Null

foreach ($camera in $config.cameras) {
    $cameraId = [Uri]::EscapeDataString($camera.cameraId)
    $baseTarget = "rtsp://${user}:${password}@${server}/$($config.truckId)/${cameraId}"
    $arguments = @(
        "-hide_banner", "-loglevel", "warning",
        "-rtsp_transport", "tcp", "-i", $camera.source,
        "-map", "0:v:0", "-c:v", "copy", "-an",
        "-f", "rtsp", "-rtsp_transport", "tcp", "$baseTarget/main",
        "-map", "0:v:0", "-vf", "scale=-2:360,fps=12",
        "-c:v", "libx264", "-preset", "veryfast", "-tune", "zerolatency",
        "-pix_fmt", "yuv420p", "-b:v", "500k", "-maxrate", "600k", "-bufsize", "1000k",
        "-g", "24", "-an", "-f", "rtsp", "-rtsp_transport", "tcp", "$baseTarget/sub"
    )
    $outputLogPath = Join-Path $logDirectory "$($camera.cameraId).out.log"
    $errorLogPath = Join-Path $logDirectory "$($camera.cameraId).error.log"
    Start-Process -FilePath $config.ffmpeg -ArgumentList $arguments -WindowStyle Hidden `
        -RedirectStandardOutput $outputLogPath -RedirectStandardError $errorLogPath
    Write-Host "Started $($camera.cameraId): main + sub"
}
