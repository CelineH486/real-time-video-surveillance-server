#!/usr/bin/env bash

set -euo pipefail

profile="${1:-}"
if [[ -z "${profile}" ]]; then
  echo "Usage: sudo $0 <NetworkManager-profile-name>" >&2
  exit 2
fi

if ! nmcli -g NAME connection show | grep -Fxq "${profile}"; then
  echo "NetworkManager profile not found: ${profile}" >&2
  exit 1
fi

nmcli connection modify "${profile}" \
  connection.autoconnect yes \
  connection.autoconnect-retries 0 \
  802-11-wireless.powersave 2

systemctl unmask NetworkManager-wait-online.service
systemctl enable NetworkManager-wait-online.service

echo "Configured automatic Wi-Fi reconnect for profile: ${profile}"
