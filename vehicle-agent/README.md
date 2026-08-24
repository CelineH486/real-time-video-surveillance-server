# Vehicle Agent

Linux boot agents for the LPA3588 vehicle computer.

## Components

- `scripts/configure-wifi.sh` configures an existing NetworkManager profile to
  reconnect forever at boot and disables Wi-Fi power saving.
- `camera-agent` captures one configured AHD input, publishes main and sub RTSP
  streams, and retries after network or pipeline failures.
- `gps-agent` starts the Quectel EG25-G GNSS engine, reads valid NMEA fixes, and
  uploads vehicle locations to the surveillance API.

Both agents include systemd units with automatic restart. Private environment
files, Wi-Fi passwords, API tokens, publish passwords, and compiled binaries
must not be committed.

## Verified hardware

- Vehicle computer: LPA3588 / RK3588, Linux ARM64
- GNSS modem: Quectel EG25-G
- GNSS NMEA port: `/dev/ttyUSB1`
- GNSS AT port: `/dev/ttyUSB2`
- AHD1 through AHD8 were each verified with one camera. See
  `docs/ahd-device-mapping.md`.

## Validation

```bash
bash -n camera-agent/camera-agent camera-agent/camera-agent-lib.sh
bash camera-agent/test.sh
go test ./...
```
