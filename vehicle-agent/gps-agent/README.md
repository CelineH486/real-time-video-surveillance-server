# EG25-G GPS Agent

This Linux agent reads continuous NMEA data from the Quectel EG25-G and uploads
valid fixes to the surveillance API. It accepts both `$GPGGA`/`$GPRMC` and
`$GNGGA`/`$GNRMC` sentences.

## Before building

Confirm that the NMEA port outputs data and close minicom before starting the
agent:

```bash
sudo timeout 20 cat /dev/ttyUSB1
```

The service expects `/dev/ttyUSB1` for NMEA and `/dev/ttyUSB2` for AT commands
by default. Override both paths when the device enumerates differently.

## Build for the RK3588 vehicle computer

From the repository root on Windows PowerShell:

```powershell
$env:GOOS = "linux"
$env:GOARCH = "arm64"
go build -trimpath -o gps-agent/bin/gps-agent ./gps-agent
Remove-Item Env:GOOS
Remove-Item Env:GOARCH
```

Copy the binary, `gps-agent.service`, and a private environment file to the
vehicle computer. Never commit the populated file or a real API token.

## Manual test on the vehicle computer

```bash
set -a
. ./gps-agent.env
set +a
sudo -E ./gps-agent
```

Only one program can own the serial ports. Exit minicom before running the
agent. Successful uploads are logged with the converted decimal coordinates.
The agent uploads the latest valid fix according to `GPS_UPLOAD_INTERVAL`
(1 second by default). If the GPS serial device closes or disconnects, the
agent stays running and retries the connection every 5 seconds.

The web location page treats a truck as disconnected when the API has not
received a new GPS location for more than 30 seconds. It keeps the last known
position visible with a warning and clears that warning automatically after
uploads resume.

## Install as a systemd service

```bash
sudo install -m 0755 gps-agent /usr/local/bin/gps-agent
sudo install -m 0600 gps-agent.env /etc/gps-agent.env
sudo install -m 0644 gps-agent.service /etc/systemd/system/gps-agent.service
sudo systemctl daemon-reload
sudo systemctl enable --now gps-agent
sudo journalctl -u gps-agent -f
```

Alternatively, after building the binary, run `./install.sh`.

Always use a truck-specific device credential outside local testing.
