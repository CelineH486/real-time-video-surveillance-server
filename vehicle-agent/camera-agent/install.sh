#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"

sudo install -m 0755 "${script_dir}/camera-agent" /usr/local/bin/camera-agent
sudo install -d -m 0755 /usr/local/lib/vehicle-agent
sudo install -m 0644 "${script_dir}/camera-agent-lib.sh" /usr/local/lib/vehicle-agent/camera-agent-lib.sh
sudo install -m 0644 "${script_dir}/camera-agent.service" /etc/systemd/system/camera-agent.service

if [[ ! -e /etc/camera-agent.env ]]; then
  sudo install -m 0600 "${script_dir}/camera-agent.env.example" /etc/camera-agent.env
  echo 'Created /etc/camera-agent.env; replace its placeholder values before starting.'
fi

sudo systemctl daemon-reload
sudo systemctl enable camera-agent.service

echo 'Camera Agent installed. Start it with: sudo systemctl restart camera-agent.service'
