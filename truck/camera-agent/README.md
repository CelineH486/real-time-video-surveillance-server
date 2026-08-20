# Linux AHD Camera Agent

This agent captures a Rockchip multi-planar V4L2 AHD input through GStreamer,
encodes main and sub H.264 streams with FFmpeg, and publishes them to MediaMTX.
It waits for the media server and retries the complete pipeline after failures.

## Requirements

- Bash, GStreamer 1.0, FFmpeg, `v4l2-ctl`, `timeout`, and `iw`
- A MediaMTX publish credential issued for the configured truck
- NetworkManager when using the optional Wi-Fi boot settings below

## Install

```bash
cd truck/camera-agent
./install.sh
sudo editor /etc/camera-agent.env
sudo systemctl restart camera-agent.service
```

Never commit `/etc/camera-agent.env`; it contains the publish password.

For a NetworkManager Wi-Fi profile named `CELINE`, enable indefinite boot
retries and disable Wi-Fi power saving with:

```bash
sudo nmcli connection modify CELINE \
  connection.autoconnect yes \
  connection.autoconnect-retries 0 \
  802-11-wireless.powersave 2
sudo systemctl unmask NetworkManager-wait-online.service
sudo systemctl enable NetworkManager-wait-online.service
```

Inspect the current boot with:

```bash
systemctl is-active camera-agent.service
sudo journalctl -b -u camera-agent.service --no-pager
```
