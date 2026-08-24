# Wi-Fi boot configuration

Create and verify the Wi-Fi connection with NetworkManager before running the
configuration script. The profile must already exist, and its private
connection file must never be committed.

```bash
nmcli connection show
sudo ./scripts/configure-wifi.sh '<profile-name>'
```

The script enables automatic connection, retries forever, disables Wi-Fi power
saving, and enables `NetworkManager-wait-online.service`.

Verify after reboot:

```bash
nmcli device status
nmcli connection show --active
iw dev wlan0 get power_save
systemctl is-enabled NetworkManager-wait-online.service
```

The camera and GPS services both start after `network-online.target` and retry
their own connections if the server is temporarily unavailable.
