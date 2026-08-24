#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
binary="${1:-${script_dir}/bin/gps-agent}"

if [[ ! -x "${binary}" ]]; then
  echo "GPS Agent binary not found or not executable: ${binary}" >&2
  echo "Build it first or pass its path as the first argument." >&2
  exit 1
fi

sudo install -m 0755 "${binary}" /usr/local/bin/gps-agent
sudo install -m 0644 "${script_dir}/gps-agent.service" /etc/systemd/system/gps-agent.service

if [[ ! -e /etc/gps-agent.env ]]; then
  sudo install -m 0600 "${script_dir}/gps-agent.env.example" /etc/gps-agent.env
  echo 'Created /etc/gps-agent.env; replace its placeholder values before starting.'
fi

sudo systemctl daemon-reload
sudo systemctl enable gps-agent.service

echo 'GPS Agent installed. Start it with: sudo systemctl restart gps-agent.service'
